package files

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return p
}

func lim() Limits { return Limits{MaxBytes: 1 << 20, MaxFiles: 100} }

func TestExtractZipOK(t *testing.T) {
	src := writeZip(t, map[string]string{"pack/run.sh": "#!/bin/sh\n", "pack/server.properties": "motd=hi\n"})
	dest := t.TempDir()
	if err := Extract(src, dest, lim(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "pack", "run.sh")); err != nil {
		t.Error("expected extracted file:", err)
	}
}

func TestExtractZipTraversal(t *testing.T) {
	src := writeZip(t, map[string]string{"../evil.txt": "x"})
	if err := Extract(src, t.TempDir(), lim(), nil); err == nil {
		t.Fatal("zip traversal not rejected")
	}
}

func TestExtractZipAbsolute(t *testing.T) {
	src := writeZip(t, map[string]string{"/etc/evil": "x"})
	if err := Extract(src, t.TempDir(), lim(), nil); err == nil {
		t.Fatal("absolute zip path not rejected")
	}
}

func TestExtractFileCountLimit(t *testing.T) {
	entries := map[string]string{}
	for i := 0; i < 20; i++ {
		entries[filepath.Join("d", string(rune('a'+i))+".txt")] = "x"
	}
	src := writeZip(t, entries)
	l := Limits{MaxBytes: 1 << 20, MaxFiles: 5}
	if err := Extract(src, t.TempDir(), l, nil); err == nil {
		t.Fatal("file-count limit not enforced")
	}
}

func TestExtractSizeLimit(t *testing.T) {
	big := make([]byte, 64*1024)
	src := writeZip(t, map[string]string{"big.bin": string(big)})
	l := Limits{MaxBytes: 1024, MaxFiles: 10}
	if err := Extract(src, t.TempDir(), l, nil); err == nil {
		t.Fatal("decompressed-size limit not enforced")
	}
}

func TestExtractTarSymlinkEscape(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.tar.gz")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{
		Name: "link", Typeflag: tar.TypeSymlink, Linkname: "../../outside",
	}); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()
	f.Close()
	if err := Extract(p, t.TempDir(), lim(), nil); err == nil {
		t.Fatal("unsafe tar symlink not rejected")
	}
}

func TestDetectFormat(t *testing.T) {
	src := writeZip(t, map[string]string{"a": "b"})
	// Rename to hide the extension; detection must inspect content.
	renamed := filepath.Join(filepath.Dir(src), "mystery.bin")
	if err := os.Rename(src, renamed); err != nil {
		t.Fatal(err)
	}
	got, err := DetectFormat(renamed)
	if err != nil {
		t.Fatal(err)
	}
	if got != "zip" {
		t.Errorf("DetectFormat = %q, want zip", got)
	}
}

func TestFindServerRoot(t *testing.T) {
	dest := t.TempDir()
	inner := filepath.Join(dest, "SomePack-1.0")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(inner, "run.sh"), []byte("#!/bin/sh\njava -jar server.jar\n"), 0o755)
	os.WriteFile(filepath.Join(inner, "server.properties"), []byte("motd=x\n"), 0o644)
	root, err := FindServerRoot(dest)
	if err != nil {
		t.Fatal(err)
	}
	if root != inner {
		t.Errorf("FindServerRoot = %q, want %q", root, inner)
	}
}

func TestExtractRespectsFreeSpaceReserve(t *testing.T) {
	src := writeZip(t, map[string]string{"pack/run.sh": "#!/bin/sh\n"})
	dest := t.TempDir()

	// A reserve larger than the whole filesystem can never be satisfied, so
	// extraction must refuse rather than filling the disk.
	huge := Limits{MaxBytes: 1 << 20, MaxFiles: 100, FreeSpaceReserve: 1 << 62}
	if err := Extract(src, dest, huge, nil); !errors.Is(err, ErrDiskSpace) {
		t.Errorf("Extract with an unsatisfiable reserve returned %v, want ErrDiskSpace", err)
	}

	// A reserve of zero disables the check entirely.
	if err := Extract(src, t.TempDir(), Limits{MaxBytes: 1 << 20, MaxFiles: 100}, nil); err != nil {
		t.Errorf("Extract without a reserve failed: %v", err)
	}
}

func TestFreeSpaceReportsSomethingForARealDirectory(t *testing.T) {
	if got := FreeSpace(t.TempDir()); got <= 0 {
		t.Errorf("FreeSpace returned %d for a real directory", got)
	}
	// An unavailable statistic must be reported as zero, never as an error.
	if got := FreeSpace("/definitely/not/a/real/path"); got != 0 {
		t.Errorf("FreeSpace on a missing path returned %d, want 0", got)
	}
}
