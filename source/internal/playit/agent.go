package playit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// FindDaemon locates the official Playit daemon without downloading or
// replacing software on the host. The Bonghos-local location allows a future
// installer to keep the dependency within BONGHOS_HOME.
func FindDaemon(home string) (string, error) {
	candidates := []string{
		filepath.Join(home, "system", "bin", "playitd"),
		"/opt/playit/playitd",
		"/usr/local/bin/playitd",
		"/usr/bin/playitd",
	}
	if path, err := exec.LookPath("playitd"); err == nil {
		candidates = append([]string{path}, candidates...)
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("official playitd executable not found")
}

func DaemonAvailable(home string) bool {
	_, err := FindDaemon(home)
	return err == nil
}

// CleanupRuntime removes only the credential and socket files created by the
// Bonghos-managed daemon. It also clears remnants after an unclean hard kill.
func CleanupRuntime(home string) {
	runtimeDir := filepath.Join(home, "system", "runtime", "playit")
	_ = os.Remove(filepath.Join(runtimeDir, "playit.toml"))
	_ = os.Remove(filepath.Join(runtimeDir, "playitd.sock"))
}

// RunManagedAgent runs the official daemon using a private runtime credential
// file. The long-lived credential remains encrypted in SQLite while stopped;
// it is materialized with mode 0600 only for the lifetime of the daemon.
func RunManagedAgent(ctx context.Context, home string, store *Store) error {
	daemon, err := FindDaemon(home)
	if err != nil {
		return err
	}
	secret, err := store.Secret()
	if err != nil {
		return err
	}
	runtimeDir := filepath.Join(home, "system", "runtime", "playit")
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(runtimeDir, 0o700); err != nil {
		return err
	}
	secretPath := filepath.Join(runtimeDir, "playit.toml")
	socketPath := filepath.Join(runtimeDir, "playitd.sock")
	logPath := filepath.Join(home, "system", "logs", "playit.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	CleanupRuntime(home)
	if err := os.WriteFile(secretPath, []byte("secret_key = "+strconv.Quote(secret)+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(secretPath, 0o600); err != nil {
		_ = os.Remove(secretPath)
		return err
	}
	defer func() {
		_ = os.Remove(secretPath)
		_ = os.Remove(socketPath)
	}()

	cmd := exec.CommandContext(ctx, daemon,
		"--secret-path", secretPath,
		"--socket-path", socketPath,
		"--log-path", logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("playitd exited: %w", err)
	}
	return nil
}
