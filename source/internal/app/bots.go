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
	Name                string  `json:"name"`
	Provider            string  `json:"provider"`
	Token               *string `json:"token"`
	DestinationID       string  `json:"destination_id"`
	Enabled             *bool   `json:"enabled"`
	NotifyServerStarted *bool   `json:"notify_server_started"`
	NotifyServerStopped *bool   `json:"notify_server_stopped"`
	NotifyPlayerJoined  *bool   `json:"notify_player_joined"`
	NotifyPlayerLeft    *bool   `json:"notify_player_left"`
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
		DestinationID:       request.DestinationID,
		Enabled:             boolDefault(request.Enabled, true),
		NotifyServerStarted: boolDefault(request.NotifyServerStarted, true),
		NotifyServerStopped: boolDefault(request.NotifyServerStopped, true),
		NotifyPlayerJoined:  boolDefault(request.NotifyPlayerJoined, true),
		NotifyPlayerLeft:    boolDefault(request.NotifyPlayerLeft, true),
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
		Name                *string `json:"name"`
		Provider            *string `json:"provider"`
		Token               *string `json:"token"`
		DestinationID       *string `json:"destination_id"`
		Enabled             *bool   `json:"enabled"`
		NotifyServerStarted *bool   `json:"notify_server_started"`
		NotifyServerStopped *bool   `json:"notify_server_stopped"`
		NotifyPlayerJoined  *bool   `json:"notify_player_joined"`
		NotifyPlayerLeft    *bool   `json:"notify_player_left"`
	}
	if err := readJSON(r, &request, 1<<16); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid request"))
		return
	}
	config, err := a.Bots.Patch(id, bot.Patch{
		Name: request.Name, Provider: request.Provider, Token: request.Token,
		DestinationID: request.DestinationID, Enabled: request.Enabled,
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

func (a *App) notifyBotEvent(event string, instanceID int64, player string) {
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
		message = fmt.Sprintf("⏹ Server stopped\n%s has fully stopped.", name)
	case bot.EventPlayerJoined:
		message = fmt.Sprintf("➡ Player joined\n%s joined %s.", player, name)
	case bot.EventPlayerLeft:
		message = fmt.Sprintf("⬅ Player left\n%s left %s.", player, name)
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
	a.botLifecycleMu.Lock()
	switch state {
	case "starting", "running", "stopping", "restarting":
		if !a.botSawOnline {
			a.botReadySent = false
			a.botStoppedSent = false
		}
		a.botSawOnline = true
	case "stopped", "crashed":
		if a.botSawOnline && !a.botStoppedSent {
			a.botStoppedSent = true
			notifyStopped = true
		}
		a.botSawOnline = false
	}
	a.botLifecycleMu.Unlock()
	if notifyStopped {
		a.notifyBotEvent(bot.EventServerStopped, instanceID, "")
	}
}
