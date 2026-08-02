// Package authorization defines Bonghos roles and the fixed granular
// permission sets backing them. Backend handlers must consult this package —
// frontend role-awareness is cosmetic only.
package authorization

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

func ValidRole(r Role) bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	}
	return false
}

type Permission string

const (
	PermServerView        Permission = "server.view"
	PermServerStart       Permission = "server.start"
	PermServerStop        Permission = "server.stop"
	PermServerRestart     Permission = "server.restart"
	PermServerForceStop   Permission = "server.force_stop"
	PermConsoleView       Permission = "server.console.view"
	PermConsoleUse        Permission = "server.console.use"
	PermPlayersView       Permission = "server.players.view"
	PermPlayersManage     Permission = "server.players.manage"
	PermFilesManage       Permission = "server.files.manage"
	PermConfigManage      Permission = "server.configuration.manage"
	PermIconManage        Permission = "server.icon.manage"
	PermImportManage      Permission = "server.import.manage"
	PermBackupsView       Permission = "server.backups.view"
	PermBackupsCreate     Permission = "server.backups.create"
	PermBackupsRestore    Permission = "server.backups.restore"
	PermSchedulesManage   Permission = "server.schedules.manage"
	PermUsersManage       Permission = "users.manage"
	PermSecurityManage    Permission = "security.manage"
	PermHostManage        Permission = "host.manage"
	PermPortabilityManage Permission = "portability.manage"
)

// rolePerms holds the fixed v1 permission sets. Members receive exactly the
// five capabilities the specification allows and nothing more.
var rolePerms = map[Role]map[Permission]bool{
	RoleOwner: allPermissions(),
	RoleAdmin: {
		PermServerView: true, PermServerStart: true, PermServerStop: true,
		PermServerRestart: true, PermServerForceStop: true,
		PermConsoleView: true, PermConsoleUse: true,
		PermPlayersView: true, PermPlayersManage: true,
		PermFilesManage: true, PermConfigManage: true, PermIconManage: true,
		PermImportManage: true,
		PermBackupsView:  true, PermBackupsCreate: true, PermBackupsRestore: true,
		PermSchedulesManage: true,
		PermUsersManage:     true, // limited: cannot touch Owners (enforced separately)
	},
	RoleMember: {
		PermServerView:    true,
		PermServerStart:   true,
		PermServerStop:    true,
		PermServerRestart: true,
		PermPlayersView:   true,
	},
	RoleViewer: {
		PermServerView:  true,
		PermPlayersView: true,
		PermConsoleView: true, // read-only console logs
		PermBackupsView: true,
	},
}

func allPermissions() map[Permission]bool {
	m := map[Permission]bool{}
	for _, p := range []Permission{
		PermServerView, PermServerStart, PermServerStop, PermServerRestart,
		PermServerForceStop, PermConsoleView, PermConsoleUse, PermPlayersView,
		PermPlayersManage, PermFilesManage, PermConfigManage, PermIconManage,
		PermImportManage, PermBackupsView, PermBackupsCreate, PermBackupsRestore,
		PermSchedulesManage, PermUsersManage, PermSecurityManage, PermHostManage,
		PermPortabilityManage,
	} {
		m[p] = true
	}
	return m
}

// Has reports whether role holds permission. Unknown roles hold nothing.
func Has(role Role, p Permission) bool {
	return rolePerms[role][p]
}

// Permissions lists the permissions of a role (for the frontend).
func Permissions(role Role) []string {
	var out []string
	for p := range rolePerms[role] {
		out = append(out, string(p))
	}
	return out
}

// CanManageUser enforces ownership boundaries for user administration:
// only Owners may modify, delete, demote or promote Owners; nobody may raise
// their own role; Admins manage Members and Viewers only.
func CanManageUser(actor Role, actorID int64, target Role, targetID int64, newRole *Role) bool {
	if actor != RoleOwner && actor != RoleAdmin {
		return false
	}
	if actorID == targetID {
		// Users cannot elevate themselves; self role change is refused here
		// (profile edits go through a different path).
		if newRole != nil && *newRole != target {
			return false
		}
	}
	if target == RoleOwner && actor != RoleOwner {
		return false
	}
	if newRole != nil && *newRole == RoleOwner && actor != RoleOwner {
		return false
	}
	if actor == RoleAdmin && target == RoleAdmin && targetID != actorID {
		// Admins may not manage other Admins in v1.
		return false
	}
	return true
}
