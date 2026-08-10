package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Sender struct {
	Client          *http.Client
	TelegramBaseURL string
	DiscordBaseURL  string
}

func NewSender() *Sender {
	return &Sender{
		Client:          &http.Client{Timeout: 12 * time.Second},
		TelegramBaseURL: "https://api.telegram.org",
		DiscordBaseURL:  "https://discord.com/api/v10",
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
	target, err := d.Store.Target(id)
	if err != nil {
		return err
	}
	return d.Sender.Send(ctx, target, "✅ Bonghos test notification\nYour "+target.Name+" bot is connected.")
}
