package bot

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/database"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "bonghos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &Store{DB: db, SecretKey: []byte("01234567890123456789012345678901")}
}

func TestStoreEncryptsAndRedactsTokens(t *testing.T) {
	store := testStore(t)
	const token = "123456789:telegram_test_token_secret"
	config, err := store.Create(CreateInput{
		Name: "Alerts", Provider: ProviderTelegram, Token: token, DestinationID: "-1001234567890",
		Enabled: true, NotifyServerStarted: true, NotifyPlayerJoined: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !config.TokenConfigured {
		t.Fatal("token should be reported as configured")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), token) || strings.Contains(string(encoded), "token_secret") {
		t.Fatalf("public config exposed token: %s", encoded)
	}
	var encrypted []byte
	if err := store.DB.QueryRow(`SELECT token_enc FROM notification_bots WHERE id=?`, config.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if string(encrypted) == token || strings.Contains(string(encrypted), "token_secret") {
		t.Fatal("database contains plaintext bot token")
	}
	target, err := store.Target(config.ID)
	if err != nil {
		t.Fatal(err)
	}
	if target.Token != token {
		t.Fatalf("decrypted token = %q", target.Token)
	}
}

func TestStoreListEmptyReturnsNonNilSlice(t *testing.T) {
	configs, err := testStore(t).List()
	if err != nil {
		t.Fatal(err)
	}
	if configs == nil || len(configs) != 0 {
		t.Fatalf("empty list = %#v, want non-nil empty slice", configs)
	}
}

func TestStorePatchAndEventFiltering(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Discord staff", Provider: ProviderDiscord,
		Token: "discord_bot_token_that_is_long_enough", DestinationID: "123456789012345678",
		Enabled: true, NotifyServerStarted: true, NotifyServerStopped: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := false
	stopped := true
	updated, err := store.Patch(config.ID, Patch{Enabled: &enabled, NotifyServerStopped: &stopped})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled || !updated.NotifyServerStopped {
		t.Fatalf("updated config = %+v", updated)
	}
	targets, err := store.TargetsFor(EventServerStopped)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 0 {
		t.Fatalf("disabled bot returned %d targets", len(targets))
	}
	enabled = true
	if _, err := store.Patch(config.ID, Patch{Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	targets, err = store.TargetsFor(EventServerStopped)
	if err != nil || len(targets) != 1 {
		t.Fatalf("targets=%+v err=%v", targets, err)
	}
	if err := store.Delete(config.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ByID(config.ID); err != ErrNotFound {
		t.Fatalf("ByID after delete = %v", err)
	}
}

func TestStoreRejectsProviderChange(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Alerts", Provider: ProviderTelegram,
		Token: "123456789:telegram_test_token_secret", DestinationID: "-1001234567890",
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	discord := ProviderDiscord
	if _, err := store.Patch(config.ID, Patch{Provider: &discord}); err == nil || !strings.Contains(err.Error(), "cannot be changed") {
		t.Fatalf("provider change error = %v", err)
	}
}

func TestStoreTelegramSupportsThreeDestinations(t *testing.T) {
	store := testStore(t)
	destinations := []Destination{
		{ID: "-1001111111111", Name: "Alpha", Type: "supergroup", ThreadID: 42, ThreadName: "Announcements"},
		{ID: "-1002222222222", Name: "Beta", Type: "supergroup"},
		{ID: "-1003333333333", Name: "Gamma", Type: "group"},
	}
	config, err := store.Create(CreateInput{
		Name: "Telegram groups", Provider: ProviderTelegram,
		Token: "123456789:telegram_multi_group_secret", Destinations: destinations,
		Enabled: true, NotifyServerStarted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Destinations) != 3 || config.DestinationID != destinations[0].ID {
		t.Fatalf("config destinations = %+v, legacy destination = %q", config.Destinations, config.DestinationID)
	}
	targets, err := store.TargetsFor(EventServerStarted)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 3 {
		t.Fatalf("event targets = %d, want 3", len(targets))
	}
	for index, target := range targets {
		if target.DestinationID != destinations[index].ID {
			t.Fatalf("target %d destination = %q, want %q", index, target.DestinationID, destinations[index].ID)
		}
	}
	if targets[0].ThreadID != 42 {
		t.Fatalf("first target thread = %d, want 42", targets[0].ThreadID)
	}

	four := append(append([]Destination{}, destinations...), Destination{ID: "-1004444444444", Name: "Delta"})
	if _, err := store.Patch(config.ID, Patch{Destinations: &four}); err == nil || !strings.Contains(err.Error(), "at most 3") {
		t.Fatalf("four-destination patch error = %v", err)
	}
}

func TestStorePersistsTelegramPhotoFileIDWithoutInlinePhoto(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Telegram photos", Provider: ProviderTelegram,
		Token: "123456789:telegram_photo_store_secret",
		Destinations: []Destination{{
			ID: "-1001111111111", Name: "Projects", Type: "supergroup",
			PhotoFileID: "telegram-small-photo", PhotoDataURL: "data:image/png;base64,should-not-persist",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Destinations) != 1 || config.Destinations[0].PhotoFileID != "telegram-small-photo" {
		t.Fatalf("saved destinations = %+v", config.Destinations)
	}
	if config.Destinations[0].PhotoDataURL != "" {
		t.Fatal("inline discovery photo was persisted")
	}
	target, fileID, err := store.TelegramPhotoTarget(config.ID, "-1001111111111")
	if err != nil {
		t.Fatal(err)
	}
	if target.Token != "123456789:telegram_photo_store_secret" || fileID != "telegram-small-photo" {
		t.Fatalf("photo target = %+v, file ID = %q", target, fileID)
	}
}

func TestStoreRetainsDiscoveredTelegramGroupsAndDeduplicatesTopics(t *testing.T) {
	store := testStore(t)
	config, err := store.Create(CreateInput{
		Name: "Telegram discovery", Provider: ProviderTelegram,
		Token:        "123456789:telegram_discovery_store_secret",
		Destinations: []Destination{{ID: "-1001111111111", Name: "Selected"}},
		DiscoveredDestinations: []Destination{
			{ID: "-1001111111111", Name: "Selected", Forum: true, Topics: []Topic{{ID: 42, Name: "Alerts"}, {ID: 43, Name: "Alerts"}}},
			{ID: "-1002222222222", Name: "Not selected"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Destinations) != 1 || len(config.DiscoveredDestinations) != 2 {
		t.Fatalf("selected=%+v discovered=%+v", config.Destinations, config.DiscoveredDestinations)
	}
	var topicCount int
	for _, destination := range config.DiscoveredDestinations {
		topicCount += len(destination.Topics)
	}
	if topicCount != 1 {
		t.Fatalf("deduplicated discoveries = %+v", config.DiscoveredDestinations)
	}
	for _, destination := range config.DiscoveredDestinations {
		if len(destination.Topics) == 1 && destination.Topics[0].ID != 43 {
			t.Fatalf("newest same-named topic was not retained: %+v", destination.Topics)
		}
	}
	merged, err := store.MergeDiscovered(config.ID, []Destination{
		{ID: "-1001111111111", Name: "Selected", Forum: true, Topics: []Topic{{ID: 42, Name: "Alerts"}}},
		{ID: "-1003333333333", Name: "New group"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 3 {
		t.Fatalf("merged discoveries = %+v", merged)
	}
}

func TestStoreLimitsBotsToOnePerProviderAndTwoTotal(t *testing.T) {
	store := testStore(t)
	if _, err := store.Create(CreateInput{
		Name: "Telegram", Provider: ProviderTelegram,
		Token: "123456789:telegram_limit_secret", DestinationID: "-1001111111111",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(CreateInput{
		Name: "Telegram duplicate", Provider: ProviderTelegram,
		Token: "123456789:telegram_limit_second", DestinationID: "-1002222222222",
	}); err == nil || !strings.Contains(err.Error(), "only one Telegram") {
		t.Fatalf("duplicate Telegram error = %v", err)
	}
	if _, err := store.Create(CreateInput{
		Name: "Discord", Provider: ProviderDiscord,
		Token: "discord_bot_token_for_limit_test", DestinationID: "123456789012345678",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(CreateInput{
		Name: "Third bot", Provider: ProviderDiscord,
		Token: "discord_bot_token_for_third_test", DestinationID: "223456789012345678",
	}); err == nil {
		t.Fatal("third notification bot was accepted")
	}
}

func TestStoreDiscordStillAllowsOneDestination(t *testing.T) {
	store := testStore(t)
	_, err := store.Create(CreateInput{
		Name: "Discord", Provider: ProviderDiscord,
		Token: "discord_bot_token_that_is_long_enough",
		Destinations: []Destination{
			{ID: "123456789012345678", Name: "One"},
			{ID: "223456789012345678", Name: "Two"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "at most 1") {
		t.Fatalf("two-destination Discord create error = %v", err)
	}
}
