package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const telegramCommandPollInterval = 3 * time.Second

// RunTelegramCommands listens for destination commands for every configured
// Telegram bot. The durable update cursor ensures a command is applied once.
func (d *Dispatcher) RunTelegramCommands(ctx context.Context) {
	if d == nil || d.Store == nil || d.Sender == nil {
		return
	}
	for {
		d.pollTelegramCommands(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(telegramCommandPollInterval):
		}
	}
}

func (d *Dispatcher) pollTelegramCommands(ctx context.Context) {
	states, err := d.Store.TelegramCommandBots()
	if err != nil {
		d.logTelegramCommand("listing Telegram command bots: %v", err)
		return
	}
	for _, state := range states {
		if ctx.Err() != nil {
			return
		}
		updates, err := d.Sender.telegramCommandUpdates(ctx, state.Token, state.LastUpdateID+1)
		if err != nil {
			d.logTelegramCommand("Telegram command polling for %s failed: %v", state.BotName, err)
			continue
		}
		if !state.Initialized {
			latest := state.LastUpdateID
			for _, update := range updates {
				if update.UpdateID > latest {
					latest = update.UpdateID
				}
			}
			if err := d.Store.InitializeTelegramUpdates(state.BotID, latest); err != nil {
				d.logTelegramCommand("initializing Telegram command cursor for %s: %v", state.BotName, err)
			}
			continue
		}
		for _, update := range updates {
			if update.Message != nil {
				d.handleTelegramCommand(ctx, state, update.Message)
			}
			if update.UpdateID > 0 {
				if err := d.Store.AdvanceTelegramUpdate(state.BotID, update.UpdateID); err != nil {
					d.logTelegramCommand("saving Telegram command cursor for %s: %v", state.BotName, err)
					break
				}
			}
		}
	}
}

func (s *Sender) telegramCommandUpdates(ctx context.Context, token string, offset int64) ([]telegramUpdate, error) {
	allowed, _ := json.Marshal([]string{"message"})
	query := url.Values{}
	query.Set("allowed_updates", string(allowed))
	query.Set("limit", "100")
	query.Set("timeout", "0")
	if offset > 0 {
		query.Set("offset", strconv.FormatInt(offset, 10))
	}
	var updates []telegramUpdate
	if err := s.telegramGet(ctx, token, "getUpdates", query.Encode(), &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func telegramCommand(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return ""
	}
	root := strings.TrimPrefix(strings.ToLower(fields[0]), "/")
	root, _, _ = strings.Cut(root, "@")
	if root != "bonghos" {
		return ""
	}
	if len(fields) == 1 {
		return "help"
	}
	subcommand := strings.ToLower(fields[1])
	switch subcommand {
	case "here", "disconnect", "where", "help":
		return subcommand
	default:
		return "help"
	}
}

func (d *Dispatcher) telegramAdmin(ctx context.Context, token string, message *telegramMessage) bool {
	if message.SenderChat != nil && message.SenderChat.ID == message.Chat.ID {
		return true // Anonymous group administrator.
	}
	if message.From == nil || message.From.ID <= 0 {
		return false
	}
	var member struct {
		Status string `json:"status"`
	}
	query := url.Values{}
	query.Set("chat_id", strconv.FormatInt(message.Chat.ID, 10))
	query.Set("user_id", strconv.FormatInt(message.From.ID, 10))
	if err := d.Sender.telegramGet(ctx, token, "getChatMember", query.Encode(), &member); err != nil {
		d.logTelegramCommand("checking Telegram group administrator: %v", err)
		return false
	}
	return member.Status == "creator" || member.Status == "administrator"
}

func (d *Dispatcher) handleTelegramCommand(ctx context.Context, state TelegramCommandState, message *telegramMessage) {
	command := telegramCommand(message.Text)
	if command == "" || (message.Chat.Type != "group" && message.Chat.Type != "supergroup") {
		return
	}
	target := Target{Provider: ProviderTelegram, Token: state.Token,
		DestinationID: strconv.FormatInt(message.Chat.ID, 10)}
	if message.MessageThreadID > 1 {
		target.ThreadID = message.MessageThreadID
	}
	reply := func(text string) {
		if err := d.Sender.Send(ctx, target, text); err != nil {
			d.logTelegramCommand("replying to Telegram command for %s: %v", state.BotName, err)
		}
	}
	if command == "help" {
		// reply("Bonghos commands:\n`/bonghos here` : Send notifications to this topic\n`/bonghos where` : Check this group's destination\n`/bonghos disconnect` : Stop notifications to this group\n\nOnly group administrators can change destinations.")
		reply("Bonghos commands:\n<code>/bonghos here</code> : Send notifications to this topic\n<code>/bonghos where</code> : Check this group's destination\n<code>/bonghos disconnect</code> : Stop notifications to this group\n\nOnly group administrators can change destinations.")
		return
	}
	if !d.telegramAdmin(ctx, state.Token, message) {
		reply("Only a group administrator can configure Bonghos notifications.")
		return
	}
	chatID := target.DestinationID
	switch command {
	case "here":
		destination := Destination{ID: chatID, Name: strings.TrimSpace(message.Chat.Title), Type: message.Chat.Type,
			Forum: message.Chat.IsForum || target.ThreadID > 0, ThreadID: target.ThreadID}
		if target.ThreadID > 0 {
			destination.ThreadName = "Selected topic"
		} else {
			destination.ThreadName = "General"
		}
		var fullChat telegramChat
		query := url.Values{}
		query.Set("chat_id", chatID)
		if d.Sender.telegramGet(ctx, state.Token, "getChat", query.Encode(), &fullChat) == nil {
			if strings.TrimSpace(fullChat.Title) != "" {
				destination.Name = strings.TrimSpace(fullChat.Title)
			}
			if fullChat.Photo != nil {
				destination.PhotoFileID = strings.TrimSpace(fullChat.Photo.SmallFileID)
			}
		}
		if err := d.Store.SetTelegramDestination(state.BotID, destination); err != nil {
			reply("Bonghos could not connect this destination: " + err.Error())
			return
		}
		reply("Bonghos notifications will be sent here.")
	case "disconnect":
		if err := d.Store.DisconnectTelegramDestination(state.BotID, chatID); err != nil {
			reply("Bonghos could not disconnect this group.")
			return
		}
		reply("Bonghos notifications are disconnected from this group.")
	case "where":
		destination, err := d.Store.TelegramDestination(state.BotID, chatID)
		if errors.Is(err, ErrNotFound) {
			reply("This group is not connected. Run /bonghos here in the topic that should receive notifications.")
			return
		}
		if err != nil {
			reply("Bonghos could not read this destination.")
			return
		}
		if destination.ThreadID == target.ThreadID {
			reply("Bonghos notifications are configured for this topic.")
		} else {
			reply("Bonghos notifications are configured for another topic in this group.")
		}
	}
}

func (d *Dispatcher) logTelegramCommand(format string, args ...any) {
	if d.Logf != nil {
		d.Logf(format, args...)
	}
}
