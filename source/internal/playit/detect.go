package playit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Detection struct {
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	State             string `json:"state"`
	ExternallyManaged bool   `json:"externally_managed"`
}

// DetectExisting finds common Playit deployments without reading agent
// secrets or container environments. An empty result means "not detected",
// not a guarantee that no custom deployment exists.
func DetectExisting(ctx context.Context) []Detection {
	if runtime.GOOS != "linux" {
		return []Detection{}
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	var found []Detection
	seen := map[string]bool{}
	add := func(item Detection) {
		key := item.Kind + "\x00" + item.Name
		if !seen[key] {
			seen[key] = true
			found = append(found, item)
		}
	}
	if _, err := exec.LookPath("systemctl"); err == nil {
		managedActive := commandState(ctx, "systemctl", "--user", "is-active", "bonghos-playit.service") == "active"
		if managedActive {
			add(Detection{Kind: "bonghos", Name: "bonghos-playit.service", State: "active", ExternallyManaged: false})
		}
		if state := commandState(ctx, "systemctl", "is-active", "playit.service"); state == "active" {
			add(Detection{Kind: "systemd-system", Name: "playit.service", State: state, ExternallyManaged: true})
		}
		if state := commandState(ctx, "systemctl", "--user", "is-active", "playit.service"); state == "active" {
			add(Detection{Kind: "systemd-user", Name: "playit.service", State: state, ExternallyManaged: true})
		}
	}
	for _, runtimeName := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(runtimeName); err != nil {
			continue
		}
		out, err := exec.CommandContext(ctx, runtimeName, "ps", "--format", "{{.Image}}\t{{.Names}}").CombinedOutput()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
			if len(parts) != 2 {
				continue
			}
			image := strings.ToLower(parts[0])
			name := strings.TrimSpace(parts[1])
			if strings.Contains(image, "playit-cloud/playit-agent") || strings.Contains(image, "playitcloud/playit") {
				add(Detection{Kind: runtimeName, Name: name, State: "running", ExternallyManaged: true})
			}
		}
	}
	// Catch custom units, tmux launches, and renamed containers that expose a
	// Playit executable in /proc. Do not read process environments.
	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil || entry.Name() == strconv.Itoa(os.Getpid()) {
			continue
		}
		exe, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue
		}
		base := strings.ToLower(filepath.Base(exe))
		if base == "playit" || base == "playit-cli" || base == "playitd" || strings.HasPrefix(base, "playit-linux-") {
			if base == "playitd" && seen["bonghos\x00bonghos-playit.service"] {
				continue
			}
			add(Detection{Kind: "process", Name: filepath.Base(exe), State: "running", ExternallyManaged: true})
		}
	}
	return found
}

func commandState(ctx context.Context, name string, args ...string) string {
	out, _ := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(out))
}
