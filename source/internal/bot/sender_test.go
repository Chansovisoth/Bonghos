package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
		Provider: ProviderTelegram, Token: "123456789:telegram_token_secret", DestinationID: "-1001234567890",
	}, "Server ready")
	if err != nil {
		t.Fatal(err)
	}
	if received["chat_id"] != "-1001234567890" || received["text"] != "Server ready" {
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

func toJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
