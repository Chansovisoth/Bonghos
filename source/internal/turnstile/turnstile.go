// Package turnstile stores Cloudflare Turnstile credentials and validates
// single-use login tokens without exposing the widget secret to the browser.
package turnstile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/security"
)

const defaultVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var (
	ErrRequired    = errors.New("complete the security check")
	ErrRejected    = errors.New("security check failed; try again")
	ErrUnavailable = errors.New("security check is temporarily unavailable; try again")
)

// Config is safe to return to an authenticated Owner. The secret itself is
// never returned after it has been saved.
type Config struct {
	Enabled          bool   `json:"enabled"`
	SiteKey          string `json:"site_key"`
	SecretConfigured bool   `json:"secret_configured"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// PublicConfig contains only what an unauthenticated login page needs.
type PublicConfig struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"site_key,omitempty"`
}

// Update retains the existing secret when SecretKey is nil. ClearSecret is
// explicit so an omitted or blank form value cannot accidentally erase it.
type Update struct {
	Enabled     bool
	SiteKey     string
	SecretKey   *string
	ClearSecret bool
	UpdatedBy   int64
}

type Store struct {
	DB        *sql.DB
	SecretKey []byte
}

type storedConfig struct {
	Config
	secret []byte
}

func (s *Store) load() (storedConfig, error) {
	var out storedConfig
	var enabled int
	var encrypted []byte
	err := s.DB.QueryRow(`SELECT enabled, site_key, secret_key_enc, updated_at
		FROM turnstile_settings WHERE id=1`).Scan(&enabled, &out.SiteKey, &encrypted, &out.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return storedConfig{}, err
	}
	out.Enabled = enabled != 0
	out.SecretConfigured = len(encrypted) > 0
	if len(encrypted) > 0 {
		plain, err := security.Decrypt(s.SecretKey, encrypted)
		if err != nil {
			return storedConfig{}, fmt.Errorf("decrypting Turnstile secret: %w", err)
		}
		out.secret = plain
	}
	return out, nil
}

func (s *Store) Config() (Config, error) {
	stored, err := s.load()
	return stored.Config, err
}

func (s *Store) PublicConfig() (PublicConfig, error) {
	stored, err := s.load()
	if err != nil {
		return PublicConfig{}, err
	}
	if !stored.Enabled || strings.TrimSpace(stored.SiteKey) == "" || len(stored.secret) == 0 {
		return PublicConfig{}, nil
	}
	return PublicConfig{Enabled: true, SiteKey: stored.SiteKey}, nil
}

func (s *Store) Update(input Update) (Config, error) {
	current, err := s.load()
	if err != nil {
		return Config{}, err
	}
	siteKey := strings.TrimSpace(input.SiteKey)
	if len(siteKey) > 256 {
		return Config{}, errors.New("Turnstile site key is too long")
	}
	secret := current.secret
	if input.ClearSecret {
		secret = nil
	}
	if input.SecretKey != nil {
		value := strings.TrimSpace(*input.SecretKey)
		if len(value) > 512 {
			return Config{}, errors.New("Turnstile secret key is too long")
		}
		if value != "" {
			secret = []byte(value)
		}
	}
	if input.Enabled && (siteKey == "" || len(secret) == 0) {
		return Config{}, errors.New("site key and secret key are required before enabling Turnstile")
	}
	var encrypted []byte
	if len(secret) > 0 {
		encrypted, err = security.Encrypt(s.SecretKey, secret)
		if err != nil {
			return Config{}, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.Exec(`INSERT INTO turnstile_settings
		(id, enabled, site_key, secret_key_enc, updated_at, updated_by)
		VALUES (1,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled, site_key=excluded.site_key,
		secret_key_enc=excluded.secret_key_enc, updated_at=excluded.updated_at, updated_by=excluded.updated_by`,
		boolInt(input.Enabled), siteKey, encrypted, now, nullableID(input.UpdatedBy))
	if err != nil {
		return Config{}, err
	}
	return Config{Enabled: input.Enabled, SiteKey: siteKey, SecretConfigured: len(secret) > 0, UpdatedAt: now}, nil
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Service combines encrypted settings with Cloudflare's Siteverify endpoint.
// Client and VerifyURL are injectable so the real login handler can be tested
// without contacting Cloudflare.
type Service struct {
	Store     *Store
	Client    httpDoer
	VerifyURL string
}

type verifyResponse struct {
	Success  bool     `json:"success"`
	Hostname string   `json:"hostname"`
	Action   string   `json:"action"`
	Errors   []string `json:"error-codes"`
}

func (s *Service) VerifyLogin(ctx context.Context, token, requestHost string) error {
	stored, err := s.Store.load()
	if err != nil {
		return ErrUnavailable
	}
	if !stored.Enabled {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return ErrRequired
	}
	form := url.Values{"secret": {string(stored.secret)}, "response": {token}}
	verifyURL := s.VerifyURL
	if verifyURL == "" {
		verifyURL = defaultVerifyURL
	}
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(verifyCtx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 6 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ErrUnavailable
	}
	var result verifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
		return ErrUnavailable
	}
	if !result.Success || result.Action != "login" || !sameHost(result.Hostname, requestHost) {
		return ErrRejected
	}
	return nil
}

func sameHost(verified, requested string) bool {
	verified = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(verified), "."))
	requested = strings.TrimSpace(requested)
	if host, _, err := net.SplitHostPort(requested); err == nil {
		requested = host
	}
	requested = strings.ToLower(strings.TrimSuffix(strings.Trim(requested, "[]"), "."))
	return verified != "" && verified == requested
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}
