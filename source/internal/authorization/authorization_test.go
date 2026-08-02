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
		PermServerForceStop, PermConsoleView, PermConsoleUse, PermPlayersManage,
		PermFilesManage, PermConfigManage, PermIconManage, PermImportManage,
		PermBackupsView, PermBackupsCreate, PermBackupsRestore, PermSchedulesManage,
		PermUsersManage, PermSecurityManage, PermHostManage, PermPortabilityManage,
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
	for _, p := range []Permission{PermServerStart, PermServerStop, PermServerRestart, PermConsoleUse,
		PermFilesManage, PermBackupsCreate, PermUsersManage} {
		if Has(RoleViewer, p) {
			t.Errorf("Viewer must NOT have %s", p)
		}
	}
}

func TestOwnerHasEverything(t *testing.T) {
	for _, p := range []Permission{PermServerForceStop, PermUsersManage, PermSecurityManage, PermPortabilityManage} {
		if !Has(RoleOwner, p) {
			t.Errorf("Owner missing %s", p)
		}
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
