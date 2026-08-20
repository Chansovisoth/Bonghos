package playit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientClaimAndAuthenticatedRunData(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/claim/setup":
			if r.Header.Get("Authorization") != "" {
				t.Error("claim setup unexpectedly sent authorization")
			}
			var request map[string]string
			_ = json.NewDecoder(r.Body).Decode(&request)
			if request["version"] != "playit 1.0.10" {
				t.Errorf("claim version = %q", request["version"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": "UserAccepted"})
		case "/claim/exchange":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"secret_key": "agent-secret"}})
		case "/v1/agents/rundata":
			if got := r.Header.Get("Authorization"); got != "Agent-Key agent-secret" {
				t.Errorf("Authorization = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{
				"agent_id": "agent-id", "tunnels": []any{}, "pending": []any{},
				"permissions": map[string]string{"account_status": "verified"},
			}})
		case "/login/guest":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"session_key": "guest-session"}})
		case "/agents/rename":
			if got := r.Header.Get("Authorization"); got != "Agent-Key agent-secret" {
				t.Errorf("rename Authorization = %q", got)
			}
			var request map[string]string
			_ = json.NewDecoder(r.Body).Decode(&request)
			if request["agent_id"] != "agent-id" || request["name"] != "Home server" {
				t.Errorf("rename request = %+v", request)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": nil})
		case "/v1/tunnels/create":
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			origin, _ := request["origin"].(map[string]any)
			if origin["type"] != "agent" {
				t.Errorf("create origin = %+v", origin)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"id": "tunnel-id"}})
		case "/v1/tunnels/config":
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			if request["tunnel_id"] != "tunnel-id" {
				t.Errorf("update tunnel id = %+v", request["tunnel_id"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": nil})
		case "/tunnels/delete":
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			if request["tunnel_id"] != "tunnel-id" {
				t.Errorf("delete tunnel id = %+v", request["tunnel_id"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": nil})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTP: server.Client(), AgentVersion: "1.0.10"}
	state, err := client.ClaimSetup(context.Background(), "0123456789")
	if err != nil || state != "UserAccepted" {
		t.Fatalf("ClaimSetup = %q, %v", state, err)
	}
	secret, err := client.ClaimExchange(context.Background(), "0123456789")
	if err != nil || secret != "agent-secret" {
		t.Fatalf("ClaimExchange = %q, %v", secret, err)
	}
	runData, err := client.RunData(context.Background(), secret)
	if err != nil || runData.AgentID != "agent-id" || runData.Permissions.AccountStatus != "verified" {
		t.Fatalf("RunData = %+v, %v", runData, err)
	}
	guestURL, err := client.GuestLogin(context.Background(), secret)
	if err != nil || guestURL != "https://playit.gg/login/guest-account/guest-session" {
		t.Fatalf("GuestLogin = %q, %v", guestURL, err)
	}
	if err := client.RenameAgent(context.Background(), secret, "agent-id", " Home server "); err != nil {
		t.Fatalf("RenameAgent: %v", err)
	}
	tunnelID, err := client.CreateMinecraftTunnel(context.Background(), secret, "agent-id", 25566)
	if err != nil || tunnelID != "tunnel-id" {
		t.Fatalf("CreateMinecraftTunnel = %q, %v", tunnelID, err)
	}
	if err := client.UpdateTunnelPort(context.Background(), secret, tunnelID, 25567); err != nil {
		t.Fatalf("UpdateTunnelPort: %v", err)
	}
	if err := client.DeleteTunnel(context.Background(), secret, tunnelID); err != nil {
		t.Fatalf("DeleteTunnel: %v", err)
	}
	if strings.Join(paths, ",") != "/claim/setup,/claim/exchange,/v1/agents/rundata,/login/guest,/agents/rename,/v1/tunnels/create,/v1/tunnels/config,/tunnels/delete" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestClientMapsProviderErrorsWithoutRawResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "fail", "data": map[string]any{"error": "TunnelTypeNotSupported", "details": "private upstream detail"},
		})
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	_, err := client.CreateMinecraftTunnel(context.Background(), "secret", "agent", 25565)
	if !IsProviderError(err, "TunnelTypeNotSupported") {
		t.Fatalf("provider error = %T %v", err, err)
	}
	if strings.Contains(err.Error(), "private upstream detail") {
		t.Fatalf("raw provider detail escaped: %v", err)
	}
}

func TestClientMapsStructuredProviderMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "error", "data": map[string]any{"type": "auth", "message": "EmailMustBeVerified"},
		})
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	_, err := client.CreateMinecraftTunnel(context.Background(), "secret", "agent", 25565)
	if !IsProviderError(err, "EmailMustBeVerified") || !strings.Contains(err.Error(), "Verify the email address") {
		t.Fatalf("structured provider error = %T %v", err, err)
	}
}

func TestClientMapsStructuredProviderErrorOnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "error", "data": map[string]any{"type": "auth", "message": "InvalidAgentKey"},
		})
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	_, err := client.RunData(context.Background(), "expired-secret")
	if !IsProviderError(err, "InvalidAgentKey") || !strings.Contains(err.Error(), "relink the agent") {
		t.Fatalf("HTTP provider error = %T %v", err, err)
	}
}

func TestClientMapsNestedProviderErrors(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		code string
		want string
	}{
		{
			name: "path not found",
			data: map[string]any{"type": "path-not-found", "message": map[string]string{"path": "/missing"}},
			code: "PathNotFound",
			want: "update Bonghos",
		},
		{
			name: "internal",
			data: map[string]any{"type": "internal", "message": map[string]string{"trace_id": "private-trace"}},
			code: "Internal",
			want: "internal error",
		},
		{
			name: "validation",
			data: map[string]any{"type": "validation", "message": "a provider validation detail"},
			code: "Validation",
			want: "provider validation detail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatal(err)
			}
			err = providerError(raw)
			if !IsProviderError(err, tt.code) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("nested provider error = %T %v", err, err)
			}
			if tt.code == "Validation" && !strings.Contains(err.Error(), "provider validation detail") {
				t.Fatalf("validation detail was not preserved: %v", err)
			}
			if strings.Contains(err.Error(), "private-trace") || strings.Contains(err.Error(), "/missing") {
				t.Fatalf("provider detail escaped: %v", err)
			}
		})
	}
}

func TestClientSanitizesProviderValidationDetail(t *testing.T) {
	for _, message := range []string{
		"Authorization header Agent-Key private-value is invalid",
		"contains\na control character",
	} {
		raw, err := json.Marshal(map[string]any{"type": "validation", "message": message})
		if err != nil {
			t.Fatal(err)
		}
		err = providerError(raw)
		if err.Error() != "Playit rejected the request as invalid" {
			t.Fatalf("unsafe validation detail escaped: %v", err)
		}
	}
}

func TestClientDoesNotExposeSecretInHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream detail", http.StatusUnauthorized)
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	_, err := client.RunData(context.Background(), "very-private-secret")
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "very-private-secret") || strings.Contains(err.Error(), "upstream detail") {
		t.Fatalf("unsafe error: %v", err)
	}
}
