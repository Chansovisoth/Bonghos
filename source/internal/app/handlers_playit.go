package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/playit"
	"github.com/Chansovisoth/Bonghos/internal/runtime/systemd"
)

type playitPayload struct {
	playit.Config
	Detections      []playit.Detection `json:"detections"`
	AgentOnline     bool               `json:"agent_online"`
	AgentPhase      string             `json:"agent_phase,omitempty"`
	AgentVersion    string             `json:"agent_version,omitempty"`
	AgentError      string             `json:"agent_error,omitempty"`
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
	payload := playitPayload{Config: config, Detections: playit.DetectExisting(ctx), DaemonAvailable: a.hasPlayitDaemon()}
	if systemd.Available() {
		payload.ManagedState = systemd.State(systemd.ServicePlayit)
	}
	if config.ManagementMode == playit.ManagementBonghos && config.SecretConfigured {
		status := a.PlayitStatus(ctx)
		payload.AgentOnline = status.Running
		payload.AgentPhase = status.Phase
		payload.AgentVersion = status.Version
		payload.AgentError = status.Error
		if !payload.DaemonAvailable {
			payload.AgentPhase = "unavailable"
			payload.AgentError = "The official Playit agent is not installed or executable"
		} else if status.Phase == "stopped" && payload.ManagedState == "failed" {
			payload.AgentPhase = "error"
			payload.AgentError = "The managed Playit service failed; check bonghos-playit.service logs"
		}
		if status.AgentID != "" && status.AgentID != config.AgentID {
			_ = a.Playit.SaveAgent(status.AgentID)
			payload.AgentID = status.AgentID
		}
		if a.foregroundPlayitRunning() {
			filtered := payload.Detections[:0]
			for _, detection := range payload.Detections {
				if detection.Kind == "process" && strings.EqualFold(detection.Name, "playitd") {
					continue
				}
				filtered = append(filtered, detection)
			}
			payload.Detections = append(filtered, playit.Detection{
				Kind: "bonghos", Name: "Bonghos foreground agent", State: status.Phase, ExternallyManaged: false,
			})
		}
	}
	if !refresh || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || !config.SecretConfigured || !payload.AgentOnline {
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
		payload.TunnelStatus = "missing"
		found := false
		for _, tunnel := range data.Tunnels {
			if tunnel.ID != config.TunnelID {
				continue
			}
			found = true
			payload.TunnelStatus = "configured"
			if tunnel.DisabledReason != "" {
				payload.TunnelStatus = tunnel.DisabledReason
				payload.PublicAddress = ""
			} else if tunnel.StatusMessage != "" {
				payload.TunnelStatus = tunnel.StatusMessage
				payload.PublicAddress = ""
			} else {
				payload.PublicAddress = tunnel.DisplayAddress
			}
			_ = a.Playit.SaveTunnel(tunnel.ID, payload.PublicAddress, config.LocalPort)
			break
		}
		if !found {
			for _, tunnel := range data.Pending {
				if tunnel.ID == config.TunnelID {
					found = true
					payload.TunnelStatus = "pending"
					payload.PublicAddress = ""
					if tunnel.StatusMessage != "" {
						payload.TunnelStatus = tunnel.StatusMessage
					}
					_ = a.Playit.SaveTunnel(tunnel.ID, "", config.LocalPort)
					break
				}
			}
		}
		if !found {
			payload.PublicAddress = ""
			_ = a.Playit.SaveTunnel(config.TunnelID, "", config.LocalPort)
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
		if !a.hasPlayitDaemon() {
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
	if !a.hasPlayitDaemon() {
		return "Install the official Playit agent, then repair Bonghos services"
	}
	a.stopForegroundPlayit()
	if err := systemd.Start(systemd.ServicePlayit); err != nil {
		_ = systemd.Stop(systemd.ServicePlayit)
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
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
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
	if req.ManagementMode == playit.ManagementExternal {
		if req.LocalPort == 0 {
			req.LocalPort = 25565
		}
		config, err = a.Playit.SaveExternalConfig(req.Enabled, req.PublicAddress, req.LocalPort, actor.ID)
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
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
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
	if !a.hasPlayitDaemon() {
		writeErr(w, http.StatusConflict, errors.New("install the official Playit agent before linking it to Bonghos"))
		return
	}
	if err := a.preparePlayitClaim(); err != nil {
		writeErr(w, http.StatusConflict, err)
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
	if err := a.preparePlayitClaim(); err != nil {
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
	oldConfig, configErr := a.Playit.Config()
	if configErr == nil && oldConfig.TunnelID != "" && oldConfig.SecretConfigured {
		oldSecret, secretErr := a.Playit.Secret()
		if secretErr != nil {
			writeErr(w, http.StatusConflict, errors.New("could not access the existing Playit tunnel"))
			return
		}
		deleteErr := a.PlayitAPI.DeleteTunnel(r.Context(), oldSecret, oldConfig.TunnelID)
		if deleteErr != nil && !playit.IsProviderError(deleteErr, "TunnelNotFound") {
			writeErr(w, http.StatusBadGateway, errors.New("could not remove the existing Playit tunnel before relinking"))
			return
		}
		if err := a.Playit.ClearTunnel(); err != nil {
			writeErr(w, http.StatusInternalServerError, errors.New("could not clear the existing Playit tunnel"))
			return
		}
		a.audit(actor.ID, actor.Username, "playit_tunnel_deleted", oldConfig.TunnelID, "during agent relink", remoteIP(r))
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

func (a *App) preparePlayitClaim() error {
	version, err := playit.DaemonVersion(a.Home)
	if err != nil {
		// Tests and specialized clients may supply a trusted version override.
		if strings.TrimSpace(a.PlayitAPI.AgentVersion) != "" {
			return nil
		}
		return errors.New("could not read the installed Playit agent version; ensure playit-cli and playitd come from the official Playit package")
	}
	a.PlayitAPI.AgentVersion = version
	return nil
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
	status := a.waitForPlayitReady(r.Context(), 15*time.Second)
	if !status.Running {
		message := status.Error
		if message == "" {
			message = "The Playit agent is still starting; wait a moment and try again"
		}
		writeErr(w, http.StatusConflict, errors.New(message))
		return
	}
	port, err := a.activeMinecraftPort()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	data, err := a.PlayitAPI.RunData(r.Context(), secret)
	if err != nil {
		status := http.StatusBadGateway
		var providerErr *playit.ProviderError
		if errors.As(err, &providerErr) {
			status = http.StatusConflict
		}
		writeErr(w, status, err)
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
		if playit.IsProviderError(err, "TunnelNotFound") {
			if clearErr := a.Playit.ClearTunnel(); clearErr != nil {
				writeErr(w, http.StatusInternalServerError, errors.New("could not clear the missing Playit tunnel"))
				return
			}
			tunnelID, err = a.PlayitAPI.CreateMinecraftTunnel(r.Context(), secret, data.AgentID, port)
			action = "playit_tunnel_recreated"
		}
	}
	if err != nil {
		status := http.StatusBadGateway
		var providerErr *playit.ProviderError
		if errors.As(err, &providerErr) {
			status = http.StatusConflict
		}
		writeErr(w, status, err)
		return
	}
	publicAddress := config.PublicAddress
	if action == "playit_tunnel_created" || action == "playit_tunnel_recreated" {
		publicAddress = ""
	}
	if err := a.Playit.SaveTunnel(tunnelID, publicAddress, port); err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not save the Playit tunnel"))
		return
	}
	a.audit(actor.ID, actor.Username, action, tunnelID, "local port "+strconv.Itoa(port), remoteIP(r))
	payload, _ := a.playitPayload(r.Context(), true)
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) waitForPlayitReady(ctx context.Context, timeout time.Duration) playit.AgentStatus {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	for {
		status := a.PlayitStatus(ctx)
		if status.Running || status.Error != "" || status.Phase == "has_invalid_secret" || status.Phase == "disabled_over_limit" {
			return status
		}
		select {
		case <-ctx.Done():
			return status
		case <-deadline.C:
			return status
		case <-ticker.C:
		}
	}
}

// schedulePlayitTunnelSync keeps a configured managed tunnel pointed at the
// active project's current server-port. It is deliberately best-effort: a
// Playit outage must never prevent project selection or Minecraft startup.
func (a *App) schedulePlayitTunnelSync() {
	config, err := a.Playit.Config()
	if err != nil || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || config.TunnelID == "" {
		return
	}
	a.playitRuntimeMu.Lock()
	parent := a.playitParent
	a.playitRuntimeMu.Unlock()
	if parent == nil {
		parent = context.Background()
	}
	go func() {
		ctx, cancel := context.WithTimeout(parent, 35*time.Second)
		defer cancel()
		updated, err := a.syncPlayitTunnelPort(ctx)
		if err != nil {
			a.Logf("Playit tunnel port sync failed: %v", err)
			return
		}
		if updated {
			config, _ := a.Playit.Config()
			a.Logf("Playit tunnel updated for active Minecraft port %d", config.LocalPort)
		}
	}()
}

func (a *App) syncPlayitTunnelPort(ctx context.Context) (bool, error) {
	if status := a.waitForPlayitReady(ctx, 30*time.Second); !status.Running {
		return false, nil
	}
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	config, err := a.Playit.Config()
	if err != nil || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || config.TunnelID == "" {
		return false, err
	}
	port, err := a.activeMinecraftPort()
	if err != nil {
		return false, err
	}
	secret, err := a.Playit.Secret()
	if err != nil {
		return false, err
	}
	if err := a.PlayitAPI.UpdateTunnelPort(ctx, secret, config.TunnelID, port); err != nil {
		if playit.IsProviderError(err, "TunnelNotFound") {
			if clearErr := a.Playit.ClearTunnel(); clearErr != nil {
				return false, clearErr
			}
			return false, errors.New("the configured Playit tunnel no longer exists; create a new tunnel in Settings")
		}
		return false, err
	}
	if err := a.Playit.SaveTunnel(config.TunnelID, config.PublicAddress, port); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) handlePlayitTunnelDelete(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	config, err := a.Playit.Config()
	if err != nil || config.TunnelID == "" || !config.SecretConfigured || config.ManagementMode != playit.ManagementBonghos {
		writeErr(w, http.StatusConflict, errors.New("Playit tunnel is not configured"))
		return
	}
	secret, err := a.Playit.Secret()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	err = a.PlayitAPI.DeleteTunnel(r.Context(), secret, config.TunnelID)
	if err != nil && !playit.IsProviderError(err, "TunnelNotFound") {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	if err := a.Playit.ClearTunnel(); err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not clear the Playit tunnel"))
		return
	}
	a.audit(actor.ID, actor.Username, "playit_tunnel_deleted", config.TunnelID, "", remoteIP(r))
	payload, _ := a.playitPayload(r.Context(), true)
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handlePlayitGuestLogin(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	config, err := a.Playit.Config()
	if err != nil || !config.SecretConfigured || config.ManagementMode != playit.ManagementBonghos || config.AccountMode != playit.AccountModeGuest {
		writeErr(w, http.StatusConflict, errors.New("a linked Playit guest account is required"))
		return
	}
	secret, err := a.Playit.Secret()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	url, err := a.PlayitAPI.GuestLogin(r.Context(), secret)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	a.audit(actor.ID, actor.Username, "playit_guest_login_created", "networking", "", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (a *App) handlePlayitRefresh(w http.ResponseWriter, r *http.Request) {
	payload, err := a.playitPayload(r.Context(), true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not refresh Playit status"))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handlePlayitAgentRename(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	var req struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	name, err := playit.NormalizeAgentName(req.Name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	config, err := a.Playit.Config()
	if err != nil || config.ManagementMode != playit.ManagementBonghos || !config.SecretConfigured {
		writeErr(w, http.StatusConflict, errors.New("link a Bonghos-managed Playit agent first"))
		return
	}
	secret, err := a.Playit.Secret()
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	agentID := config.AgentID
	if agentID == "" {
		data, runErr := a.PlayitAPI.RunData(r.Context(), secret)
		if runErr != nil {
			writeErr(w, http.StatusBadGateway, runErr)
			return
		}
		agentID = data.AgentID
		if agentID == "" {
			writeErr(w, http.StatusBadGateway, errors.New("Playit did not return an agent identifier"))
			return
		}
		_ = a.Playit.SaveAgent(agentID)
	}
	if err := a.PlayitAPI.RenameAgent(r.Context(), secret, agentID, name); err != nil {
		status := http.StatusBadGateway
		if playit.IsProviderError(err, "InvalidName") {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err)
		return
	}
	if err := a.Playit.SaveAgentName(name); err != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("Playit renamed the agent, but Bonghos could not save the name"))
		return
	}
	a.audit(actor.ID, actor.Username, "playit_agent_renamed", agentID, name, remoteIP(r))
	payload, payloadErr := a.playitPayload(r.Context(), false)
	if payloadErr != nil {
		writeErr(w, http.StatusInternalServerError, errors.New("could not reload Playit settings"))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
