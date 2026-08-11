package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Passkey is the safe, user-facing metadata for a registered WebAuthn
// credential. Credential public keys and authenticator counters never leave
// the server.
type Passkey struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	RPID           string     `json:"rp_id"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	BackupEligible bool       `json:"backup_eligible"`
	BackedUp       bool       `json:"backed_up"`
}

var ErrPasskeyNotFound = errors.New("passkey not found")

// PasskeyUser adapts a Bonghos account to the WebAuthn user interface.
type PasskeyUser struct {
	Account     *User
	Handle      []byte
	Credentials []webauthn.Credential
}

func (u *PasskeyUser) WebAuthnID() []byte                         { return u.Handle }
func (u *PasskeyUser) WebAuthnName() string                       { return u.Account.Username }
func (u *PasskeyUser) WebAuthnDisplayName() string                { return u.Account.Username }
func (u *PasskeyUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
func (u *PasskeyUser) WebAuthnIcon() string                       { return "" }

// PasskeyUserHandle returns a stable opaque identifier. It deliberately does
// not expose the numeric database user ID to authenticators or other devices.
func (s *Store) PasskeyUserHandle(userID int64) []byte {
	mac := hmac.New(sha256.New, s.SecretKey)
	_, _ = mac.Write([]byte("bonghos-passkey-user:" + strconv.FormatInt(userID, 10)))
	return mac.Sum(nil)
}

func (s *Store) PasskeyUser(userID int64, rpID string) (*PasskeyUser, error) {
	u, err := s.UserByID(userID)
	if err != nil {
		return nil, err
	}
	if u.Disabled {
		return nil, ErrInvalidCredentials
	}
	rows, err := s.DB.Query(`SELECT credential_json FROM passkeys WHERE user_id=? AND rp_id=? ORDER BY id`, userID, rpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	credentials := make([]webauthn.Credential, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var credential webauthn.Credential
		if err := json.Unmarshal(raw, &credential); err != nil {
			return nil, fmt.Errorf("decode passkey credential: %w", err)
		}
		credentials = append(credentials, credential)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &PasskeyUser{Account: u, Handle: s.PasskeyUserHandle(userID), Credentials: credentials}, nil
}

// DiscoverPasskeyUser resolves a discoverable credential without accepting a
// username from the browser. Both the opaque user handle and credential ID
// must belong to the same enabled account and relying party.
func (s *Store) DiscoverPasskeyUser(userHandle, credentialID []byte, rpID string) (*PasskeyUser, error) {
	var userID int64
	if err := s.DB.QueryRow(`SELECT user_id FROM passkeys WHERE credential_id=? AND rp_id=?`, credentialID, rpID).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !hmac.Equal(userHandle, s.PasskeyUserHandle(userID)) {
		return nil, ErrInvalidCredentials
	}
	return s.PasskeyUser(userID, rpID)
}

func (s *Store) ListPasskeys(userID int64) ([]Passkey, error) {
	rows, err := s.DB.Query(`SELECT id, name, rp_id, credential_json, created_at, last_used_at
		FROM passkeys WHERE user_id=? ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Passkey, 0)
	for rows.Next() {
		var item Passkey
		var raw []byte
		var created string
		var lastUsed sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &item.RPID, &raw, &created, &lastUsed); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if lastUsed.Valid {
			parsed, err := time.Parse(time.RFC3339, lastUsed.String)
			if err == nil {
				item.LastUsedAt = &parsed
			}
		}
		var credential webauthn.Credential
		if json.Unmarshal(raw, &credential) == nil {
			item.BackupEligible = credential.Flags.BackupEligible
			item.BackedUp = credential.Flags.BackupState
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) AddPasskey(userID int64, rpID, name string, credential *webauthn.Credential) (*Passkey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Passkey"
	}
	if len(name) > 80 {
		return nil, errors.New("passkey name must be 80 characters or fewer")
	}
	if credential == nil || len(credential.ID) == 0 {
		return nil, errors.New("invalid passkey credential")
	}
	raw, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}
	created := now()
	res, err := s.DB.Exec(`INSERT INTO passkeys
		(user_id, credential_id, credential_json, rp_id, name, created_at)
		VALUES (?,?,?,?,?,?)`, userID, credential.ID, raw, rpID, name, created)
	if err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			return nil, errors.New("this passkey is already registered")
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	createdAt, _ := time.Parse(time.RFC3339, created)
	return &Passkey{
		ID: id, Name: name, RPID: rpID, CreatedAt: createdAt,
		BackupEligible: credential.Flags.BackupEligible,
		BackedUp:       credential.Flags.BackupState,
	}, nil
}

func (s *Store) UpdatePasskeyCredential(userID int64, rpID string, credential *webauthn.Credential) error {
	if credential == nil || len(credential.ID) == 0 {
		return errors.New("invalid passkey credential")
	}
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	res, err := s.DB.Exec(`UPDATE passkeys SET credential_json=?, last_used_at=?
		WHERE user_id=? AND rp_id=? AND credential_id=?`, raw, now(), userID, rpID, credential.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrInvalidCredentials
	}
	return nil
}

// RenamePasskey changes only the user-facing label. The credential, relying
// party binding and authenticator state remain untouched.
func (s *Store) RenamePasskey(userID, passkeyID int64, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("passkey name is required")
	}
	if len(name) > 80 {
		return "", errors.New("passkey name must be 80 characters or fewer")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var previousName string
	if err := tx.QueryRow(`SELECT name FROM passkeys WHERE id=? AND user_id=?`, passkeyID, userID).Scan(&previousName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPasskeyNotFound
		}
		return "", err
	}
	res, err := tx.Exec(`UPDATE passkeys SET name=? WHERE id=? AND user_id=?`, name, passkeyID, userID)
	if err != nil {
		return "", err
	}
	rows, _ := res.RowsAffected()
	if rows != 1 {
		return "", ErrPasskeyNotFound
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return previousName, nil
}

func (s *Store) DeletePasskey(userID, passkeyID int64) error {
	res, err := s.DB.Exec(`DELETE FROM passkeys WHERE id=? AND user_id=?`, passkeyID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrPasskeyNotFound
	}
	return nil
}
