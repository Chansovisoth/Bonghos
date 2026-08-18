package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/playit"
)

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
}

func TestPlayitClaimStoresSecretEncryptedAndNeverReturnsIt(t *testing.T) {
	env := newTestEnv(t)
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
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}

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
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "has_invalid_secret", Error: "The Playit agent rejected its credential"}
	}
	status, body := owner.do(http.MethodPost, "/api/playit/tunnel", map[string]any{}, nil)
	if status != http.StatusConflict || !strings.Contains(body, "rejected its credential") {
		t.Fatalf("tunnel with unready agent: %d %s", status, body)
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
	env.app.PlayitStatus = func(context.Context) playit.AgentStatus {
		return playit.AgentStatus{Phase: "running", Running: true, AgentID: "agent-id", Version: "1.0.10"}
	}
	var updateCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/agents/rundata":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": []any{}, "permissions": map[string]string{"account_status": "verified"},
			}})
		case "/v1/tunnels/create":
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
	if status != http.StatusOK || payload["tunnel_id"] != "tunnel-id" {
		t.Fatalf("create tunnel: %d %s", status, body)
	}
	updated, err := env.app.syncPlayitTunnelPort(context.Background())
	if err != nil || !updated || updateCalls.Load() != 1 {
		t.Fatalf("automatic tunnel repair = %v, calls=%d, err=%v", updated, updateCalls.Load(), err)
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

func TestPlayitRelinkDeletesExistingRemoteTunnel(t *testing.T) {
	env := newTestEnv(t)
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
	env.app.PlayitAPI = &playit.Client{BaseURL: provider.URL, HTTP: provider.Client()}
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
