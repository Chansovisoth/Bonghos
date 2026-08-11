package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestPreUpdateCheckpointDoesNotMigrateAndUpgradePreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bonghos.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users
		(username, password_hash, totp_secret_enc, role, disabled, created_at, updated_at)
		VALUES ('existing-owner', 'hash', X'01', 'owner', 0, '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE passkeys`); err != nil {
		t.Fatal(err)
	}
	// Recreate the recovery-code table as it existed at schema version 5. The
	// database was initially opened at the latest version so tests can use the
	// real baseline schema before deliberately rolling selected objects back.
	if _, err := db.Exec(`DROP TABLE recovery_codes`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE recovery_codes (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		code_hash TEXT NOT NULL,
		used_at TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_recovery_user ON recovery_codes(user_id)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 5`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	maintenance, err := OpenForMaintenance(path)
	if err != nil {
		t.Fatal(err)
	}
	if version, err := Version(maintenance); err != nil || version != 5 {
		t.Fatalf("maintenance schema version = %d, %v; want 5", version, err)
	}
	var table string
	err = maintenance.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='passkeys'`).Scan(&table)
	if err != sql.ErrNoRows {
		t.Fatalf("maintenance open applied migration: table=%q err=%v", table, err)
	}
	if err := IntegrityCheck(maintenance); err != nil {
		t.Fatal(err)
	}
	if err := Checkpoint(maintenance); err != nil {
		t.Fatal(err)
	}
	if err := maintenance.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	if version, err := Version(upgraded); err != nil || version != 7 {
		t.Fatalf("upgraded schema version = %d, %v; want 7", version, err)
	}
	if err := upgraded.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='passkeys'`).Scan(&table); err != nil {
		t.Fatalf("passkeys migration was not applied: %v", err)
	}
	var recoveryCreatedAt string
	if err := upgraded.QueryRow(`SELECT created_at FROM recovery_codes LIMIT 1`).Scan(&recoveryCreatedAt); err != sql.ErrNoRows {
		t.Fatalf("recovery-code metadata migration result = %q, %v; want empty table", recoveryCreatedAt, err)
	}
	var users int
	if err := upgraded.QueryRow(`SELECT COUNT(*) FROM users WHERE username='existing-owner'`).Scan(&users); err != nil || users != 1 {
		t.Fatalf("existing account count = %d, %v; want 1", users, err)
	}
}
