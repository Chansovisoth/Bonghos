package app

import (
	"os"
	"path/filepath"
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
