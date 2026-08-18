package authorization

import "testing"

func TestMemberExactPermissions(t *testing.T) {
	allowed := []Permission{PermServerView, PermServerStart, PermServerStop, PermServerRestart, PermPlayersView}
	for _, p := range allowed {
		if !Has(RoleMember, p) {
			t.Errorf("Member should have %s", p)
		}
	}
	denied := []Permission{
		PermPerformanceView, PermPerformanceTest, PermServerForceStop, PermConsoleView, PermConsoleUse, PermPlayersManage,
		PermFilesManage, PermConfigManage, PermIconManage, PermImportManage,
		PermBackupsView, PermBackupsCreate, PermBackupsRestore, PermSchedulesManage,
		PermActivityView, PermUsersManage, PermRolesManage, PermBotsManage, PermHostView,
	}
	for _, p := range denied {
		if Has(RoleMember, p) {
			t.Errorf("Member must NOT have %s", p)
		}
	}
}

func TestViewerReadOnly(t *testing.T) {
	if !Has(RoleViewer, PermServerView) || !Has(RoleViewer, PermPlayersView) {
		t.Error("Viewer should view status and players")
	}
	for _, p := range []Permission{PermPerformanceView, PermPerformanceTest, PermServerStart, PermServerStop, PermServerRestart, PermConsoleUse,
		PermFilesManage, PermBackupsCreate, PermUsersManage, PermRolesManage} {
		if Has(RoleViewer, p) {
			t.Errorf("Viewer must NOT have %s", p)
		}
	}
}

func TestOwnerHasEverything(t *testing.T) {
	for _, p := range []Permission{PermServerForceStop, PermUsersManage, PermRolesManage, PermBotsManage, PermActivityView, PermHostView} {
		if !Has(RoleOwner, p) {
			t.Errorf("Owner missing %s", p)
		}
	}
}

func TestAdminCanManageBotsWithoutOwnerSecurityPermission(t *testing.T) {
	if !Has(RoleAdmin, PermBotsManage) {
		t.Fatal("Admin should manage notification bots")
	}
	if !Has(RoleAdmin, PermActivityView) || !Has(RoleAdmin, PermHostView) {
		t.Fatal("Admin should retain activity and host visibility")
	}
	if Has(RoleAdmin, PermRolesManage) {
		t.Fatal("Admin must receive role-permission management from an Owner")
	}
	if !Has(RoleAdmin, PermPerformanceTest) {
		t.Fatal("Admin should be able to run manual Internet speed tests")
	}
}

func TestPermissionCatalogDependenciesAndTargets(t *testing.T) {
	console, ok := PermissionInfo(PermConsoleUse)
	if !ok || len(console.Requires) != 1 || console.Requires[0] != PermConsoleView {
		t.Fatalf("console-use prerequisites = %v, want console view", console.Requires)
	}
	if PermissionAssignableTo(PermUsersManage, RoleViewer) {
		t.Error("Viewer must not receive users.manage")
	}
	if !PermissionAssignableTo(PermUsersManage, RoleMember) {
		t.Error("Member should be eligible for delegated users.manage")
	}
	if !PermissionAssignableTo(PermRolesManage, RoleAdmin) || PermissionAssignableTo(PermRolesManage, RoleMember) {
		t.Error("roles.manage must be Admin-only")
	}
	if !PermissionAssignableTo(PermPlayitManage, RoleAdmin) || PermissionAssignableTo(PermPlayitManage, RoleMember) {
		t.Error("playit.manage must be delegable only to Admin")
	}
	if Has(RoleAdmin, PermPlayitManage) {
		t.Error("Admin must receive playit.manage explicitly from an Owner")
	}
	if !PermissionAssignableTo(PermPerformanceTest, RoleMember) || PermissionAssignableTo(PermPerformanceTest, RoleViewer) {
		t.Error("performance.test must be an action permission assignable to Admin and Member only")
	}
}

func TestViewerCanOnlyReceiveViewPermissions(t *testing.T) {
	viewPermissions := map[Permission]bool{
		PermServerView:      true,
		PermPerformanceView: true,
		PermConsoleView:     true,
		PermPlayersView:     true,
		PermBackupsView:     true,
		PermActivityView:    true,
		PermHostView:        true,
	}
	for _, permission := range AllPermissions() {
		if got, want := PermissionAssignableTo(permission, RoleViewer), viewPermissions[permission]; got != want {
			t.Errorf("Viewer assignability for %s = %v, want %v", permission, got, want)
		}
	}
}

func TestAllPermissionsAreUniqueAndValid(t *testing.T) {
	seen := map[Permission]bool{}
	for _, definition := range PermissionCatalog() {
		permission := definition.ID
		if seen[permission] {
			t.Errorf("duplicate permission %s", permission)
		}
		seen[permission] = true
		if !ValidPermission(permission) {
			t.Errorf("listed permission %s is not valid", permission)
		}
		if definition.Group == "" || definition.Label == "" || definition.Description == "" {
			t.Errorf("permission %s has incomplete display metadata", permission)
		}
		for _, required := range definition.Requires {
			if !ValidPermission(required) {
				t.Errorf("permission %s requires unknown permission %s", permission, required)
			}
		}
	}
	if ValidPermission("unknown.permission") {
		t.Error("unknown permission was accepted")
	}
}

func TestDefaultProfilesSatisfyCatalogRules(t *testing.T) {
	for _, role := range []Role{RoleAdmin, RoleMember, RoleViewer} {
		for _, permission := range AllPermissions() {
			if !Has(role, permission) {
				continue
			}
			if !PermissionAssignableTo(permission, role) {
				t.Errorf("%s default includes unassignable permission %s", role, permission)
			}
			definition, _ := PermissionInfo(permission)
			for _, required := range definition.Requires {
				if !Has(role, required) {
					t.Errorf("%s default permission %s is missing prerequisite %s", role, permission, required)
				}
			}
		}
	}
}

func TestPermissionDependenciesAreAcyclic(t *testing.T) {
	visiting := map[Permission]bool{}
	visited := map[Permission]bool{}
	var visit func(Permission)
	visit = func(permission Permission) {
		if visiting[permission] {
			t.Fatalf("permission dependency cycle includes %s", permission)
		}
		if visited[permission] {
			return
		}
		visiting[permission] = true
		definition, _ := PermissionInfo(permission)
		for _, required := range definition.Requires {
			visit(required)
		}
		delete(visiting, permission)
		visited[permission] = true
	}
	for _, permission := range AllPermissions() {
		visit(permission)
	}
}

func TestCanManageUser(t *testing.T) {
	owner := RoleOwner
	// Admin cannot modify an Owner.
	if CanManageUser(RoleAdmin, 2, RoleOwner, 1, nil) {
		t.Error("Admin modified an Owner")
	}
	// Admin cannot promote anyone to Owner.
	if CanManageUser(RoleAdmin, 2, RoleMember, 3, &owner) {
		t.Error("Admin promoted a user to Owner")
	}
	// Admin can manage Members.
	member := RoleMember
	if !CanManageUser(RoleAdmin, 2, RoleViewer, 3, &member) {
		t.Error("Admin should manage Viewer→Member")
	}
	viewer := RoleViewer
	if !CanManageUser(RoleMember, 3, RoleViewer, 4, &viewer) {
		t.Error("a Member delegated users.manage should manage Viewers")
	}
	if CanManageUser(RoleMember, 3, RoleViewer, 4, &member) {
		t.Error("a Member delegated users.manage promoted a user to their own level")
	}
	// Users cannot raise their own role.
	admin := RoleAdmin
	if CanManageUser(RoleMember, 3, RoleMember, 3, &admin) {
		t.Error("Member self-promoted to Admin")
	}
	// Owner can manage other Owners.
	if !CanManageUser(RoleOwner, 1, RoleOwner, 4, nil) {
		t.Error("Owner should manage another Owner")
	}
}

func TestCanInviteRole(t *testing.T) {
	if !CanInviteRole(RoleOwner, RoleAdmin) || !CanInviteRole(RoleAdmin, RoleMember) || !CanInviteRole(RoleMember, RoleViewer) {
		t.Error("a manager should invite roles below their own")
	}
	if CanInviteRole(RoleAdmin, RoleAdmin) || CanInviteRole(RoleMember, RoleMember) || CanInviteRole(RoleViewer, RoleViewer) {
		t.Error("a manager invited a peer role")
	}
	if CanInviteRole(RoleOwner, RoleOwner) {
		t.Error("Owner invitations must remain prohibited")
	}
}
