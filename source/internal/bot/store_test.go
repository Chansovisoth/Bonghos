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
