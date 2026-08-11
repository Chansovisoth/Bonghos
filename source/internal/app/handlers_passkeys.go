package app

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const passkeyFlowLifetime = 5 * time.Minute

type passkeyFlow struct {
	Kind      string
	UserID    int64
	Purpose   string
	SessionID string
	RPID      string
	Origin    string
	Name      string
	Session   webauthn.SessionData
	ExpiresAt time.Time
}

func passkeyAuthority(authority string) (host, port string) {
	authority = strings.TrimSpace(authority)
	if parsedHost, parsedPort, err := net.SplitHostPort(authority); err == nil {
		host, port = parsedHost, parsedPort
	} else {
		host = strings.Trim(authority, "[]")
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host, port
}

func loopbackWebAuthnHost(host string) bool {
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// requestWebAuthn builds a relying-party configuration from the browser's
// verified same-origin request. This lets one Bonghos installation work on a
// LAN hostname and a tunnel hostname without trusting forwarded headers.
func requestWebAuthn(r *http.Request) (*webauthn.WebAuthn, string, string, error) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return nil, "", "", errors.New("the browser did not provide an origin")
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, "", "", errors.New("invalid browser origin")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopbackWebAuthnHost(parsed.Hostname())) {
		return nil, "", "", errors.New("passkeys require HTTPS or localhost")
	}
	originHost, originPort := passkeyAuthority(parsed.Host)
	requestHost, requestPort := passkeyAuthority(r.Host)
	defaultPort := "443"
	if parsed.Scheme == "http" {
		defaultPort = "80"
	}
	if originPort == "" {
		originPort = defaultPort
	}
	if requestPort == "" {
		requestPort = defaultPort
	}
	if originHost != requestHost || originPort != requestPort {
		return nil, "", "", errors.New("browser origin does not match this panel")
	}
	rpID := strings.ToLower(parsed.Hostname())
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Bonghos",
		RPID:          rpID,
		RPOrigins:     []string{origin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		},
		AttestationPreference: protocol.PreferNoAttestation,
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: passkeyFlowLifetime},
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: passkeyFlowLifetime},
		},
	})
	if err != nil {
		return nil, "", "", err
	}
	return wa, origin, rpID, nil
}

func (a *App) putPasskeyFlow(flow passkeyFlow) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()
	a.passkeyMu.Lock()
	defer a.passkeyMu.Unlock()
	if a.passkeyFlows == nil {
		a.passkeyFlows = make(map[string]passkeyFlow)
	}
	for id, existing := range a.passkeyFlows {
		if now.After(existing.ExpiresAt) {
			delete(a.passkeyFlows, id)
		}
	}
	if len(a.passkeyFlows) >= 512 {
		return "", errors.New("too many passkey requests; try again shortly")
	}
	a.passkeyFlows[token] = flow
	return token, nil
}

func (a *App) takePasskeyFlow(token, kind, origin, rpID string, userID int64) (passkeyFlow, error) {
	a.passkeyMu.Lock()
	defer a.passkeyMu.Unlock()
	flow, ok := a.passkeyFlows[token]
	delete(a.passkeyFlows, token)
	if !ok || time.Now().After(flow.ExpiresAt) || flow.Kind != kind || flow.Origin != origin || flow.RPID != rpID {
		return passkeyFlow{}, errors.New("passkey request expired or is invalid")
	}
	if userID != 0 && flow.UserID != userID {
		return passkeyFlow{}, errors.New("passkey request belongs to another account")
	}
	return flow, nil
}

func (a *App) handlePasskeyList(w http.ResponseWriter, r *http.Request) {
	items, err := a.Auth.ListPasskeys(currentUser(r).ID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handlePasskeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
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
	wa, origin, rpID, err := requestWebAuthn(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	passkeyUser, err := a.Auth.PasskeyUser(u.ID, rpID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	creation, session, err := wa.BeginRegistration(passkeyUser,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	)
	if err != nil {
		writeErr(w, 500, errors.New("could not start passkey enrollment"))
		return
	}
	flowToken, err := a.putPasskeyFlow(passkeyFlow{
		Kind: "register", UserID: u.ID, RPID: rpID, Origin: origin,
		Name: strings.TrimSpace(req.Name), Session: *session, ExpiresAt: time.Now().Add(passkeyFlowLifetime),
	})
	if err != nil {
		writeErr(w, 429, err)
		return
	}
	a.audit(u.ID, u.Username, "passkey_reauthentication", rpID, "", remoteIP(r))
	writeJSON(w, 200, map[string]any{"flow": flowToken, "options": creation})
}

func (a *App) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	wa, origin, rpID, err := requestWebAuthn(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	flow, err := a.takePasskeyFlow(r.URL.Query().Get("flow"), "register", origin, rpID, u.ID)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	passkeyUser, err := a.Auth.PasskeyUser(u.ID, rpID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	credential, err := wa.FinishRegistration(passkeyUser, flow.Session, r)
	if err != nil {
		a.Logf("passkey registration verification failed for user %d: %v", u.ID, err)
		writeErr(w, 400, errors.New("passkey could not be verified"))
		return
	}
	item, err := a.Auth.AddPasskey(u.ID, rpID, flow.Name, credential)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "passkey_added", strconv.FormatInt(item.ID, 10), fmt.Sprintf("%s on %s", item.Name, rpID), remoteIP(r))
	writeJSON(w, 201, item)
}

func (a *App) handlePasskeyUpdate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, errors.New("invalid passkey id"))
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	name := strings.TrimSpace(req.Name)
	previousName, err := a.Auth.RenamePasskey(u.ID, id, name)
	if err != nil {
		if errors.Is(err, auth.ErrPasskeyNotFound) {
			writeErr(w, 404, err)
			return
		}
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "passkey_renamed", strconv.FormatInt(id, 10), fmt.Sprintf("%s -> %s", previousName, name), remoteIP(r))
	writeJSON(w, 200, map[string]any{"id": id, "name": name})
}

func (a *App) handlePasskeyDelete(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, errors.New("invalid passkey id"))
		return
	}
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
	if err := a.takeAccountAction(req.ActionToken, u.ID, "remove_passkey", sessionToken); err != nil {
		writeErr(w, http.StatusForbidden, err)
		return
	}
	if err := a.Auth.DeletePasskey(u.ID, id); err != nil {
		writeErr(w, 404, err)
		return
	}
	a.audit(u.ID, u.Username, "passkey_removed", strconv.FormatInt(id, 10), "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handlePasskeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	wa, origin, rpID, err := requestWebAuthn(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	assertion, session, err := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		writeErr(w, 500, errors.New("could not start passkey sign-in"))
		return
	}
	flowToken, err := a.putPasskeyFlow(passkeyFlow{
		Kind: "login", RPID: rpID, Origin: origin, Session: *session,
		ExpiresAt: time.Now().Add(passkeyFlowLifetime),
	})
	if err != nil {
		writeErr(w, 429, err)
		return
	}
	writeJSON(w, 200, map[string]any{"flow": flowToken, "options": assertion})
}

func (a *App) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	wa, origin, rpID, err := requestWebAuthn(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	flow, err := a.takePasskeyFlow(r.URL.Query().Get("flow"), "login", origin, rpID, 0)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	var resolved *auth.PasskeyUser
	credential, err := wa.FinishDiscoverableLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		resolved, err = a.Auth.DiscoverPasskeyUser(userHandle, rawID, rpID)
		return resolved, err
	}, flow.Session, r)
	if err != nil || resolved == nil {
		a.Logf("passkey login verification failed: %v", err)
		writeErr(w, http.StatusUnauthorized, errors.New("passkey sign-in failed"))
		return
	}
	u := resolved.Account
	if err := a.Auth.UpdatePasskeyCredential(u.ID, rpID, credential); err != nil {
		writeErr(w, http.StatusUnauthorized, errors.New("passkey sign-in failed"))
		return
	}
	sess, err := a.Auth.CreateSession(u.ID, remoteIP(r), r.UserAgent())
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	a.setSessionCookie(w, r, sess.Token, sess.ExpiresAt)
	a.issueCSRF(w, r)
	a.audit(u.ID, u.Username, "login_passkey", rpID, "", remoteIP(r))
	writeJSON(w, 200, map[string]any{
		"id": u.ID, "username": u.Username, "role": u.Role,
		"permissions": authorization.Permissions(u.Role),
	})
}
