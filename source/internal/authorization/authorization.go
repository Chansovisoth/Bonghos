// Package authorization defines Bonghos roles and their default granular
// permission sets. Runtime overrides are stored by the app; backend handlers
// must enforce the effective permissions because frontend visibility is only
// a convenience.
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
	PermServerView      Permission = "server.view"
	PermPerformanceView Permission = "server.performance.view"
	PermPerformanceTest Permission = "server.performance.test"
	PermServerStart     Permission = "server.start"
	PermServerStop      Permission = "server.stop"
	PermServerRestart   Permission = "server.restart"
	PermServerForceStop Permission = "server.force_stop"
	PermConsoleView     Permission = "server.console.view"
	PermConsoleUse      Permission = "server.console.use"
	PermPlayersView     Permission = "server.players.view"
	PermPlayersManage   Permission = "server.players.manage"
	PermFilesManage     Permission = "server.files.manage"
	PermConfigManage    Permission = "server.configuration.manage"
	PermIconManage      Permission = "server.icon.manage"
	PermImportManage    Permission = "server.import.manage"
	PermBackupsView     Permission = "server.backups.view"
	PermBackupsCreate   Permission = "server.backups.create"
	PermBackupsRestore  Permission = "server.backups.restore"
	PermSchedulesManage Permission = "server.schedules.manage"
	PermActivityView    Permission = "activity.view"
	PermUsersManage     Permission = "users.manage"
	PermRolesManage     Permission = "roles.manage"
	PermBotsManage      Permission = "bots.manage"
	PermHostView        Permission = "host.view"
)

// PermissionDefinition is the authoritative permission catalog consumed by
// backend validation and the Web role editor.
type PermissionDefinition struct {
	ID           Permission   `json:"id"`
	Group        string       `json:"group"`
	Label        string       `json:"label"`
	Description  string       `json:"description"`
	Requires     []Permission `json:"requires"`
	AssignableTo []Role       `json:"assignable_roles"`
}

var viewAssignableRoles = []Role{RoleAdmin, RoleMember, RoleViewer}
var actionAssignableRoles = []Role{RoleAdmin, RoleMember}

var permissionCatalog = []PermissionDefinition{
	{PermServerView, "Access", "View server", "See Overview, Servers, and live status.", nil, viewAssignableRoles},
	{PermPerformanceView, "Access", "View performance", "Open machine and Java performance telemetry.", []Permission{PermServerView}, viewAssignableRoles},
	{PermPerformanceTest, "Access", "Test internet speed", "Run a manual bandwidth test that can temporarily affect connected players.", []Permission{PermPerformanceView}, actionAssignableRoles},
	{PermServerStart, "Lifecycle", "Start server", "Start the active Minecraft server.", []Permission{PermServerView}, actionAssignableRoles},
	{PermServerStop, "Lifecycle", "Stop server", "Request a clean server shutdown.", []Permission{PermServerView}, actionAssignableRoles},
	{PermServerRestart, "Lifecycle", "Restart server", "Stop and start the active server.", []Permission{PermServerView}, actionAssignableRoles},
	{PermServerForceStop, "Lifecycle", "Force stop server", "Immediately terminate a stuck server process.", []Permission{PermServerView}, actionAssignableRoles},
	{PermConsoleView, "Console and players", "View console", "Read server console output.", []Permission{PermServerView}, viewAssignableRoles},
	{PermConsoleUse, "Console and players", "Use console", "Send unrestricted Minecraft server commands.", []Permission{PermConsoleView}, actionAssignableRoles},
	{PermPlayersView, "Console and players", "View players", "See online and known players.", []Permission{PermServerView}, viewAssignableRoles},
	{PermPlayersManage, "Console and players", "Manage players", "Kick, ban, whitelist, or grant operator access to players.", []Permission{PermPlayersView}, actionAssignableRoles},
	{PermFilesManage, "Server files and configuration", "Manage files", "Read, upload, edit, move, and delete project files.", []Permission{PermServerView}, actionAssignableRoles},
	{PermConfigManage, "Server files and configuration", "Manage configuration", "Change launch settings and server properties.", []Permission{PermServerView}, actionAssignableRoles},
	{PermIconManage, "Server files and configuration", "Manage server icons", "Upload or remove project icons.", []Permission{PermServerView}, actionAssignableRoles},
	{PermImportManage, "Server files and configuration", "Manage projects", "Import, duplicate, reset, or delete server projects.", []Permission{PermServerView}, actionAssignableRoles},
	{PermBackupsView, "Backups and automation", "View backups", "See backups and their storage location.", []Permission{PermServerView}, viewAssignableRoles},
	{PermBackupsCreate, "Backups and automation", "Manage backups", "Create, verify, protect, and delete backups.", []Permission{PermBackupsView}, actionAssignableRoles},
	{PermBackupsRestore, "Backups and automation", "Restore backups", "Replace server data from a backup.", []Permission{PermBackupsView}, actionAssignableRoles},
	{PermSchedulesManage, "Backups and automation", "Manage schedules", "Create and run lifecycle, console, and backup automation.", []Permission{PermServerView, PermServerStart, PermServerStop, PermServerRestart, PermConsoleUse, PermBackupsCreate}, actionAssignableRoles},
	{PermActivityView, "People and integrations", "View activity", "Review account and server administration history.", nil, viewAssignableRoles},
	{PermUsersManage, "People and integrations", "Manage users", "Invite users and manage lower-role accounts and sessions.", nil, []Role{RoleAdmin, RoleMember}},
	{PermRolesManage, "People and integrations", "Manage role permissions", "Configure permissions for lower roles.", nil, []Role{RoleAdmin}},
	{PermBotsManage, "People and integrations", "Manage notification bots", "Configure Discord and Telegram bots.", []Permission{PermServerView}, actionAssignableRoles},
	{PermHostView, "System", "View host", "See Bonghos installation, listener, and service details.", nil, viewAssignableRoles},
}

// rolePerms holds the shipped defaults. Persisted overrides are resolved by
// the app without mutating this map.
var rolePerms = map[Role]map[Permission]bool{
	RoleOwner: allPermissions(),
	RoleAdmin: {
		PermServerView: true, PermPerformanceView: true, PermPerformanceTest: true,
		PermServerStart: true, PermServerStop: true,
		PermServerRestart: true, PermServerForceStop: true,
		PermConsoleView: true, PermConsoleUse: true,
		PermPlayersView: true, PermPlayersManage: true,
		PermFilesManage: true, PermConfigManage: true, PermIconManage: true,
		PermImportManage: true,
		PermBackupsView:  true, PermBackupsCreate: true, PermBackupsRestore: true,
		PermSchedulesManage: true,
		PermActivityView:    true,
		PermUsersManage:     true, // limited: cannot touch Owners (enforced separately)
		PermBotsManage:      true,
		PermHostView:        true,
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
	},
}

// AllPermissions returns every recognized permission in stable display order.
func AllPermissions() []Permission {
	out := make([]Permission, 0, len(permissionCatalog))
	for _, definition := range permissionCatalog {
		out = append(out, definition.ID)
	}
	return out
}

// PermissionCatalog returns a defensive copy of the definitions in stable
// display order.
func PermissionCatalog() []PermissionDefinition {
	out := make([]PermissionDefinition, len(permissionCatalog))
	for i, definition := range permissionCatalog {
		out[i] = definition
		out[i].Requires = append([]Permission(nil), definition.Requires...)
		out[i].AssignableTo = append([]Role(nil), definition.AssignableTo...)
	}
	return out
}

func PermissionInfo(permission Permission) (PermissionDefinition, bool) {
	for _, definition := range permissionCatalog {
		if definition.ID == permission {
			definition.Requires = append([]Permission(nil), definition.Requires...)
			definition.AssignableTo = append([]Role(nil), definition.AssignableTo...)
			return definition, true
		}
	}
	return PermissionDefinition{}, false
}

func PermissionAssignableTo(permission Permission, role Role) bool {
	definition, ok := PermissionInfo(permission)
	if !ok {
		return false
	}
	for _, target := range definition.AssignableTo {
		if target == role {
			return true
		}
	}
	return false
}

func allPermissions() map[Permission]bool {
	m := map[Permission]bool{}
	for _, p := range AllPermissions() {
		m[p] = true
	}
	return m
}

// ValidPermission reports whether p is a permission understood by this build.
func ValidPermission(p Permission) bool {
	_, ok := PermissionInfo(p)
	return ok
}

// Has reports whether role holds permission. Unknown roles hold nothing.
func Has(role Role, p Permission) bool {
	return rolePerms[role][p]
}

// Permissions lists the permissions of a role (for the frontend).
func Permissions(role Role) []string {
	var out []string
	for _, p := range AllPermissions() {
		if rolePerms[role][p] {
			out = append(out, string(p))
		}
	}
	return out
}

// CanManageUser enforces the fixed role hierarchy after the caller has passed
// the users.manage permission check. Owners may manage every other account;
// other delegated managers may act only on roles below their own and cannot
// promote an account to their own level or higher.
func CanManageUser(actor Role, actorID int64, target Role, targetID int64, newRole *Role) bool {
	if !ValidRole(actor) || !ValidRole(target) {
		return false
	}
	if actorID == targetID {
		if newRole != nil && *newRole != target {
			return false
		}
	}
	if actor == RoleOwner {
		return newRole == nil || ValidRole(*newRole)
	}
	if roleRank(actor) <= roleRank(target) {
		return false
	}
	return newRole == nil || (ValidRole(*newRole) && roleRank(*newRole) < roleRank(actor))
}

// CanInviteRole applies the same hierarchy to invitation creation. Owner
// invitations remain prohibited by the invitation store itself.
func CanInviteRole(actor, invited Role) bool {
	if !ValidRole(actor) || !ValidRole(invited) || invited == RoleOwner {
		return false
	}
	return actor == RoleOwner || roleRank(invited) < roleRank(actor)
}

func roleRank(role Role) int {
	switch role {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleMember:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}
