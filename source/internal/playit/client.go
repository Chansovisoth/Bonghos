package playit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.playit.gg"

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	BaseURL string
	HTTP    HTTPDoer
	Version string
}

type apiEnvelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

func (c *Client) post(ctx context.Context, path, secret string, request, response any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = defaultAPIBase
	}
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Bonghos/"+c.clientVersion())
	if secret != "" {
		req.Header.Set("Authorization", "Agent-Key "+strings.TrimSpace(secret))
	}
	doer := c.HTTP
	if doer == nil {
		doer = &http.Client{Timeout: 12 * time.Second}
	}
	res, err := doer.Do(req)
	if err != nil {
		return fmt.Errorf("Playit request failed: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusTooManyRequests {
		return errors.New("Playit is rate limiting requests; try again shortly")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Playit returned HTTP %d", res.StatusCode)
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return errors.New("Playit returned an unreadable response")
	}
	if envelope.Status != "success" {
		var detail string
		if err := json.Unmarshal(envelope.Data, &detail); err != nil || detail == "" {
			detail = strings.TrimSpace(string(envelope.Data))
		}
		if len(detail) > 180 {
			detail = detail[:180]
		}
		if detail == "" || detail == "null" || detail == "{}" {
			detail = envelope.Status
		}
		return fmt.Errorf("Playit rejected the request: %s", detail)
	}
	if response == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, response); err != nil {
		return errors.New("Playit returned an incompatible response")
	}
	return nil
}

func (c *Client) clientVersion() string {
	if version := strings.TrimSpace(c.Version); version != "" {
		return version
	}
	return "dev"
}

func (c *Client) ClaimSetup(ctx context.Context, code string) (string, error) {
	var state string
	err := c.post(ctx, "/claim/setup", "", map[string]any{
		"code": code, "agent_type": "assignable", "version": "Bonghos " + c.clientVersion(),
	}, &state)
	return state, err
}

func (c *Client) ClaimExchange(ctx context.Context, code string) (string, error) {
	var result struct {
		Secret string `json:"secret_key"`
	}
	if err := c.post(ctx, "/claim/exchange", "", map[string]string{"code": code}, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Secret) == "" {
		return "", errors.New("Playit claim completed without an agent secret")
	}
	return result.Secret, nil
}

// GuestLogin returns a short-lived Playit Web session for managing a guest
// account. It is only called after the agent credential has been encrypted.
func (c *Client) GuestLogin(ctx context.Context, secret string) (string, error) {
	var result struct {
		SessionKey string `json:"session_key"`
	}
	if err := c.post(ctx, "/login/guest", secret, struct{}{}, &result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.SessionKey) == "" {
		return "", errors.New("Playit returned an empty guest session")
	}
	return "https://playit.gg/login/guest-account/" + result.SessionKey, nil
}

type RunData struct {
	AgentID     string       `json:"agent_id"`
	Tunnels     []TunnelData `json:"tunnels"`
	Pending     []TunnelData `json:"pending"`
	Permissions struct {
		AccountStatus string `json:"account_status"`
	} `json:"permissions"`
}

type TunnelData struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayAddress string `json:"display_address"`
	StatusMessage  string `json:"status_msg"`
	DisabledReason string `json:"disabled_reason"`
}

func (c *Client) RunData(ctx context.Context, secret string) (RunData, error) {
	var result RunData
	err := c.post(ctx, "/v1/agents/rundata", secret, struct{}{}, &result)
	return result, err
}

func (c *Client) CreateMinecraftTunnel(ctx context.Context, secret, agentID string, localPort int) (string, error) {
	if localPort < 1 || localPort > 65535 {
		return "", errors.New("Minecraft port must be between 1 and 65535")
	}
	var result struct {
		ID string `json:"id"`
	}
	request := map[string]any{
		"ports": map[string]any{"type": "tunnel-type", "details": "minecraft-java"},
		"origin": map[string]any{
			"type": "agent",
			"data": map[string]any{
				"agent_id": agentID,
				"config": map[string]any{"fields": []map[string]string{
					{"name": "local_ip", "value": "127.0.0.1"},
					{"name": "local_port", "value": fmt.Sprint(localPort)},
				}},
			},
		},
		"enabled": true, "alloc": nil, "name": "Bonghos Minecraft", "firewall_id": nil,
	}
	if err := c.post(ctx, "/v1/tunnels/create", secret, request, &result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", errors.New("Playit created a tunnel without an identifier")
	}
	return result.ID, nil
}

func (c *Client) UpdateTunnelPort(ctx context.Context, secret, tunnelID string, localPort int) error {
	return c.post(ctx, "/v1/tunnels/config", secret, map[string]any{
		"tunnel_id":    tunnelID,
		"new_agent_id": nil,
		"new_config": map[string]any{"fields": []map[string]string{
			{"name": "local_ip", "value": "127.0.0.1"},
			{"name": "local_port", "value": fmt.Sprint(localPort)},
		}},
	}, nil)
}
