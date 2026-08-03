package minecraft

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RunningProcess describes a Java process that appears to be serving a
// particular directory.
type RunningProcess struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
	CWD     string `json:"cwd"`
}

// FindRunningJavaIn reports Java processes whose working directory is inside
// dir. Importing or adopting a directory that already has a live server would
// corrupt the world, so callers use this to refuse rather than to take over:
// Bonghos never kills or adopts a process it did not start.
//
// Detection is best-effort. It reads /proc, so it only sees processes owned by
// the current user, and an empty result is not proof that nothing is running.
func FindRunningJavaIn(dir string) []RunningProcess {
	target, err := filepath.EvalSymlinks(dir)
	if err != nil {
		target = filepath.Clean(dir)
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil // not Linux, or /proc unavailable
	}

	var found []RunningProcess
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a process directory
		}
		procDir := filepath.Join("/proc", e.Name())

		// Resolving cwd requires ownership; other users' processes simply fail
		// here, which is why an empty result cannot be treated as proof.
		cwd, err := os.Readlink(filepath.Join(procDir, "cwd"))
		if err != nil {
			continue
		}
		if cwd != target && !strings.HasPrefix(cwd, target+string(os.PathSeparator)) {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(procDir, "cmdline"))
		if err != nil || len(raw) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
		if len(args) == 0 {
			continue
		}
		exe := filepath.Base(args[0])
		if exe != "java" && !strings.HasPrefix(exe, "java") {
			continue
		}

		cmd := strings.Join(args, " ")
		if len(cmd) > 200 {
			cmd = cmd[:200] + "…"
		}
		found = append(found, RunningProcess{PID: pid, Command: cmd, CWD: cwd})
	}
	return found
}
