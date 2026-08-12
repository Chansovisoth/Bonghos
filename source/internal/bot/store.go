// Package bot stores encrypted notification-bot credentials and delivers
// Bonghos events through Telegram and Discord.
package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
	ID                     int64         `json:"id"`
	Name                   string        `json:"name"`
	Provider               string        `json:"provider"`
	DestinationID          string        `json:"destination_id"`
	Destinations           []Destination `json:"destinations"`
	DiscoveredDestinations []Destination `json:"discovered_destinations,omitempty"`
	Enabled                bool          `json:"enabled"`
	NotifyServerStarted    bool          `json:"notify_server_started"`
	NotifyServerStopped    bool          `json:"notify_server_stopped"`
	NotifyPlayerJoined     bool          `json:"notify_player_joined"`
	NotifyPlayerLeft       bool          `json:"notify_player_left"`
	TokenConfigured        bool          `json:"token_configured"`
	CreatedAt              string        `json:"created_at"`
	UpdatedAt              string        `json:"updated_at"`
}

type Destination struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	PhotoFileID  string  `json:"photo_file_id,omitempty"`
	PhotoDataURL string  `json:"photo_data_url,omitempty"`
	Forum        bool    `json:"forum,omitempty"`
	ThreadID     int64   `json:"thread_id,omitempty"`
	ThreadName   string  `json:"thread_name,omitempty"`
	Topics       []Topic `json:"topics,omitempty"`
}

type Topic struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type CreateInput struct {
	Name                   string
	Provider               string
	Token                  string
	DestinationID          string
	Destinations           []Destination
	DiscoveredDestinations []Destination
	Enabled                bool
	NotifyServerStarted    bool
	NotifyServerStopped    bool
	NotifyPlayerJoined     bool
	NotifyPlayerLeft       bool
}

type Patch struct {
	Name                *string
	Provider            *string
	Token               *string
	DestinationID       *string
	Destinations        *[]Destination
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
	ThreadID      int64
}

type TelegramCommandState struct {
	BotID        int64
	BotName      string
	Token        string
	LastUpdateID int64
	Initialized  bool
}

type Store struct {
	DB        *sql.DB
	SecretKey []byte
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func normalizeProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validate(name, provider, token string, requireToken bool) error {
	name = strings.TrimSpace(name)
	provider = normalizeProvider(provider)
	token = strings.TrimSpace(token)
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
	return nil
}

func normalizeDestinations(provider string, values []Destination, legacy string) ([]Destination, error) {
	if len(values) == 0 && strings.TrimSpace(legacy) != "" {
		values = []Destination{{ID: legacy}}
	}
	limit := 1
	if provider == ProviderTelegram {
		limit = 3
	}
	if len(values) == 0 && provider == ProviderTelegram {
		return []Destination{}, nil
	}
	if len(values) == 0 {
		return nil, errors.New("at least one notification destination is required")
	}
	if len(values) > limit {
		return nil, fmt.Errorf("%s supports at most %d notification destinations", provider, limit)
	}
	seen := make(map[string]bool, len(values))
	normalized := make([]Destination, 0, len(values))
	for _, value := range values {
		value.ID = strings.TrimSpace(value.ID)
		value.Name = strings.TrimSpace(value.Name)
		value.Type = strings.ToLower(strings.TrimSpace(value.Type))
		value.PhotoFileID = strings.TrimSpace(value.PhotoFileID)
		value.PhotoDataURL = ""
		value.ThreadName = strings.TrimSpace(value.ThreadName)
		value.Topics = nil
		if seen[value.ID] {
			return nil, errors.New("notification destinations must be unique")
		}
		seen[value.ID] = true
		if provider == ProviderTelegram && !telegramDestinationRE.MatchString(value.ID) {
			return nil, errors.New("Telegram chat ID must be a numeric ID or @channel username")
		}
		if provider == ProviderDiscord && !discordDestinationRE.MatchString(value.ID) {
			return nil, errors.New("Discord channel ID must be a numeric snowflake")
		}
		if value.ThreadID < 0 {
			return nil, errors.New("Telegram channel ID is invalid")
		}
		if provider != ProviderTelegram {
			value.Forum = false
			value.ThreadID = 0
			value.ThreadName = ""
		}
		if len([]rune(value.Name)) > 120 || len(value.Type) > 30 || len(value.PhotoFileID) > 512 || len([]rune(value.ThreadName)) > 128 {
			return nil, errors.New("notification destination metadata is too long")
		}
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func dedupeTopics(values []Topic) []Topic {
	byID := make(map[int64]Topic, len(values))
	for _, topic := range values {
		topic.Name = cleanTopicName(topic.Name)
		if topic.ID <= 1 {
			continue
		}
		if topic.Name == "" {
			topic.Name = "Channel " + strconv.FormatInt(topic.ID, 10)
		}
		byID[topic.ID] = topic
	}
	byName := make(map[string]Topic, len(byID))
	for _, topic := range byID {
		key := topicNameKey(topic.Name)
		if current, ok := byName[key]; !ok || topic.ID > current.ID {
			byName[key] = topic
		}
	}
	deduplicated := make([]Topic, 0, len(byName))
	for _, topic := range byName {
		deduplicated = append(deduplicated, topic)
	}
	sort.Slice(deduplicated, func(i, j int) bool {
		return strings.ToLower(deduplicated[i].Name) < strings.ToLower(deduplicated[j].Name)
	})
	return deduplicated
}

func topicNameKey(value string) string {
	return strings.ToLower(cleanTopicName(value))
}

func cleanTopicName(value string) string {
	value = strings.Map(func(r rune) rune {
		if (r >= '\u200b' && r <= '\u200d') || r == '\u2060' || r == '\ufeff' {
			return -1
		}
		return r
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func normalizeDiscovered(values []Destination) ([]Destination, error) {
	if len(values) > 100 {
		return nil, errors.New("too many discovered Telegram groups")
	}
	byID := make(map[string]Destination, len(values))
	for _, value := range values {
		value.ID = strings.TrimSpace(value.ID)
		value.Name = strings.TrimSpace(value.Name)
		value.Type = strings.ToLower(strings.TrimSpace(value.Type))
		value.PhotoFileID = strings.TrimSpace(value.PhotoFileID)
		value.PhotoDataURL = ""
		value.ThreadID = 0
		value.ThreadName = ""
		if !telegramDestinationRE.MatchString(value.ID) {
			return nil, errors.New("Telegram chat ID must be a numeric ID or @channel username")
		}
		if len([]rune(value.Name)) > 120 || len(value.Type) > 30 || len(value.PhotoFileID) > 512 {
			return nil, errors.New("discovered Telegram group metadata is too long")
		}
		topics := make([]Topic, 0, len(value.Topics))
		for _, topic := range value.Topics {
			topic.Name = strings.TrimSpace(topic.Name)
			if topic.ID <= 1 || len([]rune(topic.Name)) > 128 {
				continue
			}
			if topic.Name == "" {
				topic.Name = "Channel " + strconv.FormatInt(topic.ID, 10)
			}
			topics = append(topics, topic)
		}
		if len(topics) > 100 {
			return nil, errors.New("too many discovered Telegram channels")
		}
		value.Topics = dedupeTopics(topics)
		if existing, ok := byID[value.ID]; ok {
			value.Topics = dedupeTopics(append(existing.Topics, value.Topics...))
			value.Forum = value.Forum || existing.Forum
			if value.Name == "" {
				value.Name = existing.Name
			}
			if value.PhotoFileID == "" {
				value.PhotoFileID = existing.PhotoFileID
			}
		}
		byID[value.ID] = value
	}
	normalized := make([]Destination, 0, len(byID))
	for _, value := range byID {
		value.Topics = dedupeTopics(value.Topics)
		normalized = append(normalized, value)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return strings.ToLower(normalized[i].Name) < strings.ToLower(normalized[j].Name)
	})
	return normalized, nil
}

func insertDestinations(tx *sql.Tx, botID int64, destinations []Destination) error {
	for position, destination := range destinations {
		if _, err := tx.Exec(`INSERT INTO notification_bot_destinations
			(bot_id, destination_id, display_name, destination_type, photo_file_id, thread_id, thread_name, position)
			VALUES (?,?,?,?,?,?,?,?)`, botID, destination.ID, destination.Name, destination.Type,
			destination.PhotoFileID, destination.ThreadID, destination.ThreadName, position); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) destinations(botID int64) ([]Destination, error) {
	rows, err := s.DB.Query(`SELECT destination_id, display_name, destination_type, photo_file_id, thread_id, thread_name
		FROM notification_bot_destinations WHERE bot_id=? ORDER BY position, destination_id`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Destination, 0)
	for rows.Next() {
		var destination Destination
		if err := rows.Scan(&destination.ID, &destination.Name, &destination.Type, &destination.PhotoFileID,
			&destination.ThreadID, &destination.ThreadName); err != nil {
			return nil, err
		}
		values = append(values, destination)
	}
	return values, rows.Err()
}

func topicsFromJSON(raw string) []Topic {
	var topics []Topic
	if json.Unmarshal([]byte(raw), &topics) != nil {
		return []Topic{}
	}
	return topics
}

func mergeTopics(existing, incoming []Topic) []Topic {
	return dedupeTopics(append(existing, incoming...))
}

func mergeDiscoveredTx(tx *sql.Tx, botID int64, values []Destination) error {
	for _, value := range values {
		var raw string
		err := tx.QueryRow(`SELECT topics_json FROM notification_bot_discoveries
			WHERE bot_id=? AND destination_id=?`, botID, value.ID).Scan(&raw)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		value.Topics = mergeTopics(topicsFromJSON(raw), value.Topics)
		encoded, err := json.Marshal(value.Topics)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO notification_bot_discoveries
			(bot_id, destination_id, display_name, destination_type, photo_file_id, is_forum, topics_json, last_seen_at)
			VALUES (?,?,?,?,?,?,?,?)
			ON CONFLICT(bot_id, destination_id) DO UPDATE SET
			display_name=CASE WHEN excluded.display_name<>'' THEN excluded.display_name ELSE display_name END,
			destination_type=CASE WHEN excluded.destination_type<>'' THEN excluded.destination_type ELSE destination_type END,
			photo_file_id=CASE WHEN excluded.photo_file_id<>'' THEN excluded.photo_file_id ELSE photo_file_id END,
			is_forum=MAX(is_forum, excluded.is_forum), topics_json=excluded.topics_json,
			last_seen_at=excluded.last_seen_at`,
			botID, value.ID, value.Name, value.Type, value.PhotoFileID, boolInt(value.Forum), string(encoded), now()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) discovered(botID int64) ([]Destination, error) {
	rows, err := s.DB.Query(`SELECT destination_id, display_name, destination_type, photo_file_id,
		is_forum, topics_json FROM notification_bot_discoveries
		WHERE bot_id=? ORDER BY display_name COLLATE NOCASE, destination_id`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Destination, 0)
	for rows.Next() {
		var value Destination
		var forum int
		var raw string
		if err := rows.Scan(&value.ID, &value.Name, &value.Type, &value.PhotoFileID, &forum, &raw); err != nil {
			return nil, err
		}
		value.Forum = forum != 0
		value.Topics = topicsFromJSON(raw)
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) MergeDiscovered(botID int64, values []Destination) ([]Destination, error) {
	var provider string
	if err := s.DB.QueryRow(`SELECT provider FROM notification_bots WHERE id=?`, botID).Scan(&provider); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if provider != ProviderTelegram {
		return nil, errors.New("group discovery is available only for Telegram bots")
	}
	normalized, err := normalizeDiscovered(values)
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := mergeDiscoveredTx(tx, botID, normalized); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.discovered(botID)
}

func (s *Store) attachDestinations(config *Config) error {
	destinations, err := s.destinations(config.ID)
	if err != nil {
		return err
	}
	config.Destinations = destinations
	if len(destinations) > 0 {
		config.DestinationID = destinations[0].ID
	}
	if config.Provider == ProviderTelegram {
		discovered, err := s.discovered(config.ID)
		if err != nil {
			return err
		}
		config.DiscoveredDestinations = discovered
	}
	return nil
}

func (s *Store) Create(input CreateInput) (*Config, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Provider = normalizeProvider(input.Provider)
	input.Token = strings.TrimSpace(input.Token)
	input.DestinationID = strings.TrimSpace(input.DestinationID)
	if err := validate(input.Name, input.Provider, input.Token, true); err != nil {
		return nil, err
	}
	destinations, err := normalizeDestinations(input.Provider, input.Destinations, input.DestinationID)
	if err != nil {
		return nil, err
	}
	input.DestinationID = ""
	if len(destinations) > 0 {
		input.DestinationID = destinations[0].ID
	}
	encrypted, err := security.Encrypt(s.SecretKey, []byte(input.Token))
	if err != nil {
		return nil, err
	}
	timestamp := now()
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var totalBots, providerBots int
	if err := tx.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN provider=? THEN 1 ELSE 0 END), 0)
		FROM notification_bots`, input.Provider).Scan(&totalBots, &providerBots); err != nil {
		return nil, err
	}
	if totalBots >= 2 || providerBots > 0 {
		label := "Telegram"
		if input.Provider == ProviderDiscord {
			label = "Discord"
		}
		return nil, fmt.Errorf("only one %s bot and two notification bots total are supported", label)
	}
	result, err := tx.Exec(`INSERT INTO notification_bots
		(name, provider, token_enc, destination_id, enabled,
		 notify_server_started, notify_server_stopped, notify_player_joined, notify_player_left,
		 created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		input.Name, input.Provider, encrypted, input.DestinationID, boolInt(input.Enabled),
		boolInt(input.NotifyServerStarted), boolInt(input.NotifyServerStopped),
		boolInt(input.NotifyPlayerJoined), boolInt(input.NotifyPlayerLeft), timestamp, timestamp)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if err := insertDestinations(tx, id, destinations); err != nil {
		return nil, err
	}
	if input.Provider == ProviderTelegram {
		if _, err := tx.Exec(`INSERT INTO notification_bot_telegram_state (bot_id, last_update_id) VALUES (?, 0)`, id); err != nil {
			return nil, err
		}
		discovered, err := normalizeDiscovered(append(input.DiscoveredDestinations, destinations...))
		if err != nil {
			return nil, err
		}
		if err := mergeDiscoveredTx(tx, id, discovered); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
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
	config, err := scanConfig(s.DB.QueryRow(`SELECT `+configColumns+` FROM notification_bots WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	if err := s.attachDestinations(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (s *Store) List() ([]*Config, error) {
	rows, err := s.DB.Query(`SELECT ` + configColumns + ` FROM notification_bots ORDER BY provider, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	configs := make([]*Config, 0)
	for rows.Next() {
		config, scanErr := scanConfig(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, config := range configs {
		if err := s.attachDestinations(config); err != nil {
			return nil, err
		}
	}
	return configs, nil
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
	currentDestinations, err := s.destinations(id)
	if err != nil {
		return nil, err
	}
	originalProvider := current.Provider
	if patch.Name != nil {
		current.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Provider != nil {
		requestedProvider := normalizeProvider(*patch.Provider)
		if requestedProvider != originalProvider {
			return nil, errors.New("bot provider cannot be changed")
		}
	}
	if patch.DestinationID != nil {
		current.DestinationID = strings.TrimSpace(*patch.DestinationID)
		currentDestinations = nil
	}
	if patch.Destinations != nil {
		currentDestinations = *patch.Destinations
		current.DestinationID = ""
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
	if err := validate(current.Name, current.Provider, newToken, false); err != nil {
		return nil, err
	}
	destinations, err := normalizeDestinations(current.Provider, currentDestinations, current.DestinationID)
	if err != nil {
		return nil, err
	}
	current.DestinationID = ""
	if len(destinations) > 0 {
		current.DestinationID = destinations[0].ID
	}
	if newToken != "" {
		encrypted, err = security.Encrypt(s.SecretKey, []byte(newToken))
		if err != nil {
			return nil, err
		}
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE notification_bots SET
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
	if _, err := tx.Exec(`DELETE FROM notification_bot_destinations WHERE bot_id=?`, id); err != nil {
		return nil, err
	}
	if err := insertDestinations(tx, id, destinations); err != nil {
		return nil, err
	}
	if newToken != "" && current.Provider == ProviderTelegram {
		if _, err := tx.Exec(`DELETE FROM notification_bot_telegram_state WHERE bot_id=?`, id); err != nil {
			return nil, err
		}
	}
	if current.Provider == ProviderTelegram {
		discovered, err := normalizeDiscovered(destinations)
		if err != nil {
			return nil, err
		}
		if err := mergeDiscoveredTx(tx, id, discovered); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
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
	targets, err := s.Targets(id)
	if err != nil {
		return Target{}, err
	}
	if len(targets) == 0 {
		return Target{}, errors.New("notification bot has no destinations")
	}
	return targets[0], nil
}

// TelegramCommandBots returns Telegram credentials and durable update cursors
// for the background command listener. Tokens never leave the backend.
func (s *Store) TelegramCommandBots() ([]TelegramCommandState, error) {
	rows, err := s.DB.Query(`SELECT b.id, b.name, b.token_enc, COALESCE(state.last_update_id, 0),
		state.bot_id IS NOT NULL
		FROM notification_bots b
		LEFT JOIN notification_bot_telegram_state state ON state.bot_id=b.id
		WHERE b.provider=? ORDER BY b.id`, ProviderTelegram)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := make([]TelegramCommandState, 0, 1)
	for rows.Next() {
		var state TelegramCommandState
		var encrypted []byte
		if err := rows.Scan(&state.BotID, &state.BotName, &encrypted, &state.LastUpdateID, &state.Initialized); err != nil {
			return nil, err
		}
		token, err := security.Decrypt(s.SecretKey, encrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting Telegram bot credentials: %w", err)
		}
		state.Token = string(token)
		states = append(states, state)
	}
	return states, rows.Err()
}

func (s *Store) AdvanceTelegramUpdate(botID, updateID int64) error {
	if botID <= 0 || updateID <= 0 {
		return errors.New("invalid Telegram update cursor")
	}
	_, err := s.DB.Exec(`INSERT INTO notification_bot_telegram_state (bot_id, last_update_id)
		VALUES (?, ?) ON CONFLICT(bot_id) DO UPDATE SET
		last_update_id=MAX(last_update_id, excluded.last_update_id)`, botID, updateID)
	return err
}

func (s *Store) InitializeTelegramUpdates(botID, updateID int64) error {
	if botID <= 0 || updateID < 0 {
		return errors.New("invalid Telegram update cursor")
	}
	_, err := s.DB.Exec(`INSERT INTO notification_bot_telegram_state (bot_id, last_update_id)
		VALUES (?, ?) ON CONFLICT(bot_id) DO UPDATE SET last_update_id=excluded.last_update_id`, botID, updateID)
	return err
}

// SetTelegramDestination connects or replaces the one broadcast topic for a
// group. A Telegram bot may be connected to at most three groups.
func (s *Store) SetTelegramDestination(botID int64, destination Destination) error {
	values, err := normalizeDestinations(ProviderTelegram, []Destination{destination}, "")
	if err != nil {
		return err
	}
	destination = values[0]
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var provider string
	if err := tx.QueryRow(`SELECT provider FROM notification_bots WHERE id=?`, botID).Scan(&provider); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if provider != ProviderTelegram {
		return errors.New("destination commands are available only for Telegram bots")
	}
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM notification_bot_destinations
		WHERE bot_id=? AND destination_id<>?`, botID, destination.ID).Scan(&count); err != nil {
		return err
	}
	if count >= 3 {
		return errors.New("Telegram already has three connected groups")
	}
	var position int
	_ = tx.QueryRow(`SELECT position FROM notification_bot_destinations
		WHERE bot_id=? AND destination_id=?`, botID, destination.ID).Scan(&position)
	if position == 0 {
		_ = tx.QueryRow(`SELECT COALESCE(MAX(position), -1)+1 FROM notification_bot_destinations
			WHERE bot_id=?`, botID).Scan(&position)
	}
	if _, err := tx.Exec(`INSERT INTO notification_bot_destinations
		(bot_id, destination_id, display_name, destination_type, photo_file_id, thread_id, thread_name, position)
		VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(bot_id, destination_id) DO UPDATE SET
		display_name=excluded.display_name, destination_type=excluded.destination_type,
		photo_file_id=CASE WHEN excluded.photo_file_id<>'' THEN excluded.photo_file_id ELSE photo_file_id END,
		thread_id=excluded.thread_id, thread_name=excluded.thread_name`, botID, destination.ID,
		destination.Name, destination.Type, destination.PhotoFileID, destination.ThreadID, destination.ThreadName, position); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE notification_bots SET destination_id=(SELECT destination_id
		FROM notification_bot_destinations WHERE bot_id=? ORDER BY position LIMIT 1), updated_at=? WHERE id=?`,
		botID, now(), botID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DisconnectTelegramDestination(botID int64, destinationID string) error {
	destinationID = strings.TrimSpace(destinationID)
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM notification_bot_destinations WHERE bot_id=? AND destination_id=?`, botID, destinationID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE notification_bots SET destination_id=COALESCE((SELECT destination_id
		FROM notification_bot_destinations WHERE bot_id=? ORDER BY position LIMIT 1), ''), updated_at=? WHERE id=?`,
		botID, now(), botID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) TelegramDestination(botID int64, destinationID string) (*Destination, error) {
	var destination Destination
	err := s.DB.QueryRow(`SELECT destination_id, display_name, destination_type, photo_file_id,
		thread_id, thread_name FROM notification_bot_destinations WHERE bot_id=? AND destination_id=?`,
		botID, strings.TrimSpace(destinationID)).Scan(&destination.ID, &destination.Name, &destination.Type,
		&destination.PhotoFileID, &destination.ThreadID, &destination.ThreadName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &destination, err
}

func (s *Store) Targets(id int64) ([]Target, error) {
	var target Target
	var encrypted []byte
	err := s.DB.QueryRow(`SELECT id, name, provider, token_enc
		FROM notification_bots WHERE id=?`, id).Scan(
		&target.ID, &target.Name, &target.Provider, &encrypted)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	token, err := security.Decrypt(s.SecretKey, encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypting %s bot credentials: %w", target.Provider, err)
	}
	target.Token = string(token)
	destinations, err := s.destinations(id)
	if err != nil {
		return nil, err
	}
	targets := make([]Target, 0, len(destinations))
	for _, destination := range destinations {
		copy := target
		copy.DestinationID = destination.ID
		copy.ThreadID = destination.ThreadID
		targets = append(targets, copy)
	}
	return targets, nil
}

// TelegramPhotoTarget returns the credential and Telegram file ID needed to
// proxy one saved destination photo without exposing the bot token to the UI.
func (s *Store) TelegramPhotoTarget(id int64, destinationID string) (Target, string, error) {
	destinationID = strings.TrimSpace(destinationID)
	if destinationID == "" {
		return Target{}, "", errors.New("notification destination is required")
	}
	var target Target
	var encrypted []byte
	var photoFileID string
	err := s.DB.QueryRow(`SELECT b.id, b.name, b.provider, b.token_enc,
		COALESCE(d.destination_id, discovered.destination_id),
		COALESCE(NULLIF(d.photo_file_id, ''), discovered.photo_file_id, '')
		FROM notification_bots b
		LEFT JOIN notification_bot_destinations d
			ON d.bot_id=b.id AND d.destination_id=?
		LEFT JOIN notification_bot_discoveries discovered
			ON discovered.bot_id=b.id AND discovered.destination_id=?
		WHERE b.id=? AND b.provider=?
			AND (d.destination_id IS NOT NULL OR discovered.destination_id IS NOT NULL)`,
		destinationID, destinationID, id, ProviderTelegram).Scan(
		&target.ID, &target.Name, &target.Provider, &encrypted,
		&target.DestinationID, &photoFileID)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, "", ErrNotFound
	}
	if err != nil {
		return Target{}, "", err
	}
	photoFileID = strings.TrimSpace(photoFileID)
	token, err := security.Decrypt(s.SecretKey, encrypted)
	if err != nil {
		return Target{}, "", fmt.Errorf("decrypting Telegram bot credentials: %w", err)
	}
	target.Token = string(token)
	return target, photoFileID, nil
}

func (s *Store) TargetsFor(event string) ([]Target, error) {
	var query string
	switch event {
	case EventServerStarted:
		query = `SELECT b.id, b.name, b.provider, b.token_enc, d.destination_id, d.thread_id
			FROM notification_bots b JOIN notification_bot_destinations d ON d.bot_id=b.id
			WHERE b.enabled=1 AND b.notify_server_started=1 ORDER BY b.id, d.position`
	case EventServerStopped:
		query = `SELECT b.id, b.name, b.provider, b.token_enc, d.destination_id, d.thread_id
			FROM notification_bots b JOIN notification_bot_destinations d ON d.bot_id=b.id
			WHERE b.enabled=1 AND b.notify_server_stopped=1 ORDER BY b.id, d.position`
	case EventPlayerJoined:
		query = `SELECT b.id, b.name, b.provider, b.token_enc, d.destination_id, d.thread_id
			FROM notification_bots b JOIN notification_bot_destinations d ON d.bot_id=b.id
			WHERE b.enabled=1 AND b.notify_player_joined=1 ORDER BY b.id, d.position`
	case EventPlayerLeft:
		query = `SELECT b.id, b.name, b.provider, b.token_enc, d.destination_id, d.thread_id
			FROM notification_bots b JOIN notification_bot_destinations d ON d.bot_id=b.id
			WHERE b.enabled=1 AND b.notify_player_left=1 ORDER BY b.id, d.position`
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
		if err := rows.Scan(&target.ID, &target.Name, &target.Provider, &encrypted, &target.DestinationID, &target.ThreadID); err != nil {
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
