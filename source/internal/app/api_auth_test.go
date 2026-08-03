package app

import (
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestLoginSucceedsWithValidCredentials(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)

	c := env.newClient()
	var out struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	status, body := c.do("POST", "/api/auth/login", map[string]string{
		"username": "owner", "password": "correct horse battery",
		"code": mustTOTP(t, secret),
	}, &out)

	if status != 200 {
		t.Fatalf("login failed: %d %s", status, body)
	}
	if out.Username != "owner" || out.Role != string(authorization.RoleOwner) {
		t.Errorf("unexpected identity: %+v", out)
	}
}

// The login response must be identical whether the username exists, the
// password is wrong, or the TOTP code is wrong. Anything else lets an attacker
// enumerate accounts.
func TestLoginDoesNotRevealWhetherAccountExists(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)

	cases := []struct {
		name             string
		user, pass, code string
	}{
		{"wrong password", "owner", "wrong password here", mustTOTP(t, secret)},
		{"wrong totp", "owner", "correct horse battery", "000000"},
		{"unknown user", "ghost", "some password here", "000000"},
		{"unknown user right password", "ghost", "correct horse battery", mustTOTP(t, secret)},
	}

	var firstStatus int
	var firstBody string
	for i, tc := range cases {
		c := env.newClient()
		status, body := c.do("POST", "/api/auth/login", map[string]string{
			"username": tc.user, "password": tc.pass, "code": tc.code,
		}, nil)
		if status == 200 {
			t.Fatalf("%s: login unexpectedly succeeded", tc.name)
		}
		if i == 0 {
			firstStatus, firstBody = status, body
			continue
		}
		if status != firstStatus || body != firstBody {
			t.Errorf("%s: response differs from the wrong-password case\n got: %d %s\nwant: %d %s",
				tc.name, status, body, firstStatus, firstBody)
		}
	}
	if !strings.Contains(strings.ToLower(firstBody), "incorrect") {
		t.Errorf("error message should be generic, got: %s", firstBody)
	}
}

func TestUnauthenticatedRequestsAreRejected(t *testing.T) {
	env := newTestEnv(t)
	c := env.newClient()

	for _, path := range []string{
		"/api/servers", "/api/server/status", "/api/configuration",
		"/api/backups", "/api/schedules", "/api/users", "/api/activity",
	} {
		if status, body := c.do("GET", path, nil, nil); status != 401 {
			t.Errorf("GET %s returned %d (%s), want 401", path, status, body)
		}
	}
}

func TestAuthenticatedActivityUsesAuditTimestamp(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	var out []struct {
		Username string `json:"username"`
		Action   string `json:"action"`
		At       string `json:"at"`
	}
	status, body := c.do("GET", "/api/activity", nil, &out)
	if status != 200 {
		t.Fatalf("activity failed: %d %s", status, body)
	}
	if len(out) == 0 {
		t.Fatal("activity returned no audit events")
	}
	if out[0].Action == "" || out[0].At == "" {
		t.Fatalf("activity event missing action or timestamp: %+v", out[0])
	}
}

// State-changing requests must carry a CSRF token even with a valid session.
func TestStateChangingRequestRequiresCSRFToken(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	good := c.csrf
	c.csrf = "" // simulate a cross-site form post with no token
	if status, _ := c.do("POST", "/api/server/start", map[string]any{}, nil); status != 403 {
		t.Errorf("request without a CSRF token returned %d, want 403", status)
	}

	c.csrf = "not-the-real-token"
	if status, _ := c.do("POST", "/api/server/start", map[string]any{}, nil); status != 403 {
		t.Errorf("request with a forged CSRF token returned %d, want 403", status)
	}

	// Sanity check: the same request with the real token gets past CSRF and
	// fails for a domain reason instead.
	c.csrf = good
	if status, _ := c.do("POST", "/api/server/start", map[string]any{}, nil); status == 403 {
		t.Error("valid CSRF token was still rejected")
	}
}

func TestSessionRevocationEndsAccess(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	if status, body := c.do("GET", "/api/servers", nil, nil); status != 200 {
		t.Fatalf("authenticated request failed: %d %s", status, body)
	}

	u, err := env.app.Auth.UserByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.app.Auth.RevokeAllSessions(u.ID); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.do("GET", "/api/servers", nil, nil); status != 401 {
		t.Errorf("revoked session still works: got %d, want 401", status)
	}
}

func TestDisabledAccountCannotLogIn(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("member", "correct horse battery", authorization.RoleMember)
	u, err := env.app.Auth.UserByName("member")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.app.Auth.SetDisabled(u.ID, true); err != nil {
		t.Fatal(err)
	}
	c := env.newClient()
	if status, _ := c.login("member", "correct horse battery", secret); status == 200 {
		t.Error("a disabled account was able to sign in")
	}
}

func mustTOTP(t *testing.T, secret string) string {
	t.Helper()
	code, err := auth.TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("TOTPCode: %v", err)
	}
	return code
}
