package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerCreateAndCopy(t *testing.T) {
	root := t.TempDir()
	m := &Manager{Root: root}

	if err := m.CreateFile("notes.txt"); err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := m.CreateFile("notes.txt"); err == nil {
		t.Fatal("creating an existing file succeeded")
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Copy("notes.txt", "copies/notes.txt"); err != nil {
		t.Fatalf("copy file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "copies", "notes.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("copied file = %q, %v", data, err)
	}

	if err := os.MkdirAll(filepath.Join(root, "world", "region"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "world", "region", "r.0.0.mca"), []byte("chunk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Copy("world", "archive/world"); err != nil {
		t.Fatalf("copy folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "archive", "world", "region", "r.0.0.mca")); err != nil {
		t.Fatalf("copied folder content: %v", err)
	}
	if err := m.Copy("world", "world/nested"); err == nil {
		t.Fatal("copying a folder into itself succeeded")
	}
	if err := m.Rename("world", "world/nested"); err == nil {
		t.Fatal("moving a folder into itself succeeded")
	}
}

func TestManagerCopyAndMoveAcrossProjectRoots(t *testing.T) {
	sourceRoot := t.TempDir()
	destinationRoot := t.TempDir()
	source := &Manager{Root: sourceRoot}
	destination := &Manager{Root: destinationRoot}

	if err := os.WriteFile(filepath.Join(sourceRoot, "copy.txt"), []byte("copy me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "move.txt"), []byte("move me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := source.CopyTo("copy.txt", destination, "imports/copy.txt"); err != nil {
		t.Fatalf("copy across roots: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(destinationRoot, "imports", "copy.txt")); err != nil || string(data) != "copy me" {
		t.Fatalf("cross-root copy = %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "copy.txt")); err != nil {
		t.Fatalf("cross-root copy removed source: %v", err)
	}

	if err := source.MoveTo("move.txt", destination, "imports/move.txt"); err != nil {
		t.Fatalf("move across roots: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(destinationRoot, "imports", "move.txt")); err != nil || string(data) != "move me" {
		t.Fatalf("cross-root move = %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "move.txt")); !os.IsNotExist(err) {
		t.Fatalf("cross-root move left source behind: %v", err)
	}

	if err := source.CopyTo("copy.txt", destination, "../outside.txt"); err == nil {
		t.Fatal("cross-root copy escaped the destination project")
	}
}
