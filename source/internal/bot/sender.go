package bot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Sender struct {
	Client              *http.Client
	TelegramBaseURL     string
	TelegramFileBaseURL string
	DiscordBaseURL      string
}

type TelegramDiscovery struct {
	BotUsername string        `json:"bot_username"`
	Groups      []Destination `json:"groups"`
}

type telegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
	IsForum  bool   `json:"is_forum"`
	Photo    *struct {
		SmallFileID string `json:"small_file_id"`
	} `json:"photo,omitempty"`
}

type telegramTopicEvent struct {
	Name string `json:"name"`
}

type telegramMessage struct {
	MessageID         int64               `json:"message_id"`
	MessageThreadID   int64               `json:"message_thread_id"`
	IsTopicMessage    bool                `json:"is_topic_message"`
	Text              string              `json:"text"`
	Chat              telegramChat        `json:"chat"`
	From              *telegramUser       `json:"from"`
	SenderChat        *telegramChat       `json:"sender_chat"`
	ForumTopicCreated *telegramTopicEvent `json:"forum_topic_created"`
	ForumTopicEdited  *telegramTopicEvent `json:"forum_topic_edited"`
	ReplyToMessage    *struct {
		ForumTopicCreated *telegramTopicEvent `json:"forum_topic_created"`
		ForumTopicEdited  *telegramTopicEvent `json:"forum_topic_edited"`
	} `json:"reply_to_message"`
}

type telegramUser struct {
	ID int64 `json:"id"`
}

type telegramUpdate struct {
	UpdateID          int64            `json:"update_id"`
	Message           *telegramMessage `json:"message"`
	EditedMessage     *telegramMessage `json:"edited_message"`
	ChannelPost       *telegramMessage `json:"channel_post"`
	EditedChannelPost *telegramMessage `json:"edited_channel_post"`
	MyChatMember      *struct {
		Chat          telegramChat `json:"chat"`
		NewChatMember struct {
			Status string `json:"status"`
		} `json:"new_chat_member"`
	} `json:"my_chat_member"`
}

func (s *Sender) telegramGet(ctx context.Context, token, method, query string, output any) error {
	endpoint := strings.TrimRight(s.TelegramBaseURL, "/") + "/bot" + url.PathEscape(token) + "/" + method
	if query != "" {
		endpoint += "?" + query
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "Bonghos/notification-bot")
	response, err := s.Client.Do(request)
	if err != nil {
		return errors.New("Telegram request failed")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return errors.New("reading Telegram response failed")
	}
	var envelope struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return errors.New("Telegram returned an invalid response")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !envelope.OK {
		detail := strings.TrimSpace(envelope.Description)
		if len(detail) > 200 {
			detail = detail[:200]
		}
		if detail != "" {
			return fmt.Errorf("Telegram returned HTTP %d: %s", response.StatusCode, detail)
		}
		return fmt.Errorf("Telegram returned HTTP %d", response.StatusCode)
	}
	if err := json.Unmarshal(envelope.Result, output); err != nil {
		return errors.New("Telegram returned invalid result data")
	}
	return nil
}

func (s *Sender) TelegramGroupPhoto(ctx context.Context, token, fileID string) ([]byte, string, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil, "", errors.New("Telegram group photo is unavailable")
	}
	var file struct {
		Path string `json:"file_path"`
	}
	query := url.Values{}
	query.Set("file_id", fileID)
	if err := s.telegramGet(ctx, token, "getFile", query.Encode(), &file); err != nil {
		return nil, "", err
	}
	parts := strings.Split(strings.Trim(file.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return nil, "", errors.New("Telegram returned an invalid group photo path")
	}
	for index, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, "", errors.New("Telegram returned an invalid group photo path")
		}
		parts[index] = url.PathEscape(part)
	}
	baseURL := s.TelegramFileBaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = s.TelegramBaseURL
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/file/bot" + url.PathEscape(token) + "/" + strings.Join(parts, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("User-Agent", "Bonghos/notification-bot")
	response, err := s.Client.Do(request)
	if err != nil {
		return nil, "", errors.New("Telegram group photo request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return nil, "", fmt.Errorf("Telegram group photo returned HTTP %d", response.StatusCode)
	}
	const maxPhotoBytes = 512 << 10
	data, err := io.ReadAll(io.LimitReader(response.Body, maxPhotoBytes+1))
	if err != nil {
		return nil, "", errors.New("reading Telegram group photo failed")
	}
	if len(data) == 0 || len(data) > maxPhotoBytes {
		return nil, "", errors.New("Telegram group photo is invalid or too large")
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return nil, "", errors.New("Telegram group photo has an unsupported format")
	}
	return data, contentType, nil
}

func (s *Sender) TelegramGroupPhotoForChat(ctx context.Context, token, chatID string) ([]byte, string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, "", errors.New("Telegram group is unavailable")
	}
	var fullChat telegramChat
	query := url.Values{}
	query.Set("chat_id", chatID)
	if err := s.telegramGet(ctx, token, "getChat", query.Encode(), &fullChat); err != nil {
		return nil, "", err
	}
	if fullChat.Photo == nil || strings.TrimSpace(fullChat.Photo.SmallFileID) == "" {
		return nil, "", errors.New("Telegram group has no profile photo")
	}
	return s.TelegramGroupPhoto(ctx, token, fullChat.Photo.SmallFileID)
}

// DiscoverTelegramGroups returns groups visible in the bot's pending updates.
// Discovery never enables a group; the owner must explicitly save selections.
func (s *Sender) DiscoverTelegramGroups(ctx context.Context, token string) (TelegramDiscovery, error) {
	token = strings.TrimSpace(token)
	if len(token) < 20 || strings.ContainsAny(token, " \t\r\n") {
		return TelegramDiscovery{}, errors.New("Telegram bot token is invalid")
	}
	var me struct {
		Username string `json:"username"`
	}
	if err := s.telegramGet(ctx, token, "getMe", "", &me); err != nil {
		return TelegramDiscovery{}, err
	}
	allowed, _ := json.Marshal([]string{"message", "edited_message", "channel_post", "edited_channel_post", "my_chat_member"})
	query := url.Values{}
	query.Set("allowed_updates", string(allowed))
	query.Set("limit", "100")
	query.Set("timeout", "0")
	var updates []telegramUpdate
	if err := s.telegramGet(ctx, token, "getUpdates", query.Encode(), &updates); err != nil {
		return TelegramDiscovery{}, err
	}
	byID := make(map[string]Destination)
	topicsByChat := make(map[string]map[int64]Topic)
	add := func(chat telegramChat) {
		if chat.Type != "group" && chat.Type != "supergroup" {
			return
		}
		id := strconv.FormatInt(chat.ID, 10)
		name := strings.TrimSpace(chat.Title)
		if name == "" && chat.Username != "" {
			name = "@" + chat.Username
		}
		if name == "" {
			name = id
		}
		current := byID[id]
		current.ID = id
		current.Name = name
		current.Type = chat.Type
		current.Forum = current.Forum || chat.IsForum
		byID[id] = current
	}
	addMessage := func(message *telegramMessage) {
		if message == nil {
			return
		}
		add(message.Chat)
		if message.Chat.Type != "group" && message.Chat.Type != "supergroup" {
			return
		}
		threadID := message.MessageThreadID
		if threadID <= 1 && message.ForumTopicCreated != nil {
			threadID = message.MessageID
		}
		if threadID <= 1 || (!message.IsTopicMessage && message.ForumTopicCreated == nil && message.ForumTopicEdited == nil) {
			return
		}
		chatID := strconv.FormatInt(message.Chat.ID, 10)
		if topicsByChat[chatID] == nil {
			topicsByChat[chatID] = make(map[int64]Topic)
		}
		name := ""
		for _, event := range []*telegramTopicEvent{
			message.ForumTopicCreated, message.ForumTopicEdited,
		} {
			if event != nil && strings.TrimSpace(event.Name) != "" {
				name = strings.TrimSpace(event.Name)
			}
		}
		if message.ReplyToMessage != nil {
			for _, event := range []*telegramTopicEvent{
				message.ReplyToMessage.ForumTopicCreated, message.ReplyToMessage.ForumTopicEdited,
			} {
				if event != nil && strings.TrimSpace(event.Name) != "" {
					name = strings.TrimSpace(event.Name)
				}
			}
		}
		if name == "" {
			if existing, ok := topicsByChat[chatID][threadID]; ok {
				name = existing.Name
			}
		}
		if name == "" {
			name = "Channel " + strconv.FormatInt(threadID, 10)
		}
		topicsByChat[chatID][threadID] = Topic{ID: threadID, Name: name}
		current := byID[chatID]
		current.Forum = true
		byID[chatID] = current
	}
	for _, update := range updates {
		addMessage(update.Message)
		addMessage(update.EditedMessage)
		addMessage(update.ChannelPost)
		addMessage(update.EditedChannelPost)
		if update.MyChatMember != nil {
			status := update.MyChatMember.NewChatMember.Status
			if status == "left" || status == "kicked" {
				delete(byID, strconv.FormatInt(update.MyChatMember.Chat.ID, 10))
			} else {
				add(update.MyChatMember.Chat)
			}
		}
	}
	groups := make([]Destination, 0, len(byID))
	for chatID, destination := range byID {
		for _, topic := range topicsByChat[chatID] {
			destination.Topics = append(destination.Topics, topic)
		}
		destination.Topics = dedupeTopics(destination.Topics)
		groups = append(groups, destination)
	}
	sort.Slice(groups, func(i, j int) bool {
		left, right := strings.ToLower(groups[i].Name), strings.ToLower(groups[j].Name)
		if left == right {
			return groups[i].ID < groups[j].ID
		}
		return left < right
	})
	// Telegram update objects contain only basic chat details. Fetch the full
	// chat for its 160px photo, but keep discovery useful if an individual
	// group photo is missing or cannot be downloaded.
	const maxPhotoGroups = 20
	for index := range groups {
		if index >= maxPhotoGroups {
			break
		}
		var fullChat telegramChat
		chatQuery := url.Values{}
		chatQuery.Set("chat_id", groups[index].ID)
		if err := s.telegramGet(ctx, token, "getChat", chatQuery.Encode(), &fullChat); err != nil || fullChat.Photo == nil {
			if err == nil {
				groups[index].Forum = groups[index].Forum || fullChat.IsForum
			}
			continue
		}
		groups[index].Forum = groups[index].Forum || fullChat.IsForum
		groups[index].PhotoFileID = strings.TrimSpace(fullChat.Photo.SmallFileID)
		if groups[index].PhotoFileID == "" {
			continue
		}
		data, contentType, err := s.TelegramGroupPhoto(ctx, token, groups[index].PhotoFileID)
		if err == nil {
			groups[index].PhotoDataURL = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
		}
	}
	return TelegramDiscovery{BotUsername: me.Username, Groups: groups}, nil
}

func NewSender() *Sender {
	return &Sender{
		Client:              &http.Client{Timeout: 12 * time.Second},
		TelegramBaseURL:     "https://api.telegram.org",
		TelegramFileBaseURL: "https://api.telegram.org",
		DiscordBaseURL:      "https://discord.com/api/v10",
	}
}

func (s *Sender) Send(ctx context.Context, target Target, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("notification message is empty")
	}
	switch target.Provider {
	case ProviderTelegram:
		return s.sendTelegram(ctx, target, message)
	case ProviderDiscord:
		return s.sendDiscord(ctx, target, message)
	default:
		return errors.New("unsupported notification provider")
	}
}

func (s *Sender) sendTelegram(ctx context.Context, target Target, message string) error {
	body := map[string]any{
		"chat_id": target.DestinationID,
		"text":    message,
	}
	if target.ThreadID > 0 {
		body["message_thread_id"] = target.ThreadID
	}
	endpoint := strings.TrimRight(s.TelegramBaseURL, "/") + "/bot" + url.PathEscape(target.Token) + "/sendMessage"
	return s.postJSON(ctx, endpoint, "", body, ProviderTelegram)
}

func (s *Sender) sendDiscord(ctx context.Context, target Target, message string) error {
	body := map[string]any{
		"content": message,
		"allowed_mentions": map[string]any{
			"parse": []string{},
		},
	}
	endpoint := strings.TrimRight(s.DiscordBaseURL, "/") + "/channels/" + url.PathEscape(target.DestinationID) + "/messages"
	return s.postJSON(ctx, endpoint, "Bot "+target.Token, body, ProviderDiscord)
}

func (s *Sender) postJSON(ctx context.Context, endpoint, authorization string, payload any, provider string) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Bonghos/notification-bot")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := s.Client.Do(request)
	if err != nil {
		// A Telegram token is part of the request URL. Do not wrap url.Error here,
		// because its formatted value can expose that URL (and therefore the token)
		// in the application log or API response.
		return fmt.Errorf("%s request failed", provider)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d", provider, response.StatusCode)
	}
	return nil
}

type Dispatcher struct {
	Store  *Store
	Sender *Sender
	Logf   func(string, ...any)
}

func (d *Dispatcher) Notify(event, message string) {
	if d == nil || d.Store == nil || d.Sender == nil {
		return
	}
	targets, err := d.Store.TargetsFor(event)
	if err != nil {
		if d.Logf != nil {
			d.Logf("notification targets for %s: %v", event, err)
		}
		return
	}
	for _, target := range targets {
		target := target
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			if err := d.Sender.Send(ctx, target, message); err != nil && d.Logf != nil {
				d.Logf("%s notification through %s failed: %v", event, target.Name, err)
			}
		}()
	}
}

func (d *Dispatcher) Test(ctx context.Context, id int64) error {
	targets, err := d.Store.Targets(id)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return errors.New("notification bot has no destinations")
	}
	for _, target := range targets {
		if err := d.Sender.Send(ctx, target, "✅ Bonghos test notification\nYour "+target.Name+" bot is connected."); err != nil {
			return err
		}
	}
	return nil
}
