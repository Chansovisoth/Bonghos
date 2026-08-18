// Package systemd generates and manages the Bonghos systemd user
// services. Normal operation never requires root; lingering is only ever
// explained, never enabled silently.
package systemd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ServiceControlPlane = "bonghos.service"
	ServiceMinecraft    = "bonghos-minecraft.service"
	ServicePlayit       = "bonghos-playit.service"
)

// unitDir returns ~/.config/systemd/user.
func unitDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

// unitQuote returns one systemd.syntax quoted value. Besides the usual string
// escapes, percent signs must be doubled so paths are not interpreted as unit
// specifiers. This keeps custom Bonghos homes with spaces or percent signs
// valid in Environment, WorkingDirectory and ExecStart directives.
func unitQuote(value string) string {
	value = strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`%`, `%%`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	).Replace(value)
	return `"` + value + `"`
}

// unitPath escapes a path used by a scalar directive such as
// WorkingDirectory. Unlike ExecStart, systemd retains surrounding quotes for
// this directive, so whitespace and control bytes use \xNN escapes instead.
func unitPath(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c == '%':
			b.WriteString("%%")
		case c <= ' ' || c == '\\' || c == '"':
			fmt.Fprintf(&b, `\x%02x`, c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// ControlPlaneUnit renders bonghos.service for the given runtime root.
func ControlPlaneUnit(bonghosHome, binPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Bonghos Minecraft Hosting Control Panel
After=network.target

[Service]
Type=exec
Environment=%s
WorkingDirectory=%s
ExecStart=%s --home %s serve
Restart=always
RestartSec=5s
NoNewPrivileges=yes
PrivateTmp=yes
RestrictSUIDSGID=yes

[Install]
WantedBy=default.target
`, unitQuote("BONGHOS_HOME="+bonghosHome), unitPath(bonghosHome), unitQuote(binPath), unitQuote(bonghosHome))
}

// MinecraftUnit renders bonghos-minecraft.service. Deliberately no [Install]
// section: the control plane starts it on demand / after validating autostart,
// so the Minecraft service is never unconditionally enabled at boot.
func MinecraftUnit(bonghosHome, binPath string, gracefulStopSeconds int) string {
	return fmt.Sprintf(`[Unit]
Description=Bonghos Minecraft Supervisor
After=network.target

[Service]
Type=exec
Environment=%s
WorkingDirectory=%s
ExecStart=%s --home %s supervisor
Restart=on-failure
RestartSec=5s
KillMode=control-group
TimeoutStopSec=%d
NoNewPrivileges=yes
RestrictSUIDSGID=yes

# No [Install] section on purpose: started on demand by the Bonghos control
# plane, which validates the active project and prevents duplicate starts.
`, unitQuote("BONGHOS_HOME="+bonghosHome), unitPath(bonghosHome), unitQuote(binPath), unitQuote(bonghosHome), gracefulStopSeconds+30)
}

// PlayitUnit renders the optional Playit daemon service. It is installed but
// not enabled; linking an agent starts it on demand, and the control panel can
// start it again after reboot when Playit remains enabled.
func PlayitUnit(bonghosHome, binPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Bonghos Managed Playit.gg Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
Environment=%s
WorkingDirectory=%s
ExecStart=%s --home %s playit-agent
Restart=on-failure
RestartSec=5s
NoNewPrivileges=yes
PrivateTmp=yes
RestrictSUIDSGID=yes

`, unitQuote("BONGHOS_HOME="+bonghosHome), unitPath(bonghosHome), unitQuote(binPath), unitQuote(bonghosHome))
}

// Available reports whether a systemd user manager is reachable.
func Available() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	cmd := exec.Command("systemctl", "--user", "is-system-running")
	out, _ := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	return s != "" && s != "offline" && !strings.Contains(s, "Failed to connect")
}

// Install writes all unit files and reloads the user daemon.
func Install(bonghosHome string, gracefulStopSeconds int) error {
	dir, err := unitDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	binPath := filepath.Join(bonghosHome, "system", "bin", "bonghos")
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("bonghos executable not found at %s", binPath)
	}
	if err := os.WriteFile(filepath.Join(dir, ServiceControlPlane),
		[]byte(ControlPlaneUnit(bonghosHome, binPath)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ServiceMinecraft),
		[]byte(MinecraftUnit(bonghosHome, binPath, gracefulStopSeconds)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ServicePlayit),
		[]byte(PlayitUnit(bonghosHome, binPath)), 0o644); err != nil {
		return err
	}
	return DaemonReload()
}

func DaemonReload() error {
	return run("systemctl", "--user", "daemon-reload")
}

// Repair regenerates all units (after BONGHOS_HOME moves) then reloads.
func Repair(bonghosHome string, gracefulStopSeconds int) error {
	return Install(bonghosHome, gracefulStopSeconds)
}

// Uninstall stops and removes all unit files (data is untouched).
func Uninstall() error {
	run("systemctl", "--user", "stop", ServiceMinecraft)
	run("systemctl", "--user", "stop", ServicePlayit)
	run("systemctl", "--user", "stop", ServiceControlPlane)
	run("systemctl", "--user", "disable", ServiceControlPlane)
	dir, err := unitDir()
	if err != nil {
		return err
	}
	os.Remove(filepath.Join(dir, ServiceControlPlane))
	os.Remove(filepath.Join(dir, ServiceMinecraft))
	os.Remove(filepath.Join(dir, ServicePlayit))
	return DaemonReload()
}

// Start / Stop / Enable / Status wrappers -------------------------------------

func Start(unit string) error   { return run("systemctl", "--user", "start", unit) }
func Stop(unit string) error    { return run("systemctl", "--user", "stop", unit) }
func Restart(unit string) error { return run("systemctl", "--user", "restart", unit) }
func Enable(unit string) error  { return run("systemctl", "--user", "enable", unit) }
func Disable(unit string) error { return run("systemctl", "--user", "disable", unit) }

// Logs returns recent journal entries, or follows them until interrupted.
func Logs(unit string, follow bool) error {
	args := []string{"--user", "-u", unit, "--no-pager", "-n", "100"}
	if follow {
		args = append(args, "--follow")
	}
	cmd := exec.Command("journalctl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsActive reports whether a unit is currently active.
func IsActive(unit string) bool {
	return State(unit) == "active"
}

// State returns the concise systemd activity state used by the API and Web UI.
// is-active intentionally exits non-zero for useful states such as inactive
// and failed, so preserve its output instead of treating that as an error.
func State(unit string) string {
	out, _ := exec.Command("systemctl", "--user", "is-active", unit).CombinedOutput()
	state := strings.TrimSpace(string(out))
	if state == "" {
		return "unknown"
	}
	return state
}

// Status returns human-readable unit status text.
func Status(unit string) string {
	out, _ := exec.Command("systemctl", "--user", "status", "--no-pager", unit).CombinedOutput()
	return string(out)
}

func run(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %v: %s", bin, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// LingerHint returns the exact command the user may run manually. Bonghos
// never runs it silently.
func LingerHint() (string, error) {
	u := os.Getenv("USER")
	if u == "" {
		return "", errors.New("cannot determine username")
	}
	return "sudo loginctl enable-linger " + u, nil
}

// LingerEnabled reports whether the current user's systemd manager is allowed
// to start at boot and remain available without an interactive login session.
func LingerEnabled() (bool, error) {
	u := os.Getenv("USER")
	if u == "" {
		return false, errors.New("cannot determine username")
	}
	out, err := exec.Command("loginctl", "show-user", u, "--property=Linger", "--value").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("checking linger for %s: %w: %s", u, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)) == "yes", nil
}
