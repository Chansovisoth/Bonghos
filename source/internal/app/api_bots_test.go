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
	status, body = client.do(http.MethodPost, "/api/bots/"+itoa(created.ID)+"/test", map[string]any{}, nil)
	if status != http.StatusConflict || !strings.Contains(body, "disabled") {
		t.Fatalf("test disabled bot: %d %s", status, body)
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

func TestBotAPIEmptyListReturnsJSONArray(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	var listed []*bot.Config
	status, body := client.do(http.MethodGet, "/api/bots", nil, &listed)
	if status != http.StatusOK {
		t.Fatalf("list empty bots: %d %s", status, body)
	}
	if strings.TrimSpace(body) != "[]" {
		t.Fatalf("empty bot list JSON = %q, want []", body)
	}
	if listed == nil || len(listed) != 0 {
		t.Fatalf("decoded empty bot list = %#v, want non-nil empty slice", listed)
	}
}

func TestTelegramBotCanBeCreatedBeforeHereCommand(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	var created bot.Config
	status, body := client.do(http.MethodPost, "/api/bots", map[string]any{
		"name": "Telegram commands", "provider": "telegram",
		"token": "123456789:telegram_command_secret",
	}, &created)
	if status != http.StatusCreated || len(created.Destinations) != 0 || created.DestinationID != "" {
		t.Fatalf("create destination-free Telegram bot: %d %s %+v", status, body, created)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"bonghos_test_bot"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
	}))
	defer provider.Close()
	env.app.BotNotify.Sender.TelegramBaseURL = provider.URL
	var discovery bot.TelegramDiscovery
	status, body = client.do(http.MethodPost, "/api/bots/"+itoa(created.ID)+"/telegram/discover", map[string]any{}, &discovery)
	if status != http.StatusOK || discovery.BotUsername != "bonghos_test_bot" {
		t.Fatalf("discover destination-free Telegram bot: %d %s %+v", status, body, discovery)
	}
}

func TestDiscordBotCanBeCreatedBeforeSlashCommand(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	var created bot.Config
	status, body := client.do(http.MethodPost, "/api/bots", map[string]any{
		"name": "Discord commands", "provider": "discord",
		"token": "discord_bot_token_for_command_setup",
	}, &created)
	if status != http.StatusCreated || len(created.Destinations) != 0 || created.DestinationID != "" {
		t.Fatalf("create destination-free Discord bot: %d %s %+v", status, body, created)
	}
}

func TestBotInviteEndpointResolvesProviderLinkServerSide(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"bonghos_test_bot"}}`))
	}))
	defer provider.Close()
	env.app.BotNotify.Sender.TelegramBaseURL = provider.URL
	const token = "123456789:telegram_invite_secret"
	var created bot.Config
	status, body := client.do(http.MethodPost, "/api/bots", map[string]any{
		"name": "Invite bot", "provider": "telegram", "token": token,
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create bot: %d %s", status, body)
	}
	var invite map[string]string
	status, body = client.do(http.MethodGet, "/api/bots/"+itoa(created.ID)+"/invite", nil, &invite)
	if status != http.StatusOK || invite["url"] != "https://t.me/bonghos_test_bot?startgroup&admin=manage_chat" || strings.Contains(body, token) {
		t.Fatalf("invite response: %d %s", status, body)
	}
}

func TestBotAPIRejectsRolesWithoutBotPermission(t *testing.T) {
	for _, role := range []authorization.Role{
		authorization.RoleMember,
		authorization.RoleViewer,
	} {
		t.Run(string(role), func(t *testing.T) {
			env := newTestEnv(t)
			username := string(role)
			secret := env.createUser(username, "correct horse battery", role)
			client := env.newClient()
			client.mustLogin(username, "correct horse battery", secret)
			if status, _ := client.do(http.MethodGet, "/api/bots", nil, nil); status != http.StatusForbidden {
				t.Fatalf("%s GET /api/bots = %d, want 403", role, status)
			}
			if status, _ := client.do(http.MethodPost, "/api/bots/telegram/discover", map[string]any{"token": "not-used"}, nil); status != http.StatusForbidden {
				t.Fatalf("%s POST /api/bots/telegram/discover = %d, want 403", role, status)
			}
			if status, _ := client.do(http.MethodGet, "/api/bots/1/telegram/destinations/-1001/photo", nil, nil); status != http.StatusForbidden {
				t.Fatalf("%s GET Telegram group photo = %d, want 403", role, status)
			}
			if status, _ := client.do(http.MethodGet, "/api/bots/1/invite", nil, nil); status != http.StatusForbidden {
				t.Fatalf("%s GET bot invite = %d, want 403", role, status)
			}
		})
	}
}

func TestAdminCanAddEditListAndRemoveBots(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("admin", "correct horse battery", authorization.RoleAdmin)
	client := env.newClient()
	client.mustLogin("admin", "correct horse battery", secret)

	const token = "123456789:admin_bot_management_secret"
	var created bot.Config
	status, body := client.do(http.MethodPost, "/api/bots", map[string]any{
		"name": "Admin alerts", "provider": "telegram", "token": token,
	}, &created)
	if status != http.StatusCreated || created.Name != "Admin alerts" || strings.Contains(body, token) {
		t.Fatalf("admin create bot: %d %s %+v", status, body, created)
	}

	var updated bot.Config
	status, body = client.do(http.MethodPatch, "/api/bots/"+itoa(created.ID), map[string]any{
		"name": "Admin edited alerts", "enabled": false,
	}, &updated)
	if status != http.StatusOK || updated.Name != "Admin edited alerts" || updated.Enabled {
		t.Fatalf("admin edit bot: %d %s %+v", status, body, updated)
	}

	var listed []*bot.Config
	status, body = client.do(http.MethodGet, "/api/bots", nil, &listed)
	if status != http.StatusOK || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("admin list bots: %d %s %+v", status, body, listed)
	}

	status, body = client.do(http.MethodDelete, "/api/bots/"+itoa(created.ID), nil, nil)
	if status != http.StatusOK {
		t.Fatalf("admin remove bot: %d %s", status, body)
	}
}

func TestTelegramBotDiscoveryAndMultipleDestinations(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	sent := 0
	updatesCalls := 0
	photo := []byte("\x89PNG\r\n\x1a\nprofile")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"bonghos_test_bot"}}`))
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			updatesCalls++
			if updatesCalls == 1 {
				_, _ = w.Write([]byte(`{"ok":true,"result":[
					{"message":{"message_id":42,"message_thread_id":42,"is_topic_message":true,"chat":{"id":-1001111111111,"type":"supergroup","title":"Projects","is_forum":true},"forum_topic_created":{"name":"Alerts"}}},
					{"message":{"message_id":43,"message_thread_id":42,"is_topic_message":true,"chat":{"id":-1001111111111,"type":"supergroup","title":"Projects","is_forum":true},"reply_to_message":{"forum_topic_created":{"name":"Alerts"}}}},
					{"message":{"message_id":45,"message_thread_id":45,"is_topic_message":true,"chat":{"id":-1001111111111,"type":"supergroup","title":"Projects","is_forum":true},"reply_to_message":{"forum_topic_created":{"name":"Alerts"}}}},
					{"my_chat_member":{"chat":{"id":-1002222222222,"type":"group","title":"Staff"}}}
				]}`))
			} else {
				_, _ = w.Write([]byte(`{"ok":true,"result":[
					{"message":{"message_id":44,"message_thread_id":42,"is_topic_message":true,"chat":{"id":-1001111111111,"type":"supergroup","title":"Projects","is_forum":true},"reply_to_message":{"forum_topic_created":{"name":"Alerts"}}}}
				]}`))
			}
		case strings.HasSuffix(r.URL.Path, "/getChat"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":-1001111111111,"type":"supergroup","title":"Projects","photo":{"small_file_id":"group-photo"}}}`))
		case strings.HasSuffix(r.URL.Path, "/getFile"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"file_path":"photos/group.png"}}`))
		case strings.HasPrefix(r.URL.Path, "/file/bot"):
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(photo)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			sent++
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	env.app.BotNotify.Sender.TelegramBaseURL = server.URL
	env.app.BotNotify.Sender.TelegramFileBaseURL = server.URL

	const token = "123456789:telegram_discovery_secret"
	var discovery bot.TelegramDiscovery
	status, body := client.do(http.MethodPost, "/api/bots/telegram/discover", map[string]any{"token": token}, &discovery)
	if status != http.StatusOK || len(discovery.Groups) != 2 || strings.Contains(body, token) {
		t.Fatalf("discover groups: %d %s %+v", status, body, discovery)
	}

	var created bot.Config
	status, body = client.do(http.MethodPost, "/api/bots", map[string]any{
		"name": "Telegram groups", "provider": "telegram", "token": token,
		"destinations":            discovery.Groups[:1],
		"discovered_destinations": discovery.Groups,
	}, &created)
	if status != http.StatusCreated || len(created.Destinations) != 1 || len(created.DiscoveredDestinations) != 2 || strings.Contains(body, token) {
		t.Fatalf("create multi-group bot: %d %s %+v", status, body, created)
	}
	for _, group := range created.DiscoveredDestinations {
		if group.DiscoveredAt == "" {
			t.Fatalf("created discovery has no stable timestamp: %+v", created.DiscoveredDestinations)
		}
	}
	var refreshed bot.TelegramDiscovery
	status, body = client.do(http.MethodPost, "/api/bots/"+itoa(created.ID)+"/telegram/discover", map[string]any{}, &refreshed)
	if status != http.StatusOK || len(refreshed.Groups) != 2 {
		t.Fatalf("persisted discovery refresh: %d %s %+v", status, body, refreshed)
	}
	for _, group := range refreshed.Groups {
		if group.ID == "-1001111111111" && len(group.Topics) != 1 {
			t.Fatalf("duplicate topic was persisted: %+v", group.Topics)
		}
	}
	if created.Destinations[0].PhotoFileID == "" {
		t.Fatalf("saved group photo metadata = %+v", created.Destinations)
	}
	// Older saved bots have no photo metadata. The image endpoint resolves the
	// current group photo directly so they do not need to be re-saved first.
	if _, err := env.app.DB.Exec(`UPDATE notification_bot_destinations SET photo_file_id='' WHERE bot_id=?`, created.ID); err != nil {
		t.Fatal(err)
	}
	status, body = client.do(http.MethodGet, "/api/bots/"+itoa(created.ID)+"/telegram/destinations/"+created.Destinations[0].ID+"/photo", nil, nil)
	if status != http.StatusOK || body != strings.TrimSpace(string(photo)) || strings.Contains(body, token) {
		t.Fatalf("group photo: %d %q", status, body)
	}

	status, body = client.do(http.MethodPost, "/api/bots/"+itoa(created.ID)+"/test", map[string]any{}, nil)
	if status != http.StatusOK || sent != 1 {
		t.Fatalf("test multi-group bot: %d %s, sent=%d", status, body, sent)
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
	messages := make(chan string, 5)
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
	env.app.observeBotLifecycle("stopping")
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
	for _, expected := range []string{"Server started", "Named Pack", "Steve joined", "Steve left", "Server stopping", "shutting down"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("notifications missing %q: %v", expected, received)
		}
	}
	select {
	case duplicate := <-messages:
		t.Fatalf("duplicate stop notification after process exit: %q", duplicate)
	case <-time.After(150 * time.Millisecond):
	}
}
