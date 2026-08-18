package app

import (
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

type rolePermissionsTestPayload struct {
	Catalog []authorization.PermissionDefinition `json:"catalog"`
	Roles   map[string]struct {
		Permissions []string `json:"permissions"`
		Revision    int64    `json:"revision"`
		Customized  bool     `json:"customized"`
	} `json:"roles"`
}

func getRolePermissions(t *testing.T, c *client) rolePermissionsTestPayload {
	t.Helper()
	var payload rolePermissionsTestPayload
	if status, body := c.do("GET", "/api/roles/permissions", nil, &payload); status != 200 {
		t.Fatalf("GET role permissions = %d (%s)", status, body)
	}
	return payload
}

func putRolePermissions(c *client, role string, permissions []string, revision int64, out any) (int, string) {
	return c.do("PUT", "/api/roles/"+role+"/permissions", map[string]any{
		"permissions": permissions,
		"revision":    revision,
	}, out)
}

func withoutPermission(permissions []string, removed authorization.Permission) []string {
	out := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		if permission != string(removed) {
			out = append(out, permission)
		}
	}
	return out
}

func TestOwnerCanDelegateRolePermissionManagementToAdmin(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	adminSecret := env.createUser("admin", "correct horse battery", authorization.RoleAdmin)

	adminClient := env.newClient()
	adminClient.mustLogin("admin", "correct horse battery", adminSecret)
	if status, body := adminClient.do("GET", "/api/roles/permissions", nil, nil); status != 403 {
		t.Fatalf("default Admin role-manager access = %d (%s), want 403", status, body)
	}

	ownerClient := env.newClient()
	ownerClient.mustLogin("owner", "correct horse battery", ownerSecret)
	permissions := withoutPermission(authorization.Permissions(authorization.RoleAdmin), authorization.PermHostView)
	permissions = append(permissions, string(authorization.PermRolesManage))
	if status, body := putRolePermissions(ownerClient, "admin", permissions, 0, nil); status != 200 {
		t.Fatalf("Owner granting roles.manage to Admin = %d (%s), want 200", status, body)
	}

	adminPayload := getRolePermissions(t, adminClient)
	if status, body := putRolePermissions(adminClient, "admin", permissions, adminPayload.Roles["admin"].Revision, nil); status != 403 {
		t.Fatalf("Admin editing Admin role = %d (%s), want 403", status, body)
	}
	if status, body := putRolePermissions(adminClient, "member", []string{
		string(authorization.PermServerView), string(authorization.PermHostView),
	}, adminPayload.Roles["member"].Revision, nil); status != 403 {
		t.Fatalf("Admin granting a permission they lack = %d (%s), want 403", status, body)
	}
	if status, body := putRolePermissions(adminClient, "member", []string{
		string(authorization.PermServerView), string(authorization.PermConsoleView),
	}, adminPayload.Roles["member"].Revision, nil); status != 200 {
		t.Fatalf("Admin editing a lower role = %d (%s), want 200", status, body)
	}
}

func TestRolePermissionOverridesDriveHTTPAuthorization(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	viewerSecret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)

	ownerClient := env.newClient()
	ownerClient.mustLogin("owner", "correct horse battery", ownerSecret)
	desired := []string{string(authorization.PermServerView), string(authorization.PermPerformanceView)}
	if status, body := putRolePermissions(ownerClient, "viewer", desired, 0, nil); status != 200 {
		t.Fatalf("updating Viewer permissions = %d (%s), want 200", status, body)
	}

	viewerClient := env.newClient()
	viewerClient.mustLogin("viewer", "correct horse battery", viewerSecret)
	if status, body := viewerClient.do("GET", "/api/metrics/config", nil, nil); status != 200 {
		t.Fatalf("Viewer with performance override = %d (%s), want 200", status, body)
	}
	if status, body := viewerClient.do("GET", "/api/players", nil, nil); status != 403 {
		t.Fatalf("Viewer without players override = %d (%s), want 403", status, body)
	}
	if !env.app.websocketTopicAllowed(authorization.RoleViewer, "performance") {
		t.Error("Viewer performance WebSocket subscription did not use the override")
	}
	if !env.app.websocketTopicAllowed(authorization.RoleViewer, "overview_performance") {
		t.Error("Viewer Overview performance subscription did not use the override")
	}
	if env.app.websocketTopicAllowed(authorization.RoleViewer, "console") {
		t.Error("Viewer console WebSocket subscription ignored the removed permission")
	}
	status, body := viewerClient.do("GET", "/api/auth/me", nil, nil)
	if status != 200 || !strings.Contains(body, string(authorization.PermPerformanceView)) || strings.Contains(body, string(authorization.PermPlayersView)) {
		t.Fatalf("effective permissions in /api/auth/me = %d (%s)", status, body)
	}

	if status, body := putRolePermissions(ownerClient, "owner", nil, 0, nil); status != 403 {
		t.Fatalf("editing Owner permissions = %d (%s), want 403", status, body)
	}
	if status, body := putRolePermissions(ownerClient, "viewer", []string{"unknown.permission"}, 1, nil); status != 400 {
		t.Fatalf("unknown permission = %d (%s), want 400", status, body)
	}
	if status, body := putRolePermissions(ownerClient, "viewer", []string{string(authorization.PermRolesManage)}, 1, nil); status != 400 {
		t.Fatalf("roles.manage on Viewer = %d (%s), want 400", status, body)
	}
	if status, body := putRolePermissions(ownerClient, "viewer", []string{
		string(authorization.PermServerView), string(authorization.PermServerStart),
	}, 1, nil); status != 400 {
		t.Fatalf("server.start on Viewer = %d (%s), want 400", status, body)
	}
}

func TestRolePermissionDependenciesAreRequired(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", ownerSecret)

	for _, permissions := range [][]string{
		{string(authorization.PermConsoleUse)},
		{string(authorization.PermPlayersManage)},
		{string(authorization.PermBackupsRestore)},
		{string(authorization.PermServerView), string(authorization.PermSchedulesManage)},
	} {
		if status, body := putRolePermissions(c, "member", permissions, 0, nil); status != 400 {
			t.Fatalf("invalid dependency set %v = %d (%s), want 400", permissions, status, body)
		}
	}
	if status, body := putRolePermissions(c, "viewer", []string{string(authorization.PermUsersManage)}, 0, nil); status != 400 {
		t.Fatalf("users.manage on Viewer = %d (%s), want 400", status, body)
	}
}

func TestViewerLegacyActionPermissionIsFiltered(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", ownerSecret)

	if status, body := putRolePermissions(c, "viewer", authorization.Permissions(authorization.RoleViewer), 0, nil); status != 200 {
		t.Fatalf("saving Viewer profile = %d (%s), want 200", status, body)
	}
	if _, err := env.app.DB.Exec(`UPDATE role_permissions SET allowed=1 WHERE role='viewer' AND permission=?`,
		string(authorization.PermServerStart)); err != nil {
		t.Fatal(err)
	}
	if env.app.hasPermission(authorization.RoleViewer, authorization.PermServerStart) {
		t.Error("Viewer retained a legacy action permission that is no longer assignable")
	}
	payload := getRolePermissions(t, c)
	for _, permission := range payload.Roles["viewer"].Permissions {
		if permission == string(authorization.PermServerStart) {
			t.Error("Viewer payload exposed a legacy action permission")
		}
	}
}

func TestCustomizedRoleIsAnExplicitSnapshot(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", ownerSecret)

	defaults := authorization.Permissions(authorization.RoleMember)
	customized := append([]string(nil), defaults...)
	customized = append(customized, string(authorization.PermConsoleView))
	var payload rolePermissionsTestPayload
	if status, body := putRolePermissions(c, "member", customized, 0, &payload); status != 200 {
		t.Fatalf("saving customized permissions = %d (%s), want 200", status, body)
	}
	var count int
	if err := env.app.DB.QueryRow(`SELECT COUNT(*) FROM role_permissions WHERE role='member'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != len(authorization.AllPermissions()) {
		t.Fatalf("custom profile stored %d permissions, want %d", count, len(authorization.AllPermissions()))
	}
	if !payload.Roles["member"].Customized || payload.Roles["member"].Revision != 1 {
		t.Fatalf("saved Member profile = %+v", payload.Roles["member"])
	}
	if _, err := env.app.DB.Exec(`DELETE FROM role_permissions WHERE role='member' AND permission=?`, string(authorization.PermServerView)); err != nil {
		t.Fatal(err)
	}
	if env.app.hasPermission(authorization.RoleMember, authorization.PermServerView) {
		t.Error("a missing customized permission inherited the shipped default")
	}
	if env.app.hasPermission(authorization.RoleMember, authorization.PermServerStart) {
		t.Error("a customized permission remained effective without its prerequisite")
	}
}

func TestRolePermissionSaveRejectsStaleRevisionAndAuditsChanges(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", ownerSecret)

	permissions := authorization.Permissions(authorization.RoleViewer)
	permissions = append(permissions, string(authorization.PermHostView))
	if status, body := putRolePermissions(c, "viewer", permissions, 0, nil); status != 200 {
		t.Fatalf("first save = %d (%s)", status, body)
	}
	if status, body := putRolePermissions(c, "viewer", permissions, 0, nil); status != 409 {
		t.Fatalf("stale save = %d (%s), want 409", status, body)
	}
	var detail string
	if err := env.app.DB.QueryRow(`SELECT detail FROM audit_log WHERE action='role_permissions_changed' ORDER BY id DESC LIMIT 1`).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, `"granted":["host.view"]`) || !strings.Contains(detail, `"revoked":`) {
		t.Fatalf("audit detail does not contain exact changes: %s", detail)
	}
}

func TestRolePermissionNoOpDoesNotCustomizeOrRevise(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", ownerSecret)

	var payload rolePermissionsTestPayload
	if status, body := putRolePermissions(c, "viewer", authorization.Permissions(authorization.RoleViewer), 0, &payload); status != 200 {
		t.Fatalf("no-op Viewer save = %d (%s), want 200", status, body)
	}
	if payload.Roles["viewer"].Revision != 0 || payload.Roles["viewer"].Customized {
		t.Fatalf("no-op Viewer save changed profile: %+v", payload.Roles["viewer"])
	}
	var profiles, audits int
	if err := env.app.DB.QueryRow(`SELECT COUNT(*) FROM role_permission_profiles WHERE role='viewer'`).Scan(&profiles); err != nil {
		t.Fatal(err)
	}
	if err := env.app.DB.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action='role_permissions_changed' AND target='viewer'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if profiles != 0 || audits != 0 {
		t.Fatalf("no-op Viewer save created profiles=%d audits=%d, want zero", profiles, audits)
	}
}

func TestDelegatedUserManagementFollowsRoleHierarchy(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	memberSecret := env.createUser("member", "correct horse battery", authorization.RoleMember)
	viewer := func() int64 {
		env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
		user, err := env.app.Auth.UserByName("viewer")
		if err != nil {
			t.Fatal(err)
		}
		return user.ID
	}()

	ownerClient := env.newClient()
	ownerClient.mustLogin("owner", "correct horse battery", ownerSecret)
	permissions := authorization.Permissions(authorization.RoleMember)
	permissions = append(permissions, string(authorization.PermUsersManage))
	if status, body := putRolePermissions(ownerClient, "member", permissions, 0, nil); status != 200 {
		t.Fatalf("delegating users.manage to Member = %d (%s), want 200", status, body)
	}

	memberClient := env.newClient()
	memberClient.mustLogin("member", "correct horse battery", memberSecret)
	if status, body := memberClient.do("POST", "/api/users/"+itoa(viewer)+"/disable",
		map[string]bool{"disabled": true}, nil); status != 200 {
		t.Fatalf("Member managing lower Viewer = %d (%s), want 200", status, body)
	}
	if status, body := memberClient.do("POST", "/api/users/invite",
		map[string]string{"role": "viewer"}, nil); status != 200 {
		t.Fatalf("Member inviting Viewer = %d (%s), want 200", status, body)
	}
	if status, body := memberClient.do("POST", "/api/users/invite",
		map[string]string{"role": "member"}, nil); status != 403 {
		t.Fatalf("Member inviting peer Member = %d (%s), want 403", status, body)
	}
}
