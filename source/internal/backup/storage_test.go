package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyStorageVerified(t *testing.T) {
	base := t.TempDir()
	current := filepath.Join(base, "current")
	target := filepath.Join(base, "target")
	if err := os.MkdirAll(filepath.Join(current, "project", "metadata"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"project/backup.tar.zst":       "archive",
		"project/metadata/backup.json": "metadata",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(current, filepath.FromSlash(name)), []byte(body), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if err := CopyStorageVerified(current, target); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(name)))
		if err != nil || string(got) != want {
			t.Fatalf("copied %s = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestCopyStorageRejectsExistingAndNestedTargets(t *testing.T) {
	base := t.TempDir()
	current := filepath.Join(base, "current")
	if err := os.Mkdir(current, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(base, "existing")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CopyStorageVerified(current, existing); err == nil {
		t.Fatal("existing destination was accepted")
	}
	if err := ValidateStorageRoot(current, filepath.Join(current, "nested")); err == nil {
		t.Fatal("nested destination was accepted")
	}
}

func TestStorageOperationsRejectSymlinkRoot(t *testing.T) {
	base := t.TempDir()
	realRoot := filepath.Join(base, "real")
	if err := os.Mkdir(realRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(realRoot, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := StorageEmpty(link); err == nil {
		t.Fatal("StorageEmpty accepted a symbolic-link root")
	}
	if err := CopyStorageVerified(link, filepath.Join(base, "target")); err == nil {
		t.Fatal("CopyStorageVerified accepted a symbolic-link root")
	}
}
