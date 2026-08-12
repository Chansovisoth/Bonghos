package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramHereConnectsCurrentTopicOnce(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Telegram commands", Provider: ProviderTelegram,
		Token: "123456789:telegram_command_secret", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Destinations) != 0 {
		t.Fatalf("new Telegram destinations = %+v, want none", config.Destinations)
	}
	updatesCalls := 0
	acknowledgements := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			updatesCalls++
			if updatesCalls > 1 {
				if r.URL.Query().Get("offset") != "78" {
					t.Errorf("second update offset = %q, want 78", r.URL.Query().Get("offset"))
				}
				_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":77,"message":{"message_id":9,"message_thread_id":42,"is_topic_message":true,"text":"/bonghos here","from":{"id":123},"chat":{"id":-100222,"type":"supergroup","title":"Projects","is_forum":true}}}]}`))
		case strings.HasSuffix(r.URL.Path, "/getChatMember"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"status":"administrator"}}`))
		case strings.HasSuffix(r.URL.Path, "/getChat"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":-100222,"type":"supergroup","title":"Projects","is_forum":true,"photo":{"small_file_id":"group-photo"}}}`))
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			acknowledgements++
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if payload["message_thread_id"] != float64(42) {
				t.Errorf("acknowledgement payload = %+v", payload)
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	dispatcher := &Dispatcher{Store: store, Sender: sender}
	dispatcher.pollTelegramCommands(context.Background())

	updated, err := store.ByID(config.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Destinations) != 1 || updated.Destinations[0].ID != "-100222" || updated.Destinations[0].ThreadID != 42 {
		t.Fatalf("connected destination = %+v", updated.Destinations)
	}
	if acknowledgements != 1 {
		t.Fatalf("acknowledgements = %d, want 1", acknowledgements)
	}
	dispatcher.pollTelegramCommands(context.Background())
	if acknowledgements != 1 {
		t.Fatalf("command replayed; acknowledgements = %d", acknowledgements)
	}
}

func TestTelegramHereRejectsNonAdministrator(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Telegram commands", Provider: ProviderTelegram,
		Token: "123456789:telegram_command_secret", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":12,"message":{"text":"/bonghos here","from":{"id":456},"chat":{"id":-100333,"type":"supergroup","title":"Private"}}}]}`))
		case strings.HasSuffix(r.URL.Path, "/getChatMember"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"status":"member"}}`))
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	(&Dispatcher{Store: store, Sender: sender}).pollTelegramCommands(context.Background())
	updated, err := store.ByID(config.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Destinations) != 0 {
		t.Fatalf("non-admin connected destinations = %+v", updated.Destinations)
	}
}

func TestTelegramCommandParsing(t *testing.T) {
	for input, want := range map[string]string{
		"/bonghos here": "here", "/BONGHOS@Bonghos_Bot HERE": "here",
		"/bonghos disconnect": "disconnect", "/bonghos@bonghos_bot where": "where",
		"/bonghos help": "help", "/bonghos": "help", "/bonghos unknown": "help",
		"/here": "", "ordinary message": "", "/unknown": "",
	} {
		if got := telegramCommand(input); got != want {
			t.Errorf("telegramCommand(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTelegramHelpListsNamespacedCommandsWithoutAdminCheck(t *testing.T) {
	var text string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Fatalf("unexpected Telegram help request: %s", r.URL.Path)
		}
		var payload struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		text = payload.Text
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer server.Close()
	sender := NewSender()
	sender.TelegramBaseURL = server.URL
	(&Dispatcher{Sender: sender}).handleTelegramCommand(context.Background(), TelegramCommandState{
		Token: "123456789:telegram_help_secret",
	}, &telegramMessage{Text: "/bonghos help", Chat: telegramChat{ID: -1001, Type: "supergroup"}})
	for _, command := range []string{"/bonghos here", "/bonghos where", "/bonghos disconnect"} {
		if !strings.Contains(text, command) {
			t.Errorf("help response %q does not contain %q", text, command)
		}
	}
}
