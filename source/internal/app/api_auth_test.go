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

// The invitation enrolment endpoint must always return the secret and URI a
// user can type by hand. The QR is an extra convenience, never a replacement.
func TestInvitationTOTPKeepsManualFallback(t *testing.T) {
	env := newTestEnv(t)
	env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner, err := env.app.Auth.UserByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := env.app.Auth.CreateInvitation(owner.ID, authorization.RoleMember, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	var out struct {
		Secret string `json:"secret"`
		URI    string `json:"uri"`
		QRSVG  string `json:"qr_svg"`
	}
	status, body := c.do("POST", "/api/invitations/"+inv.Token+"/totp",
		map[string]string{"username": "newcomer"}, &out)
	if status != 200 {
		t.Fatalf("enrolment failed: %d %s", status, body)
	}

	if out.Secret == "" {
		t.Error("no secret returned; manual enrolment would be impossible")
	}
	if !strings.HasPrefix(out.URI, "otpauth://totp/") {
		t.Errorf("URI is not an otpauth provisioning URI: %q", out.URI)
	}
	if !strings.Contains(out.URI, out.Secret) {
		t.Error("the URI does not carry the same secret that was returned")
	}

	// The QR is optional, but if present it must encode that same URI and
	// contain nothing script-like, because the page injects it as markup.
	if out.QRSVG != "" {
		if !strings.HasPrefix(out.QRSVG, "<svg ") {
			t.Errorf("qr_svg is not an SVG document: %.60s", out.QRSVG)
		}
		for _, bad := range []string{"<script", "onload=", "javascript:"} {
			if strings.Contains(strings.ToLower(out.QRSVG), bad) {
				t.Errorf("qr_svg contains %q", bad)
			}
		}
		// The QR encodes the URI; it must not embed the raw secret as text.
		if strings.Contains(out.QRSVG, out.Secret) {
			t.Error("qr_svg leaks the secret as literal text")
		}
	}
}

// Enrolment must never write the secret or the provisioning URI to the audit
// trail, where other administrators could read it.
func TestTOTPEnrolmentIsNotAudited(t *testing.T) {
	env := newTestEnv(t)
	env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner, err := env.app.Auth.UserByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := env.app.Auth.CreateInvitation(owner.ID, authorization.RoleMember, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	var out struct {
		Secret string `json:"secret"`
		URI    string `json:"uri"`
	}
	if status, body := c.do("POST", "/api/invitations/"+inv.Token+"/totp",
		map[string]string{"username": "newcomer"}, &out); status != 200 {
		t.Fatalf("enrolment failed: %d %s", status, body)
	}

	rows, err := env.app.DB.Query(`SELECT action, target, detail FROM audit_log`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var action, target, detail string
		if err := rows.Scan(&action, &target, &detail); err != nil {
			t.Fatal(err)
		}
		joined := action + " " + target + " " + detail
		if strings.Contains(joined, out.Secret) {
			t.Errorf("audit row leaks the TOTP secret: %q", joined)
		}
		if strings.Contains(joined, "otpauth://") {
			t.Errorf("audit row leaks the provisioning URI: %q", joined)
		}
	}
}

// The Activity page failed with "no such column: created_at" because the query
// and the schema disagreed. It compiled fine and only broke when a user opened
// the tab, so it needs a test that actually hits the endpoint.
func TestActivityEndpointMatchesTheSchema(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	// The login above writes an audit row, so there is something to return.
	var rows []map[string]any
	status, body := c.do("GET", "/api/activity", nil, &rows)
	if status != 200 {
		t.Fatalf("GET /api/activity returned %d: %s", status, body)
	}
	if len(rows) == 0 {
		t.Fatal("no audit rows returned; logging in should have recorded one")
	}
	for _, key := range []string{"action", "username", "at"} {
		if _, ok := rows[0][key]; !ok {
			t.Errorf("audit row is missing %q: %v", key, rows[0])
		}
	}
}
