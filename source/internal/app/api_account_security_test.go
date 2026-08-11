package app

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func accountReauth(t *testing.T, c *client, purpose, password, secret string) string {
	t.Helper()
	var out struct {
		ActionToken string `json:"action_token"`
	}
	status, body := c.do(http.MethodPost, "/api/account/reauth/password", map[string]string{
		"purpose": purpose, "password": password, "code": mustTOTP(t, secret),
	}, &out)
	if status != http.StatusOK || out.ActionToken == "" {
		t.Fatalf("account reauthentication = %d %s", status, body)
	}
	return out.ActionToken
}

func TestAccountSecurityPasswordAndTOTPChanges(t *testing.T) {
	env := newTestEnv(t)
	oldSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	primary := env.newClient()
	primary.mustLogin("owner", "correct horse battery", oldSecret)
	other := env.newClient()
	other.mustLogin("owner", "correct horse battery", oldSecret)

	var initialCodes []auth.RecoveryCode
	status, body := primary.do(http.MethodGet, "/api/account/recovery-codes", nil, &initialCodes)
	if status != http.StatusOK || len(initialCodes) != 8 {
		t.Fatalf("initial recovery metadata = %d %s %+v", status, body, initialCodes)
	}
	if strings.Contains(body, "code_hash") || strings.Contains(body, "correct horse") {
		t.Fatalf("recovery metadata exposed sensitive material: %s", body)
	}

	// Grants are bound to the exact browser session, not merely the account.
	grant := accountReauth(t, primary, "change_password", "correct horse battery", oldSecret)
	if status, _ := other.do(http.MethodPost, "/api/account/password", map[string]string{
		"action_token": grant, "new_password": "new correct horse battery",
	}, nil); status != http.StatusForbidden {
		t.Fatalf("another session used an action grant: got %d, want 403", status)
	}

	grant = accountReauth(t, primary, "change_password", "correct horse battery", oldSecret)
	status, body = primary.do(http.MethodPost, "/api/account/password", map[string]string{
		"action_token": grant, "new_password": "new correct horse battery",
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("password change = %d %s", status, body)
	}
	if status, _ := primary.do(http.MethodPost, "/api/account/password", map[string]string{
		"action_token": grant, "new_password": "another correct horse battery",
	}, nil); status != http.StatusForbidden {
		t.Fatalf("action grant reuse = %d, want 403", status)
	}
	if status, _ := primary.do(http.MethodGet, "/api/auth/me", nil, nil); status != http.StatusOK {
		t.Fatalf("current session was revoked after password change: %d", status)
	}
	if status, _ := other.do(http.MethodGet, "/api/auth/me", nil, nil); status != http.StatusUnauthorized {
		t.Fatalf("other session survived password change: %d", status)
	}

	grant = accountReauth(t, primary, "change_totp", "new correct horse battery", oldSecret)
	var setup struct {
		SetupToken string `json:"setup_token"`
		Secret     string `json:"secret"`
		URI        string `json:"uri"`
	}
	status, body = primary.do(http.MethodPost, "/api/account/totp/begin", map[string]string{
		"action_token": grant,
	}, &setup)
	if status != http.StatusOK || setup.SetupToken == "" || setup.Secret == "" || setup.URI == "" {
		t.Fatalf("TOTP setup = %d %s %+v", status, body, setup)
	}
	if status, _ := primary.do(http.MethodPost, "/api/account/totp/finish", map[string]string{
		"setup_token": setup.SetupToken, "code": "123",
	}, nil); status != http.StatusBadRequest {
		t.Fatalf("invalid new TOTP = %d, want 400", status)
	}
	newCode, err := auth.TOTPCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var changed struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	status, body = primary.do(http.MethodPost, "/api/account/totp/finish", map[string]string{
		"setup_token": setup.SetupToken, "code": newCode,
	}, &changed)
	if status != http.StatusOK || len(changed.RecoveryCodes) != 8 {
		t.Fatalf("TOTP confirmation = %d %s %+v", status, body, changed)
	}
	if status, _ := primary.do(http.MethodGet, "/api/auth/me", nil, nil); status != http.StatusOK {
		t.Fatalf("current session was revoked after TOTP change: %d", status)
	}

	oldLogin := env.newClient()
	if status, _ := oldLogin.login("owner", "new correct horse battery", oldSecret); status == http.StatusOK {
		t.Fatal("old authenticator still signed in after replacement")
	}
	newLogin := env.newClient()
	newLogin.mustLogin("owner", "new correct horse battery", setup.Secret)
}

func TestRecoveryCodeManagementRequiresFreshVerification(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	if status, _ := c.do(http.MethodPost, "/api/account/recovery-codes/regenerate", map[string]string{
		"action_token": "missing",
	}, nil); status != http.StatusForbidden {
		t.Fatalf("regenerate without verification = %d, want 403", status)
	}
	grant := accountReauth(t, c, "regenerate_recovery_codes", "correct horse battery", secret)
	var regenerated struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	status, body := c.do(http.MethodPost, "/api/account/recovery-codes/regenerate", map[string]string{
		"action_token": grant,
	}, &regenerated)
	if status != http.StatusOK || len(regenerated.RecoveryCodes) != 8 {
		t.Fatalf("regenerate recovery codes = %d %s", status, body)
	}

	var metadata []auth.RecoveryCode
	status, body = c.do(http.MethodGet, "/api/account/recovery-codes", nil, &metadata)
	if status != http.StatusOK || len(metadata) != 8 || metadata[0].CreatedAt.IsZero() {
		t.Fatalf("recovery metadata = %d %s %+v", status, body, metadata)
	}
}
