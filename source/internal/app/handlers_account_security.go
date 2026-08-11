package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/qrcode"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	accountActionLifetime  = 5 * time.Minute
	totpEnrollmentLifetime = 10 * time.Minute
)

type accountAction struct {
	UserID    int64
	Purpose   string
	SessionID string
	ExpiresAt time.Time
}

type totpEnrollment struct {
	UserID    int64
	Secret    string
	SessionID string
	ExpiresAt time.Time
}

func validAccountPurpose(purpose string) bool {
	switch purpose {
	case "change_password", "change_totp", "regenerate_recovery_codes", "remove_passkey":
		return true
	default:
		return false
	}
}

func (a *App) putAccountAction(userID int64, purpose, sessionToken string) (string, error) {
	if !validAccountPurpose(purpose) {
		return "", errors.New("invalid account action")
	}
	if sessionToken == "" {
		return "", errors.New("authentication required")
	}
	token, _, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	if a.accountActions == nil {
		a.accountActions = make(map[string]accountAction)
	}
	for id, action := range a.accountActions {
		if now.After(action.ExpiresAt) {
			delete(a.accountActions, id)
		}
	}
	if len(a.accountActions) >= 512 {
		return "", errors.New("too many verification requests; try again shortly")
	}
	a.accountActions[token] = accountAction{
		UserID: userID, Purpose: purpose, SessionID: auth.HashToken(sessionToken),
		ExpiresAt: now.Add(accountActionLifetime),
	}
	return token, nil
}

func (a *App) takeAccountAction(token string, userID int64, purpose, sessionToken string) error {
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	action, ok := a.accountActions[token]
	delete(a.accountActions, token)
	if !ok || action.UserID != userID || action.Purpose != purpose ||
		action.SessionID != auth.HashToken(sessionToken) || time.Now().After(action.ExpiresAt) {
		return errors.New("identity verification expired or is invalid")
	}
	return nil
}

func (a *App) putTOTPEnrollment(userID int64, secret, sessionToken string) (string, error) {
	token, _, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	if a.totpEnrollments == nil {
		a.totpEnrollments = make(map[string]totpEnrollment)
	}
	for id, enrollment := range a.totpEnrollments {
		if now.After(enrollment.ExpiresAt) {
			delete(a.totpEnrollments, id)
		}
	}
	if len(a.totpEnrollments) >= 512 {
		return "", errors.New("too many authenticator setup requests; try again shortly")
	}
	a.totpEnrollments[token] = totpEnrollment{
		UserID: userID, Secret: secret, SessionID: auth.HashToken(sessionToken),
		ExpiresAt: now.Add(totpEnrollmentLifetime),
	}
	return token, nil
}

// verifyAndTakeTOTPEnrollment leaves an enrollment available after a mistyped
// code, but consumes it atomically once a valid new-authenticator code arrives.
func (a *App) verifyAndTakeTOTPEnrollment(token string, userID int64, code, sessionToken string) (string, error) {
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	enrollment, ok := a.totpEnrollments[token]
	if !ok || enrollment.UserID != userID || enrollment.SessionID != auth.HashToken(sessionToken) ||
		time.Now().After(enrollment.ExpiresAt) {
		delete(a.totpEnrollments, token)
		return "", errors.New("authenticator setup expired or is invalid")
	}
	if !auth.VerifyTOTP(enrollment.Secret, code, time.Now()) {
		return "", errors.New("authentication code did not match the new authenticator")
	}
	delete(a.totpEnrollments, token)
	return enrollment.Secret, nil
}

func currentSessionToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return "", errors.New("authentication required")
	}
	return cookie.Value, nil
}

func (a *App) handleAccountPasswordReauth(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Purpose  string `json:"purpose"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil || !validAccountPurpose(req.Purpose) {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	verified, err := a.Auth.Login(u.Username, req.Password, req.Code, remoteIP(r))
	if err != nil || verified.ID != u.ID {
		status := http.StatusForbidden
		if errors.Is(err, auth.ErrRateLimited) {
			status = http.StatusTooManyRequests
		}
		writeErr(w, status, auth.ErrInvalidCredentials)
		return
	}
	sessionToken, sessionErr := currentSessionToken(r)
	if sessionErr != nil {
		writeErr(w, http.StatusUnauthorized, sessionErr)
		return
	}
	token, err := a.putAccountAction(u.ID, req.Purpose, sessionToken)
	if err != nil {
		writeErr(w, http.StatusTooManyRequests, err)
		return
	}
	a.audit(u.ID, u.Username, "account_reauthenticated", req.Purpose, "password and second factor", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"action_token": token})
}

func (a *App) handleAccountPasskeyReauthBegin(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Purpose string `json:"purpose"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil || !validAccountPurpose(req.Purpose) {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	sessionToken, err := currentSessionToken(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err)
		return
	}
	wa, origin, rpID, err := requestWebAuthn(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	passkeyUser, err := a.Auth.PasskeyUser(u.ID, rpID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if len(passkeyUser.Credentials) == 0 {
		writeErr(w, http.StatusBadRequest, errors.New("no passkey is registered for this panel address"))
		return
	}
	assertion, session, err := wa.BeginLogin(passkeyUser,
		webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not start passkey verification"))
		return
	}
	flowToken, err := a.putPasskeyFlow(passkeyFlow{
		Kind: "reauth", UserID: u.ID, Purpose: req.Purpose, SessionID: auth.HashToken(sessionToken), RPID: rpID, Origin: origin,
		Session: *session, ExpiresAt: time.Now().Add(passkeyFlowLifetime),
	})
	if err != nil {
		writeErr(w, http.StatusTooManyRequests, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flow": flowToken, "options": assertion})
}

func (a *App) handleAccountPasskeyReauthFinish(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	sessionToken, sessionErr := currentSessionToken(r)
	if sessionErr != nil {
		writeErr(w, http.StatusUnauthorized, sessionErr)
		return
	}
	wa, origin, rpID, err := requestWebAuthn(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	flow, err := a.takePasskeyFlow(r.URL.Query().Get("flow"), "reauth", origin, rpID, u.ID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if flow.SessionID != auth.HashToken(sessionToken) {
		writeErr(w, http.StatusBadRequest, errors.New("passkey request belongs to another session"))
		return
	}
	passkeyUser, err := a.Auth.PasskeyUser(u.ID, rpID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, errors.New("passkey verification failed"))
		return
	}
	credential, err := wa.FinishLogin(passkeyUser, flow.Session, r)
	if err != nil {
		a.Logf("passkey reauthentication failed for user %d: %v", u.ID, err)
		writeErr(w, http.StatusUnauthorized, errors.New("passkey verification failed"))
		return
	}
	if err := a.Auth.UpdatePasskeyCredential(u.ID, rpID, credential); err != nil {
		writeErr(w, http.StatusUnauthorized, errors.New("passkey verification failed"))
		return
	}
	token, err := a.putAccountAction(u.ID, flow.Purpose, sessionToken)
	if err != nil {
		writeErr(w, http.StatusTooManyRequests, err)
		return
	}
	a.audit(u.ID, u.Username, "account_reauthenticated", flow.Purpose, "passkey", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"action_token": token})
}

func (a *App) handleAccountPasswordChange(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		ActionToken string `json:"action_token"`
		NewPassword string `json:"new_password"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	sessionToken, err := currentSessionToken(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err)
		return
	}
	if err := a.takeAccountAction(req.ActionToken, u.ID, "change_password", sessionToken); err != nil {
		writeErr(w, http.StatusForbidden, err)
		return
	}
	if err := a.Auth.ChangePassword(u.ID, sessionToken, req.NewPassword); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	a.Hub.DisconnectUser(u.ID)
	a.audit(u.ID, u.Username, "password_changed", u.Username, "other sessions revoked", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleAccountTOTPBegin(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		ActionToken string `json:"action_token"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	sessionToken, err := currentSessionToken(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err)
		return
	}
	if err := a.takeAccountAction(req.ActionToken, u.ID, "change_totp", sessionToken); err != nil {
		writeErr(w, http.StatusForbidden, err)
		return
	}
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	setupToken, err := a.putTOTPEnrollment(u.ID, secret, sessionToken)
	if err != nil {
		writeErr(w, http.StatusTooManyRequests, err)
		return
	}
	uri := auth.TOTPProvisioningURI(u.Username, secret)
	out := map[string]string{"setup_token": setupToken, "secret": secret, "uri": uri}
	if svg, err := qrcode.SVG(uri, 4); err == nil {
		out["qr_svg"] = svg
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleAccountTOTPFinish(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		SetupToken string `json:"setup_token"`
		Code       string `json:"code"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	sessionToken, err := currentSessionToken(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err)
		return
	}
	secret, err := a.verifyAndTakeTOTPEnrollment(req.SetupToken, u.ID, req.Code, sessionToken)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	codes, err := a.Auth.ChangeTOTP(u.ID, sessionToken, secret)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	a.Hub.DisconnectUser(u.ID)
	a.audit(u.ID, u.Username, "totp_changed", u.Username, "recovery codes rotated; other sessions revoked", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (a *App) handleRecoveryCodeList(w http.ResponseWriter, r *http.Request) {
	items, err := a.Auth.ListRecoveryCodes(currentUser(r).ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleRecoveryCodeRegenerate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		ActionToken string `json:"action_token"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	sessionToken, err := currentSessionToken(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err)
		return
	}
	if err := a.takeAccountAction(req.ActionToken, u.ID, "regenerate_recovery_codes", sessionToken); err != nil {
		writeErr(w, http.StatusForbidden, err)
		return
	}
	codes, err := a.Auth.ReplaceRecoveryCodes(u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	a.audit(u.ID, u.Username, "recovery_codes_regenerated", u.Username, "previous codes revoked", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}
