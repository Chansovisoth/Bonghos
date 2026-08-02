// Package config handles BONGHOS_HOME resolution, directory layout and the
// bonghos.toml configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Well-known relative paths inside BONGHOS_HOME. All internal storage uses
// these relative paths so the runtime directory remains portable.
const (
	DirServers = "servers"
	DirBackups = "backups"
	DirSystem  = "system"
	DirBin     = "system/bin"
	DirWeb     = "system/web"
	DirConfig  = "system/config"
	DirData    = "system/data"
	DirLogs    = "system/logs"
	DirOpLogs  = "system/logs/operations"
	DirRuntime = "system/runtime"
	DirTemp    = "system/temp"
	DirUploads = "system/temp/uploads"
	DirStaging = "system/temp/staging"

	FileConfig    = "system/config/bonghos.toml"
	FileSecretKey = "system/config/secret.key"
	FileDatabase  = "system/data/bonghos.db"
	FileLog       = "system/logs/bonghos.log"
	FileAuditLog  = "system/logs/audit.log"
	FileActive    = "system/runtime/active-instance.json"
	FileSupState  = "system/runtime/supervisor-state.json"
	FileOpLock    = "system/runtime/operation.lock"
	FileSupSocket = "system/runtime/supervisor.sock"

	ServersJavaModded = "servers/minecraft-java/modded"
	BackupsJavaModded = "backups/minecraft-java/modded"
)

// ResolveHome resolves the Bonghos runtime root in priority order:
// 1. explicit --home flag value, 2. BONGHOS_HOME env, 3. ~/bonghos.
func ResolveHome(flagValue string) (string, error) {
	if flagValue != "" {
		return absClean(flagValue)
	}
	if env := os.Getenv("BONGHOS_HOME"); env != "" {
		return absClean(env)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine user home directory: %w", err)
	}
	return filepath.Join(home, "bonghos"), nil
}

func absClean(p string) (string, error) {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// Layout returns all directories that must exist inside the runtime root.
func Layout() []string {
	return []string{
		DirServers, ServersJavaModded, "servers/minecraft-java/vanilla",
		"servers/minecraft-bedrock/vanilla",
		DirBackups, BackupsJavaModded, "backups/minecraft-java/vanilla",
		"backups/minecraft-bedrock/vanilla",
		DirSystem, DirBin, DirWeb, DirConfig, DirData, DirLogs, DirOpLogs,
		DirRuntime, DirTemp, DirUploads, DirStaging,
	}
}

// InitHome creates the portable runtime directory structure.
func InitHome(home string) error {
	for _, d := range Layout() {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}
	for _, d := range []string{DirConfig, DirData, DirRuntime} {
		if err := os.Chmod(filepath.Join(home, d), 0o700); err != nil {
			return err
		}
	}
	return nil
}

// Config models system/config/bonghos.toml. A deliberately small hand-rolled
// TOML subset (key = value, [section]) keeps runtime dependencies at zero.
type Config struct {
	BindAddress          string
	Port                 int
	AllowInsecureHTTPURL bool
	TrustedDownloadHosts []string
	MaxUploadBytes       int64
	MaxArchiveBytes      int64
	MaxArchiveFiles      int64
	GracefulStopSeconds  int
	MetricsIntervalSec   int
	MetricsRetentionDays int
	SessionHours         int
	FreeSpaceReserveMB   int64
	LogLevel             string
}

func Default() *Config {
	return &Config{
		BindAddress:          "127.0.0.1",
		Port:                 8080,
		AllowInsecureHTTPURL: false,
		MaxUploadBytes:       32 << 30,
		MaxArchiveBytes:      64 << 30,
		MaxArchiveFiles:      1000000,
		GracefulStopSeconds:  120,
		MetricsIntervalSec:   10,
		MetricsRetentionDays: 14,
		SessionHours:         72,
		FreeSpaceReserveMB:   1024,
		LogLevel:             "info",
	}
}

// Load reads bonghos.toml under home, applying defaults for missing keys.
func Load(home string) (*Config, error) {
	c := Default()
	path := filepath.Join(home, FileConfig)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"`)
		switch k {
		case "bind_address":
			c.BindAddress = v
		case "port":
			c.Port = atoi(v, c.Port)
		case "allow_insecure_http_url":
			c.AllowInsecureHTTPURL = v == "true"
		case "trusted_download_hosts":
			v = strings.Trim(v, "[]")
			for _, h := range strings.Split(v, ",") {
				h = strings.Trim(strings.TrimSpace(h), `"`)
				if h != "" {
					c.TrustedDownloadHosts = append(c.TrustedDownloadHosts, h)
				}
			}
		case "max_upload_bytes":
			c.MaxUploadBytes = atoi64(v, c.MaxUploadBytes)
		case "max_archive_bytes":
			c.MaxArchiveBytes = atoi64(v, c.MaxArchiveBytes)
		case "max_archive_files":
			c.MaxArchiveFiles = atoi64(v, c.MaxArchiveFiles)
		case "graceful_stop_seconds":
			c.GracefulStopSeconds = atoi(v, c.GracefulStopSeconds)
		case "metrics_interval_seconds":
			c.MetricsIntervalSec = atoi(v, c.MetricsIntervalSec)
		case "metrics_retention_days":
			c.MetricsRetentionDays = atoi(v, c.MetricsRetentionDays)
		case "session_hours":
			c.SessionHours = atoi(v, c.SessionHours)
		case "free_space_reserve_mb":
			c.FreeSpaceReserveMB = atoi64(v, c.FreeSpaceReserveMB)
		case "log_level":
			c.LogLevel = v
		}
	}
	return c, nil
}

// Save writes bonghos.toml atomically.
func Save(home string, c *Config) error {
	var b strings.Builder
	b.WriteString("# Bonghos configuration\n\n[web]\n")
	fmt.Fprintf(&b, "bind_address = %q\n", c.BindAddress)
	fmt.Fprintf(&b, "port = %d\n", c.Port)
	fmt.Fprintf(&b, "session_hours = %d\n\n", c.SessionHours)
	b.WriteString("[downloads]\n")
	fmt.Fprintf(&b, "allow_insecure_http_url = %t\n", c.AllowInsecureHTTPURL)
	hosts := make([]string, len(c.TrustedDownloadHosts))
	for i, h := range c.TrustedDownloadHosts {
		hosts[i] = strconv.Quote(h)
	}
	fmt.Fprintf(&b, "trusted_download_hosts = [%s]\n", strings.Join(hosts, ", "))
	fmt.Fprintf(&b, "max_upload_bytes = %d\n", c.MaxUploadBytes)
	fmt.Fprintf(&b, "max_archive_bytes = %d\n", c.MaxArchiveBytes)
	fmt.Fprintf(&b, "max_archive_files = %d\n\n", c.MaxArchiveFiles)
	b.WriteString("[runtime]\n")
	fmt.Fprintf(&b, "graceful_stop_seconds = %d\n", c.GracefulStopSeconds)
	fmt.Fprintf(&b, "free_space_reserve_mb = %d\n\n", c.FreeSpaceReserveMB)
	b.WriteString("[monitoring]\n")
	fmt.Fprintf(&b, "metrics_interval_seconds = %d\n", c.MetricsIntervalSec)
	fmt.Fprintf(&b, "metrics_retention_days = %d\n\n", c.MetricsRetentionDays)
	b.WriteString("[logging]\n")
	fmt.Fprintf(&b, "log_level = %q\n", c.LogLevel)
	return AtomicWrite(filepath.Join(home, FileConfig), []byte(b.String()), 0o600)
}

// AtomicWrite writes data to path via a same-directory temp file and rename.
func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".bonghos-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func atoi64(s string, def int64) int64 {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return def
}
