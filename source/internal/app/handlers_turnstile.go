package app

import (
	"errors"
	"net/http"

	"github.com/Chansovisoth/Bonghos/internal/turnstile"
)

func (a *App) handleTurnstilePublic(w http.ResponseWriter, _ *http.Request) {
	config, err := a.Turnstile.Store.PublicConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not load login protection"))
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (a *App) verifyTurnstileLogin(w http.ResponseWriter, r *http.Request, token string) bool {
	err := a.Turnstile.VerifyLogin(r.Context(), token, r.Host)
	if err == nil {
		return true
	}
	status := http.StatusForbidden
	if errors.Is(err, turnstile.ErrUnavailable) {
		status = http.StatusServiceUnavailable
	}
	writeErr(w, status, err)
	return false
}

// The public passkey begin handler accepts only the Turnstile token. The
// existing WebAuthn handler remains focused on creating the assertion flow.
func (a *App) handleTurnstilePasskeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TurnstileToken string `json:"turnstile_token"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	if !a.verifyTurnstileLogin(w, r, req.TurnstileToken) {
		return
	}
	a.handlePasskeyLoginBegin(w, r)
}

func (a *App) handleTurnstileSettings(w http.ResponseWriter, _ *http.Request) {
	config, err := a.Turnstile.Store.Config()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not load Turnstile settings"))
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (a *App) handleTurnstileSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	var req struct {
		Enabled     bool    `json:"enabled"`
		SiteKey     string  `json:"site_key"`
		SecretKey   *string `json:"secret_key"`
		ClearSecret bool    `json:"clear_secret"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	config, err := a.Turnstile.Store.Update(turnstile.Update{
		Enabled: req.Enabled, SiteKey: req.SiteKey, SecretKey: req.SecretKey,
		ClearSecret: req.ClearSecret, UpdatedBy: actor.ID,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	a.audit(actor.ID, actor.Username, "turnstile_settings_updated", "login", enabledDetail(config.Enabled), remoteIP(r))
	writeJSON(w, http.StatusOK, config)
}

func enabledDetail(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
