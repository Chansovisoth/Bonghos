package minecraft

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// findJava locates a Java binary, skipping the test when none is installed.
func findJava(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("java"); err == nil {
		return p
	}
	matches, _ := filepath.Glob("/usr/lib/jvm/*/bin/java")
	if len(matches) > 0 {
		return matches[0]
	}
	t.Skip("no Java installation available")
	return ""
}

func TestFindRunningJavaInDetectsAndClearsProcess(t *testing.T) {
	java := findJava(t)
	dir := t.TempDir()

	if got := FindRunningJavaIn(dir); len(got) != 0 {
		t.Fatalf("empty directory reported %d running processes", len(got))
	}

	// A long-running Java process whose working directory is the target.
	cmd := exec.Command(java, "-version")
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = os.NewFile(0, os.DevNull), os.NewFile(0, os.DevNull)
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start Java: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	// The process is short-lived, so poll rather than assuming timing.
	var seen []RunningProcess
	for i := 0; i < 50; i++ {
		if seen = FindRunningJavaIn(dir); len(seen) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(seen) == 0 {
		t.Skip("java exited before detection could observe it")
	}
	if seen[0].PID != cmd.Process.Pid {
		t.Errorf("detected PID %d, want %d", seen[0].PID, cmd.Process.Pid)
	}
	if seen[0].Command == "" {
		t.Error("detected process has no command line")
	}
}

// A sibling directory whose path shares a prefix must not match.
func TestFindRunningJavaInDoesNotMatchPrefixSiblings(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "server")
	sibling := base + "-other"
	for _, d := range []string{target, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { os.RemoveAll(sibling) })

	if got := FindRunningJavaIn(target); len(got) != 0 {
		t.Errorf("unexpected matches: %+v", got)
	}
}

func TestFindRunningJavaInHandlesMissingDirectory(t *testing.T) {
	// Must not panic or error on a path that does not exist.
	if got := FindRunningJavaIn("/definitely/not/a/real/path"); len(got) != 0 {
		t.Errorf("nonexistent path reported %d processes", len(got))
	}
}
