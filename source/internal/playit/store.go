// Package playit implements Bonghos's optional Playit.gg integration.
// Settings are global to one Bonghos installation because the installation
// runs at most one active Minecraft project at a time.
package playit

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/security"
)

const (
	AccountModeAccount = "account"
	AccountModeGuest   = "guest"

	ManagementNone     = "none"
	ManagementExternal = "external"
	ManagementBonghos  = "bonghos"
)

// Config is safe to return to the browser. Agent credentials are represented
// only by SecretConfigured and are never serialized or logged.
type Config struct {
	Enabled          bool   `json:"enabled"`
	AccountMode      string `json:"account_mode"`
	ManagementMode   string `json:"management_mode"`
	SecretConfigured bool   `json:"secret_configured"`
	AgentID          string `json:"agent_id,omitempty"`
	TunnelID         string `json:"tunnel_id,omitempty"`
	PublicAddress    string `json:"public_address,omitempty"`
	LocalPort        int    `json:"local_port"`
	ClaimPending     bool   `json:"claim_pending"`
	ClaimURL         string `json:"claim_url,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type storedConfig struct {
	Config
	secret         []byte
	claimCode      string
	claimStartedAt string
}

type Store struct {
	DB        *sql.DB
	SecretKey []byte
}

func validAccountMode(mode string) bool {
	return mode == AccountModeAccount || mode == AccountModeGuest
}

func (s *Store) load(decryptSecret bool) (storedConfig, error) {
	var out storedConfig
	var enabled int
	var encrypted []byte
	err := s.DB.QueryRow(`SELECT enabled, account_mode, management_mode, agent_secret_enc,
		agent_id, tunnel_id, public_address, local_port, claim_code, claim_started_at, updated_at
		FROM playit_settings WHERE id=1`).Scan(&enabled, &out.AccountMode, &out.ManagementMode,
		&encrypted, &out.AgentID, &out.TunnelID, &out.PublicAddress, &out.LocalPort,
		&out.claimCode, &out.claimStartedAt, &out.UpdatedAt)
	if err != nil {
		return storedConfig{}, err
	}
	out.Enabled = enabled != 0
	out.SecretConfigured = len(encrypted) > 0
	if decryptSecret && len(encrypted) > 0 {
		plain, err := security.Decrypt(s.SecretKey, encrypted)
		if err != nil {
			return storedConfig{}, fmt.Errorf("decrypting Playit agent secret: %w", err)
		}
		out.secret = plain
	}
	if out.claimCode != "" {
		started, err := time.Parse(time.RFC3339, out.claimStartedAt)
		out.ClaimPending = err == nil && time.Since(started) < 20*time.Minute
		if out.ClaimPending {
			out.ClaimURL = "https://playit.gg/claim/" + out.claimCode
		}
	}
	return out, nil
}

func (s *Store) Config() (Config, error) {
	stored, err := s.load(false)
	return stored.Config, err
}

// SetPreference records explicit intent without provisioning credentials or
// contacting Playit. Disabling preserves credentials so a temporary opt-out
// does not destroy the user's public address.
func (s *Store) SetPreference(enabled bool, accountMode, managementMode string, updatedBy int64) (Config, error) {
	accountMode = strings.ToLower(strings.TrimSpace(accountMode))
	managementMode = strings.ToLower(strings.TrimSpace(managementMode))
	if !validAccountMode(accountMode) {
		return Config{}, errors.New("Playit account mode must be account or guest")
	}
	if managementMode != ManagementNone && managementMode != ManagementExternal && managementMode != ManagementBonghos {
		return Config{}, errors.New("invalid Playit management mode")
	}
	if enabled && managementMode == ManagementNone {
		return Config{}, errors.New("choose Bonghos-managed or external Playit")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`UPDATE playit_settings SET enabled=?, account_mode=?, management_mode=?,
		claim_code=CASE WHEN ?=0 THEN '' ELSE claim_code END,
		claim_started_at=CASE WHEN ?=0 THEN '' ELSE claim_started_at END,
		updated_at=?, updated_by=? WHERE id=1`, boolInt(enabled), accountMode, managementMode,
		boolInt(enabled), boolInt(enabled), now, nullableID(updatedBy))
	if err != nil {
		return Config{}, err
	}
	return s.Config()
}

func (s *Store) BeginClaim(accountMode, code string, updatedBy int64) (Config, error) {
	accountMode = strings.ToLower(strings.TrimSpace(accountMode))
	code = strings.ToLower(strings.TrimSpace(code))
	if !validAccountMode(accountMode) {
		return Config{}, errors.New("Playit account mode must be account or guest")
	}
	if len(code) != 10 {
		return Config{}, errors.New("invalid Playit claim code")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`UPDATE playit_settings SET enabled=1, account_mode=?, management_mode='bonghos',
		claim_code=?, claim_started_at=?, updated_at=?, updated_by=? WHERE id=1`, accountMode,
		code, now, now, nullableID(updatedBy))
	if err != nil {
		return Config{}, err
	}
	return s.Config()
}

func (s *Store) Claim() (code, accountMode string, err error) {
	stored, err := s.load(false)
	if err != nil {
		return "", "", err
	}
	if !stored.ClaimPending || stored.claimCode == "" {
		return "", "", errors.New("no active Playit claim")
	}
	return stored.claimCode, stored.AccountMode, nil
}

func (s *Store) CancelClaim() error {
	_, err := s.DB.Exec(`UPDATE playit_settings SET claim_code='', claim_started_at=?, updated_at=? WHERE id=1`,
		"", time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) CompleteClaim(secret string, updatedBy int64) (Config, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" || len(secret) > 4096 {
		return Config{}, errors.New("Playit returned an invalid agent secret")
	}
	encrypted, err := security.Encrypt(s.SecretKey, []byte(secret))
	if err != nil {
		return Config{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.Exec(`UPDATE playit_settings SET enabled=1, management_mode='bonghos',
		agent_secret_enc=?, agent_id='', tunnel_id='', public_address='',
		claim_code='', claim_started_at='', updated_at=?, updated_by=? WHERE id=1`,
		encrypted, now, nullableID(updatedBy))
	if err != nil {
		return Config{}, err
	}
	return s.Config()
}

func (s *Store) Secret() (string, error) {
	stored, err := s.load(true)
	if err != nil {
		return "", err
	}
	if len(stored.secret) == 0 {
		return "", errors.New("Playit agent is not linked")
	}
	return string(stored.secret), nil
}

func (s *Store) SaveAgent(agentID string) error {
	agentID = strings.TrimSpace(agentID)
	_, err := s.DB.Exec(`UPDATE playit_settings SET agent_id=?, updated_at=? WHERE id=1`,
		agentID, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) SaveTunnel(tunnelID, publicAddress string, localPort int) error {
	if localPort < 1 || localPort > 65535 {
		return errors.New("Minecraft port must be between 1 and 65535")
	}
	_, err := s.DB.Exec(`UPDATE playit_settings SET tunnel_id=?, public_address=?, local_port=?, updated_at=? WHERE id=1`,
		strings.TrimSpace(tunnelID), strings.TrimSpace(publicAddress), localPort,
		time.Now().UTC().Format(time.RFC3339))
	return err
}

// ClearTunnel forgets only Bonghos's association with a Playit tunnel. Callers
// must first delete the remote tunnel, or establish that it no longer exists.
func (s *Store) ClearTunnel() error {
	_, err := s.DB.Exec(`UPDATE playit_settings SET tunnel_id='', public_address='', updated_at=? WHERE id=1`,
		time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) SaveExternalAddress(publicAddress string, localPort int, updatedBy int64) (Config, error) {
	publicAddress = strings.TrimSpace(publicAddress)
	if publicAddress == "" || len(publicAddress) > 255 || strings.ContainsAny(publicAddress, "\r\n\t /\\") {
		return Config{}, errors.New("enter a valid Playit public host and optional port")
	}
	if localPort < 1 || localPort > 65535 {
		return Config{}, errors.New("Minecraft port must be between 1 and 65535")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`UPDATE playit_settings SET enabled=1, management_mode='external',
		public_address=?, local_port=?, updated_at=?, updated_by=? WHERE id=1`,
		publicAddress, localPort, now, nullableID(updatedBy))
	if err != nil {
		return Config{}, err
	}
	return s.Config()
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
