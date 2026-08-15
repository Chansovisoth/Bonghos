package backup_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/Chansovisoth/Bonghos/internal/instance"
)

func TestManagerUsesExternalStorageForArchiveLifecycle(t *testing.T) {
	home := t.TempDir()
	if err := config.InitHome(home); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(home, config.FileDatabase))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	inst := &instance.Instance{
		Slug: "external-storage-test", DisplayName: "External storage test",
		ServerDirectory: filepath.ToSlash(filepath.Join(config.ServersJavaModded, "external-storage-test")),
	}
	if err := (&instance.Store{DB: db}).Create(inst); err != nil {
		t.Fatal(err)
	}
	serverDir := inst.AbsoluteDir(home)
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "server.properties"), []byte("motd=test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "backups")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	m := &backup.Manager{Home: home, Root: external, DB: db}
	rec, err := m.Create(inst, backup.TypeConfig, "offline", "manual", 0, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	archiveRel := filepath.FromSlash(rec.ArchivePath[len(config.DirBackups)+1:])
	if _, err := os.Stat(filepath.Join(external, archiveRel)); err != nil {
		t.Fatalf("archive was not written to external storage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, filepath.FromSlash(rec.ArchivePath))); !os.IsNotExist(err) {
		t.Fatalf("archive unexpectedly exists under BONGHOS_HOME: %v", err)
	}
	parked := filepath.Join(t.TempDir(), filepath.Base(archiveRel))
	archivePath := filepath.Join(external, archiveRel)
	if err := os.Rename(archivePath, parked); err != nil {
		t.Fatal(err)
	}
	list, err := m.List(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("List returned %d backup(s) while the archive was outside active storage", len(list))
	}
	if _, err := m.Get(rec.BackupID); !errors.Is(err, backup.ErrArchiveUnavailable) {
		t.Fatalf("Get() error = %v, want ErrArchiveUnavailable", err)
	}
	if err := os.Rename(parked, archivePath); err != nil {
		t.Fatal(err)
	}
	list, err = m.List(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].BackupID != rec.BackupID {
		t.Fatalf("returned archive was not rediscovered: %+v", list)
	}
	if err := m.Verify(rec.BackupID); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(rec.BackupID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(external, archiveRel)); !os.IsNotExist(err) {
		t.Fatalf("external archive was not deleted: %v", err)
	}
}
