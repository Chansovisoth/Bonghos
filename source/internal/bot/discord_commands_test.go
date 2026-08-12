package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func discordTestInteraction(command, permissions string) discordInteraction {
	var interaction discordInteraction
	interaction.ID = "323456789012345678"
	interaction.Token = "interaction-token"
	interaction.Type = 2
	interaction.GuildID = "123456789012345678"
	interaction.ChannelID = "223456789012345678"
	interaction.Member = &struct {
		Permissions string `json:"permissions"`
	}{Permissions: permissions}
	interaction.Channel = &struct {
		Name string `json:"name"`
	}{Name: "alerts"}
	interaction.Data.Name = "bonghos"
	interaction.Data.Options = append(interaction.Data.Options, struct {
		Type int    `json:"type"`
		Name string `json:"name"`
	}{Type: 1, Name: command})
	return interaction
}

func TestDiscordInteractionCommandsEnforceAdministrator(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Discord commands", Provider: ProviderDiscord,
		Token: "discord_bot_token_for_interactions",
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := &Dispatcher{Store: store, Sender: NewSender()}
	state := DiscordCommandState{BotID: config.ID, BotName: config.Name}
	denied := dispatcher.handleDiscordInteraction(state, discordTestInteraction("here", "0"))
	if denied == nil || !strings.Contains(denied.Content, "administrator") {
		t.Fatalf("non-admin reply = %+v", denied)
	}
	deniedWhere := dispatcher.handleDiscordInteraction(state, discordTestInteraction("where", "0"))
	if deniedWhere == nil || !strings.Contains(deniedWhere.Content, "administrator") {
		t.Fatalf("non-admin where reply = %+v", deniedWhere)
	}
	connected := dispatcher.handleDiscordInteraction(state, discordTestInteraction("here", "8"))
	if connected == nil || !strings.Contains(connected.Content, "will be sent") {
		t.Fatalf("admin reply = %+v", connected)
	}
	destination, err := store.DiscordDestination(config.ID, "223456789012345678")
	if err != nil || destination.Name != "alerts" {
		t.Fatalf("connected destination = %+v, %v", destination, err)
	}
	where := dispatcher.handleDiscordInteraction(state, discordTestInteraction("where", "32"))
	if where == nil || !strings.Contains(where.Content, "configured") {
		t.Fatalf("where reply = %+v", where)
	}
	disconnected := dispatcher.handleDiscordInteraction(state, discordTestInteraction("disconnect", "32"))
	if disconnected == nil || !strings.Contains(disconnected.Content, "disconnected") {
		t.Fatalf("disconnect reply = %+v", disconnected)
	}
}

func TestDiscordApplicationRegistrationAndInteractionResponse(t *testing.T) {
	const token = "discord_bot_token_for_rest_requests"
	registrations := 0
	callbacks := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/@me":
			if r.Header.Get("Authorization") != "Bot "+token {
				t.Errorf("identity authorization = %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"id":"1536799744431755275","username":"Bonghos","bot":true}`))
		case strings.HasSuffix(r.URL.Path, "/commands"):
			registrations++
			var definitions []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&definitions); err != nil {
				t.Error(err)
			}
			if len(definitions) != 1 || definitions[0]["name"] != "bonghos" || definitions[0]["default_member_permissions"] != "32" {
				t.Errorf("command definitions = %+v", definitions)
			}
			_, _ = w.Write([]byte(`[]`))
		case strings.Contains(r.URL.Path, "/interactions/"):
			callbacks++
			if r.Header.Get("Authorization") != "" {
				t.Error("interaction callback unexpectedly included bot authorization")
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	sender := NewSender()
	sender.DiscordBaseURL = server.URL
	application, err := sender.resolveDiscordApplication(context.Background(), token)
	if err != nil || application.ID != "1536799744431755275" {
		t.Fatalf("application = %+v, %v", application, err)
	}
	if err := sender.registerDiscordCommands(context.Background(), token, application.ID, "123456789012345678"); err != nil {
		t.Fatal(err)
	}
	interaction := discordTestInteraction("help", "0")
	if err := sender.respondDiscordInteraction(context.Background(), interaction, discordReply{Content: "help", Flags: 64}); err != nil {
		t.Fatal(err)
	}
	if registrations != 1 || callbacks != 1 {
		t.Fatalf("registrations=%d callbacks=%d", registrations, callbacks)
	}
}

func TestDiscordGatewayDispatchesInteraction(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Discord gateway", Provider: ProviderDiscord,
		Token: "discord_bot_token_for_gateway_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	callback := make(chan struct{}, 1)
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/commands") {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		if strings.Contains(r.URL.Path, "/interactions/") {
			callback <- struct{}{}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer rest.Close()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	identified := make(chan struct{}, 1)
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{"op": 10, "d": map[string]any{"heartbeat_interval": 45000}})
		var identify map[string]any
		if err := conn.ReadJSON(&identify); err != nil {
			t.Error(err)
			return
		}
		data, _ := identify["d"].(map[string]any)
		if data["intents"] != float64(discordGuildsIntent) {
			t.Errorf("Discord identify intents = %v, want %d", data["intents"], discordGuildsIntent)
		}
		identified <- struct{}{}
		_ = conn.WriteJSON(map[string]any{"op": 0, "t": "READY", "s": 1, "d": map[string]any{
			"guilds": []map[string]string{{"id": "123456789012345678"}},
		}})
		_ = conn.WriteJSON(map[string]any{"op": 0, "t": "GUILD_CREATE", "s": 2, "d": map[string]any{
			"id": "123456789012345678", "name": "Bonghos Lab", "icon": "guild-icon-hash",
		}})
		interaction := discordTestInteraction("help", "8")
		_ = conn.WriteJSON(map[string]any{"op": 0, "t": "INTERACTION_CREATE", "s": 3, "d": interaction})
		<-time.After(500 * time.Millisecond)
	}))
	defer gateway.Close()

	sender := NewSender()
	sender.DiscordBaseURL = rest.URL
	sender.DiscordGatewayURL = "ws" + strings.TrimPrefix(gateway.URL, "http")
	dispatcher := &Dispatcher{Store: store, Sender: sender}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = dispatcher.runDiscordGateway(ctx, DiscordCommandState{
			BotID: config.ID, BotName: config.Name, Token: "discord_bot_token_for_gateway_test",
		}, "1536799744431755275")
		close(done)
	}()
	select {
	case <-identified:
	case <-time.After(2 * time.Second):
		t.Fatal("gateway did not identify")
	}
	select {
	case <-callback:
	case <-time.After(2 * time.Second):
		t.Fatal("interaction callback was not sent")
	}
	observed, err := store.ByID(config.ID)
	if err != nil || len(observed.Destinations) != 0 || len(observed.DiscoveredDestinations) != 1 || observed.DiscoveredDestinations[0].GuildName != "Bonghos Lab" {
		t.Fatalf("gateway server discovery = %+v, %v", observed.DiscoveredDestinations, err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("gateway did not stop")
	}
}
