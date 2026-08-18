package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/monitoring"
	"github.com/Chansovisoth/Bonghos/internal/qrcode"
	"github.com/Chansovisoth/Bonghos/internal/runtime/systemd"
)

// routes builds the complete HTTP handler.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// --- session / auth -----------------------------------------------------
	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/passkey/begin", a.handlePasskeyLoginBegin)
	mux.HandleFunc("POST /api/auth/passkey/finish", a.handlePasskeyLoginFinish)
	mux.HandleFunc("POST /api/auth/logout", a.requireAuth(a.handleLogout))
	mux.HandleFunc("GET /api/auth/me", a.requireAuth(a.handleMe))
	mux.HandleFunc("GET /api/auth/csrf", a.handleCSRF)
	mux.HandleFunc("GET /api/invitations/{token}", a.handleInvitationCheck)
	mux.HandleFunc("POST /api/invitations/{token}/activate", a.handleInvitationActivate)
	mux.HandleFunc("POST /api/invitations/{token}/totp", a.handleInvitationTOTP)
	mux.HandleFunc("GET /api/passkeys", a.requireAuth(a.handlePasskeyList))
	mux.HandleFunc("POST /api/passkeys/register/begin", a.requireAuth(a.handlePasskeyRegisterBegin))
	mux.HandleFunc("POST /api/passkeys/register/finish", a.requireAuth(a.handlePasskeyRegisterFinish))
	mux.HandleFunc("PATCH /api/passkeys/{id}", a.requireAuth(a.handlePasskeyUpdate))
	mux.HandleFunc("DELETE /api/passkeys/{id}", a.requireAuth(a.handlePasskeyDelete))
	mux.HandleFunc("POST /api/account/reauth/password", a.requireAuth(a.handleAccountPasswordReauth))
	mux.HandleFunc("POST /api/account/reauth/passkey/begin", a.requireAuth(a.handleAccountPasskeyReauthBegin))
	mux.HandleFunc("POST /api/account/reauth/passkey/finish", a.requireAuth(a.handleAccountPasskeyReauthFinish))
	mux.HandleFunc("POST /api/account/password", a.requireAuth(a.handleAccountPasswordChange))
	mux.HandleFunc("POST /api/account/totp/begin", a.requireAuth(a.handleAccountTOTPBegin))
	mux.HandleFunc("POST /api/account/totp/finish", a.requireAuth(a.handleAccountTOTPFinish))
	mux.HandleFunc("GET /api/account/recovery-codes", a.requireAuth(a.handleRecoveryCodeList))
	mux.HandleFunc("POST /api/account/recovery-codes/regenerate", a.requireAuth(a.handleRecoveryCodeRegenerate))

	// --- users --------------------------------------------------------------
	mux.HandleFunc("GET /api/users", a.requireAnyPerm([]authorization.Permission{authorization.PermUsersManage, authorization.PermRolesManage}, a.handleUserList))
	mux.HandleFunc("POST /api/users/invite", a.requirePerm(authorization.PermUsersManage, a.handleUserInvite))
	mux.HandleFunc("POST /api/users/{id}/role", a.requirePerm(authorization.PermUsersManage, a.handleUserRole))
	mux.HandleFunc("POST /api/users/{id}/disable", a.requirePerm(authorization.PermUsersManage, a.handleUserDisable))
	mux.HandleFunc("POST /api/users/{id}/revoke-sessions", a.requirePerm(authorization.PermUsersManage, a.handleUserRevoke))
	mux.HandleFunc("DELETE /api/users/{id}", a.requirePerm(authorization.PermUsersManage, a.handleUserDelete))
	mux.HandleFunc("GET /api/roles/permissions", a.requirePerm(authorization.PermRolesManage, a.handleRolePermissions))
	mux.HandleFunc("PUT /api/roles/{role}/permissions", a.requirePerm(authorization.PermRolesManage, a.handleRolePermissionsUpdate))

	// --- notification bots -------------------------------------------------
	mux.HandleFunc("GET /api/bots", a.requirePerm(authorization.PermBotsManage, a.handleBotList))
	mux.HandleFunc("POST /api/bots", a.requirePerm(authorization.PermBotsManage, a.handleBotCreate))
	mux.HandleFunc("POST /api/bots/telegram/discover", a.requirePerm(authorization.PermBotsManage, a.handleBotTelegramDiscover))
	mux.HandleFunc("PATCH /api/bots/{id}", a.requirePerm(authorization.PermBotsManage, a.handleBotUpdate))
	mux.HandleFunc("DELETE /api/bots/{id}", a.requirePerm(authorization.PermBotsManage, a.handleBotDelete))
	mux.HandleFunc("POST /api/bots/{id}/test", a.requirePerm(authorization.PermBotsManage, a.handleBotTest))
	mux.HandleFunc("GET /api/bots/{id}/invite", a.requirePerm(authorization.PermBotsManage, a.handleBotInvite))
	mux.HandleFunc("POST /api/bots/{id}/telegram/discover", a.requirePerm(authorization.PermBotsManage, a.handleBotTelegramDiscoverExisting))
	mux.HandleFunc("GET /api/bots/{id}/telegram/destinations/{destination}/photo", a.requirePerm(authorization.PermBotsManage, a.handleBotTelegramGroupPhoto))

	// --- servers ------------------------------------------------------------
	mux.HandleFunc("GET /api/servers", a.requirePerm(authorization.PermServerView, a.handleServerList))
	mux.HandleFunc("POST /api/servers", a.requirePerm(authorization.PermImportManage, a.handleServerCreate))
	mux.HandleFunc("GET /api/servers/{id}", a.requirePerm(authorization.PermServerView, a.handleServerGet))
	mux.HandleFunc("PATCH /api/servers/{id}", a.requirePerm(authorization.PermConfigManage, a.handleServerUpdate))
	mux.HandleFunc("DELETE /api/servers/{id}", a.requirePerm(authorization.PermImportManage, a.handleServerDelete))
	mux.HandleFunc("POST /api/servers/{id}/select", a.requirePerm(authorization.PermConfigManage, a.handleServerSelect))
	mux.HandleFunc("POST /api/servers/{id}/duplicate", a.requirePerm(authorization.PermImportManage, a.handleServerDuplicate))
	mux.HandleFunc("POST /api/servers/{id}/world/reset", a.requirePerm(authorization.PermConfigManage, a.handleWorldReset))
	mux.HandleFunc("GET /api/servers/{id}/world.zip", a.requirePerm(authorization.PermFilesManage, a.handleWorldDownload))
	mux.HandleFunc("GET /api/servers/{id}/icon", a.requireAuth(a.handleIconGet))
	mux.HandleFunc("POST /api/servers/{id}/icon", a.requirePerm(authorization.PermIconManage, a.handleIconUpload))
	mux.HandleFunc("DELETE /api/servers/{id}/icon", a.requirePerm(authorization.PermIconManage, a.handleIconDelete))
	mux.HandleFunc("GET /api/servers/{id}/detect", a.requirePerm(authorization.PermConfigManage, a.handleServerDetect))
	mux.HandleFunc("POST /api/servers/slug-preview", a.requireAuth(a.handleSlugPreview))

	// --- imports ------------------------------------------------------------
	mux.HandleFunc("POST /api/imports/upload", a.requirePerm(authorization.PermImportManage, a.handleImportUpload))
	mux.HandleFunc("POST /api/imports/upload/start", a.requirePerm(authorization.PermImportManage, a.handleImportUploadStart))
	mux.HandleFunc("PUT /api/imports/upload/{id}/chunk", a.requirePerm(authorization.PermImportManage, a.handleImportUploadChunk))
	mux.HandleFunc("POST /api/imports/upload/{id}/finish", a.requirePerm(authorization.PermImportManage, a.handleImportUploadFinish))
	mux.HandleFunc("POST /api/imports/url", a.requirePerm(authorization.PermImportManage, a.handleImportURL))
	mux.HandleFunc("POST /api/imports/local-archive", a.requirePerm(authorization.PermImportManage, a.handleImportLocal))
	mux.HandleFunc("POST /api/imports/existing-directory", a.requirePerm(authorization.PermImportManage, a.handleImportExisting))
	mux.HandleFunc("GET /api/operations", a.requirePerm(authorization.PermServerView, a.handleOperationList))
	mux.HandleFunc("POST /api/operations/{id}/cancel", a.requirePerm(authorization.PermImportManage, a.handleOperationCancel))

	// --- lifecycle ----------------------------------------------------------
	mux.HandleFunc("POST /api/server/start", a.requirePerm(authorization.PermServerStart, a.handleStart))
	mux.HandleFunc("POST /api/server/stop", a.requirePerm(authorization.PermServerStop, a.handleStop))
	mux.HandleFunc("POST /api/server/restart", a.requirePerm(authorization.PermServerRestart, a.handleRestart))
	mux.HandleFunc("POST /api/server/force-stop", a.requirePerm(authorization.PermServerForceStop, a.handleForceStop))
	mux.HandleFunc("GET /api/server/status", a.requirePerm(authorization.PermServerView, a.handleStatus))
	mux.HandleFunc("GET /api/server/console/history", a.requirePerm(authorization.PermConsoleView, a.handleConsoleHistory))
	mux.HandleFunc("POST /api/server/command", a.requirePerm(authorization.PermConsoleUse, a.handleCommand))

	// --- players ------------------------------------------------------------
	mux.HandleFunc("GET /api/players", a.requirePerm(authorization.PermPlayersView, a.handlePlayerList))
	mux.HandleFunc("GET /api/players/avatar", a.requirePerm(authorization.PermPlayersView, a.handlePlayerAvatar))
	mux.HandleFunc("POST /api/players/action", a.requirePerm(authorization.PermPlayersManage, a.handlePlayerAction))

	// --- monitoring ---------------------------------------------------------
	mux.HandleFunc("GET /api/metrics", a.requirePerm(authorization.PermPerformanceView, a.handleMetrics))
	mux.HandleFunc("GET /api/metrics/config", a.requirePerm(authorization.PermPerformanceView, a.handleMetricsConfig))
	mux.HandleFunc("GET /api/metrics/storage", a.requirePerm(authorization.PermPerformanceView, a.handleMetricsStorage))
	mux.HandleFunc("GET /api/metrics/internet", a.requirePerm(authorization.PermPerformanceView, a.handleMetricsInternet))
	mux.HandleFunc("POST /api/metrics/internet/refresh", a.requirePerm(authorization.PermPerformanceView, a.handleRefreshMetricsInternet))
	mux.HandleFunc("POST /api/metrics/internet/speed-test", a.requirePerm(authorization.PermPerformanceTest, a.handleInternetSpeedTest))
	mux.HandleFunc("GET /api/overview", a.requirePerm(authorization.PermServerView, a.handleOverview))

	// --- files --------------------------------------------------------------
	mux.HandleFunc("GET /api/files", a.requirePerm(authorization.PermFilesManage, a.handleFileList))
	mux.HandleFunc("GET /api/files/content", a.requirePerm(authorization.PermFilesManage, a.handleFileRead))
	mux.HandleFunc("POST /api/files/content", a.requirePerm(authorization.PermFilesManage, a.handleFileWrite))
	mux.HandleFunc("POST /api/files/create", a.requirePerm(authorization.PermFilesManage, a.handleFileCreate))
	mux.HandleFunc("POST /api/files/mkdir", a.requirePerm(authorization.PermFilesManage, a.handleFileMkdir))
	mux.HandleFunc("POST /api/files/copy", a.requirePerm(authorization.PermFilesManage, a.handleFileCopy))
	mux.HandleFunc("POST /api/files/move", a.requirePerm(authorization.PermFilesManage, a.handleFileMove))
	mux.HandleFunc("POST /api/files/rename", a.requirePerm(authorization.PermFilesManage, a.handleFileRename))
	mux.HandleFunc("POST /api/files/delete", a.requirePerm(authorization.PermFilesManage, a.handleFileDelete))
	mux.HandleFunc("POST /api/files/upload", a.requirePerm(authorization.PermFilesManage, a.handleFileUpload))
	mux.HandleFunc("GET /api/files/preview", a.requirePerm(authorization.PermFilesManage, a.handleFilePreview))
	mux.HandleFunc("GET /api/files/download", a.requirePerm(authorization.PermFilesManage, a.handleFileDownload))

	// --- configuration ------------------------------------------------------
	mux.HandleFunc("GET /api/configuration", a.requirePerm(authorization.PermConfigManage, a.handleConfigGet))
	mux.HandleFunc("POST /api/configuration/jvm", a.requirePerm(authorization.PermConfigManage, a.handleConfigJVM))
	mux.HandleFunc("POST /api/configuration/startup-script", a.requirePerm(authorization.PermConfigManage, a.handleConfigStartup))
	mux.HandleFunc("POST /api/configuration/property", a.requirePerm(authorization.PermConfigManage, a.handleConfigProperty))
	mux.HandleFunc("POST /api/configuration/eula", a.requirePerm(authorization.PermConfigManage, a.handleConfigEULA))
	mux.HandleFunc("GET /api/java", a.requirePerm(authorization.PermConfigManage, a.handleJavaList))

	// --- backups ------------------------------------------------------------
	mux.HandleFunc("GET /api/backups", a.requirePerm(authorization.PermBackupsView, a.handleBackupList))
	mux.HandleFunc("GET /api/backups/storage", a.requirePerm(authorization.PermBackupsView, a.handleBackupStorage))
	mux.HandleFunc("POST /api/backups", a.requirePerm(authorization.PermBackupsCreate, a.handleBackupCreate))
	mux.HandleFunc("POST /api/backups/{id}/verify", a.requirePerm(authorization.PermBackupsCreate, a.handleBackupVerify))
	mux.HandleFunc("POST /api/backups/{id}/restore", a.requirePerm(authorization.PermBackupsRestore, a.handleBackupRestore))
	mux.HandleFunc("POST /api/backups/{id}/protect", a.requirePerm(authorization.PermBackupsCreate, a.handleBackupProtect))
	mux.HandleFunc("DELETE /api/backups/{id}", a.requirePerm(authorization.PermBackupsCreate, a.handleBackupDelete))

	// --- schedules ----------------------------------------------------------
	mux.HandleFunc("GET /api/schedules", a.requirePerm(authorization.PermSchedulesManage, a.handleScheduleList))
	mux.HandleFunc("POST /api/schedules", a.requirePerm(authorization.PermSchedulesManage, a.handleScheduleCreate))
	mux.HandleFunc("PATCH /api/schedules/{id}", a.requirePerm(authorization.PermSchedulesManage, a.handleScheduleUpdate))
	mux.HandleFunc("DELETE /api/schedules/{id}", a.requirePerm(authorization.PermSchedulesManage, a.handleScheduleDelete))
	mux.HandleFunc("POST /api/schedules/{id}/run", a.requirePerm(authorization.PermSchedulesManage, a.handleScheduleRunNow))
	mux.HandleFunc("GET /api/schedules/{id}/history", a.requirePerm(authorization.PermSchedulesManage, a.handleScheduleHistory))

	// --- activity / host ----------------------------------------------------
	mux.HandleFunc("GET /api/activity", a.requirePerm(authorization.PermActivityView, a.handleActivity))
	// The server's own timeline. Unlike the audit trail this is what the
	// server did rather than what a person did, so anyone who can see the
	// dashboard can see it.
	mux.HandleFunc("GET /api/events", a.requirePerm(authorization.PermServerView, a.handleEvents))
	mux.HandleFunc("GET /api/host", a.requirePerm(authorization.PermHostView, a.handleHost))
	mux.HandleFunc("GET /api/version", a.handleVersion)

	// --- websocket ----------------------------------------------------------
	mux.HandleFunc("GET /api/ws", a.handleWS)

	// --- frontend -----------------------------------------------------------
	mux.Handle("/", a.spaHandler())

	return a.secureHeaders(a.csrfProtect(mux))
}

// ---------------------------------------------------------------------------
// auth
// ---------------------------------------------------------------------------

func (a *App) handleCSRF(w http.ResponseWriter, r *http.Request) {
	tok := a.issueCSRF(w, r)
	writeJSON(w, 200, map[string]string{"csrf": tok})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	u, err := a.Auth.Login(req.Username, req.Password, req.Code, remoteIP(r))
	if err != nil {
		status := 401
		if errors.Is(err, auth.ErrRateLimited) {
			status = 429
		}
		writeErr(w, status, err)
		return
	}
	sess, err := a.Auth.CreateSession(u.ID, remoteIP(r), r.UserAgent())
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	a.setSessionCookie(w, r, sess.Token, sess.ExpiresAt)
	a.issueCSRF(w, r)
	a.audit(u.ID, u.Username, "login_success", "", "", remoteIP(r))
	writeJSON(w, 200, map[string]any{
		"id": u.ID, "username": u.Username, "role": u.Role,
		"permissions": a.permissions(u.Role),
	})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	u, _ := a.sessionUser(r)
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = a.Auth.RevokeSession(c.Value)
	}
	if u != nil {
		a.Hub.DisconnectUser(u.ID)
	}
	a.clearSessionCookie(w, r)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	writeJSON(w, 200, map[string]any{
		"id": u.ID, "username": u.Username, "role": u.Role,
		"permissions": a.permissions(u.Role),
	})
}

// invitation activation ------------------------------------------------------

func (a *App) handleInvitationCheck(w http.ResponseWriter, r *http.Request) {
	role, _, err := a.Auth.CheckInvitation(r.PathValue("token"))
	if err != nil {
		writeErr(w, 404, errors.New("invitation is invalid or expired"))
		return
	}
	writeJSON(w, 200, map[string]any{"role": role})
}

// handleInvitationTOTP issues a fresh TOTP secret for the enrollment page.
func (a *App) handleInvitationTOTP(w http.ResponseWriter, r *http.Request) {
	if _, _, err := a.Auth.CheckInvitation(r.PathValue("token")); err != nil {
		writeErr(w, 404, errors.New("invitation is invalid or expired"))
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	_ = readJSON(r, &req, 1<<14)
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	uri := auth.TOTPProvisioningURI(req.Username, secret)
	out := map[string]string{"secret": secret, "uri": uri}
	// The QR is rendered server-side as SVG so the browser needs no QR library
	// and the enrolment page keeps working offline. Failure is not fatal: the
	// activation page falls back to the secret and URI it already shows.
	if svg, err := qrcode.SVG(uri, 4); err == nil {
		out["qr_svg"] = svg
	}
	writeJSON(w, 200, out)
}

func (a *App) handleInvitationActivate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Secret   string `json:"totp_secret"`
		Code     string `json:"totp_code"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	u, recovery, err := a.Auth.ActivateInvitation(r.PathValue("token"), req.Username, req.Password, req.Secret, req.Code)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "invitation_used", "", "", remoteIP(r))
	writeJSON(w, 200, map[string]any{"username": u.Username, "recovery_codes": recovery})
}

// ---------------------------------------------------------------------------
// users
// ---------------------------------------------------------------------------

func (a *App) handleUserList(w http.ResponseWriter, r *http.Request) {
	users, err := a.Auth.ListUsers()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, users)
}

func (a *App) handleUserInvite(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Role string `json:"role"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	role := authorization.Role(req.Role)
	switch role {
	case authorization.RoleAdmin, authorization.RoleMember, authorization.RoleViewer:
	default:
		writeErr(w, 400, errors.New("role must be Admin, Member or Viewer"))
		return
	}
	if !authorization.CanInviteRole(u.Role, role) {
		writeErr(w, 403, errors.New("cannot invite a user at that role"))
		return
	}
	inv, err := a.Auth.CreateInvitation(u.ID, role, 72*time.Hour)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "invitation_created", string(role), "", remoteIP(r))
	writeJSON(w, 200, map[string]any{
		"token":           inv.Token,
		"role":            inv.Role,
		"expires_at":      inv.ExpiresAt.Format(time.RFC3339),
		"activation_path": "/activate/" + inv.Token,
	})
}

func (a *App) targetUser(r *http.Request) (*auth.User, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	return a.Auth.UserByID(id)
}

func (a *App) handleUserRole(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	target, err := a.targetUser(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	newRole := authorization.Role(req.Role)
	if !authorization.CanManageUser(actor.Role, actor.ID, target.Role, target.ID, &newRole) {
		writeErr(w, 403, errors.New("permission denied"))
		return
	}
	if err := a.Auth.SetRole(target.ID, newRole); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.Hub.DisconnectUser(target.ID)
	a.audit(actor.ID, actor.Username, "role_changed", target.Username, string(newRole), remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleUserDisable(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	target, err := a.targetUser(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if !authorization.CanManageUser(actor.Role, actor.ID, target.Role, target.ID, nil) || actor.ID == target.ID {
		writeErr(w, 403, errors.New("permission denied"))
		return
	}
	if err := a.Auth.SetDisabled(target.ID, req.Disabled); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.Hub.DisconnectUser(target.ID)
	a.audit(actor.ID, actor.Username, "account_disabled", target.Username, fmt.Sprint(req.Disabled), remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleUserRevoke(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	target, err := a.targetUser(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	if !authorization.CanManageUser(actor.Role, actor.ID, target.Role, target.ID, nil) && actor.ID != target.ID {
		writeErr(w, 403, errors.New("permission denied"))
		return
	}
	if err := a.Auth.RevokeAllSessions(target.ID); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.Hub.DisconnectUser(target.ID)
	a.audit(actor.ID, actor.Username, "sessions_revoked", target.Username, "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	target, err := a.targetUser(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	if !authorization.CanManageUser(actor.Role, actor.ID, target.Role, target.ID, nil) || actor.ID == target.ID {
		writeErr(w, 403, errors.New("permission denied"))
		return
	}
	if err := a.Auth.DeleteUser(target.ID); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.Hub.DisconnectUser(target.ID)
	a.audit(actor.ID, actor.Username, "user_deleted", target.Username, "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------------------------------------------------------------------------
// servers
// ---------------------------------------------------------------------------

func (a *App) handleServerList(w http.ResponseWriter, r *http.Request) {
	list, err := a.Instances.List()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	// Older imports may predate version detection. Refresh only metadata that
	// can be confirmed from the pack and persist changes so known loaders do
	// not need to be rescanned on later requests.
	for _, inst := range list {
		a.refreshServerMetadata(inst)
	}
	activeID, _ := a.Instances.ActiveID()
	writeJSON(w, 200, map[string]any{"servers": list, "active_id": activeID})
}

func (a *App) refreshServerMetadata(inst *instance.Instance) {
	if inst.MinecraftVersion != "" && inst.Modloader != "" && inst.ModloaderVersion != "" {
		return
	}
	meta, err := minecraft.DetectServerMetadata(inst.AbsoluteDir(a.Home))
	if err != nil {
		return
	}
	changed := false
	if meta.MinecraftVersion != "" && meta.MinecraftVersion != inst.MinecraftVersion {
		inst.MinecraftVersion = meta.MinecraftVersion
		changed = true
	}
	if meta.Modloader != "" && meta.Modloader != inst.Modloader {
		inst.Modloader = meta.Modloader
		changed = true
	}
	if meta.ModloaderVersion != "" && meta.ModloaderVersion != inst.ModloaderVersion {
		inst.ModloaderVersion = meta.ModloaderVersion
		changed = true
	}
	if changed {
		_ = a.Instances.Update(inst)
	}
}

func (a *App) handleSlugPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	writeJSON(w, 200, map[string]string{"slug": instance.GenerateSlug(req.Name)})
}

// handleServerCreate creates an empty project shell (used before import).
func (a *App) handleServerCreate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		DisplayName string `json:"display_name"`
		Slug        string `json:"slug"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		writeErr(w, 400, errors.New("a project display name is required"))
		return
	}
	slug := req.Slug
	if slug == "" {
		slug = instance.GenerateSlug(req.DisplayName)
	}
	if err := instance.ValidateSlug(slug); err != nil {
		writeErr(w, 400, err)
		return
	}
	inst := &instance.Instance{
		Slug: slug, DisplayName: strings.TrimSpace(req.DisplayName),
		ServerType: "minecraft-java-modded", SourceType: "archive-upload",
	}
	if err := a.Instances.Create(inst); err != nil {
		writeErr(w, 409, err)
		return
	}
	if err := os.MkdirAll(inst.AbsoluteDir(a.Home), 0o755); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "project_created", slug, "", remoteIP(r))
	writeJSON(w, 200, inst)
}

func (a *App) pathInstance(r *http.Request) (*instance.Instance, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return nil, errors.New("invalid server id")
	}
	return a.Instances.ByID(id)
}

func (a *App) handleServerGet(w http.ResponseWriter, r *http.Request) {
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	writeJSON(w, 200, inst)
}

func (a *App) handleServerUpdate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	previousDisplayName := inst.DisplayName
	var req struct {
		DisplayName                 *string `json:"display_name"`
		StartupScript               *string `json:"startup_script"`
		JavaSelection               *string `json:"java_selection"`
		AutostartEnabled            *bool   `json:"autostart_enabled"`
		BootDelaySeconds            *int    `json:"boot_delay_seconds"`
		RecoverAfterUncleanShutdown *bool   `json:"recover_after_unclean_shutdown"`
		RestartPolicy               *string `json:"restart_policy"`
		RestartDelaySeconds         *int    `json:"restart_delay_seconds"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName == "" {
			writeErr(w, 400, errors.New("display name is required"))
			return
		}
		if utf8.RuneCountInString(displayName) > 120 {
			writeErr(w, 400, errors.New("display name must be 120 characters or fewer"))
			return
		}
		inst.DisplayName = displayName
	}
	if req.StartupScript != nil {
		rel := filepath.ToSlash(filepath.Clean(*req.StartupScript))
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			writeErr(w, 400, errors.New("startup script must be inside the project"))
			return
		}
		inst.StartupScript = rel
	}
	if req.JavaSelection != nil {
		if _, err := minecraft.ResolveJava(*req.JavaSelection); err != nil {
			writeErr(w, 400, err)
			return
		}
		inst.JavaSelection = *req.JavaSelection
	}
	if req.AutostartEnabled != nil {
		inst.AutostartEnabled = *req.AutostartEnabled
	}
	if req.BootDelaySeconds != nil && *req.BootDelaySeconds >= 0 {
		inst.BootDelaySeconds = *req.BootDelaySeconds
	}
	if req.RecoverAfterUncleanShutdown != nil {
		inst.RecoverAfterUncleanShutdown = *req.RecoverAfterUncleanShutdown
	}
	if req.RestartPolicy != nil {
		switch *req.RestartPolicy {
		case "never", "on-failure", "always":
			inst.RestartPolicy = *req.RestartPolicy
		default:
			writeErr(w, 400, errors.New("restart policy must be never, on-failure or always"))
			return
		}
	}
	if req.RestartDelaySeconds != nil && *req.RestartDelaySeconds >= 0 {
		inst.RestartDelaySeconds = *req.RestartDelaySeconds
	}
	if err := a.Instances.Update(inst); err != nil {
		writeErr(w, 500, err)
		return
	}
	detail := ""
	if inst.DisplayName != previousDisplayName {
		detail = fmt.Sprintf("display_name=%q -> %q", previousDisplayName, inst.DisplayName)
	}
	a.audit(u.ID, u.Username, "project_updated", inst.Slug, detail, remoteIP(r))
	writeJSON(w, 200, inst)
}

func (a *App) handleServerDelete(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	activeID, _ := a.Instances.ActiveID()
	if activeID == inst.ID && a.Runner.Online() {
		writeErr(w, 409, errors.New("cannot delete the running project; stop the server first"))
		return
	}
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	release, err := a.OpLock.Acquire("project-delete")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	defer release()

	if deleteFiles && !inst.ExternalDirectory {
		dir := inst.AbsoluteDir(a.Home)
		within := strings.HasPrefix(dir, filepath.Join(a.Home, config.DirServers)+string(os.PathSeparator))
		if within {
			_ = os.RemoveAll(dir)
		}
	}
	if err := a.Instances.Delete(inst.ID); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "project_deleted", inst.Slug, fmt.Sprintf("files_deleted=%v", deleteFiles), remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleServerSelect(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	if a.Runner.Online() {
		writeErr(w, 409, errors.New("stop the running server before switching the active project"))
		return
	}
	if err := a.Instances.SetActive(inst.ID); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "project_selected", inst.Slug, "", remoteIP(r))
	a.broadcastStatus()
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// handleServerDetect runs startup/JVM detection for the setup UI.
func (a *App) handleServerDetect(w http.ResponseWriter, r *http.Request) {
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	dir := inst.AbsoluteDir(a.Home)
	scripts, err := minecraft.DetectStartupScripts(dir, 3)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	var jvm *minecraft.JVMConfig
	if inst.StartupScript != "" {
		jvm, _ = minecraft.DetectJVMConfig(dir, inst.StartupScript)
	} else if len(scripts) > 0 {
		jvm, _ = minecraft.DetectJVMConfig(dir, scripts[0].Path)
	}
	metadata, _ := minecraft.DetectServerMetadata(dir)
	writeJSON(w, 200, map[string]any{
		"scripts":  scripts,
		"jvm":      jvm,
		"java":     minecraft.DiscoverJava(),
		"metadata": metadata,
	})
}

// ---------------------------------------------------------------------------
// lifecycle
// ---------------------------------------------------------------------------

func (a *App) handleStart(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := a.Runner.Start(ctx); err != nil {
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "server_start", "", "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleStop(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(a.Cfg.GracefulStopSeconds+60)*time.Second)
	defer cancel()
	if err := a.Runner.Stop(ctx); err != nil {
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "server_stop", "", "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleRestart(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(a.Cfg.GracefulStopSeconds+120)*time.Second)
	defer cancel()
	if err := a.Runner.Restart(ctx); err != nil {
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "server_restart", "", "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleForceStop(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil || !req.Confirm {
		writeErr(w, 400, errors.New("force stop requires explicit confirmation"))
		return
	}
	if err := a.Runner.ForceStop(); err != nil {
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "server_force_stop", "", "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	st, ps := a.Runner.State()
	out := map[string]any{"state": st}
	if ps != nil {
		out["detail"] = ps
	}
	writeJSON(w, 200, out)
}

func (a *App) handleCommand(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Command string `json:"command"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	cmd := strings.TrimSpace(req.Command)
	if cmd == "" || len(cmd) > 1024 || strings.ContainsAny(cmd, "\n\r") {
		writeErr(w, 400, errors.New("invalid console command"))
		return
	}
	if err := a.Runner.SendCommand(cmd); err != nil {
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "console_command", "", cmd, remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------------------------------------------------------------------------
// overview / metrics / host
// ---------------------------------------------------------------------------

func (a *App) handleOverview(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	st, ps := a.Runner.State()
	out := map[string]any{"state": st, "version": Version}
	if ip := localLANIPv4(a.Cfg.BindAddress); ip != "" {
		out["lan_ip"] = ip
	}
	if ps != nil {
		out["supervisor"] = ps
	}
	if inst, err := a.activeInstance(); err == nil {
		out["instance"] = inst
		dir := inst.AbsoluteDir(a.Home)
		if props, err := minecraft.ReadProperties(dir); err == nil {
			out["motd"] = props["motd"]
			out["port"] = props["server-port"]
			out["max_players"] = props["max-players"]
		}
	}
	if a.hasPermission(u.Role, authorization.PermPerformanceView) {
		out["sample"] = a.collectSample()
	}

	// last backup
	if inst, err := a.activeInstance(); err == nil {
		if a.hasPermission(u.Role, authorization.PermBackupsView) {
			if list, err := a.Backups.List(inst.ID); err == nil && len(list) > 0 {
				out["last_backup"] = list[0]
			}
		}
		if a.hasPermission(u.Role, authorization.PermSchedulesManage) {
			scheds, err := a.Sched.List(inst.ID)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			var next *time.Time
			for _, s := range scheds {
				if !s.Enabled || s.NextRunAt == "" {
					continue
				}
				if t, err := time.Parse(time.RFC3339, s.NextRunAt); err == nil {
					if next == nil || t.Before(*next) {
						next = &t
					}
				}
			}
			if next != nil {
				out["next_schedule_at"] = next.Format(time.RFC3339)
			}
		}
	}
	writeJSON(w, 200, out)
}

func localLANIPv4(bindAddress string) string {
	if ip := net.ParseIP(bindAddress); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	if conn, err := net.DialTimeout("udp", "8.8.8.8:80", 100*time.Millisecond); err == nil {
		defer conn.Close()
		if address, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			if ipv4 := address.IP.To4(); ipv4 != nil && !ipv4.IsLoopback() && !ipv4.IsUnspecified() {
				return ipv4.String()
			}
		}
	}

	var fallback string
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil || ip.IsLoopback() {
				continue
			}
			ipv4 := ip.To4()
			if ipv4 == nil || ipv4.IsLinkLocalUnicast() {
				continue
			}
			if ipv4.IsPrivate() {
				return ipv4.String()
			}
			if fallback == "" {
				fallback = ipv4.String()
			}
		}
	}
	return fallback
}

func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	hours := 1
	if h, err := strconv.Atoi(r.URL.Query().Get("hours")); err == nil && h > 0 && h <= 24*14 {
		hours = h
	}
	samples, err := monitoring.History(a.DB, time.Now().Add(-time.Duration(hours)*time.Hour), 5000)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, samples)
}

func (a *App) handleMetricsConfig(w http.ResponseWriter, r *http.Request) {
	interval := a.Cfg.MetricsIntervalSec
	if interval < 2 {
		interval = 10
	}
	writeJSON(w, 200, map[string]int{"interval_seconds": interval})
}

func (a *App) handleMetricsStorage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.collectStorageSnapshot())
}

func (a *App) handleHost(w http.ResponseWriter, r *http.Request) {
	memT, memA := monitoring.HostMemory()
	storage := a.cachedStorageSnapshot()
	out := map[string]any{
		"bind_address":      a.Cfg.BindAddress,
		"port":              a.Cfg.Port,
		"session_hours":     a.Cfg.SessionHours,
		"home":              a.Home,
		"mem_total":         memT,
		"mem_available":     memA,
		"disk_total":        storage.DiskTotal,
		"disk_free":         storage.DiskFree,
		"load1":             monitoring.LoadAvg(),
		"systemd":           systemd.Available(),
		"service_bonghos":   systemd.State(systemd.ServiceControlPlane),
		"service_minecraft": systemd.State(systemd.ServiceMinecraft),
		"version":           Version,
		"note":              "Local listening does not prove public accessibility. Port forwarding, firewalls and tunnels remain your responsibility.",
	}
	writeJSON(w, 200, out)
}

func (a *App) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"version": Version})
}

// ---------------------------------------------------------------------------
// activity
// ---------------------------------------------------------------------------

// handleEvents returns the recent server timeline: starting, loading mods,
// ready, crashed, restarted, backups. It answers "what is happening now" and
// "what happened recently" without the operator reading journalctl.
func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	var instID int64
	if inst, err := a.activeInstance(); err == nil {
		instID = inst.ID
	}
	events, err := a.listEvents(instID, limit)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"events": events})
}

func (a *App) handleActivity(w http.ResponseWriter, r *http.Request) {
	limit := 100
	rows, err := a.DB.Query(`SELECT id, user_id, username, action, target, detail, remote_addr, occurred_at
		FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var userID *int64
		var username, action, target, detail, remote, at string
		if rows.Scan(&id, &userID, &username, &action, &target, &detail, &remote, &at) != nil {
			continue
		}
		out = append(out, map[string]any{
			"id": id, "username": username, "action": action,
			"target": target, "detail": detail, "remote_addr": remote, "at": at,
		})
	}
	writeJSON(w, 200, out)
}

func websocketTopicAllowed(role authorization.Role, topic string) bool {
	permission, ok := websocketTopicPermission(topic)
	return ok && authorization.Has(role, permission)
}

func websocketTopicPermission(topic string) (authorization.Permission, bool) {
	switch topic {
	case "overview", "servers":
		return authorization.PermServerView, true
	case "overview_performance":
		return authorization.PermPerformanceView, true
	case "performance":
		return authorization.PermPerformanceView, true
	case "console":
		return authorization.PermConsoleView, true
	case "console_use":
		return authorization.PermConsoleUse, true
	case "players":
		return authorization.PermPlayersView, true
	case "backups":
		return authorization.PermBackupsView, true
	case "schedules":
		return authorization.PermSchedulesManage, true
	case "activity":
		return authorization.PermActivityView, true
	default:
		return "", false
	}
}

func (a *App) websocketTopicAllowed(role authorization.Role, topic string) bool {
	permission, ok := websocketTopicPermission(topic)
	return ok && a.hasPermission(role, permission)
}

// handleWS upgrades the authenticated websocket.
func (a *App) handleWS(w http.ResponseWriter, r *http.Request) {
	u, err := a.sessionUser(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	permissionRevision, err := a.rolePermissionRevision(u.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	canUse := func(topic string) bool { return a.websocketTopicAllowed(u.Role, topic) }
	cookie, _ := r.Cookie(sessionCookie)
	stillAuthorized := func() bool {
		if cookie == nil || cookie.Value == "" {
			return false
		}
		current, err := a.Auth.ValidateSession(cookie.Value)
		if err != nil || current.ID != u.ID || current.Role != u.Role {
			return false
		}
		currentRevision, err := a.rolePermissionRevision(current.Role)
		return err == nil && currentRevision == permissionRevision
	}
	onCommand := func(cmd string) {
		if !a.hasPermission(u.Role, authorization.PermConsoleUse) {
			return
		}
		cmd = strings.TrimSpace(cmd)
		if cmd == "" || len(cmd) > 1024 || strings.ContainsAny(cmd, "\n\r") {
			return
		}
		if err := a.Runner.SendCommand(cmd); err == nil {
			a.audit(u.ID, u.Username, "console_command", "", cmd, remoteIP(r))
		}
	}
	a.Hub.Serve(w, r, u.ID, canUse, stillAuthorized, onCommand)
}
