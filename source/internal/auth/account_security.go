package auth

import (
	"database/sql"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/security"
)

// RecoveryCode is safe account-facing metadata. Recovery-code hashes and
// plaintext values are deliberately never returned by the listing API.
type RecoveryCode struct {
	ID        int64      `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

func (s *Store) ListRecoveryCodes(userID int64) ([]RecoveryCode, error) {
	rows, err := s.DB.Query(`SELECT id, created_at, used_at
		FROM recovery_codes WHERE user_id=? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RecoveryCode, 0)
	for rows.Next() {
		var item RecoveryCode
		var created string
		var used sql.NullString
		if err := rows.Scan(&item.ID, &created, &used); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if used.Valid {
			parsed, err := time.Parse(time.RFC3339, used.String)
			if err == nil {
				item.UsedAt = &parsed
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ReplaceRecoveryCodes revokes every existing code and returns a fresh set.
// Plaintext values exist only in this return value and are never persisted.
func (s *Store) ReplaceRecoveryCodes(userID int64) ([]string, error) {
	codes, err := GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := replaceRecoveryCodes(tx, userID, codes); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return codes, nil
}

func replaceRecoveryCodes(tx *sql.Tx, userID int64, codes []string) error {
	if _, err := tx.Exec(`DELETE FROM recovery_codes WHERE user_id=?`, userID); err != nil {
		return err
	}
	created := now()
	for _, code := range codes {
		if _, err := tx.Exec(`INSERT INTO recovery_codes (user_id, code_hash, created_at)
			VALUES (?,?,?)`, userID, HashToken(code), created); err != nil {
			return err
		}
	}
	return nil
}

// ChangePassword updates the password and revokes every session except the
// browser session that completed the fresh identity check.
func (s *Store) ChangePassword(userID int64, currentSessionToken, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`, hash, now(), userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	if _, err := tx.Exec(`UPDATE sessions SET revoked=1 WHERE user_id=? AND id<>?`,
		userID, HashToken(currentSessionToken)); err != nil {
		return err
	}
	return tx.Commit()
}

// ChangeTOTP atomically confirms the replacement secret, rotates all recovery
// codes, and revokes other sessions. The previous secret remains valid until
// this transaction commits.
func (s *Store) ChangeTOTP(userID int64, currentSessionToken, secret string) ([]string, error) {
	enc, err := security.Encrypt(s.SecretKey, []byte(secret))
	if err != nil {
		return nil, err
	}
	codes, err := GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE users SET totp_secret_enc=?, updated_at=? WHERE id=?`, enc, now(), userID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, sql.ErrNoRows
	}
	if err := replaceRecoveryCodes(tx, userID, codes); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE sessions SET revoked=1 WHERE user_id=? AND id<>?`,
		userID, HashToken(currentSessionToken)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return codes, nil
}
