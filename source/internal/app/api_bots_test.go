package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/bot"
)

func TestBotAPIEncryptsTokenAndSupportsToggles(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	const token = "123456789:telegram_api_secret_token"
	var created bot.Config
	status, body := client.do(http.MethodPost, "/api/bots", map[string]any{
		"name": "Server alerts", "provider": "telegram", "token": token,
		"destination_id": "-1001234567890", "enabled": true,
		"notify_server_started": true, "notify_server_stopped": true,
		"notify_player_joined": true, "notify_player_left": true,
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create bot: %d %s", status, body)
	}
	if !created.TokenConfigured || strings.Contains(body, token) {
		t.Fatalf("token was not safely redacted: %s", body)
	}

	var encrypted []byte
	if err := env.app.DB.QueryRow(`SELECT token_enc FROM notification_bots WHERE id=?`, created.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if string(encrypted) == token || strings.Contains(string(encrypted), "api_secret") {
		t.Fatal("database stored plaintext bot token")
	}
	status, body = client.do(http.MethodPatch, "/api/bots/"+itoa(created.ID), map[string]any{
		"provider": "discord",
	}, nil)
	if status != http.StatusBadRequest || !strings.Contains(body, "cannot be changed") {
		t.Fatalf("change provider: %d %s", status, body)
	}

	var updated bot.Config
	status, body = client.do(http.MethodPatch, "/api/bots/"+itoa(created.ID), map[string]any{
		"enabled": false, "notify_player_joined": false,
	}, &updated)
	if status != http.StatusOK || updated.Enabled || updated.NotifyPlayerJoined {
		t.Fatalf("patch bot: %d %s %+v", status, body, updated)
	}

	var listed []*bot.Config
	status, body = client.do(http.MethodGet, "/api/bots", nil, &listed)
	if status != http.StatusOK || len(listed) != 1 || strings.Contains(body, token) {
		t.Fatalf("list bots: %d %s", status, body)
	}

	status, body = client.do(http.MethodDelete, "/api/bots/"+itoa(created.ID), nil, nil)
	if status != http.StatusOK {
		t.Fatalf("delete bot: %d %s", status, body)
	}
}

func TestBotAPIRequiresOwnerSecurityPermission(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("member", "correct horse battery", authorization.RoleMember)
	client := env.newClient()
	client.mustLogin("member", "correct horse battery", secret)
	if status, _ := client.do(http.MethodGet, "/api/bots", nil, nil); status != http.StatusForbidden {
		t.Fatalf("member GET /api/bots = %d, want 403", status)
	}
}

func TestBotNotificationsIncludeServerAndPlayerNames(t *testing.T) {
	env := newTestEnv(t)
	instance := env.newServerProject(t, "named-pack")
	instance.DisplayName = "Named Pack"
	if err := env.app.Instances.Update(instance); err != nil {
		t.Fatal(err)
	}
	if err := env.app.Instances.SetActive(instance.ID); err != nil {
		t.Fatal(err)
	}
	_, err := env.app.Bots.Create(bot.CreateInput{
		Name: "Telegram", Provider: bot.ProviderTelegram,
		Token: "123456789:telegram_event_secret", DestinationID: "-1001234567890",
		Enabled: true, NotifyServerStarted: true, NotifyServerStopped: true,
		NotifyPlayerJoined: true, NotifyPlayerLeft: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	messages := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		messages <- payload.Text
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	env.app.BotNotify.Sender.TelegramBaseURL = server.URL

	env.app.handleConsoleLine(`[12:00:00] [Server thread/INFO]: Done (1.25s)! For help, type "help"`, true)
	env.app.handleConsoleLine(`[12:01:00] [Server thread/INFO]: Steve joined the game`, true)
	env.app.handleConsoleLine(`[12:02:00] [Server thread/INFO]: Steve left the game`, true)
	env.app.observeBotLifecycle("stopped")

	var received []string
	deadline := time.After(3 * time.Second)
	for len(received) < 4 {
		select {
		case message := <-messages:
			received = append(received, message)
		case <-deadline:
			t.Fatalf("received %d notifications: %v", len(received), received)
		}
	}
	joined := strings.Join(received, "\n")
	for _, expected := range []string{"Server started", "Named Pack", "Steve joined", "Steve left", "Server stopped"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("notifications missing %q: %v", expected, received)
		}
	}
}
