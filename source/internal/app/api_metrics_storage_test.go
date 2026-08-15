package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/config"
)

func TestStorageSnapshotRefreshesOnlyWhenRequested(t *testing.T) {
	home := t.TempDir()
	serverDir := filepath.Join(home, config.DirServers)
	if err := os.MkdirAll(serverDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "first.bin"), []byte("1234"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := &App{Home: home}
	first := a.collectStorageSnapshot()
	if first.DiskTotal <= 0 || first.BonghosDirBytes < 4 || first.ServerDirBytes != 4 {
		t.Fatalf("unexpected first storage snapshot: %+v", first)
	}

	if err := os.WriteFile(filepath.Join(serverDir, "second.bin"), []byte("1234567"), 0o600); err != nil {
		t.Fatal(err)
	}
	cached := a.cachedStorageSnapshot()
	if cached.ServerDirBytes != first.ServerDirBytes {
		t.Fatalf("cached storage changed without a refresh: got %d, want %d", cached.ServerDirBytes, first.ServerDirBytes)
	}

	refreshed := a.collectStorageSnapshot()
	if refreshed.ServerDirBytes != first.ServerDirBytes+7 {
		t.Fatalf("manual refresh did not include the new file: first=%d refreshed=%d", first.ServerDirBytes, refreshed.ServerDirBytes)
	}
}

func TestStorageSnapshotExcludesExternalBackupDirectory(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, config.DirBackups), 0o700); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external-backups")
	if err := os.MkdirAll(external, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "archive.tar.zst"), []byte("external archive"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := &App{Home: home, Cfg: &config.Config{BackupDirectory: external}}
	snapshot := a.collectStorageSnapshot()
	if snapshot.BackupDirBytes != 0 {
		t.Fatalf("BackupDirBytes = %d, want 0 for storage outside BONGHOS_HOME", snapshot.BackupDirBytes)
	}
	if snapshot.BonghosDirBytes != 0 {
		t.Fatalf("BonghosDirBytes = %d, want 0 for an otherwise empty home", snapshot.BonghosDirBytes)
	}
}

func TestNewRejectsCustomBackupDirectoryInsideHome(t *testing.T) {
	home := t.TempDir()
	if err := config.InitHome(home); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(home, "custom-backups")
	if err := os.MkdirAll(custom, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.BackupDirectory = custom
	if err := config.Save(home, cfg); err != nil {
		t.Fatal(err)
	}
	_, err := New(home, nil)
	if err == nil || !strings.Contains(err.Error(), "must be outside BONGHOS_HOME") {
		t.Fatalf("New() error = %v, want internal custom-directory rejection", err)
	}
}
