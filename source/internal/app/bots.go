package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/bot"
)

type botRequest struct {
	Name                   string            `json:"name"`
	Provider               string            `json:"provider"`
	Token                  *string           `json:"token"`
	DNSServer              string            `json:"dns_server"`
	DestinationID          string            `json:"destination_id"`
	Destinations           []bot.Destination `json:"destinations"`
	DiscoveredDestinations []bot.Destination `json:"discovered_destinations"`
	Enabled                *bool             `json:"enabled"`
	NotifyServerStarted    *bool             `json:"notify_server_started"`
	NotifyServerStopped    *bool             `json:"notify_server_stopped"`
	NotifyPlayerJoined     *bool             `json:"notify_player_joined"`
	NotifyPlayerLeft       *bool             `json:"notify_player_left"`
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func (a *App) handleBotList(w http.ResponseWriter, _ *http.Request) {
	configs, err := a.Bots.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

func (a *App) handleBotCreate(w http.ResponseWriter, r *http.Request) {
	var request botRequest
	if err := readJSON(r, &request, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	token := ""
	if request.Token != nil {
		token = *request.Token
	}
	config, err := a.Bots.Create(bot.CreateInput{
		Name: request.Name, Provider: request.Provider, Token: token,
		DNSServer:              request.DNSServer,
		DestinationID:          request.DestinationID,
		Destinations:           request.Destinations,
		DiscoveredDestinations: request.DiscoveredDestinations,
		Enabled:                boolDefault(request.Enabled, true),
		NotifyServerStarted:    boolDefault(request.NotifyServerStarted, true),
		NotifyServerStopped:    boolDefault(request.NotifyServerStopped, true),
		NotifyPlayerJoined:     boolDefault(request.NotifyPlayerJoined, true),
		NotifyPlayerLeft:       boolDefault(request.NotifyPlayerLeft, true),
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	user := currentUser(r)
	a.audit(user.ID, user.Username, "notification_bot_created", config.Name, config.Provider, remoteIP(r))
	writeJSON(w, http.StatusCreated, config)
}

func botID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid bot id")
	}
	return id, nil
}

func (a *App) handleBotUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := botID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var request struct {
		Name                *string            `json:"name"`
		Provider            *string            `json:"provider"`
		Token               *string            `json:"token"`
		DNSServer           *string            `json:"dns_server"`
		DestinationID       *string            `json:"destination_id"`
		Destinations        *[]bot.Destination `json:"destinations"`
		Enabled             *bool              `json:"enabled"`
		NotifyServerStarted *bool              `json:"notify_server_started"`
		NotifyServerStopped *bool              `json:"notify_server_stopped"`
		NotifyPlayerJoined  *bool              `json:"notify_player_joined"`
		NotifyPlayerLeft    *bool              `json:"notify_player_left"`
	}
	if err := readJSON(r, &request, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	config, err := a.Bots.Patch(id, bot.Patch{
		Name: request.Name, Provider: request.Provider, Token: request.Token,
		DNSServer:     request.DNSServer,
		DestinationID: request.DestinationID, Destinations: request.Destinations, Enabled: request.Enabled,
		NotifyServerStarted: request.NotifyServerStarted,
		NotifyServerStopped: request.NotifyServerStopped,
		NotifyPlayerJoined:  request.NotifyPlayerJoined,
		NotifyPlayerLeft:    request.NotifyPlayerLeft,
	})
	if errors.Is(err, bot.ErrNotFound) {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	user := currentUser(r)
	a.audit(user.ID, user.Username, "notification_bot_updated", config.Name, config.Provider, remoteIP(r))
	writeJSON(w, http.StatusOK, config)
}

func (a *App) telegramDiscovery(w http.ResponseWriter, r *http.Request, token, dnsServer, auditTarget string, persistBotID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	discovery, err := a.BotNotify.Sender.WithDNS(dnsServer).DiscoverTelegramGroups(ctx, token)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	if persistBotID > 0 {
		discovery.Groups, err = a.Bots.MergeDiscovered(persistBotID, discovery.Groups)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	user := currentUser(r)
	a.audit(user.ID, user.Username, "notification_bot_groups_discovered", auditTarget, "", remoteIP(r))
	writeJSON(w, http.StatusOK, discovery)
}

func (a *App) handleBotTelegramDiscover(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Token     string `json:"token"`
		DNSServer string `json:"dns_server"`
	}
	if err := readJSON(r, &request, 1<<16); err != nil || strings.TrimSpace(request.Token) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("Telegram bot token is required"))
		return
	}
	a.telegramDiscovery(w, r, request.Token, request.DNSServer, "new Telegram bot", 0)
}

func (a *App) handleBotTelegramDiscoverExisting(w http.ResponseWriter, r *http.Request) {
	id, err := botID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	config, err := a.Bots.ByID(id)
	if errors.Is(err, bot.ErrNotFound) {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if config.Provider != bot.ProviderTelegram {
		writeErr(w, http.StatusBadRequest, errors.New("group discovery is available only for Telegram bots"))
		return
	}
	var request struct {
		Token string `json:"token"`
	}
	if err := readJSON(r, &request, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	token := strings.TrimSpace(request.Token)
	if token == "" {
		target, targetErr := a.Bots.Credential(id)
		if targetErr != nil {
			writeErr(w, http.StatusBadRequest, targetErr)
			return
		}
		token = target.Token
	}
	a.telegramDiscovery(w, r, token, config.DNSServer, config.Name, id)
}

func (a *App) handleBotTelegramGroupPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := botID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	target, fileID, err := a.Bots.TelegramPhotoTarget(id, r.PathValue("destination"))
	if errors.Is(err, bot.ErrNotFound) {
		writeErr(w, http.StatusNotFound, errors.New("Telegram group photo not found"))
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	var data []byte
	var contentType string
	if fileID != "" {
		data, contentType, err = a.BotNotify.Sender.WithDNS(target.DNSServer).TelegramGroupPhoto(ctx, target.Token, fileID)
	}
	if fileID == "" || err != nil {
		data, contentType, err = a.BotNotify.Sender.WithDNS(target.DNSServer).TelegramGroupPhotoForChat(ctx, target.Token, target.DestinationID)
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, errors.New("Telegram group photo is unavailable"))
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (a *App) handleBotDelete(w http.ResponseWriter, r *http.Request) {
	id, err := botID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	config, getErr := a.Bots.ByID(id)
	if errors.Is(getErr, bot.ErrNotFound) {
		writeErr(w, http.StatusNotFound, getErr)
		return
	}
	if getErr != nil {
		writeErr(w, http.StatusInternalServerError, getErr)
		return
	}
	if err := a.Bots.Delete(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	user := currentUser(r)
	a.audit(user.ID, user.Username, "notification_bot_deleted", config.Name, config.Provider, remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleBotTest(w http.ResponseWriter, r *http.Request) {
	id, err := botID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := a.BotNotify.Test(ctx, id); err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, bot.ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, bot.ErrDisabled) || errors.Is(err, bot.ErrNoDestinations) {
			status = http.StatusConflict
		}
		writeErr(w, status, err)
		return
	}
	config, _ := a.Bots.ByID(id)
	user := currentUser(r)
	target := strconv.FormatInt(id, 10)
	if config != nil {
		target = config.Name
	}
	a.audit(user.ID, user.Username, "notification_bot_tested", target, "", remoteIP(r))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleBotInvite(w http.ResponseWriter, r *http.Request) {
	id, err := botID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	credential, err := a.Bots.Credential(id)
	if errors.Is(err, bot.ErrNotFound) {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	inviteURL, err := a.BotNotify.Sender.InviteURL(ctx, credential.Provider, credential.Token, credential.DNSServer)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": inviteURL})
}

func (a *App) notifyBotEvent(event string, instanceID int64, subject string) {
	if a.BotNotify == nil || instanceID == 0 {
		return
	}
	instance, err := a.Instances.ByID(instanceID)
	if err != nil {
		return
	}
	name := strings.TrimSpace(instance.DisplayName)
	if name == "" {
		name = instance.Slug
	}
	var message string
	switch event {
	case bot.EventServerStarted:
		message = fmt.Sprintf("✅ Server started\n%s is fully started and accepting players.", name)
	case bot.EventServerStopped:
		switch subject {
		case "stopping", "restarting":
			message = fmt.Sprintf("⏹ Server stopping\n%s is shutting down.", name)
		case "crashed":
			message = fmt.Sprintf("⚠️ Server stopped unexpectedly\n%s is no longer running.", name)
		default:
			message = fmt.Sprintf("⏹ Server stopped\n%s is no longer running.", name)
		}
	case bot.EventPlayerJoined:
		message = fmt.Sprintf("➡ Player joined\n%s joined %s.", subject, name)
	case bot.EventPlayerLeft:
		message = fmt.Sprintf("⬅ Player left\n%s left %s.", subject, name)
	default:
		return
	}
	a.BotNotify.Notify(event, message)
}

func (a *App) markBotReady(instanceID int64) {
	a.botLifecycleMu.Lock()
	if a.botReadySent {
		a.botLifecycleMu.Unlock()
		return
	}
	a.botSawOnline = true
	a.botReadySent = true
	a.botStoppedSent = false
	a.botLifecycleMu.Unlock()
	a.notifyBotEvent(bot.EventServerStarted, instanceID, "")
}

func (a *App) observeBotLifecycle(state string) {
	instanceID := a.activeInstanceIDQuiet()
	notifyStopped := false
	stopState := ""
	a.botLifecycleMu.Lock()
	switch state {
	case "starting", "running":
		if !a.botSawOnline {
			a.botReadySent = false
			a.botStoppedSent = false
		}
		a.botSawOnline = true
	case "stopping", "restarting":
		if a.botSawOnline && !a.botStoppedSent {
			a.botStoppedSent = true
			notifyStopped = true
			stopState = state
		}
		a.botSawOnline = false
	case "stopped", "crashed":
		if a.botSawOnline && !a.botStoppedSent {
			a.botStoppedSent = true
			notifyStopped = true
			stopState = state
		}
		a.botSawOnline = false
	}
	a.botLifecycleMu.Unlock()
	if notifyStopped {
		a.notifyBotEvent(bot.EventServerStopped, instanceID, stopState)
	}
}
