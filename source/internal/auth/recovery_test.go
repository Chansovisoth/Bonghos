package auth_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/database"
)

func TestRecoveryCodeLoginIsSingleUseAndRequiresCorrectPassword(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "bonghos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := &auth.Store{
		DB: db, SecretKey: []byte("01234567890123456789012345678901"), Sessions: time.Hour,
	}
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	user, recovery, err := store.CreateUser("owner", "correct horse battery", secret, authorization.RoleOwner)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Login("owner", "wrong password", recovery[0], "wrong-password"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("wrong-password login error = %v", err)
	}
	if _, err := store.Login("owner", "correct horse battery", recovery[0], "correct-password"); err != nil {
		t.Fatalf("recovery code was consumed by the wrong-password attempt: %v", err)
	}
	if _, err := store.Login("owner", "correct horse battery", recovery[0], "reused-code"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("reused recovery code error = %v", err)
	}

	// Numeric-only codes are valid hexadecimal recovery codes. The Web UI must
	// not mistake this shape for a six-digit TOTP value.
	const numericRecovery = "12345-67890"
	if _, err := db.Exec(`INSERT INTO recovery_codes (user_id, code_hash) VALUES (?,?)`,
		user.ID, auth.HashToken(numericRecovery)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Login("owner", "correct horse battery", numericRecovery, "numeric-code"); err != nil {
		t.Fatalf("numeric-only recovery code rejected: %v", err)
	}
}
