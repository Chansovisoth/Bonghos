// Package tmux manages the optional on-demand console session named exactly
// "bonghos". tmux is only a console client: it is never created during boot,
// startup, autostart, scheduling or backups, and is never authoritative for
// server state. Killing tmux never affects Minecraft.
package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const SessionName = "bonghos"

var ErrNotInstalled = errors.New("tmux is not installed; use 'bonghos console --direct' or install tmux")

// Installed reports whether tmux is available.
func Installed() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// SessionExists checks for the "bonghos" tmux session.
func SessionExists() bool {
	cmd := exec.Command("tmux", "has-session", "-t", SessionName)
	return cmd.Run() == nil
}

// sessionCommand returns the command running in the session's first pane.
func sessionCommand() (string, error) {
	out, err := exec.Command("tmux", "display-message", "-p", "-t", SessionName,
		"#{pane_start_command}").CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// OwnedByBonghos verifies the existing session was created by Bonghos (its
// pane runs the bonghos console client). Unknown sessions are never killed,
// replaced, renamed or taken over.
func OwnedByBonghos() bool {
	cmd, err := sessionCommand()
	if err != nil {
		return false
	}
	return strings.Contains(cmd, "console attach --direct")
}

// CreateAndAttach creates the session running only the console client and
// attaches the current terminal. Arguments are passed as an argv array —
// no shell interpolation is involved.
func CreateAndAttach(bonghosBin, home string) error {
	if !Installed() {
		return ErrNotInstalled
	}
	// tmux new-session with a command vector: tmux joins the args itself.
	args := []string{
		"new-session", "-s", SessionName,
		bonghosBin, "--home", home, "console", "attach", "--direct",
	}
	cmd := exec.Command("tmux", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Attach attaches the current terminal to the existing session.
func Attach() error {
	if !Installed() {
		return ErrNotInstalled
	}
	cmd := exec.Command("tmux", "attach", "-t", SessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Console implements `bonghos console`: create the session lazily or attach,
// reporting a conflict for foreign sessions instead of taking them over.
func Console(bonghosBin, home string) error {
	if !Installed() {
		return ErrNotInstalled
	}
	if SessionExists() {
		if !OwnedByBonghos() {
			return fmt.Errorf("a tmux session named %q exists but was not created by Bonghos; "+
				"attach manually with 'tmux attach -t %s' or use 'bonghos console --direct'",
				SessionName, SessionName)
		}
		return Attach()
	}
	return CreateAndAttach(bonghosBin, home)
}
