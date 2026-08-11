package auth_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/go-webauthn/webauthn/webauthn"
)

func passkeyStore(t *testing.T) (*auth.Store, int64) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "bonghos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	created := time.Now().UTC().Format(time.RFC3339)
	result, err := db.Exec(`INSERT INTO users
		(username, password_hash, totp_secret_enc, role, created_at, updated_at)
		VALUES (?,?,?,?,?,?)`, "owner", "unused", []byte("unused"), "owner", created, created)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := result.LastInsertId()
	return &auth.Store{
		DB: db, SecretKey: []byte("01234567890123456789012345678901"), Sessions: time.Hour,
	}, userID
}

func TestPasskeyLifecycleAndDiscoverableLookup(t *testing.T) {
	store, userID := passkeyStore(t)
	credential := &webauthn.Credential{
		ID: []byte("credential-one"), PublicKey: []byte("public-key"),
		Flags: webauthn.CredentialFlags{UserPresent: true, UserVerified: true, BackupEligible: true},
	}
	created, err := store.AddPasskey(userID, "panel.example.test", "Laptop", credential)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Laptop" || !created.BackupEligible {
		t.Fatalf("created passkey = %+v", created)
	}

	items, err := store.ListPasskeys(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "Laptop" || items[0].LastUsedAt != nil {
		t.Fatalf("listed passkeys = %+v", items)
	}
	previousName, err := store.RenamePasskey(userID, created.ID, "  Office security key  ")
	if err != nil {
		t.Fatal(err)
	}
	if previousName != "Laptop" {
		t.Fatalf("previous passkey name = %q", previousName)
	}
	items, err = store.ListPasskeys(userID)
	if err != nil || len(items) != 1 || items[0].Name != "Office security key" {
		t.Fatalf("renamed passkeys = %+v, %v", items, err)
	}

	user, err := store.DiscoverPasskeyUser(store.PasskeyUserHandle(userID), credential.ID, "panel.example.test")
	if err != nil {
		t.Fatal(err)
	}
	if user.Account.ID != userID || len(user.Credentials) != 1 {
		t.Fatalf("discovered user = %+v", user)
	}
	if _, err := store.DiscoverPasskeyUser([]byte("wrong-handle"), credential.ID, "panel.example.test"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("wrong handle error = %v", err)
	}
	if _, err := store.DiscoverPasskeyUser(store.PasskeyUserHandle(userID), credential.ID, "other.example.test"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("wrong RP error = %v", err)
	}

	credential.Flags.BackupState = true
	credential.Authenticator.SignCount = 2
	if err := store.UpdatePasskeyCredential(userID, "panel.example.test", credential); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListPasskeys(userID)
	if err != nil {
		t.Fatal(err)
	}
	if !items[0].BackedUp || items[0].LastUsedAt == nil {
		t.Fatalf("updated passkey = %+v", items[0])
	}

	if err := store.DeletePasskey(userID, created.ID); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListPasskeys(userID)
	if err != nil || len(items) != 0 {
		t.Fatalf("passkeys after delete = %+v, %v", items, err)
	}
}

func TestPasskeyNameValidationAndUniqueness(t *testing.T) {
	store, userID := passkeyStore(t)
	credential := &webauthn.Credential{ID: []byte("same-id"), PublicKey: []byte("key")}
	created, err := store.AddPasskey(userID, "localhost", "", credential)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddPasskey(userID, "localhost", "Duplicate", credential); err == nil {
		t.Fatal("duplicate credential should fail")
	}
	if _, err := store.AddPasskey(userID, "localhost", string(make([]byte, 81)), &webauthn.Credential{ID: []byte("other")}); err == nil {
		t.Fatal("overlong name should fail")
	}
	if _, err := store.RenamePasskey(userID, created.ID, "   "); err == nil {
		t.Fatal("empty renamed name should fail")
	}
	if _, err := store.RenamePasskey(userID, created.ID, string(make([]byte, 81))); err == nil {
		t.Fatal("overlong renamed name should fail")
	}
	if _, err := store.RenamePasskey(userID+1, created.ID, "Not mine"); !errors.Is(err, auth.ErrPasskeyNotFound) {
		t.Fatalf("cross-account rename error = %v", err)
	}
}
