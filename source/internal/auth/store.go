package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/security"
)

var (
	ErrInvalidCredentials = errors.New("the username, password, or authentication code is incorrect")
	ErrRateLimited        = errors.New("too many attempts; try again shortly")
	ErrLastOwner          = errors.New("the final Owner cannot be deleted, demoted or disabled")
	ErrUsernameTaken      = errors.New("username is not available")
)

type User struct {
	ID        int64
	Username  string
	Role      authorization.Role
	Disabled  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Store struct {
	DB        *sql.DB
	SecretKey []byte
	Sessions  time.Duration
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func NormalizeUsername(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

func ValidateUsername(u string) error {
	u = strings.TrimSpace(u)
	if len(u) < 3 || len(u) > 32 {
		return errors.New("username must be 3-32 characters")
	}
	for _, r := range u {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return errors.New("username may contain letters, numbers, '_', '-', '.'")
		}
	}
	return nil
}

func ValidatePassword(p string) error {
	if len(p) < 10 {
		return errors.New("password must be at least 10 characters")
	}
	if len(p) > 256 {
		return errors.New("password too long")
	}
	return nil
}

// CreateUser creates an account with an encrypted TOTP secret and hashed
// recovery codes, returning the user and plaintext recovery codes.
func (s *Store) CreateUser(username, password, totpSecret string, role authorization.Role) (*User, []string, error) {
	if err := ValidateUsername(username); err != nil {
		return nil, nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, nil, err
	}
	if !authorization.ValidRole(role) {
		return nil, nil, errors.New("invalid role")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, nil, err
	}
	enc, err := security.Encrypt(s.SecretKey, []byte(totpSecret))
	if err != nil {
		return nil, nil, err
	}
	codes, err := GenerateRecoveryCodes(8)
	if err != nil {
		return nil, nil, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO users (username, password_hash, totp_secret_enc, role, created_at, updated_at)
		VALUES (?,?,?,?,?,?)`, username, hash, enc, string(role), now(), now())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, nil, ErrUsernameTaken
		}
		return nil, nil, err
	}
	id, _ := res.LastInsertId()
	created := now()
	for _, c := range codes {
		if _, err := tx.Exec(`INSERT INTO recovery_codes (user_id, code_hash, created_at) VALUES (?,?,?)`,
			id, HashToken(c), created); err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return &User{ID: id, Username: username, Role: role}, codes, nil
}

func scanUser(row *sql.Row) (*User, string, []byte, error) {
	var u User
	var role, hash, created, updated string
	var enc []byte
	var disabled int
	err := row.Scan(&u.ID, &u.Username, &hash, &enc, &role, &disabled, &created, &updated)
	if err != nil {
		return nil, "", nil, err
	}
	u.Role = authorization.Role(role)
	u.Disabled = disabled != 0
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &u, hash, enc, nil
}

const userCols = `id, username, password_hash, totp_secret_enc, role, disabled, created_at, updated_at`

func (s *Store) UserByID(id int64) (*User, error) {
	u, _, _, err := scanUser(s.DB.QueryRow(`SELECT `+userCols+` FROM users WHERE id = ?`, id))
	return u, err
}

func (s *Store) UserByName(username string) (*User, error) {
	u, _, _, err := scanUser(s.DB.QueryRow(`SELECT `+userCols+` FROM users WHERE username = ?`, username))
	return u, err
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.DB.Query(`SELECT id, username, role, disabled, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var role, created string
		var disabled int
		if err := rows.Scan(&u.ID, &u.Username, &role, &disabled, &created); err != nil {
			return nil, err
		}
		u.Role = authorization.Role(role)
		u.Disabled = disabled != 0
		u.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) countActiveOwners() (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='owner' AND disabled=0`).Scan(&n)
	return n, err
}

// guardLastOwner returns ErrLastOwner if the change would leave no active Owner.
func (s *Store) guardLastOwner(targetID int64) error {
	var role string
	var disabled int
	if err := s.DB.QueryRow(`SELECT role, disabled FROM users WHERE id=?`, targetID).Scan(&role, &disabled); err != nil {
		return err
	}
	if role != "owner" || disabled != 0 {
		return nil
	}
	n, err := s.countActiveOwners()
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastOwner
	}
	return nil
}

func (s *Store) SetRole(targetID int64, role authorization.Role) error {
	if !authorization.ValidRole(role) {
		return errors.New("invalid role")
	}
	if role != authorization.RoleOwner {
		if err := s.guardLastOwner(targetID); err != nil {
			return err
		}
	}
	_, err := s.DB.Exec(`UPDATE users SET role=?, updated_at=? WHERE id=?`, string(role), now(), targetID)
	return err
}

func (s *Store) SetDisabled(targetID int64, disabled bool) error {
	if disabled {
		if err := s.guardLastOwner(targetID); err != nil {
			return err
		}
	}
	if _, err := s.DB.Exec(`UPDATE users SET disabled=?, updated_at=? WHERE id=?`,
		boolInt(disabled), now(), targetID); err != nil {
		return err
	}
	if disabled {
		return s.RevokeAllSessions(targetID)
	}
	return nil
}

func (s *Store) DeleteUser(targetID int64) error {
	if err := s.guardLastOwner(targetID); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM users WHERE id=?`, targetID)
	return err
}

// ----- rate limiting ---------------------------------------------------------

const (
	rateWindow   = 10 * time.Minute
	rateMaxFails = 8
)

func (s *Store) recordAttempt(identifier string, success bool) {
	s.DB.Exec(`INSERT INTO login_attempts (identifier, attempted_at, success) VALUES (?,?,?)`,
		identifier, now(), boolInt(success))
	s.DB.Exec(`DELETE FROM login_attempts WHERE attempted_at < ?`,
		time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339))
}

func (s *Store) rateLimited(identifiers ...string) bool {
	since := time.Now().UTC().Add(-rateWindow).Format(time.RFC3339)
	for _, id := range identifiers {
		var fails int
		s.DB.QueryRow(`SELECT COUNT(*) FROM login_attempts
			WHERE identifier=? AND success=0 AND attempted_at > ?`, id, since).Scan(&fails)
		if fails >= rateMaxFails {
			return true
		}
	}
	return false
}

// ----- login -----------------------------------------------------------------

// Login performs the complete two-step verification in one backend call:
// username + password + TOTP (or recovery code). Work is equalized whether or
// not the account exists, and one generic error is returned for any failure.
func (s *Store) Login(username, password, code, remoteAddr string) (*User, error) {
	norm := NormalizeUsername(username)
	if s.rateLimited("u:"+norm, "ip:"+remoteAddr) {
		return nil, ErrRateLimited
	}

	fail := func() (*User, error) {
		s.recordAttempt("u:"+norm, false)
		s.recordAttempt("ip:"+remoteAddr, false)
		return nil, ErrInvalidCredentials
	}

	row := s.DB.QueryRow(`SELECT `+userCols+` FROM users WHERE username = ?`, username)
	u, hash, enc, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		// equalize work for nonexistent accounts
		DummyVerify(password)
		DummyTOTPWork(code)
		return fail()
	}
	if err != nil {
		return nil, err
	}
	passOK := VerifyPassword(password, hash)

	secret, decErr := security.Decrypt(s.SecretKey, enc)
	codeOK := false
	recoveryOK := false
	if decErr == nil {
		codeOK = VerifyTOTP(string(secret), code, time.Now())
	} else {
		DummyTOTPWork(code)
	}
	if !codeOK {
		// Verify a recovery code without spending it. It is consumed atomically
		// only after the password and account state also pass, so a password typo
		// cannot destroy a valid one-time code.
		recoveryOK = s.recoveryCodeAvailable(u.ID, code)
		codeOK = recoveryOK
	}
	if !passOK || !codeOK || u.Disabled {
		return fail()
	}
	if recoveryOK && !s.consumeRecoveryCode(u.ID, code) {
		// Another concurrent login may have consumed the same code after our
		// availability check. Only the request that updates the row succeeds.
		return fail()
	}
	s.recordAttempt("u:"+norm, true)
	return u, nil
}

func (s *Store) recoveryCodeAvailable(userID int64, code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) < 8 {
		return false
	}
	var available int
	err := s.DB.QueryRow(`SELECT EXISTS(
		SELECT 1 FROM recovery_codes
		WHERE user_id=? AND code_hash=? AND used_at IS NULL
	)`, userID, HashToken(code)).Scan(&available)
	return err == nil && available == 1
}

func (s *Store) consumeRecoveryCode(userID int64, code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) < 8 {
		return false
	}
	res, err := s.DB.Exec(`UPDATE recovery_codes SET used_at=?
		WHERE user_id=? AND code_hash=? AND used_at IS NULL`, now(), userID, HashToken(code))
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n == 1
}

// TOTPSecretFor decrypts a user's TOTP secret (setup/QR display during
// activation only; never exposed after enrollment completes).
func (s *Store) TOTPSecretFor(userID int64) (string, error) {
	var enc []byte
	if err := s.DB.QueryRow(`SELECT totp_secret_enc FROM users WHERE id=?`, userID).Scan(&enc); err != nil {
		return "", err
	}
	pt, err := security.Decrypt(s.SecretKey, enc)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// ----- sessions --------------------------------------------------------------

type Session struct {
	Token     string // plaintext, only at creation
	UserID    int64
	ExpiresAt time.Time
}

func (s *Store) CreateSession(userID int64, remoteAddr, userAgent string) (*Session, error) {
	tok, hash, err := NewToken()
	if err != nil {
		return nil, err
	}
	exp := time.Now().UTC().Add(s.Sessions)
	_, err = s.DB.Exec(`INSERT INTO sessions (id, user_id, created_at, expires_at, last_seen_at, remote_addr, user_agent)
		VALUES (?,?,?,?,?,?,?)`, hash, userID, now(), exp.Format(time.RFC3339), now(), remoteAddr, userAgent)
	if err != nil {
		return nil, err
	}
	return &Session{Token: tok, UserID: userID, ExpiresAt: exp}, nil
}

// ValidateSession resolves a cookie token to an active, non-disabled user.
func (s *Store) ValidateSession(token string) (*User, error) {
	if token == "" {
		return nil, ErrInvalidCredentials
	}
	hash := HashToken(token)
	var userID int64
	var expires string
	var revoked int
	err := s.DB.QueryRow(`SELECT user_id, expires_at, revoked FROM sessions WHERE id=?`, hash).
		Scan(&userID, &expires, &revoked)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	exp, _ := time.Parse(time.RFC3339, expires)
	if revoked != 0 || time.Now().UTC().After(exp) {
		return nil, ErrInvalidCredentials
	}
	u, err := s.UserByID(userID)
	if err != nil || u.Disabled {
		return nil, ErrInvalidCredentials
	}
	s.DB.Exec(`UPDATE sessions SET last_seen_at=? WHERE id=?`, now(), hash)
	return u, nil
}

func (s *Store) RevokeSession(token string) error {
	_, err := s.DB.Exec(`UPDATE sessions SET revoked=1 WHERE id=?`, HashToken(token))
	return err
}

func (s *Store) RevokeAllSessions(userID int64) error {
	_, err := s.DB.Exec(`UPDATE sessions SET revoked=1 WHERE user_id=?`, userID)
	return err
}

// ----- invitations -----------------------------------------------------------

type Invitation struct {
	ID        int64
	Token     string // plaintext at creation only
	Role      authorization.Role
	ExpiresAt time.Time
}

func (s *Store) CreateInvitation(createdBy int64, role authorization.Role, ttl time.Duration) (*Invitation, error) {
	if role == authorization.RoleOwner {
		return nil, errors.New("invitations cannot grant the Owner role")
	}
	if !authorization.ValidRole(role) {
		return nil, errors.New("invalid role")
	}
	tok, hash, err := NewToken()
	if err != nil {
		return nil, err
	}
	exp := time.Now().UTC().Add(ttl)
	res, err := s.DB.Exec(`INSERT INTO invitations (token_hash, role, created_by, created_at, expires_at)
		VALUES (?,?,?,?,?)`, hash, string(role), createdBy, now(), exp.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Invitation{ID: id, Token: tok, Role: role, ExpiresAt: exp}, nil
}

// CheckInvitation validates an unused, unexpired, unrevoked token.
func (s *Store) CheckInvitation(token string) (authorization.Role, int64, error) {
	var id int64
	var role, expires string
	var usedBy sql.NullInt64
	var revoked int
	err := s.DB.QueryRow(`SELECT id, role, expires_at, used_by, revoked FROM invitations WHERE token_hash=?`,
		HashToken(token)).Scan(&id, &role, &expires, &usedBy, &revoked)
	if err != nil {
		return "", 0, errors.New("invitation is invalid")
	}
	exp, _ := time.Parse(time.RFC3339, expires)
	if revoked != 0 || usedBy.Valid || time.Now().UTC().After(exp) {
		return "", 0, errors.New("invitation is invalid or expired")
	}
	return authorization.Role(role), id, nil
}

// ActivateInvitation atomically consumes an invitation and creates the user.
func (s *Store) ActivateInvitation(token, username, password, totpSecret, totpCode string) (*User, []string, error) {
	role, invID, err := s.CheckInvitation(token)
	if err != nil {
		return nil, nil, err
	}
	if !VerifyTOTP(totpSecret, totpCode, time.Now()) {
		return nil, nil, errors.New("authentication code did not match; scan the QR again and retry")
	}
	u, codes, err := s.CreateUser(username, password, totpSecret, role)
	if err != nil {
		return nil, nil, err
	}
	res, err := s.DB.Exec(`UPDATE invitations SET used_by=?, used_at=? WHERE id=? AND used_by IS NULL AND revoked=0`,
		u.ID, now(), invID)
	if err != nil {
		return nil, nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		s.DB.Exec(`DELETE FROM users WHERE id=?`, u.ID)
		return nil, nil, errors.New("invitation was already used")
	}
	return u, codes, nil
}

func (s *Store) RevokeInvitation(id int64) error {
	_, err := s.DB.Exec(`UPDATE invitations SET revoked=1 WHERE id=?`, id)
	return err
}

// ----- audit -----------------------------------------------------------------

func (s *Store) Audit(userID int64, username, action, target, detail, remoteAddr string) {
	s.DB.Exec(`INSERT INTO audit_log (occurred_at, user_id, username, action, target, detail, remote_addr)
		VALUES (?,?,?,?,?,?,?)`, now(), userID, username, action, target, detail, remoteAddr)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

var _ = fmt.Sprintf

// ResetPassword replaces a user's password hash (CLI recovery path).
func (s *Store) ResetPassword(userID int64, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`,
		hash, now(), userID)
	return err
}
