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
		d.logCommand("listing Telegram command bots: %v", err)
		return
	}
	for _, state := range states {
		if ctx.Err() != nil {
			return
		}
		updates, err := d.Sender.WithDNS(state.DNSServer).telegramCommandUpdates(ctx, state.Token, state.LastUpdateID+1)
		if err != nil {
			d.logCommand("Telegram command polling for %s failed: %v", state.BotName, err)
			continue
		}
		if !state.Initialized {
			latest := state.LastUpdateID
			for _, update := range updates {
				d.observeTelegramMembership(state, update)
				if update.UpdateID > latest {
					latest = update.UpdateID
				}
			}
			if err := d.Store.InitializeTelegramUpdates(state.BotID, latest); err != nil {
				d.logCommand("initializing Telegram command cursor for %s: %v", state.BotName, err)
			}
			continue
		}
		for _, update := range updates {
			d.observeTelegramMembership(state, update)
			if update.Message != nil {
				d.handleTelegramCommand(ctx, state, update.Message)
			}
			if update.UpdateID > 0 {
				if err := d.Store.AdvanceTelegramUpdate(state.BotID, update.UpdateID); err != nil {
					d.logCommand("saving Telegram command cursor for %s: %v", state.BotName, err)
					break
				}
			}
		}
	}
}

func (s *Sender) telegramCommandUpdates(ctx context.Context, token string, offset int64) ([]telegramUpdate, error) {
	allowed, _ := json.Marshal([]string{"message", "my_chat_member"})
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

func telegramGroupContainer(chat telegramChat) Destination {
	return Destination{
		ID: strconv.FormatInt(chat.ID, 10), Name: strings.TrimSpace(chat.Title),
		Type: chat.Type, Forum: chat.IsForum,
	}
}

func (d *Dispatcher) observeTelegramMembership(state TelegramCommandState, update telegramUpdate) {
	membership := update.MyChatMember
	if membership == nil || (membership.Chat.Type != "group" && membership.Chat.Type != "supergroup") {
		return
	}
	chatID := strconv.FormatInt(membership.Chat.ID, 10)
	status := strings.ToLower(strings.TrimSpace(membership.NewChatMember.Status))
	if status == "left" || status == "kicked" ||
		(status == "restricted" && membership.NewChatMember.IsMember != nil && !*membership.NewChatMember.IsMember) {
		if err := d.Store.ForgetDiscovered(state.BotID, chatID); err != nil {
			d.logCommand("forgetting Telegram group for %s: %v", state.BotName, err)
		}
		return
	}
	if _, err := d.Store.MergeDiscovered(state.BotID, []Destination{telegramGroupContainer(membership.Chat)}); err != nil {
		d.logCommand("remembering Telegram group for %s: %v", state.BotName, err)
	}
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

func (d *Dispatcher) telegramAdmin(ctx context.Context, state TelegramCommandState, message *telegramMessage) bool {
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
	if err := d.Sender.WithDNS(state.DNSServer).telegramGet(ctx, state.Token, "getChatMember", query.Encode(), &member); err != nil {
		d.logCommand("checking Telegram group administrator: %v", err)
		return false
	}
	return member.Status == "creator" || member.Status == "administrator"
}

func (d *Dispatcher) handleTelegramCommand(ctx context.Context, state TelegramCommandState, message *telegramMessage) {
	if message.Chat.Type != "group" && message.Chat.Type != "supergroup" {
		return
	}
	if d.Store != nil && state.BotID > 0 {
		if _, err := d.Store.MergeDiscovered(state.BotID, []Destination{telegramGroupContainer(message.Chat)}); err != nil {
			d.logCommand("remembering Telegram command group for %s: %v", state.BotName, err)
		}
	}
	command := telegramCommand(message.Text)
	if command == "" {
		return
	}
	target := Target{Provider: ProviderTelegram, Token: state.Token, DNSServer: state.DNSServer,
		DestinationID: strconv.FormatInt(message.Chat.ID, 10)}
	if message.MessageThreadID > 1 {
		target.ThreadID = message.MessageThreadID
	}
	// Replies use Telegram HTML parse mode so command names render as tappable
	// monospace. Keep provider and storage errors in server logs rather than
	// exposing their details inside a group chat.
	reply := func(text string) {
		if err := d.Sender.SendHTML(ctx, target, text); err != nil {
			d.logCommand("replying to Telegram command for %s: %v", state.BotName, err)
		}
	}
	if command == "help" {
		reply("Bonghos commands:\n<code>/bonghos here</code> : Send notifications to this topic\n<code>/bonghos where</code> : Check this group's destination\n<code>/bonghos disconnect</code> : Stop notifications to this group\n\nOnly group administrators can change destinations.")
		return
	}
	if !d.telegramAdmin(ctx, state, message) {
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
		if d.Sender.WithDNS(state.DNSServer).telegramGet(ctx, state.Token, "getChat", query.Encode(), &fullChat) == nil {
			if strings.TrimSpace(fullChat.Title) != "" {
				destination.Name = strings.TrimSpace(fullChat.Title)
			}
			if fullChat.Photo != nil {
				destination.PhotoFileID = strings.TrimSpace(fullChat.Photo.SmallFileID)
			}
		}
		if err := d.Store.SetTelegramDestination(state.BotID, destination); err != nil {
			if strings.Contains(err.Error(), "three connected groups") {
				reply("Bonghos already has three connected Telegram groups.")
				return
			}
			d.logCommand("connecting a Telegram destination for %s: %v", state.BotName, err)
			reply("Bonghos could not connect this destination. Check the Bonghos logs for details.")
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
			reply("This group is not connected. Run <code>/bonghos here</code> in the topic that should receive notifications.")
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

func (d *Dispatcher) logCommand(format string, args ...any) {
	if d.Logf != nil {
		d.Logf(format, args...)
	}
}
