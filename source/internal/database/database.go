// Package database opens the Bonghos SQLite database and applies versioned
// migrations. The database stores Bonghos metadata only — Minecraft server
// files remain normal files on disk.
package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Chansovisoth/Bonghos/migrations"
)

func open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&_txlock=immediate", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // sqlite single-writer; serialize through one conn
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Open opens (creating if needed) the SQLite database at path with WAL mode,
// foreign keys and busy timeout, then applies pending migrations.
func Open(path string) (*sql.DB, error) {
	db, err := open(path)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// OpenForMaintenance opens the database without applying migrations. The
// installer uses it to integrity-check and checkpoint an older database before
// taking the pre-update snapshot; applying a migration first would make that
// snapshot unsuitable for a rollback to the previous executable.
func OpenForMaintenance(path string) (*sql.DB, error) {
	return open(path)
}

// IntegrityCheck runs PRAGMA integrity_check and returns an error on failure.
func IntegrityCheck(db *sql.DB) error {
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("sqlite integrity check failed: %s", result)
	}
	return nil
}

// Checkpoint forces a WAL checkpoint (used before backups/exports/updates).
func Checkpoint(db *sql.DB) error {
	_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// Version returns the current schema version.
func Version(db *sql.DB) (int, error) {
	var v int
	err := db.QueryRow("PRAGMA user_version").Scan(&v)
	return v, err
}

// Migrate applies all pending migrations transactionally in version order.
// Migration files are named NNNN_description.sql.
func Migrate(db *sql.DB) error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return err
	}
	type mig struct {
		version int
		name    string
	}
	var migs []mig
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		numStr, _, ok := strings.Cut(name, "_")
		if !ok {
			continue
		}
		v, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		migs = append(migs, mig{v, name})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	current, err := Version(db)
	if err != nil {
		return err
	}
	for _, m := range migs {
		if m.version <= current {
			continue
		}
		body, err := migrations.FS.ReadFile(m.name)
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s (set version): %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %s (commit): %w", m.name, err)
		}
	}
	return nil
}
