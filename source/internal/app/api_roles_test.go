package app

import (
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

// The shipped Member role starts with exactly five capabilities. These
// defaults are enforced in the backend, not merely hidden in the interface,
// so they are asserted against the real HTTP API rather than only the table.
func TestMemberIsLimitedToExactlyTheAllowedActions(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("member", "correct horse battery", authorization.RoleMember)
	c := env.newClient()
	c.mustLogin("member", "correct horse battery", secret)

	// Permitted: viewing status and players.
	for _, path := range []string{"/api/server/status", "/api/players"} {
		if status, body := c.do("GET", path, nil, nil); status == 403 {
			t.Errorf("Member was denied %s, which their role allows: %s", path, body)
		}
	}

	// Forbidden reads.
	forbiddenGET := []string{
		"/api/configuration",
		"/api/files?path=.",
		"/api/backups",
		"/api/schedules",
		"/api/users",
		"/api/activity",
		"/api/host",
		"/api/metrics?hours=1",
		"/api/metrics/config",
		"/api/metrics/storage",
		"/api/metrics/internet",
	}
	for _, path := range forbiddenGET {
		if status, body := c.do("GET", path, nil, nil); status != 403 {
			t.Errorf("GET %s as Member returned %d (%s), want 403", path, status, body)
		}
	}

	// Forbidden writes.
	forbiddenPOST := []struct {
		path string
		body any
	}{
		{"/api/server/force-stop", map[string]any{"confirm": true}},
		{"/api/server/command", map[string]string{"command": "list"}},
		{"/api/backups", map[string]string{"type": "world"}},
		{"/api/configuration/jvm", map[string]string{"xms": "1G", "xmx": "2G"}},
		{"/api/configuration/eula", map[string]bool{"accept": true}},
		{"/api/schedules", map[string]any{"name": "x", "schedule_type": "daily"}},
		{"/api/users/invite", map[string]string{"role": "viewer"}},
		{"/api/players/action", map[string]string{"action": "kick", "player": "Steve"}},
		{"/api/metrics/internet/refresh", nil},
		{"/api/metrics/internet/speed-test", map[string]any{}},
	}
	for _, tc := range forbiddenPOST {
		if status, body := c.do("POST", tc.path, tc.body, nil); status != 403 {
			t.Errorf("POST %s as Member returned %d (%s), want 403", tc.path, status, body)
		}
	}
}

// Viewer may look but never touch, including the lifecycle controls a Member
// is allowed to use.
func TestViewerIsReadOnly(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
	c := env.newClient()
	c.mustLogin("viewer", "correct horse battery", secret)

	if status, body := c.do("GET", "/api/server/status", nil, nil); status == 403 {
		t.Errorf("Viewer was denied server status: %s", body)
	}

	for _, path := range []string{
		"/api/server/start", "/api/server/stop", "/api/server/restart", "/api/metrics/internet/refresh", "/api/metrics/internet/speed-test",
	} {
		if status, body := c.do("POST", path, map[string]any{}, nil); status != 403 {
			t.Errorf("POST %s as Viewer returned %d (%s), want 403", path, status, body)
		}
	}

	// A Viewer's permitted view list does not include backups, files,
	// configuration, schedules, the audit trail or host settings.
	for _, path := range []string{
		"/api/backups", "/api/files?path=.", "/api/configuration",
		"/api/schedules", "/api/activity", "/api/host", "/api/users",
		"/api/metrics?hours=1",
		"/api/metrics/config", "/api/metrics/storage", "/api/metrics/internet",
	} {
		if status, body := c.do("GET", path, nil, nil); status != 403 {
			t.Errorf("GET %s as Viewer returned %d (%s), want 403", path, status, body)
		}
	}
}

// An Admin runs servers but must not be able to touch Owner accounts or grant
// themselves the Owner role.
func TestAdminCannotEscalateOrModifyOwner(t *testing.T) {
	env := newTestEnv(t)
	env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	adminSecret := env.createUser("admin", "correct horse battery", authorization.RoleAdmin)

	owner, err := env.app.Auth.UserByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := env.app.Auth.UserByName("admin")
	if err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("admin", "correct horse battery", adminSecret)

	// Cannot promote themselves.
	if status, body := c.do("POST", "/api/users/"+itoa(admin.ID)+"/role",
		map[string]string{"role": "owner"}, nil); status < 400 {
		t.Errorf("Admin promoted themselves to Owner: %d %s", status, body)
	}
	// Cannot modify an Owner.
	if status, body := c.do("POST", "/api/users/"+itoa(owner.ID)+"/role",
		map[string]string{"role": "member"}, nil); status < 400 {
		t.Errorf("Admin demoted an Owner: %d %s", status, body)
	}
	// Cannot delete an Owner.
	if status, body := c.do("DELETE", "/api/users/"+itoa(owner.ID), nil, nil); status < 400 {
		t.Errorf("Admin deleted an Owner: %d %s", status, body)
	}
	// Cannot invite a new Owner.
	if status, body := c.do("POST", "/api/users/invite",
		map[string]string{"role": "owner"}, nil); status < 400 {
		t.Errorf("Admin invited a new Owner: %d %s", status, body)
	}
}

// There must always be at least one active Owner.
func TestLastOwnerCannotBeRemovedOrDemoted(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	owner, err := env.app.Auth.UserByName("owner")
	if err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	if status, body := c.do("POST", "/api/users/"+itoa(owner.ID)+"/role",
		map[string]string{"role": "admin"}, nil); status < 400 {
		t.Errorf("the last Owner was demoted: %d %s", status, body)
	}
	if status, body := c.do("DELETE", "/api/users/"+itoa(owner.ID), nil, nil); status < 400 {
		t.Errorf("the last Owner was deleted: %d %s", status, body)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
