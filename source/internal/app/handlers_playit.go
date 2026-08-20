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

type remotePlayitTunnel struct {
	data    playit.TunnelData
	pending bool
}

func playitTunnelConfigPort(tunnel playit.TunnelData) int {
	for _, field := range tunnel.AgentConfig.Fields {
		if field.Name != "local_port" {
			continue
		}
		port, err := strconv.Atoi(strings.TrimSpace(field.Value))
		if err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	return 0
}

func isManagedPlayitTunnel(tunnel playit.TunnelData, localPort int, pending bool) bool {
	if tunnel.ID == "" {
		return false
	}
	managedName := strings.EqualFold(strings.TrimSpace(tunnel.Name), playit.ManagedTunnelName)
	if pending {
		// Pending run data does not expose the local agent configuration. A
		// unique Minecraft tunnel on this linked agent is still safe to adopt;
		// an untyped pending tunnel must retain Bonghos's generated name.
		return tunnel.TunnelType == "minecraft-java" || (tunnel.TunnelType == "" && managedName)
	}
	if tunnel.TunnelType != "" && tunnel.TunnelType != "minecraft-java" {
		return false
	}
	configuredPort := playitTunnelConfigPort(tunnel)
	if configuredPort == 0 {
		return managedName
	}
	if localPort > 0 && configuredPort != localPort {
		return false
	}
	return tunnel.TunnelType == "minecraft-java" || managedName
}

func discoverManagedPlayitTunnel(data playit.RunData, localPort int, excluded map[string]bool) (remotePlayitTunnel, bool, bool) {
	candidates := make([]remotePlayitTunnel, 0, 2)
	seen := make(map[string]bool)
	consider := func(tunnel playit.TunnelData, pending bool) {
		if seen[tunnel.ID] || excluded[tunnel.ID] || !isManagedPlayitTunnel(tunnel, localPort, pending) {
			return
		}
		seen[tunnel.ID] = true
		candidates = append(candidates, remotePlayitTunnel{data: tunnel, pending: pending})
	}
	for _, tunnel := range data.Tunnels {
		consider(tunnel, false)
	}
	for _, tunnel := range data.Pending {
		consider(tunnel, true)
	}
	if len(candidates) == 1 {
		return candidates[0], true, false
	}
	return remotePlayitTunnel{}, false, len(candidates) > 1
}

func findPlayitTunnel(data playit.RunData, tunnelID string) (remotePlayitTunnel, bool) {
	for _, tunnel := range data.Tunnels {
		if tunnel.ID == tunnelID {
			return remotePlayitTunnel{data: tunnel}, true
		}
	}
	for _, tunnel := range data.Pending {
		if tunnel.ID == tunnelID {
			return remotePlayitTunnel{data: tunnel, pending: true}, true
		}
	}
	return remotePlayitTunnel{}, false
}

func playitTunnelDisplay(remote remotePlayitTunnel) (string, string) {
	if remote.pending {
		if remote.data.StatusMessage != "" {
			return remote.data.StatusMessage, ""
		}
		return "pending", ""
	}
	if remote.data.DisabledReason != "" {
		return remote.data.DisabledReason, ""
	}
	if remote.data.StatusMessage != "" {
		return remote.data.StatusMessage, ""
	}
	return "configured", remote.data.DisplayAddress
}

func (a *App) cachePlayitTunnel(payload *playitPayload, remote remotePlayitTunnel, localPort int) {
	status, address := playitTunnelDisplay(remote)
	payload.TunnelID = remote.data.ID
	payload.TunnelStatus = status
	payload.PublicAddress = address
	_ = a.Playit.SaveTunnel(remote.data.ID, address, localPort)
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
		a.Logf("Playit status refresh failed: %v", err)
		payload.Notice = "Playit account status is temporarily unavailable"
		return payload, nil
	}
	payload.AccountStatus = data.Permissions.AccountStatus
	if data.AgentID != "" && data.AgentID != config.AgentID {
		_ = a.Playit.SaveAgent(data.AgentID)
		payload.AgentID = data.AgentID
	}
	if config.TunnelID == "" {
		remote, found, ambiguous := discoverManagedPlayitTunnel(data, config.LocalPort, nil)
		if found {
			a.cachePlayitTunnel(&payload, remote, config.LocalPort)
			a.Logf("Playit adopted remote Bonghos tunnel %s", remote.data.ID)
			a.audit(0, "system", "playit_tunnel_adopted", remote.data.ID, "discovered during status refresh", "")
		} else if ambiguous {
			payload.Notice = "Multiple Playit tunnels match the active server port; remove duplicates before refreshing"
		} else if len(data.Tunnels)+len(data.Pending) > 0 {
			payload.Notice = "Playit has tunnels, but none match the active server port"
		}
		return payload, nil
	}
	remote, found := findPlayitTunnel(data, config.TunnelID)
	if found {
		a.cachePlayitTunnel(&payload, remote, config.LocalPort)
	} else {
		payload.TunnelStatus = "missing"
		payload.PublicAddress = ""
		_ = a.Playit.SaveTunnel(config.TunnelID, "", config.LocalPort)
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

func (a *App) runPlayitServiceReconcile(config playit.Config) string {
	if a.PlayitServiceReconcile != nil {
		return a.PlayitServiceReconcile(config)
	}
	return a.reconcilePlayitService(config)
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
	notice := a.runPlayitServiceReconcile(config)
	a.audit(actor.ID, actor.Username, "playit_settings_updated", "networking",
		enabledDetail(config.Enabled), remoteIP(r))
	payload, payloadErr := a.playitPayload(r.Context(), false)
	if payloadErr != nil {
		writeJSON(w, http.StatusOK, config)
		return
	}
	payload.Notice = notice
	writeJSON(w, http.StatusOK, payload)
	if config.Enabled && config.ManagementMode == playit.ManagementBonghos && config.SecretConfigured {
		a.schedulePlayitTunnelSync()
	}
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
	notice := a.runPlayitServiceReconcile(config)
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
		a.schedulePlayitTunnelSync()
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": "complete", "config": config})
	a.schedulePlayitTunnelSync()
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

func playitTunnelIDs(data playit.RunData) map[string]bool {
	ids := make(map[string]bool, len(data.Tunnels)+len(data.Pending))
	for _, tunnel := range data.Tunnels {
		ids[tunnel.ID] = true
	}
	for _, tunnel := range data.Pending {
		ids[tunnel.ID] = true
	}
	return ids
}

func (a *App) recoverCreatedPlayitTunnel(ctx context.Context, secret string, before playit.RunData, localPort int) (remotePlayitTunnel, bool) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	excluded := playitTunnelIDs(before)
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	for {
		data, err := a.PlayitAPI.RunData(ctx, secret)
		if err == nil {
			if remote, found, ambiguous := discoverManagedPlayitTunnel(data, localPort, excluded); found && !ambiguous {
				return remote, true
			}
		}
		select {
		case <-ctx.Done():
			return remotePlayitTunnel{}, false
		case <-ticker.C:
		}
	}
}

func (a *App) waitForConfiguredPlayitTunnel(ctx context.Context, secret, tunnelID string) (remotePlayitTunnel, bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	for {
		if data, err := a.PlayitAPI.RunData(ctx, secret); err == nil {
			if remote, found := findPlayitTunnel(data, tunnelID); found {
				return remote, true
			}
		}
		select {
		case <-ctx.Done():
			return remotePlayitTunnel{}, false
		case <-ticker.C:
		}
	}
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
		a.Logf("Playit tunnel status request failed: %v", err)
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
		remote, found, ambiguous := discoverManagedPlayitTunnel(data, port, nil)
		if ambiguous {
			writeErr(w, http.StatusConflict, errors.New("multiple Bonghos Playit tunnels already exist; remove duplicates in Playit.gg, then refresh"))
			return
		}
		if found {
			tunnelID = remote.data.ID
			action = "playit_tunnel_adopted"
		} else {
			tunnelID, err = a.PlayitAPI.CreateMinecraftTunnel(r.Context(), secret, data.AgentID, port)
			action = "playit_tunnel_created"
			if err != nil {
				if recovered, ok := a.recoverCreatedPlayitTunnel(r.Context(), secret, data, port); ok {
					tunnelID = recovered.data.ID
					err = nil
					action = "playit_tunnel_recovered"
				}
			}
		}
	} else if remote, found := findPlayitTunnel(data, tunnelID); found && config.LocalPort == port {
		// The connection is already correct. Refreshing should be harmless and
		// must not send a schema update that can fail on otherwise healthy
		// Playit agents.
		_, config.PublicAddress = playitTunnelDisplay(remote)
		action = "playit_tunnel_refreshed"
	} else {
		err = a.PlayitAPI.UpdateTunnelPort(r.Context(), secret, tunnelID, port)
		if playit.IsProviderError(err, "TunnelNotFound") {
			if clearErr := a.Playit.ClearTunnel(); clearErr != nil {
				writeErr(w, http.StatusInternalServerError, errors.New("could not clear the missing Playit tunnel"))
				return
			}
			tunnelID, err = a.PlayitAPI.CreateMinecraftTunnel(r.Context(), secret, data.AgentID, port)
			action = "playit_tunnel_recreated"
			if err != nil {
				if recovered, ok := a.recoverCreatedPlayitTunnel(r.Context(), secret, data, port); ok {
					tunnelID = recovered.data.ID
					err = nil
					action = "playit_tunnel_recovered"
				}
			}
		}
	}
	if err != nil {
		a.Logf("Playit tunnel %s failed: %v", action, err)
		status := http.StatusBadGateway
		var providerErr *playit.ProviderError
		if errors.As(err, &providerErr) {
			status = http.StatusConflict
		}
		writeErr(w, status, err)
		return
	}
	publicAddress := config.PublicAddress
	if action == "playit_tunnel_created" || action == "playit_tunnel_recreated" || action == "playit_tunnel_adopted" || action == "playit_tunnel_recovered" {
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
	if err != nil || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || !config.SecretConfigured {
		return
	}
	a.playitRuntimeMu.Lock()
	parent := a.playitParent
	a.playitRuntimeMu.Unlock()
	if parent == nil {
		parent = context.Background()
	}
	a.playitSyncMu.Lock()
	if a.playitSyncRunning {
		a.playitSyncPending = true
		a.playitSyncMu.Unlock()
		return
	}
	a.playitSyncRunning = true
	a.playitSyncMu.Unlock()
	go func() {
		defer func() {
			a.playitSyncMu.Lock()
			pending := a.playitSyncPending
			a.playitSyncRunning = false
			a.playitSyncPending = false
			a.playitSyncMu.Unlock()
			if pending {
				a.schedulePlayitTunnelSync()
			}
		}()
		ctx, cancel := context.WithTimeout(parent, 90*time.Second)
		defer cancel()
		for attempt := 0; attempt < 4; attempt++ {
			updated, err := a.syncPlayitTunnelPort(ctx)
			if err == nil {
				if updated {
					config, _ := a.Playit.Config()
					a.Logf("Playit tunnel synchronized for active game-server port %d", config.LocalPort)
				}
				return
			}
			a.Logf("Playit tunnel sync attempt %d failed: %v", attempt+1, err)
			delay := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}()
}

func (a *App) syncPlayitTunnelPort(ctx context.Context) (bool, error) {
	if status := a.waitForPlayitReady(ctx, 30*time.Second); !status.Running {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if status.Error != "" {
			return false, errors.New(status.Error)
		}
		return false, errors.New("the Playit agent is still starting")
	}
	a.playitMu.Lock()
	defer a.playitMu.Unlock()
	config, err := a.Playit.Config()
	if err != nil || !config.Enabled || config.ManagementMode != playit.ManagementBonghos || !config.SecretConfigured {
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
	if config.TunnelID == "" {
		data, runErr := a.PlayitAPI.RunData(ctx, secret)
		if runErr != nil {
			return false, runErr
		}
		if data.AgentID == "" {
			return false, errors.New("Playit did not return an agent identifier")
		}
		remote, found, ambiguous := discoverManagedPlayitTunnel(data, port, nil)
		if ambiguous {
			return false, errors.New("multiple Bonghos Playit tunnels exist; remove duplicates in Playit.gg")
		}
		if found {
			_, address := playitTunnelDisplay(remote)
			if err := a.Playit.SaveTunnel(remote.data.ID, address, port); err != nil {
				return false, err
			}
			a.audit(0, "system", "playit_tunnel_adopted", remote.data.ID, "discovered during automatic synchronization", "")
			return true, nil
		}

		tunnelID, createErr := a.PlayitAPI.CreateMinecraftTunnel(ctx, secret, data.AgentID, port)
		if createErr != nil {
			if recovered, ok := a.recoverCreatedPlayitTunnel(ctx, secret, data, port); ok {
				remote = recovered
				tunnelID = recovered.data.ID
				createErr = nil
			}
		}
		if createErr != nil {
			return false, createErr
		}
		address := ""
		if remote.data.ID == "" {
			if activated, ok := a.recoverCreatedPlayitTunnel(ctx, secret, data, port); ok {
				remote = activated
			}
		}
		if remote.data.ID != "" {
			_, address = playitTunnelDisplay(remote)
		}
		if err := a.Playit.SaveTunnel(tunnelID, address, port); err != nil {
			return false, err
		}
		a.audit(0, "system", "playit_tunnel_created", tunnelID, "created during automatic synchronization", "")
		return true, nil
	}
	if config.LocalPort == port {
		data, runErr := a.PlayitAPI.RunData(ctx, secret)
		if runErr != nil {
			return false, runErr
		}
		remote, found := findPlayitTunnel(data, config.TunnelID)
		if !found {
			remote, found = a.waitForConfiguredPlayitTunnel(ctx, secret, config.TunnelID)
		}
		if !found {
			if clearErr := a.Playit.ClearTunnel(); clearErr != nil {
				return false, clearErr
			}
			if data.AgentID == "" {
				return false, errors.New("Playit did not return an agent identifier")
			}
			tunnelID, createErr := a.PlayitAPI.CreateMinecraftTunnel(ctx, secret, data.AgentID, port)
			if createErr != nil {
				if recovered, ok := a.recoverCreatedPlayitTunnel(ctx, secret, data, port); ok {
					tunnelID = recovered.data.ID
					createErr = nil
				}
			}
			if createErr != nil {
				return false, createErr
			}
			if err := a.Playit.SaveTunnel(tunnelID, "", port); err != nil {
				return false, err
			}
			a.audit(0, "system", "playit_tunnel_recreated", tunnelID, "recreated during automatic synchronization", "")
			return true, nil
		}
		_, address := playitTunnelDisplay(remote)
		if err := a.Playit.SaveTunnel(config.TunnelID, address, port); err != nil {
			return false, err
		}
		return address != config.PublicAddress, nil
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
			a.Logf("Playit agent lookup before rename failed: %v", runErr)
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
		} else {
			var providerErr *playit.ProviderError
			if errors.As(err, &providerErr) {
				status = http.StatusConflict
			}
		}
		a.Logf("Playit agent rename failed: %v", err)
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
