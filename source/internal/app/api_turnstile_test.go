package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestTurnstileSettingsAreOwnerOnlyAndEnforcedAtLogin(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	viewerSecret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
	owner := env.newClient()
	owner.mustLogin("owner", "correct horse battery", ownerSecret)
	viewer := env.newClient()
	viewer.mustLogin("viewer", "correct horse battery", viewerSecret)

	if status, _ := viewer.do(http.MethodGet, "/api/security/turnstile", nil, nil); status != http.StatusForbidden {
		t.Fatalf("Viewer read Turnstile settings: status %d", status)
	}

	const privateKey = "private-widget-secret"
	var saved map[string]any
	status, body := owner.do(http.MethodPut, "/api/security/turnstile", map[string]any{
		"enabled": true, "site_key": "public-widget-key", "secret_key": privateKey,
	}, &saved)
	if status != http.StatusOK {
		t.Fatalf("Owner could not configure Turnstile: %d %s", status, body)
	}
	if strings.Contains(body, privateKey) || saved["secret_key"] != nil || saved["secret_configured"] != true {
		t.Fatalf("settings response exposed or lost the secret: %s", body)
	}
	var encrypted []byte
	if err := env.app.DB.QueryRow(`SELECT secret_key_enc FROM turnstile_settings WHERE id=1`).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encrypted), privateKey) {
		t.Fatal("Turnstile secret was stored as plaintext")
	}

	verify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":  form.Get("response") == "valid-login-token",
			"hostname": "127.0.0.1",
			"action":   "login",
		})
	}))
	defer verify.Close()
	env.app.Turnstile.VerifyURL = verify.URL

	public := env.newClient()
	var publicConfig map[string]any
	if status, body := public.do(http.MethodGet, "/api/auth/turnstile", nil, &publicConfig); status != http.StatusOK {
		t.Fatalf("public config failed: %d %s", status, body)
	}
	if publicConfig["enabled"] != true || publicConfig["site_key"] != "public-widget-key" || strings.Contains(mustJSON(publicConfig), privateKey) {
		t.Fatalf("unsafe public config: %+v", publicConfig)
	}

	code, err := auth.TOTPCode(ownerSecret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	credentials := map[string]string{
		"username": "owner", "password": "correct horse battery", "code": code,
	}
	if status, _ := public.do(http.MethodPost, "/api/auth/login", credentials, nil); status != http.StatusForbidden {
		t.Fatalf("login without Turnstile token returned %d", status)
	}
	credentials["turnstile_token"] = "invalid-login-token"
	if status, _ := public.do(http.MethodPost, "/api/auth/login", credentials, nil); status != http.StatusForbidden {
		t.Fatalf("login with rejected Turnstile token returned %d", status)
	}
	credentials["turnstile_token"] = "valid-login-token"
	if status, body := public.do(http.MethodPost, "/api/auth/login", credentials, nil); status != http.StatusOK {
		t.Fatalf("login with valid Turnstile token failed: %d %s", status, body)
	}
}

func mustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
