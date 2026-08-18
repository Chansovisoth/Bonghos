package turnstile_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/Chansovisoth/Bonghos/internal/turnstile"
)

func newStore(t *testing.T) *turnstile.Store {
	t.Helper()
	db, err := database.Open(t.TempDir() + "/turnstile.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &turnstile.Store{DB: db, SecretKey: []byte("0123456789abcdef0123456789abcdef")}
}

func TestStoreEncryptsSecretAndNeverReturnsIt(t *testing.T) {
	store := newStore(t)
	secret := "0x-secret-widget-key"
	config, err := store.Update(turnstile.Update{
		Enabled: true, SiteKey: "1x-public-site-key", SecretKey: &secret,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !config.Enabled || !config.SecretConfigured || config.SiteKey != "1x-public-site-key" {
		t.Fatalf("saved config = %+v", config)
	}
	var encrypted []byte
	if err := store.DB.QueryRow(`SELECT secret_key_enc FROM turnstile_settings WHERE id=1`).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if len(encrypted) == 0 || strings.Contains(string(encrypted), secret) {
		t.Fatalf("secret was not encrypted at rest: %q", encrypted)
	}
	public, err := store.PublicConfig()
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(public)
	if !public.Enabled || strings.Contains(string(encoded), secret) {
		t.Fatalf("unsafe public config: %s", encoded)
	}
}

func TestStoreRetainsConfiguredSecretWhenUpdateOmitsIt(t *testing.T) {
	store := newStore(t)
	secret := "existing-secret"
	if _, err := store.Update(turnstile.Update{SiteKey: "site-one", SecretKey: &secret}); err != nil {
		t.Fatal(err)
	}
	config, err := store.Update(turnstile.Update{Enabled: true, SiteKey: "site-two"})
	if err != nil {
		t.Fatal(err)
	}
	if !config.Enabled || !config.SecretConfigured || config.SiteKey != "site-two" {
		t.Fatalf("updated config = %+v", config)
	}
}

func TestVerifyLoginRequiresSingleValidHostBoundToken(t *testing.T) {
	store := newStore(t)
	secret := "server-secret"
	if _, err := store.Update(turnstile.Update{Enabled: true, SiteKey: "public-site", SecretKey: &secret}); err != nil {
		t.Fatal(err)
	}
	var received url.Values
	verify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received, _ = url.ParseQuery(string(body))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":  received.Get("response") == "valid-token",
			"hostname": "panel.example.com",
			"action":   "login",
		})
	}))
	defer verify.Close()
	service := &turnstile.Service{Store: store, VerifyURL: verify.URL}

	if err := service.VerifyLogin(context.Background(), "", "panel.example.com"); err != turnstile.ErrRequired {
		t.Fatalf("missing token error = %v", err)
	}
	if err := service.VerifyLogin(context.Background(), "invalid", "panel.example.com"); err != turnstile.ErrRejected {
		t.Fatalf("invalid token error = %v", err)
	}
	if err := service.VerifyLogin(context.Background(), "valid-token", "other.example.com"); err != turnstile.ErrRejected {
		t.Fatalf("wrong-host token error = %v", err)
	}
	if err := service.VerifyLogin(context.Background(), "valid-token", "panel.example.com:443"); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if received.Get("secret") != secret {
		t.Fatalf("Siteverify received secret %q", received.Get("secret"))
	}
}
