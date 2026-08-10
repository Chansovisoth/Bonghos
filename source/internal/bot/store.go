// Package bot stores encrypted notification-bot credentials and delivers
// Bonghos events through Telegram and Discord.
package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/security"
)

const (
	ProviderTelegram = "telegram"
	ProviderDiscord  = "discord"

	EventServerStarted = "server_started"
	EventServerStopped = "server_stopped"
	EventPlayerJoined  = "player_joined"
	EventPlayerLeft    = "player_left"
)

var (
	telegramDestinationRE = regexp.MustCompile(`^(?:-?\d+|@[A-Za-z0-9_]{5,})$`)
	discordDestinationRE  = regexp.MustCompile(`^\d{10,25}$`)
	ErrNotFound           = errors.New("notification bot not found")
)

type Config struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	Provider            string `json:"provider"`
	DestinationID       string `json:"destination_id"`
	Enabled             bool   `json:"enabled"`
	NotifyServerStarted bool   `json:"notify_server_started"`
	NotifyServerStopped bool   `json:"notify_server_stopped"`
	NotifyPlayerJoined  bool   `json:"notify_player_joined"`
	NotifyPlayerLeft    bool   `json:"notify_player_left"`
	TokenConfigured     bool   `json:"token_configured"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type CreateInput struct {
	Name                string
	Provider            string
	Token               string
	DestinationID       string
	Enabled             bool
	NotifyServerStarted bool
	NotifyServerStopped bool
	NotifyPlayerJoined  bool
	NotifyPlayerLeft    bool
}

type Patch struct {
	Name                *string
	Provider            *string
	Token               *string
	DestinationID       *string
	Enabled             *bool
	NotifyServerStarted *bool
	NotifyServerStopped *bool
	NotifyPlayerJoined  *bool
	NotifyPlayerLeft    *bool
}

type Target struct {
	ID            int64
	Name          string
	Provider      string
	Token         string
	DestinationID string
}

type Store struct {
	DB        *sql.DB
	SecretKey []byte
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func normalizeProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validate(name, provider, token, destination string, requireToken bool) error {
	name = strings.TrimSpace(name)
	provider = normalizeProvider(provider)
	token = strings.TrimSpace(token)
	destination = strings.TrimSpace(destination)
	if name == "" || len([]rune(name)) > 80 {
		return errors.New("bot name must be between 1 and 80 characters")
	}
	if provider != ProviderTelegram && provider != ProviderDiscord {
		return errors.New("provider must be telegram or discord")
	}
	if requireToken && token == "" {
		return errors.New("bot token is required")
	}
	if token != "" && (len(token) < 20 || strings.ContainsAny(token, " \t\r\n")) {
		return errors.New("bot token is invalid")
	}
	if provider == ProviderTelegram && !telegramDestinationRE.MatchString(destination) {
		return errors.New("Telegram chat ID must be a numeric ID or @channel username")
	}
	if provider == ProviderDiscord && !discordDestinationRE.MatchString(destination) {
		return errors.New("Discord channel ID must be a numeric snowflake")
	}
	return nil
}

func (s *Store) Create(input CreateInput) (*Config, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Provider = normalizeProvider(input.Provider)
	input.Token = strings.TrimSpace(input.Token)
	input.DestinationID = strings.TrimSpace(input.DestinationID)
	if err := validate(input.Name, input.Provider, input.Token, input.DestinationID, true); err != nil {
		return nil, err
	}
	encrypted, err := security.Encrypt(s.SecretKey, []byte(input.Token))
	if err != nil {
		return nil, err
	}
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO notification_bots
		(name, provider, token_enc, destination_id, enabled,
		 notify_server_started, notify_server_stopped, notify_player_joined, notify_player_left,
		 created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		input.Name, input.Provider, encrypted, input.DestinationID, boolInt(input.Enabled),
		boolInt(input.NotifyServerStarted), boolInt(input.NotifyServerStopped),
		boolInt(input.NotifyPlayerJoined), boolInt(input.NotifyPlayerLeft), timestamp, timestamp)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.ByID(id)
}

const configColumns = `id, name, provider, destination_id, enabled,
 notify_server_started, notify_server_stopped, notify_player_joined, notify_player_left,
 created_at, updated_at, length(token_enc) > 0`

func scanConfig(row interface{ Scan(...any) error }) (*Config, error) {
	var config Config
	var enabled, started, stopped, joined, left, tokenConfigured int
	if err := row.Scan(&config.ID, &config.Name, &config.Provider, &config.DestinationID,
		&enabled, &started, &stopped, &joined, &left, &config.CreatedAt, &config.UpdatedAt,
		&tokenConfigured); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	config.Enabled = enabled != 0
	config.NotifyServerStarted = started != 0
	config.NotifyServerStopped = stopped != 0
	config.NotifyPlayerJoined = joined != 0
	config.NotifyPlayerLeft = left != 0
	config.TokenConfigured = tokenConfigured != 0
	return &config, nil
}

func (s *Store) ByID(id int64) (*Config, error) {
	return scanConfig(s.DB.QueryRow(`SELECT `+configColumns+` FROM notification_bots WHERE id=?`, id))
}

func (s *Store) List() ([]*Config, error) {
	rows, err := s.DB.Query(`SELECT ` + configColumns + ` FROM notification_bots ORDER BY provider, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*Config
	for rows.Next() {
		config, scanErr := scanConfig(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		configs = append(configs, config)
	}
	return configs, rows.Err()
}

func (s *Store) Patch(id int64, patch Patch) (*Config, error) {
	var current CreateInput
	var encrypted []byte
	var enabled, started, stopped, joined, left int
	err := s.DB.QueryRow(`SELECT name, provider, token_enc, destination_id, enabled,
		notify_server_started, notify_server_stopped, notify_player_joined, notify_player_left
		FROM notification_bots WHERE id=?`, id).Scan(
		&current.Name, &current.Provider, &encrypted, &current.DestinationID,
		&enabled, &started, &stopped, &joined, &left)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	current.Enabled = enabled != 0
	current.NotifyServerStarted = started != 0
	current.NotifyServerStopped = stopped != 0
	current.NotifyPlayerJoined = joined != 0
	current.NotifyPlayerLeft = left != 0
	originalProvider := current.Provider
	if patch.Name != nil {
		current.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Provider != nil {
		current.Provider = normalizeProvider(*patch.Provider)
	}
	if patch.DestinationID != nil {
		current.DestinationID = strings.TrimSpace(*patch.DestinationID)
	}
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.NotifyServerStarted != nil {
		current.NotifyServerStarted = *patch.NotifyServerStarted
	}
	if patch.NotifyServerStopped != nil {
		current.NotifyServerStopped = *patch.NotifyServerStopped
	}
	if patch.NotifyPlayerJoined != nil {
		current.NotifyPlayerJoined = *patch.NotifyPlayerJoined
	}
	if patch.NotifyPlayerLeft != nil {
		current.NotifyPlayerLeft = *patch.NotifyPlayerLeft
	}
	newToken := ""
	if patch.Token != nil {
		newToken = strings.TrimSpace(*patch.Token)
	}
	if current.Provider != originalProvider && newToken == "" {
		return nil, errors.New("a new token is required when changing provider")
	}
	if err := validate(current.Name, current.Provider, newToken, current.DestinationID, false); err != nil {
		return nil, err
	}
	if newToken != "" {
		encrypted, err = security.Encrypt(s.SecretKey, []byte(newToken))
		if err != nil {
			return nil, err
		}
	}
	result, err := s.DB.Exec(`UPDATE notification_bots SET
		name=?, provider=?, token_enc=?, destination_id=?, enabled=?,
		notify_server_started=?, notify_server_stopped=?, notify_player_joined=?, notify_player_left=?,
		updated_at=? WHERE id=?`, current.Name, current.Provider, encrypted, current.DestinationID,
		boolInt(current.Enabled), boolInt(current.NotifyServerStarted), boolInt(current.NotifyServerStopped),
		boolInt(current.NotifyPlayerJoined), boolInt(current.NotifyPlayerLeft), now(), id)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}
	return s.ByID(id)
}

func (s *Store) Delete(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM notification_bots WHERE id=?`, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Target(id int64) (Target, error) {
	var target Target
	var encrypted []byte
	err := s.DB.QueryRow(`SELECT id, name, provider, token_enc, destination_id
		FROM notification_bots WHERE id=?`, id).Scan(
		&target.ID, &target.Name, &target.Provider, &encrypted, &target.DestinationID)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	if err != nil {
		return Target{}, err
	}
	token, err := security.Decrypt(s.SecretKey, encrypted)
	if err != nil {
		return Target{}, fmt.Errorf("decrypting %s bot credentials: %w", target.Provider, err)
	}
	target.Token = string(token)
	return target, nil
}

func (s *Store) TargetsFor(event string) ([]Target, error) {
	var query string
	switch event {
	case EventServerStarted:
		query = `SELECT id, name, provider, token_enc, destination_id
			FROM notification_bots WHERE enabled=1 AND notify_server_started=1 ORDER BY id`
	case EventServerStopped:
		query = `SELECT id, name, provider, token_enc, destination_id
			FROM notification_bots WHERE enabled=1 AND notify_server_stopped=1 ORDER BY id`
	case EventPlayerJoined:
		query = `SELECT id, name, provider, token_enc, destination_id
			FROM notification_bots WHERE enabled=1 AND notify_player_joined=1 ORDER BY id`
	case EventPlayerLeft:
		query = `SELECT id, name, provider, token_enc, destination_id
			FROM notification_bots WHERE enabled=1 AND notify_player_left=1 ORDER BY id`
	default:
		return nil, errors.New("unknown notification event")
	}
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []Target
	for rows.Next() {
		var target Target
		var encrypted []byte
		if err := rows.Scan(&target.ID, &target.Name, &target.Provider, &encrypted, &target.DestinationID); err != nil {
			return nil, err
		}
		token, decryptErr := security.Decrypt(s.SecretKey, encrypted)
		if decryptErr != nil {
			return nil, decryptErr
		}
		target.Token = string(token)
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
