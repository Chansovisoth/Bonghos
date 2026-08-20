package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/playit"
)

func managedTunnelJSON(id, address string, port int) map[string]any {
	return map[string]any{
		"id": id, "name": playit.ManagedTunnelName, "tunnel_type": "minecraft-java", "display_address": address,
		"agent_config": map[string]any{"fields": []map[string]string{
			{"name": "local_ip", "value": "127.0.0.1"},
			{"name": "local_port", "value": strconv.Itoa(port)},
		}},
	}
}

func TestPlayitBootRetriesUntilDaemonIsAvailable(t *testing.T) {
	env := newTestEnv(t)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	var available atomic.Bool
	env.app.PlayitDaemonAvailable = available.Load
	env.app.PlayitBootRetryInterval = 5 * time.Millisecond
	reconciled := make(chan playit.Config, 1)
	env.app.PlayitServiceReconcile = func(config playit.Config) string {
		reconciled <- config
		return ""
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "stopped", Error: "test stop"}
	}
	ctx, cancel := context.WithCancel(context.Background())
	env.app.playitParent = ctx
	defer cancel()
	go env.app.bootPlayit(ctx)

	select {
	case <-reconciled:
		t.Fatal("Playit service was reconciled before playitd became available")
	case <-time.After(20 * time.Millisecond):
	}
	available.Store(true)
	select {
	case config := <-reconciled:
		if !config.Enabled || config.ManagementMode != playit.ManagementBonghos {
			t.Fatalf("startup reconciliation config = %+v", config)
		}
	case <-time.After(time.Second):
		t.Fatal("Playit startup did not retry after playitd became available")
	}
}

func TestPlayitSettingsAreOwnerOnlyAndExistingInstallsDefaultToManual(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	adminSecret := env.createUser("admin", "correct horse battery", authorization.RoleAdmin)
	viewerSecret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)

	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	admin := env.newClient()
	admin.mustLogin("admin", "correct horse battery", adminSecret)
	viewer := env.newClient()
	viewer.mustLogin("viewer", "correct horse battery", viewerSecret)

	for role, client := range map[string]*client{"admin": admin, "viewer": viewer} {
		if status, _ := client.do(http.MethodGet, "/api/playit", nil, nil); status != http.StatusForbidden {
			t.Fatalf("%s read Playit settings: status %d", role, status)
		}
	}

	var settings map[string]any
	status, body := owner.do(http.MethodGet, "/api/playit", nil, &settings)
	if status != http.StatusOK {
		t.Fatalf("owner Playit settings: %d %s", status, body)
	}
	if settings["enabled"] != false || settings["management_mode"] != playit.ManagementNone {
		t.Fatalf("existing-install default changed networking: %s", body)
	}

	status, body = owner.do(http.MethodPut, "/api/playit", map[string]any{
		"enabled": true, "account_mode": "account", "management_mode": "external",
		"public_address": "example.gl.joinmc.link:25565", "local_port": 25565,
	}, &settings)
	if status != http.StatusOK || settings["management_mode"] != playit.ManagementExternal {
		t.Fatalf("save external Playit settings: %d %s", status, body)
	}

	var overview map[string]any
	status, body = owner.do(http.MethodGet, "/api/overview", nil, &overview)
	if status != http.StatusOK || overview["playit_address"] != "example.gl.joinmc.link:25565" {
		t.Fatalf("overview Playit address: %d %s", status, body)
	}

	status, body = owner.do(http.MethodPut, "/api/playit", map[string]any{
		"enabled": false, "account_mode": "account", "management_mode": "external",
		"public_address": "disabled.gl.joinmc.link:25565", "local_port": 25566,
	}, &settings)
	if status != http.StatusOK || settings["enabled"] != false || settings["public_address"] != "disabled.gl.joinmc.link:25565" {
		t.Fatalf("save disabled external Playit settings: %d %s", status, body)
	}
}

func TestPlayitClaimStoresSecretEncryptedAndNeverReturnsIt(t *testing.T) {
	env := newTestEnv(t)
	env.app.PlayitDaemonAvailable = func() bool { return true }
	daemon := filepath.Join(env.home, "system", "bin", "playitd")
	if err := os.WriteFile(daemon, []byte("test daemon"), 0o755); err != nil {
		t.Fatal(err)
	}
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)

	const agentSecret = "private-playit-agent-secret"
	var setupCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/claim/setup":
			state := "WaitingForUserVisit"
			if setupCalls.Add(1) > 1 {
				state = "UserAccepted"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": state})
		case "/claim/exchange":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"secret_key": agentSecret}})
		case "/v1/agents/rundata":
			if r.Header.Get("Authorization") != "Agent-Key "+agentSecret {
				t.Error("rundata did not receive the linked agent credential")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": []any{},
				"permissions": map[string]string{"account_status": "verified"},
			}})
		case "/login/guest":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"session_key": "guest-session"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client(), AgentVersion: "1.0.10"}

	var started map[string]any
	status, body := owner.do(http.MethodPost, "/api/playit/claim", map[string]string{"account_mode": "account"}, &started)
	if status != http.StatusOK || started["claim_pending"] != true {
		t.Fatalf("start claim: %d %s", status, body)
	}
	claimURL, _ := started["claim_url"].(string)
	if !strings.HasPrefix(claimURL, "https://playit.gg/claim/") {
		t.Fatalf("unsafe claim URL: %q", claimURL)
	}

	var completed map[string]any
	status, body = owner.do(http.MethodPost, "/api/playit/claim/poll", map[string]any{}, &completed)
	if status != http.StatusOK || completed["state"] != "complete" {
		t.Fatalf("complete claim: %d %s", status, body)
	}
	if strings.Contains(body, agentSecret) || strings.Contains(body, "agent_secret") {
		t.Fatalf("claim response exposed the credential: %s", body)
	}
	var encrypted []byte
	if err := env.app.DB.QueryRow(`SELECT agent_secret_enc FROM playit_settings WHERE id=1`).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encrypted), agentSecret) {
		t.Fatal("Playit credential was stored as plaintext")
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.Enabled || !config.SecretConfigured {
		t.Fatalf("linking while disabled changed activation state: %+v, %v", config, err)
	}
}

func TestPlayitClaimRequiresInstalledDaemon(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	status, body := owner.do(http.MethodPost, "/api/playit/claim", map[string]string{"account_mode": "account"}, nil)
	if status != http.StatusConflict || !strings.Contains(body, "install the official Playit agent") {
		t.Fatalf("claim without daemon: %d %s", status, body)
	}
}

func TestPlayitTunnelRequiresDaemonReportedReady(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "has_invalid_secret", Error: "The Playit agent rejected its credential"}
	}
	status, body := owner.do(http.MethodPost, "/api/playit/tunnel", map[string]any{}, nil)
	if status != http.StatusConflict || !strings.Contains(body, "rejected its credential") {
		t.Fatalf("tunnel with unready agent: %d %s", status, body)
	}
}

func TestPlayitTunnelExplainsIncompatibleAgentRegistration(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "playit-incompatible")
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id", Version: "1.0.10"}
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/agents/rundata":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": []any{}, "permissions": map[string]string{"account_status": "verified"},
			}})
		case "/tunnels/create":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "fail", "data": "TunnelTypeNotSupported"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	status, body := owner.do(http.MethodPost, "/api/playit/tunnel", map[string]any{}, nil)
	if status != http.StatusConflict || !strings.Contains(body, "relink the agent") {
		t.Fatalf("incompatible tunnel registration: %d %s", status, body)
	}
}

func TestPlayitRefreshAdoptsUniqueRemoteTunnelAndTracksPendingActivation(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id"}
	}
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pendingTunnel := managedTunnelJSON("remote-tunnel", "", 25565)
		pendingTunnel["name"] = "Created in Playit"
		data := map[string]any{
			"agent_id": "agent-id", "tunnels": []any{},
			"pending":     []any{pendingTunnel},
			"permissions": map[string]string{"account_status": "verified"},
		}
		if calls.Add(1) > 1 {
			activeTunnel := managedTunnelJSON("remote-tunnel", "public.example:25565", 25565)
			activeTunnel["name"] = "Created in Playit"
			data["tunnels"] = []any{activeTunnel}
			data["pending"] = []any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	var payload map[string]any
	status, body := owner.do(http.MethodGet, "/api/playit", nil, &payload)
	if status != http.StatusOK || payload["tunnel_id"] != "remote-tunnel" || payload["public_address"] != nil {
		t.Fatalf("adopt pending remote tunnel: %d %s", status, body)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "remote-tunnel" || config.PublicAddress != "" {
		t.Fatalf("pending tunnel cache = %+v, %v", config, err)
	}

	payload = nil
	status, body = owner.do(http.MethodPost, "/api/playit/refresh", map[string]any{}, &payload)
	if status != http.StatusOK || payload["public_address"] != "public.example:25565" {
		t.Fatalf("activate pending remote tunnel: %d %s", status, body)
	}
	config, err = env.app.Playit.Config()
	if err != nil || config.PublicAddress != "public.example:25565" {
		t.Fatalf("active tunnel cache = %+v, %v", config, err)
	}
}

func TestPlayitTunnelCreateRecoversRemoteTunnelAfterIncompatibleResponse(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "playit-create-recovery")
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id"}
	}
	var runDataCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/agents/rundata":
			data := map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": []any{},
				"permissions": map[string]string{"account_status": "verified"},
			}
			if runDataCalls.Add(1) > 1 {
				data["pending"] = []any{managedTunnelJSON("recovered-tunnel", "", 25565)}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
		case "/tunnels/create":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": "unexpected-response-shape"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	var payload map[string]any
	status, body := owner.do(http.MethodPost, "/api/playit/tunnel", map[string]any{}, &payload)
	if status != http.StatusOK || payload["tunnel_id"] != "recovered-tunnel" {
		t.Fatalf("recover created tunnel: %d %s", status, body)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "recovered-tunnel" {
		t.Fatalf("recovered tunnel cache = %+v, %v", config, err)
	}
}

func TestDiscoverManagedPlayitTunnelIsConservative(t *testing.T) {
	matching := playit.TunnelData{ID: "matching", Name: playit.ManagedTunnelName, TunnelType: "minecraft-java"}
	unrelated := playit.TunnelData{ID: "unrelated", Name: "Another app", TunnelType: "minecraft-java"}
	if remote, found, ambiguous := discoverManagedPlayitTunnel(playit.RunData{Tunnels: []playit.TunnelData{unrelated, matching}}, 25565, nil); !found || ambiguous || remote.data.ID != "matching" {
		t.Fatalf("unique tunnel discovery = %+v, found=%v ambiguous=%v", remote, found, ambiguous)
	}
	second := matching
	second.ID = "second"
	if _, found, ambiguous := discoverManagedPlayitTunnel(playit.RunData{Tunnels: []playit.TunnelData{matching, second}}, 25565, nil); found || !ambiguous {
		t.Fatalf("duplicate tunnel discovery found=%v ambiguous=%v", found, ambiguous)
	}
	if _, found, ambiguous := discoverManagedPlayitTunnel(playit.RunData{Tunnels: []playit.TunnelData{unrelated}}, 25565, nil); found || ambiguous {
		t.Fatalf("unrelated tunnel discovery found=%v ambiguous=%v", found, ambiguous)
	}
	var manual playit.TunnelData
	raw, err := json.Marshal(managedTunnelJSON("manual", "manual.example:25565", 25565))
	if err != nil || json.Unmarshal(raw, &manual) != nil {
		t.Fatal("could not prepare manual Playit tunnel fixture")
	}
	manual.Name = "My manually created server"
	if remote, found, ambiguous := discoverManagedPlayitTunnel(playit.RunData{Tunnels: []playit.TunnelData{manual}}, 25565, nil); !found || ambiguous || remote.data.ID != "manual" {
		t.Fatalf("manual matching tunnel discovery = %+v, found=%v ambiguous=%v", remote, found, ambiguous)
	}
	manual.AgentConfig.Fields[1].Value = "25566"
	if _, found, ambiguous := discoverManagedPlayitTunnel(playit.RunData{Tunnels: []playit.TunnelData{manual}}, 25565, nil); found || ambiguous {
		t.Fatalf("wrong-port tunnel discovery found=%v ambiguous=%v", found, ambiguous)
	}
}

func TestPlayitAgentCanBeRenamedWhileDisabled(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if err := env.app.Playit.SaveAgent("agent-id"); err != nil {
		t.Fatal(err)
	}
	var renameCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/agents/rename" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Agent-Key agent-secret" {
			t.Error("rename did not use the linked agent credential")
		}
		renameCalls.Add(1)
		var request map[string]string
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request["agent_id"] != "agent-id" || request["name"] != "Bonghos home" {
			t.Errorf("rename request = %+v", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": nil})
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client(), AgentVersion: "1.0.10"}

	var payload map[string]any
	status, body := owner.do(http.MethodPut, "/api/playit/agent", map[string]string{"name": " Bonghos home "}, &payload)
	if status != http.StatusOK || payload["agent_name"] != "Bonghos home" || payload["enabled"] != false {
		t.Fatalf("rename disabled agent: %d %s", status, body)
	}
	status, body = owner.do(http.MethodPut, "/api/playit/agent", map[string]string{"name": "  "}, nil)
	if status != http.StatusBadRequest || renameCalls.Load() != 1 {
		t.Fatalf("invalid rename reached Playit: %d calls=%d %s", status, renameCalls.Load(), body)
	}
}

func TestPlayitAgentRenameProviderFailureIsConflict(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if err := env.app.Playit.SaveAgent("missing-agent"); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "fail", "data": "AgentNotFound"})
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	status, body := owner.do(http.MethodPut, "/api/playit/agent", map[string]string{"name": "Bonghos home"}, nil)
	if status != http.StatusConflict || !strings.Contains(body, "could not be found") {
		t.Fatalf("rename missing agent: %d %s", status, body)
	}
}

func TestPlayitRefreshHidesInactiveRemoteTunnelAddresses(t *testing.T) {
	tests := []struct {
		name       string
		tunnels    []any
		pending    []any
		wantStatus string
	}{
		{name: "missing", wantStatus: "missing"},
		{name: "disabled", tunnels: []any{map[string]any{
			"id": "managed-tunnel", "display_address": "disabled.example:25565", "disabled_reason": "account disabled",
		}}, wantStatus: "account disabled"},
		{name: "pending", pending: []any{map[string]any{
			"id": "managed-tunnel", "display_address": "pending.example:25565", "status_msg": "tunnel type not supported",
		}}, wantStatus: "tunnel type not supported"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newTestEnv(t)
			ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
			owner := env.newClient()
			owner.mustLogin("owner", "correct horse battery", ownerSecret)
			if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
				t.Fatal(err)
			}
			if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
				t.Fatal(err)
			}
			if err := env.app.Playit.SaveTunnel("managed-tunnel", "stale.example:25565", 25565); err != nil {
				t.Fatal(err)
			}
			env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
				return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id"}
			}
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
					"agent_id": "agent-id", "tunnels": test.tunnels, "pending": test.pending,
					"permissions": map[string]string{"account_status": "verified"},
				}})
			}))
			defer provider.Close()
			env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

			var payload map[string]any
			status, body := owner.do(http.MethodGet, "/api/playit", nil, &payload)
			if _, hasAddress := payload["public_address"]; status != http.StatusOK || hasAddress || payload["tunnel_status"] != test.wantStatus {
				t.Fatalf("refresh inactive tunnel: %d %s", status, body)
			}
			config, err := env.app.Playit.Config()
			if err != nil || config.TunnelID != "managed-tunnel" || config.PublicAddress != "" {
				t.Fatalf("inactive tunnel cache = %+v, %v", config, err)
			}
			var overview map[string]any
			status, body = owner.do(http.MethodGet, "/api/overview", nil, &overview)
			if _, hasAddress := overview["playit_address"]; status != http.StatusOK || hasAddress {
				t.Fatalf("overview advertised inactive tunnel: %d %s", status, body)
			}
		})
	}
}

func TestPlayitUpdateRecreatesMissingRemoteTunnel(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "playit-recreate")
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	if err := env.app.Playit.SaveTunnel("missing-tunnel", "stale.example:25565", 25565); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id"}
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/agents/rundata":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": []any{},
				"permissions": map[string]string{"account_status": "verified"},
			}})
		case "/v1/tunnels/config":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "fail", "data": "TunnelNotFound"})
		case "/tunnels/create":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"id": "replacement-tunnel"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	var payload map[string]any
	status, body := owner.do(http.MethodPost, "/api/playit/tunnel", map[string]any{}, &payload)
	if status != http.StatusOK || payload["tunnel_id"] != "replacement-tunnel" {
		t.Fatalf("recreate missing tunnel: %d %s", status, body)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "replacement-tunnel" {
		t.Fatalf("replacement tunnel cache = %+v, %v", config, err)
	}
}

func TestPlayitTunnelCreateAndMissingRemoteDelete(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "playit-test")
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id", Version: "1.0.10"}
	}
	var updateCalls atomic.Int32
	var tunnelCreated atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/agents/rundata":
			tunnels := []any{}
			if tunnelCreated.Load() {
				tunnels = []any{managedTunnelJSON("tunnel-id", "public.example:25565", 25565)}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": tunnels, "pending": []any{}, "permissions": map[string]string{"account_status": "verified"},
			}})
		case "/tunnels/create":
			tunnelCreated.Store(true)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"id": "tunnel-id"}})
		case "/v1/tunnels/config":
			updateCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": nil})
		case "/tunnels/delete":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "fail", "data": "TunnelNotFound"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	var payload map[string]any
	status, body := owner.do(http.MethodPost, "/api/playit/tunnel", map[string]any{}, &payload)
	if status != http.StatusOK || payload["tunnel_id"] != "tunnel-id" || payload["public_address"] != "public.example:25565" {
		t.Fatalf("create tunnel: %d %s", status, body)
	}
	updated, err := env.app.syncPlayitTunnelPort(context.Background())
	if err != nil || updated || updateCalls.Load() != 0 {
		t.Fatalf("idempotent tunnel sync = %v, calls=%d, err=%v", updated, updateCalls.Load(), err)
	}
	payload = nil
	status, body = owner.do(http.MethodDelete, "/api/playit/tunnel", nil, &payload)
	_, hasTunnelID := payload["tunnel_id"]
	if status != http.StatusOK || hasTunnelID {
		t.Fatalf("delete missing remote tunnel: %d %s", status, body)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "" || config.PublicAddress != "" {
		t.Fatalf("local tunnel state was not cleared: %+v, %v", config, err)
	}
}

func TestPlayitAutomaticSyncAdoptsExistingRemoteTunnel(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "playit-auto-adopt")
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeAccount, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id"}
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
			"agent_id": "agent-id",
			"tunnels":  []any{managedTunnelJSON("startup-tunnel", "startup.example:25565", 25565)},
			"pending":  []any{}, "permissions": map[string]string{"account_status": "verified"},
		}})
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	updated, err := env.app.syncPlayitTunnelPort(context.Background())
	if err != nil || !updated {
		t.Fatalf("automatic adoption updated=%v err=%v", updated, err)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "startup-tunnel" || config.PublicAddress != "startup.example:25565" {
		t.Fatalf("automatically adopted tunnel = %+v, %v", config, err)
	}
}

func TestPlayitAutomaticSyncCreatesMissingTunnel(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "playit-auto-create")
	if _, err := env.app.Playit.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.Playit.SetPreference(true, playit.AccountModeGuest, playit.ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id"}
	}
	var createCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/agents/rundata":
			pending := []any{}
			if createCalls.Load() > 0 {
				pending = []any{managedTunnelJSON("automatic-tunnel", "", 25565)}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": pending,
				"permissions": map[string]string{"account_status": "guest"},
			}})
		case "/tunnels/create":
			createCalls.Add(1)
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			if request["tunnel_type"] != "minecraft-java" || request["port_type"] != "tcp" {
				t.Errorf("automatic create request = %+v", request)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"id": "automatic-tunnel"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

	updated, err := env.app.syncPlayitTunnelPort(context.Background())
	if err != nil || !updated || createCalls.Load() != 1 {
		t.Fatalf("automatic create updated=%v calls=%d err=%v", updated, createCalls.Load(), err)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "automatic-tunnel" || config.LocalPort != 25565 {
		t.Fatalf("automatically created tunnel = %+v, %v", config, err)
	}
}

func TestPlayitRelinkDeletesExistingRemoteTunnel(t *testing.T) {
	env := newTestEnv(t)
	env.app.PlayitDaemonAvailable = func() bool { return true }
	daemon := filepath.Join(env.home, "system", "bin", "playitd")
	if err := os.WriteFile(daemon, []byte("test daemon"), 0o755); err != nil {
		t.Fatal(err)
	}
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	if _, err := env.app.Playit.CompleteClaim("old-secret", 0); err != nil {
		t.Fatal(err)
	}
	if err := env.app.Playit.SaveTunnel("old-tunnel", "old.example", 25565); err != nil {
		t.Fatal(err)
	}
	var setupCalls atomic.Int32
	var deleteBeforeExchange atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/claim/setup":
			state := "WaitingForUserVisit"
			if setupCalls.Add(1) > 1 {
				state = "UserAccepted"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": state})
		case "/tunnels/delete":
			if r.Header.Get("Authorization") != "Agent-Key old-secret" {
				t.Error("old tunnel was not deleted with its original credential")
			}
			deleteBeforeExchange.Store(true)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": nil})
		case "/claim/exchange":
			if !deleteBeforeExchange.Load() {
				t.Error("claim was exchanged before the old tunnel was removed")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"secret_key": "new-secret"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client(), AgentVersion: "1.0.10"}
	if status, body := owner.do(http.MethodPost, "/api/playit/claim", map[string]string{"account_mode": "account"}, nil); status != http.StatusOK {
		t.Fatalf("start relink: %d %s", status, body)
	}
	var completed map[string]any
	if status, body := owner.do(http.MethodPost, "/api/playit/claim/poll", map[string]any{}, &completed); status != http.StatusOK || completed["state"] != "complete" {
		t.Fatalf("complete relink: %d %s", status, body)
	}
	config, err := env.app.Playit.Config()
	if err != nil || config.TunnelID != "" {
		t.Fatalf("relink retained old tunnel: %+v, %v", config, err)
	}
	secret, err := env.app.Playit.Secret()
	if err != nil || secret != "new-secret" {
		t.Fatalf("relink secret = %q, %v", secret, err)
	}
}
