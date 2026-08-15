package portability

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/config"
)

func TestExternalBackupsExportIntoPortableDefault(t *testing.T) {
	home := filepath.Join(t.TempDir(), "source-home")
	if err := config.InitHome(home); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external-backups")
	archiveRel := filepath.Join("java", "vanilla", "example", "backup.tar.zst")
	if err := os.MkdirAll(filepath.Join(external, filepath.Dir(archiveRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("portable external backup")
	if err := os.WriteFile(filepath.Join(external, archiveRel), want, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.BackupDirectory = external
	if err := config.Save(home, cfg); err != nil {
		t.Fatal(err)
	}

	exportPath := filepath.Join(t.TempDir(), "backups.tar.zst")
	if _, err := Export(home, ExportOptions{Scope: ScopeBackups, Output: exportPath, Version: "test"}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target-home")
	if _, err := Import(exportPath, target, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, config.DirBackups, archiveRel))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("imported backup = %q, want %q", got, want)
	}
	importedCfg, err := config.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	if importedCfg.BackupDirectory != "" {
		t.Fatalf("import retained source backup directory %q", importedCfg.BackupDirectory)
	}
}
