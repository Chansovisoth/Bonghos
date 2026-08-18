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

	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
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
	if strings.Join(paths, ",") != "/claim/setup,/claim/exchange,/v1/agents/rundata,/login/guest,/v1/tunnels/create,/v1/tunnels/config,/tunnels/delete" {
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
