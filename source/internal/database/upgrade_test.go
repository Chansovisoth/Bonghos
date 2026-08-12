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
	if _, err := db.Exec(`INSERT INTO notification_bots
		(name, provider, token_enc, destination_id, enabled,
		 notify_server_started, notify_server_stopped, notify_player_joined, notify_player_left,
		 created_at, updated_at)
		VALUES ('existing-alerts', 'telegram', X'01', '-1001234567890', 1, 1, 1, 1, 1,
		 '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TRIGGER notification_bots_limit_insert`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE notification_bot_telegram_state`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE notification_bot_discoveries`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE notification_bot_destinations`); err != nil {
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
	if version, err := Version(upgraded); err != nil || version != 10 {
		t.Fatalf("upgraded schema version = %d, %v; want 10", version, err)
	}
	if err := upgraded.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='passkeys'`).Scan(&table); err != nil {
		t.Fatalf("passkeys migration was not applied: %v", err)
	}
	if err := upgraded.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='notification_bot_telegram_state'`).Scan(&table); err != nil {
		t.Fatalf("Telegram command-state migration was not applied: %v", err)
	}
	var recoveryCreatedAt string
	if err := upgraded.QueryRow(`SELECT created_at FROM recovery_codes LIMIT 1`).Scan(&recoveryCreatedAt); err != sql.ErrNoRows {
		t.Fatalf("recovery-code metadata migration result = %q, %v; want empty table", recoveryCreatedAt, err)
	}
	var users int
	if err := upgraded.QueryRow(`SELECT COUNT(*) FROM users WHERE username='existing-owner'`).Scan(&users); err != nil || users != 1 {
		t.Fatalf("existing account count = %d, %v; want 1", users, err)
	}
	var destinations int
	if err := upgraded.QueryRow(`SELECT COUNT(*) FROM notification_bot_destinations
		WHERE bot_id=(SELECT id FROM notification_bots WHERE name='existing-alerts')`).Scan(&destinations); err != nil || destinations != 0 {
		t.Fatalf("legacy Telegram destinations = %d, %v; want reset", destinations, err)
	}
	var destination string
	if err := upgraded.QueryRow(`SELECT destination_id FROM notification_bots
		WHERE name='existing-alerts'`).Scan(&destination); err != nil || destination != "" {
		t.Fatalf("legacy Telegram destination field = %q, %v; want empty", destination, err)
	}
}
