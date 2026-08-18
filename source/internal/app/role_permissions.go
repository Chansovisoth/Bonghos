package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

type rolePermissionQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type rolePermissionState struct {
	Permissions   map[authorization.Permission]bool
	Revision      int64
	Customized    bool
	ProfileExists bool
}

func defaultRolePermissionState(role authorization.Role) rolePermissionState {
	permissions := make(map[authorization.Permission]bool, len(authorization.AllPermissions()))
	for _, permission := range authorization.AllPermissions() {
		permissions[permission] = authorization.Has(role, permission)
	}
	return rolePermissionState{Permissions: permissions}
}

func loadRolePermissionState(q rolePermissionQuerier, role authorization.Role) (rolePermissionState, error) {
	state := defaultRolePermissionState(role)
	if role == authorization.RoleOwner {
		return state, nil
	}
	if !authorization.ValidRole(role) {
		return rolePermissionState{}, errors.New("invalid role")
	}
	var customized int
	err := q.QueryRow(`SELECT customized, revision FROM role_permission_profiles WHERE role=?`, string(role)).
		Scan(&customized, &state.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return rolePermissionState{}, err
	}
	state.ProfileExists = true
	state.Customized = customized != 0
	if !state.Customized {
		return state, nil
	}

	// Customized profiles are explicit snapshots. Missing or newly introduced
	// permissions fail closed instead of inheriting a future shipped default.
	for permission := range state.Permissions {
		state.Permissions[permission] = false
	}
	rows, err := q.Query(`SELECT permission, allowed FROM role_permissions WHERE role=?`, string(role))
	if err != nil {
		return rolePermissionState{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		var allowed int
		if err := rows.Scan(&raw, &allowed); err != nil {
			return rolePermissionState{}, err
		}
		permission := authorization.Permission(raw)
		if authorization.ValidPermission(permission) && authorization.PermissionAssignableTo(permission, role) {
			state.Permissions[permission] = allowed != 0
		}
	}
	if err := rows.Err(); err != nil {
		return rolePermissionState{}, err
	}
	enforcePermissionDependencies(state.Permissions)
	return state, nil
}

func enforcePermissionDependencies(permissions map[authorization.Permission]bool) {
	for changed := true; changed; {
		changed = false
		for permission, allowed := range permissions {
			if !allowed {
				continue
			}
			definition, ok := authorization.PermissionInfo(permission)
			if !ok {
				permissions[permission] = false
				changed = true
				continue
			}
			for _, required := range definition.Requires {
				if permissions[required] {
					continue
				}
				permissions[permission] = false
				changed = true
				break
			}
		}
	}
}

func permissionList(state rolePermissionState) []string {
	out := make([]string, 0, len(authorization.AllPermissions()))
	for _, permission := range authorization.AllPermissions() {
		if state.Permissions[permission] {
			out = append(out, string(permission))
		}
	}
	return out
}

func (a *App) hasPermission(role authorization.Role, permission authorization.Permission) bool {
	if !authorization.ValidPermission(permission) {
		return false
	}
	state, err := loadRolePermissionState(a.DB, role)
	return err == nil && state.Permissions[permission]
}

func (a *App) permissions(role authorization.Role) []string {
	state, err := loadRolePermissionState(a.DB, role)
	if err != nil {
		return nil
	}
	return permissionList(state)
}

func (a *App) rolePermissionRevision(role authorization.Role) (int64, error) {
	state, err := loadRolePermissionState(a.DB, role)
	return state.Revision, err
}

func canEditRolePermissions(actor, target authorization.Role) bool {
	if target == authorization.RoleOwner || !authorization.ValidRole(target) {
		return false
	}
	if actor == authorization.RoleOwner {
		return true
	}
	return actor == authorization.RoleAdmin && (target == authorization.RoleMember || target == authorization.RoleViewer)
}

func (a *App) rolePermissionsPayload(actor authorization.Role) (map[string]any, error) {
	roles := map[string]any{}
	for _, role := range []authorization.Role{
		authorization.RoleOwner, authorization.RoleAdmin, authorization.RoleMember, authorization.RoleViewer,
	} {
		state, err := loadRolePermissionState(a.DB, role)
		if err != nil {
			return nil, err
		}
		defaults := defaultRolePermissionState(role)
		roles[string(role)] = map[string]any{
			"permissions": permissionList(state),
			"defaults":    permissionList(defaults),
			"editable":    canEditRolePermissions(actor, role),
			"revision":    state.Revision,
			"customized":  state.Customized,
		}
	}
	return map[string]any{
		"catalog": authorization.PermissionCatalog(),
		"roles":   roles,
	}, nil
}

func (a *App) handleRolePermissions(w http.ResponseWriter, r *http.Request) {
	payload, err := a.rolePermissionsPayload(currentUser(r).Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func validateDesiredPermissions(target authorization.Role, requested []string) (map[authorization.Permission]bool, error) {
	desired := make(map[authorization.Permission]bool, len(requested))
	for _, raw := range requested {
		permission := authorization.Permission(raw)
		if !authorization.ValidPermission(permission) {
			return nil, fmt.Errorf("unknown permission %q", raw)
		}
		if !authorization.PermissionAssignableTo(permission, target) {
			return nil, fmt.Errorf("permission %q cannot be assigned to %s", raw, target)
		}
		desired[permission] = true
	}
	for permission := range desired {
		definition, _ := authorization.PermissionInfo(permission)
		for _, required := range definition.Requires {
			if !desired[required] {
				return nil, fmt.Errorf("permission %q requires %q", permission, required)
			}
		}
	}
	return desired, nil
}

func permissionChanges(before map[authorization.Permission]bool, after map[authorization.Permission]bool) (granted, revoked []string) {
	for _, permission := range authorization.AllPermissions() {
		switch {
		case !before[permission] && after[permission]:
			granted = append(granted, string(permission))
		case before[permission] && !after[permission]:
			revoked = append(revoked, string(permission))
		}
	}
	return granted, revoked
}

func samePermissions(left, right map[authorization.Permission]bool) bool {
	for _, permission := range authorization.AllPermissions() {
		if left[permission] != right[permission] {
			return false
		}
	}
	return true
}

func (a *App) handleRolePermissionsUpdate(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	target := authorization.Role(r.PathValue("role"))
	if !canEditRolePermissions(actor.Role, target) {
		writeErr(w, http.StatusForbidden, errors.New("permission denied"))
		return
	}
	var req struct {
		Permissions []string `json:"permissions"`
		Revision    int64    `json:"revision"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil || req.Revision < 0 {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	desired, err := validateDesiredPermissions(target, req.Permissions)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()
	current, err := loadRolePermissionState(tx, target)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if current.Revision != req.Revision {
		writeErr(w, http.StatusConflict, errors.New("role permissions changed elsewhere; reload and try again"))
		return
	}
	if actor.Role != authorization.RoleOwner {
		actorState, err := loadRolePermissionState(tx, actor.Role)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !actorState.Permissions[authorization.PermRolesManage] {
			writeErr(w, http.StatusForbidden, errors.New("permission denied"))
			return
		}
		for permission := range desired {
			if !actorState.Permissions[permission] && !current.Permissions[permission] {
				writeErr(w, http.StatusForbidden, fmt.Errorf("cannot grant permission %q", permission))
				return
			}
		}
	}
	if samePermissions(current.Permissions, desired) {
		if err := tx.Rollback(); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		payload, err := a.rolePermissionsPayload(actor.Role)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
		return
	}

	nextRevision := current.Revision + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if current.ProfileExists {
		result, err := tx.Exec(`UPDATE role_permission_profiles
			SET customized=1, revision=?, updated_at=?, updated_by=?
			WHERE role=? AND revision=?`, nextRevision, now, actor.ID, string(target), current.Revision)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		changed, err := result.RowsAffected()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if changed != 1 {
			writeErr(w, http.StatusConflict, errors.New("role permissions changed elsewhere; reload and try again"))
			return
		}
	} else {
		result, err := tx.Exec(`INSERT INTO role_permission_profiles
			(role, customized, revision, updated_at, updated_by) VALUES (?,1,?,?,?)
			ON CONFLICT(role) DO NOTHING`, string(target), nextRevision, now, actor.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		changed, err := result.RowsAffected()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if changed != 1 {
			writeErr(w, http.StatusConflict, errors.New("role permissions changed elsewhere; reload and try again"))
			return
		}
	}
	if _, err := tx.Exec(`DELETE FROM role_permissions WHERE role=?`, string(target)); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	for _, permission := range authorization.AllPermissions() {
		allowed := 0
		if desired[permission] {
			allowed = 1
		}
		if _, err := tx.Exec(`INSERT INTO role_permissions (role, permission, allowed) VALUES (?,?,?)`,
			string(target), string(permission), allowed); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}

	rows, err := tx.Query(`SELECT id FROM users WHERE role=?`, string(target))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rows.Close()
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	for _, userID := range userIDs {
		a.Hub.DisconnectUser(userID)
	}
	granted, revoked := permissionChanges(current.Permissions, desired)
	detail, _ := json.Marshal(map[string]any{
		"revision": nextRevision,
		"granted":  granted,
		"revoked":  revoked,
	})
	a.audit(actor.ID, actor.Username, "role_permissions_changed", string(target), string(detail), remoteIP(r))
	payload, err := a.rolePermissionsPayload(actor.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
