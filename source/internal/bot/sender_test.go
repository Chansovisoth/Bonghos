package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSenderInviteURLs(t *testing.T) {
	telegram := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"bonghos_test_bot"}}`))
	}))
	defer telegram.Close()
	sender := NewSender()
	sender.TelegramBaseURL = telegram.URL
	telegramURL, err := sender.InviteURL(context.Background(), ProviderTelegram, "123456789:telegram_invite_secret")
	if err != nil || telegramURL != "https://t.me/bonghos_test_bot?startgroup&admin=manage_chat" {
		t.Fatalf("Telegram invite = %q, %v", telegramURL, err)
	}

	discord := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"1536799744431755275","username":"Bonghos","bot":true}`))
	}))
	defer discord.Close()
	sender.DiscordBaseURL = discord.URL
	discordURL, err := sender.InviteURL(context.Background(), ProviderDiscord, "discord_bot_token_for_invite")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(discordURL, "client_id=1536799744431755275") || !strings.Contains(discordURL, "permissions=274877910016") || !strings.Contains(discordURL, "scope=bot+applications.commands") {
		t.Fatalf("Discord invite = %q", discordURL)
	}
}

func TestSenderTelegram(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123456789:telegram_token_secret/sendMessage" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	err := sender.Send(context.Background(), Target{
		Provider: ProviderTelegram, Token: "123456789:telegram_token_secret", DestinationID: "-1001234567890", ThreadID: 42,
	}, "Server ready")
	if err != nil {
		t.Fatal(err)
	}
	if received["chat_id"] != "-1001234567890" || received["text"] != "Server ready" || received["message_thread_id"] != float64(42) {
		t.Fatalf("payload = %+v", received)
	}
}

func TestSenderDiscord(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/channels/123456789012345678/messages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bot discord_bot_token_secret" {
			t.Errorf("authorization = %q", authorization)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	sender := NewSender()
	sender.DiscordBaseURL = server.URL
	err := sender.Send(context.Background(), Target{
		Provider: ProviderDiscord, Token: "discord_bot_token_secret", DestinationID: "123456789012345678",
	}, "Steve joined")
	if err != nil {
		t.Fatal(err)
	}
	if received["content"] != "Steve joined" {
		t.Fatalf("payload = %+v", received)
	}
	allowed, ok := received["allowed_mentions"].(map[string]any)
	if !ok || !strings.Contains(strings.TrimSpace(toJSON(allowed)), `"parse":[]`) {
		t.Fatalf("allowed_mentions = %+v", received["allowed_mentions"])
	}
}

func TestTelegramNetworkErrorDoesNotExposeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()

	const token = "123456789:telegram_token_must_stay_secret"
	sender := NewSender()
	sender.TelegramBaseURL = baseURL
	err := sender.Send(context.Background(), Target{
		Provider: ProviderTelegram, Token: token, DestinationID: "-1001234567890",
	}, "Server ready")
	if err == nil {
		t.Fatal("send unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("network error exposed token: %v", err)
	}
}

func TestDiscoverTelegramGroupsReturnsUniqueGroupChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"bonghos_test_bot"}}`))
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			_, _ = w.Write([]byte(`{"ok":true,"result":[
				{"message":{"chat":{"id":42,"type":"private","username":"owner"}}},
				{"message":{"chat":{"id":-100222,"type":"supergroup","title":"Projects"}}},
				{"message":{"message_id":101,"message_thread_id":99,"is_topic_message":true,"chat":{"id":-100222,"type":"supergroup","title":"Projects","is_forum":true},"reply_to_message":{"forum_topic_created":{"name":"Announcements"}}}},
				{"message":{"message_id":102,"message_thread_id":100,"is_topic_message":true,"chat":{"id":-100222,"type":"supergroup","title":"Projects","is_forum":true},"reply_to_message":{"forum_topic_created":{"name":"Announcements"}}}},
				{"my_chat_member":{"chat":{"id":-100111,"type":"group","title":"Alerts"}}},
				{"my_chat_member":{"chat":{"id":-100333,"type":"group","title":"Removed"},"new_chat_member":{"status":"kicked"}}},
				{"edited_message":{"chat":{"id":-100222,"type":"supergroup","title":"Projects"}}}
			]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	discovery, err := sender.DiscoverTelegramGroups(context.Background(), "123456789:telegram_discovery_secret")
	if err != nil {
		t.Fatal(err)
	}
	if discovery.BotUsername != "bonghos_test_bot" || len(discovery.Groups) != 2 {
		t.Fatalf("discovery = %+v", discovery)
	}
	if discovery.Groups[0].Name != "Alerts" || discovery.Groups[1].Name != "Projects" {
		t.Fatalf("groups are not sorted or deduplicated: %+v", discovery.Groups)
	}
	if !discovery.Groups[1].Forum || len(discovery.Groups[1].Topics) != 1 || discovery.Groups[1].Topics[0].Name != "Announcements" || discovery.Groups[1].Topics[0].ID != 100 {
		t.Fatalf("forum topics = %+v", discovery.Groups[1])
	}
}

func TestDiscoverTelegramGroupsIncludesSafeProfilePhoto(t *testing.T) {
	const token = "123456789:telegram_photo_secret"
	photo := []byte("\x89PNG\r\n\x1a\nprofile")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"bonghos_test_bot"}}`))
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			_, _ = w.Write([]byte(`{"ok":true,"result":[{"message":{"chat":{"id":-100111,"type":"group","title":"Alerts"}}}]}`))
		case strings.HasSuffix(r.URL.Path, "/getChat"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":-100111,"type":"group","title":"Alerts","photo":{"small_file_id":"photo-small"}}}`))
		case strings.HasSuffix(r.URL.Path, "/getFile"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"file_path":"photos/group.png"}}`))
		case strings.HasPrefix(r.URL.Path, "/file/bot"):
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(photo)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	sender.TelegramFileBaseURL = server.URL
	discovery, err := sender.DiscoverTelegramGroups(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Groups) != 1 || discovery.Groups[0].PhotoFileID != "photo-small" {
		t.Fatalf("discovery group = %+v", discovery.Groups)
	}
	if !strings.HasPrefix(discovery.Groups[0].PhotoDataURL, "data:image/png;base64,") {
		t.Fatalf("photo data URL = %q", discovery.Groups[0].PhotoDataURL)
	}
}

func TestDiscoverTelegramGroupsErrorDoesNotExposeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer server.Close()

	const token = "123456789:telegram_discovery_secret"
	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	_, err := sender.DiscoverTelegramGroups(context.Background(), token)
	if err == nil {
		t.Fatal("discovery unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("discovery error exposed token: %v", err)
	}
}

func TestDispatcherTestSendsToEveryTelegramDestination(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Telegram groups", Provider: ProviderTelegram,
		Token:   "123456789:telegram_multi_send_secret",
		Enabled: true,
		Destinations: []Destination{
			{ID: "-1001111111111", Name: "Projects"},
			{ID: "-1002222222222", Name: "Staff"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var chatIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ChatID string `json:"chat_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		chatIDs = append(chatIDs, payload.ChatID)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	dispatcher := &Dispatcher{Store: store, Sender: sender}
	if err := dispatcher.Test(context.Background(), config.ID); err != nil {
		t.Fatal(err)
	}
	if len(chatIDs) != 2 || chatIDs[0] != "-1001111111111" || chatIDs[1] != "-1002222222222" {
		t.Fatalf("test notification chat IDs = %v", chatIDs)
	}
}

func TestDispatcherTestRejectsDisabledBot(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Disabled Telegram", Provider: ProviderTelegram,
		Token:        "123456789:telegram_disabled_test_secret",
		Destinations: []Destination{{ID: "-1001111111111", Name: "Projects"}},
		Enabled:      false,
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &Dispatcher{Store: store, Sender: NewSender()}
	if err := dispatcher.Test(context.Background(), config.ID); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled bot test error = %v, want %v", err, ErrDisabled)
	}
}

func toJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
