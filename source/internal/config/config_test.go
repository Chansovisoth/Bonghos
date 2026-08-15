package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupRootDefaultsInsideHome(t *testing.T) {
	home := t.TempDir()
	if got, want := BackupRoot(home, Default()), filepath.Join(home, DirBackups); got != want {
		t.Fatalf("BackupRoot() = %q, want %q", got, want)
	}
}

func TestBackupDirectoryRoundTrip(t *testing.T) {
	home := t.TempDir()
	if err := InitHome(home); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(t.TempDir(), "bonghos-backups")
	c := Default()
	c.BackupDirectory = want
	if err := Save(home, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.BackupDirectory != want || BackupRoot(home, loaded) != want {
		t.Fatalf("loaded backup directory = %q (%q), want %q", loaded.BackupDirectory, BackupRoot(home, loaded), want)
	}
	if _, err := os.Stat(filepath.Join(home, FileConfig)); err != nil {
		t.Fatal(err)
	}
}

func TestRelativeBackupDirectoryStaysInsideHome(t *testing.T) {
	home := t.TempDir()
	c := Default()
	c.BackupDirectory = "storage/backups"
	if got, want := BackupRoot(home, c), filepath.Join(home, "storage", "backups"); got != want {
		t.Fatalf("BackupRoot() = %q, want %q", got, want)
	}
}
