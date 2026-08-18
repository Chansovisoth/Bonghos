package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/playit"
	"github.com/Chansovisoth/Bonghos/internal/runtime/systemd"
)

type playitPayload struct {
	playit.Config
	Detections      []playit.Detection `json:"detections"`
	AgentOnline     bool               `json:"agent_online"`
	AccountStatus   string             `json:"account_status,omitempty"`
	TunnelStatus    string             `json:"tunnel_status,omitempty"`
	Notice          string             `json:"notice,omitempty"`
	DaemonAvailable bool               `json:"daemon_available"`
	ManagedState    string             `json:"managed_state,omitempty"`
	GuestLoginURL   string             `json:"guest_login_url,omitempty"`
}

func (a *App) playitPayload(ctx context.Context, refresh bool) (playitPayload, error) {
	config, err := a.Playit.Config()
	if err != nil {
		return playitPayload{}, err
	}
	payload := playitPayload{Config: config, Detections: playit.DetectExisting(ctx), DaemonAvailable: playit.DaemonAvailable(a.Home)}
	if systemd.Available() {
		payload.ManagedState = systemd.State(systemd.ServicePlayit)
	}
	for _, detection := range payload.Detections {
		if detection.State == "active" || detection.State == "running" {
			payload.AgentOnline = true
			break
		}
	}
	if !refresh || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || !config.SecretConfigured {
		return payload, nil
	}
	secret, err := a.Playit.Secret()
	if err != nil {
		return payload, nil
	}
	data, err := a.PlayitAPI.RunData(ctx, secret)
	if err != nil {
		payload.Notice = "Playit account status is temporarily unavailable"
		return payload, nil
	}
	payload.AccountStatus = data.Permissions.AccountStatus
	if data.AgentID != "" && data.AgentID != config.AgentID {
		_ = a.Playit.SaveAgent(data.AgentID)
		payload.AgentID = data.AgentID
	}
	if config.TunnelID != "" {
		payload.TunnelStatus = "pending"
		for _, tunnel := range data.Tunnels {
			if tunnel.ID != config.TunnelID {
				continue
			}
			payload.TunnelStatus = "configured"
			payload.PublicAddress = tunnel.DisplayAddress
			_ = a.Playit.SaveTunnel(tunnel.ID, tunnel.DisplayAddress, config.LocalPort)
			break
		}
		for _, tunnel := range data.Pending {
			if tunnel.ID == config.TunnelID && tunnel.StatusMessage != "" {
				payload.TunnelStatus = tunnel.StatusMessage
			}
		}
	}
	return payload, nil
}

func (a *App) reconcilePlayitService(config playit.Config) string {
	shouldRun := config.Enabled && config.ManagementMode == playit.ManagementBonghos && config.SecretConfigured
	if !shouldRun {
		a.stopForegroundPlayit()
	}
	if !systemd.Available() {
		if !shouldRun {
			return ""
		}
		if !playit.DaemonAvailable(a.Home) {
			return "Install the official Playit agent before starting the managed tunnel"
		}
		a.startForegroundPlayit(nil)
		return ""
	}
	if !shouldRun {
		if systemd.IsActive(systemd.ServicePlayit) {
			if err := systemd.Stop(systemd.ServicePlayit); err != nil {
				return "Playit settings were saved, but the managed agent could not be stopped"
			}
		}
		playit.CleanupRuntime(a.Home)
		return ""
	}
	if !playit.DaemonAvailable(a.Home) {
		return "Install the official Playit agent, then repair Bonghos services"
	}
	if err := systemd.Start(systemd.ServicePlayit); err != nil {
		a.startForegroundPlayit(nil)
		return "The Playit service is unavailable, so the agent is running inside Bonghos; run bonghos service repair to restore the separate service"
	}
	return ""
}

func (a *App) handlePlayitGet(w http.ResponseWriter, r *http.Request) {
	payload, err := a.playitPayload(r.Context(), true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not load Playit settings"))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handlePlayitUpdate(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	var req struct {
		Enabled        bool   `json:"enabled"`
		AccountMode    string `json:"account_mode"`
		ManagementMode string `json:"management_mode"`
		PublicAddress  string `json:"public_address"`
		LocalPort      int    `json:"local_port"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	if req.AccountMode != playit.AccountModeAccount && req.AccountMode != playit.AccountModeGuest {
		writeErr(w, http.StatusBadRequest, errors.New("Playit account mode must be account or guest"))
		return
	}
	var (
		config playit.Config
		err    error
	)
	if req.Enabled && req.ManagementMode == playit.ManagementExternal {
		if req.LocalPort == 0 {
			req.LocalPort = 25565
		}
		config, err = a.Playit.SaveExternalAddress(req.PublicAddress, req.LocalPort, actor.ID)
	} else {
		config, err = a.Playit.SetPreference(req.Enabled, req.AccountMode, req.ManagementMode, actor.ID)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	notice := a.reconcilePlayitService(config)
	a.audit(actor.ID, actor.Username, "playit_settings_updated", "networking",
		enabledDetail(config.Enabled), remoteIP(r))
	payload, payloadErr := a.playitPayload(r.Context(), false)
	if payloadErr != nil {
		writeJSON(w, http.StatusOK, config)
		return
	}
	payload.Notice = notice
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handlePlayitClaimStart(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	var req struct {
		AccountMode string `json:"account_mode"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	if req.AccountMode != playit.AccountModeAccount && req.AccountMode != playit.AccountModeGuest {
		writeErr(w, http.StatusBadRequest, errors.New("Playit account mode must be account or guest"))
		return
	}
	codeBytes := make([]byte, 5)
	if _, err := rand.Read(codeBytes); err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not create a Playit claim"))
		return
	}
	code := hex.EncodeToString(codeBytes)
	if _, err := a.PlayitAPI.ClaimSetup(r.Context(), code); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	config, err := a.Playit.BeginClaim(req.AccountMode, code, actor.ID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	a.audit(actor.ID, actor.Username, "playit_claim_started", "networking", req.AccountMode, remoteIP(r))
	writeJSON(w, http.StatusOK, config)
}

func (a *App) handlePlayitClaimPoll(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	code, accountMode, err := a.Playit.Claim()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	state, err := a.PlayitAPI.ClaimSetup(r.Context(), code)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	switch state {
	case "WaitingForUserVisit", "WaitingForUser":
		writeJSON(w, http.StatusOK, map[string]any{"state": "waiting", "claim_url": "https://playit.gg/claim/" + code})
		return
	case "UserRejected":
		_ = a.Playit.CancelClaim()
		writeJSON(w, http.StatusOK, map[string]any{"state": "rejected"})
		return
	case "UserAccepted":
		// continue below
	default:
		writeErr(w, http.StatusBadGateway, errors.New("Playit returned an unknown claim state"))
		return
	}
	secret, err := a.PlayitAPI.ClaimExchange(r.Context(), code)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	config, err := a.Playit.CompleteClaim(secret, actor.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not save the Playit agent"))
		return
	}
	data, statusErr := a.PlayitAPI.RunData(r.Context(), secret)
	if statusErr == nil && data.AgentID != "" {
		_ = a.Playit.SaveAgent(data.AgentID)
		config.AgentID = data.AgentID
	}
	notice := a.reconcilePlayitService(config)
	guestLoginURL := ""
	if accountMode == playit.AccountModeGuest {
		guestLoginURL, _ = a.PlayitAPI.GuestLogin(r.Context(), secret)
	}
	a.audit(actor.ID, actor.Username, "playit_claim_completed", "networking", config.AccountMode, remoteIP(r))
	payload, payloadErr := a.playitPayload(r.Context(), false)
	if payloadErr == nil {
		payload.Notice = notice
		payload.GuestLoginURL = guestLoginURL
		writeJSON(w, http.StatusOK, map[string]any{"state": "complete", "config": payload})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": "complete", "config": config})
}

func (a *App) activeMinecraftPort() (int, error) {
	inst, err := a.activeInstance()
	if err != nil {
		return 0, err
	}
	port := 25565
	if props, err := minecraft.ReadProperties(inst.AbsoluteDir(a.Home)); err == nil {
		if configured, parseErr := strconv.Atoi(strings.TrimSpace(props["server-port"])); parseErr == nil && configured > 0 && configured <= 65535 {
			port = configured
		}
	}
	return port, nil
}

func (a *App) handlePlayitTunnel(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	config, err := a.Playit.Config()
	if err != nil || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || !config.SecretConfigured {
		writeErr(w, http.StatusConflict, errors.New("link a Playit agent first"))
		return
	}
	secret, err := a.Playit.Secret()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	port, err := a.activeMinecraftPort()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	data, err := a.PlayitAPI.RunData(r.Context(), secret)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	if data.AgentID == "" {
		writeErr(w, http.StatusBadGateway, errors.New("Playit did not return an agent identifier"))
		return
	}
	_ = a.Playit.SaveAgent(data.AgentID)
	tunnelID := config.TunnelID
	action := "playit_tunnel_updated"
	if tunnelID == "" {
		tunnelID, err = a.PlayitAPI.CreateMinecraftTunnel(r.Context(), secret, data.AgentID, port)
		action = "playit_tunnel_created"
	} else {
		err = a.PlayitAPI.UpdateTunnelPort(r.Context(), secret, tunnelID, port)
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	if err := a.Playit.SaveTunnel(tunnelID, config.PublicAddress, port); err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not save the Playit tunnel"))
		return
	}
	a.audit(actor.ID, actor.Username, action, tunnelID, "local port "+strconv.Itoa(port), remoteIP(r))
	payload, _ := a.playitPayload(r.Context(), true)
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handlePlayitRefresh(w http.ResponseWriter, r *http.Request) {
	payload, err := a.playitPayload(r.Context(), true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not refresh Playit status"))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
