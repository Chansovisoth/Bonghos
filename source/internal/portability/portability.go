// Package portability implements doctor diagnostics, repair, and
// portable export/import of a complete BONGHOS_HOME.
package portability

import (
	"archive/tar"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/security"
)

// CheckStatus classifies a diagnostic result.
type CheckStatus string

const (
	StatusOK      CheckStatus = "ok"
	StatusWarning CheckStatus = "warning"
	StatusError   CheckStatus = "error"
	StatusFixed   CheckStatus = "fixed"
	StatusSkipped CheckStatus = "skipped"
)

// Check is a single diagnostic result.
type Check struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

// Report is a full doctor run.
type Report struct {
	Home      string    `json:"home"`
	Repair    bool      `json:"repair"`
	StartedAt time.Time `json:"started_at"`
	Checks    []Check   `json:"checks"`
	Errors    int       `json:"errors"`
	Warnings  int       `json:"warnings"`
	Fixed     int       `json:"fixed"`
}

func (r *Report) add(name string, st CheckStatus, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: st, Detail: detail})
	switch st {
	case StatusError:
		r.Errors++
	case StatusWarning:
		r.Warnings++
	case StatusFixed:
		r.Fixed++
	}
}

// Doctor runs diagnostics on a Bonghos home. When repair is true it fixes
// safe problems (stale PIDs, stale locks, stale sockets, WAL checkpoint,
// migrations, service paths) but never touches Minecraft server files and
// never starts Minecraft.
func Doctor(home string, repair bool) (*Report, error) {
	r := &Report{Home: home, Repair: repair, StartedAt: time.Now().UTC()}

	// 1. Home structure.
	info, err := os.Stat(home)
	if err != nil || !info.IsDir() {
		r.add("home", StatusError, "Bonghos home not found at "+home)
		return r, fmt.Errorf("home not found: %s", home)
	}
	r.add("home", StatusOK, home)

	for _, d := range []string{config.DirServers, config.DirBackups, "system"} {
		p := filepath.Join(home, d)
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			if repair {
				if err := os.MkdirAll(p, 0o755); err == nil {
					r.add("dir:"+d, StatusFixed, "recreated")
					continue
				}
			}
			r.add("dir:"+d, StatusError, "missing "+p)
		} else {
			r.add("dir:"+d, StatusOK, "")
		}
	}
	if repair {
		if err := config.InitHome(home); err != nil {
			r.add("init", StatusError, err.Error())
		}
	}

	// 2. Ownership and permissions.
	if st, err := os.Stat(home); err == nil {
		if sys, ok := st.Sys().(*syscall.Stat_t); ok {
			if int(sys.Uid) != os.Getuid() {
				r.add("ownership", StatusWarning, fmt.Sprintf(
					"home owned by uid %d but running as uid %d; run: sudo chown -R %d:%d %s",
					sys.Uid, os.Getuid(), os.Getuid(), os.Getgid(), home))
			} else {
				r.add("ownership", StatusOK, "")
			}
		}
	}
	for _, f := range []string{config.FileSecretKey} {
		p := filepath.Join(home, f)
		if st, err := os.Stat(p); err == nil {
			if st.Mode().Perm()&0o077 != 0 {
				if repair {
					if err := os.Chmod(p, 0o600); err == nil {
						r.add("perm:"+f, StatusFixed, "tightened to 0600")
						continue
					}
				}
				r.add("perm:"+f, StatusWarning, "should be 0600")
			} else {
				r.add("perm:"+f, StatusOK, "")
			}
		}
	}

	// 3. Secret key.
	keyPath := filepath.Join(home, config.FileSecretKey)
	if _, err := os.Stat(keyPath); err != nil {
		r.add("secret.key", StatusError,
			"secret.key missing — encrypted TOTP secrets are unrecoverable without it; a new key will be generated on next serve, requiring account TOTP re-enrollment")
	} else if _, err := security.LoadSecretKey(keyPath); err != nil {
		r.add("secret.key", StatusError, "secret.key unreadable or invalid: "+err.Error())
	} else {
		r.add("secret.key", StatusOK, "")
	}

	// 4. Stale runtime state.
	doctorRuntimeState(home, repair, r)

	// 5. Database.
	doctorDatabase(home, repair, r)

	// 6. Executable architecture.
	binPath := filepath.Join(home, config.DirBin, "bonghos")
	if _, err := os.Stat(binPath); err != nil {
		r.add("executable", StatusWarning, "no installed executable at "+binPath)
	} else if archOK, detail := checkBinaryArch(binPath); !archOK {
		r.add("executable", StatusError, detail)
	} else {
		r.add("executable", StatusOK, detail)
	}

	// 7. External tools.
	for _, tool := range []struct {
		name     string
		optional bool
		hint     string
	}{
		{"tmux", true, "optional console client; install with: sudo apt install tmux"},
		{"tar", false, ""},
		{"unzip", true, "needed for .zip imports via external fallback"},
		{"xz", true, "needed for .tar.xz archives"},
		{"zstd", true, "external zstd not required (built-in Go zstd is used)"},
		{"7z", true, "needed only for .7z archives"},
		{"unrar", true, "needed only for .rar archives"},
	} {
		if _, err := exec.LookPath(tool.name); err != nil {
			if tool.optional {
				r.add("tool:"+tool.name, StatusWarning, "not installed — "+tool.hint)
			} else {
				r.add("tool:"+tool.name, StatusError, "required tool missing")
			}
		} else {
			r.add("tool:"+tool.name, StatusOK, "")
		}
	}

	// 8. Java installations.
	javas := minecraft.DiscoverJava()
	if len(javas) == 0 {
		r.add("java", StatusWarning, "no Java installations found; install e.g. openjdk-21-jre-headless")
	} else {
		var names []string
		for _, j := range javas {
			names = append(names, fmt.Sprintf("%s (%s)", j.Path, j.Version))
		}
		r.add("java", StatusOK, strings.Join(names, "; "))
	}

	// 9. systemd services.
	doctorSystemd(home, repair, r)

	// 10. Server projects.
	doctorProjects(home, r)

	return r, nil
}

func doctorRuntimeState(home string, repair bool, r *Report) {
	// Supervisor state file with stale PID.
	statePath := filepath.Join(home, config.FileSupState)
	if b, err := os.ReadFile(statePath); err == nil {
		var st struct {
			ScriptPID int    `json:"script_pid"`
			JavaPID   int    `json:"java_pid"`
			State     string `json:"state"`
		}
		if json.Unmarshal(b, &st) == nil && st.ScriptPID > 0 {
			if !pidAlive(st.ScriptPID) {
				if repair {
					_ = os.Remove(statePath)
					r.add("runtime:supervisor-state", StatusFixed, "removed stale supervisor state (PID gone)")
				} else {
					r.add("runtime:supervisor-state", StatusWarning, fmt.Sprintf("stale supervisor state references dead PID %d", st.ScriptPID))
				}
			} else {
				r.add("runtime:supervisor-state", StatusOK, fmt.Sprintf("supervisor PID %d alive", st.ScriptPID))
			}
		}
	}

	// Operation lock.
	lockPath := filepath.Join(home, config.FileOpLock)
	if b, err := os.ReadFile(lockPath); err == nil {
		var lk struct {
			PID int `json:"pid"`
		}
		if json.Unmarshal(b, &lk) == nil {
			if !pidAlive(lk.PID) {
				if repair {
					_ = os.Remove(lockPath)
					r.add("runtime:operation-lock", StatusFixed, "removed stale operation lock")
				} else {
					r.add("runtime:operation-lock", StatusWarning, fmt.Sprintf("stale operation lock held by dead PID %d", lk.PID))
				}
			} else {
				r.add("runtime:operation-lock", StatusWarning, fmt.Sprintf("operation lock held by running PID %d", lk.PID))
			}
		}
	}

	// Supervisor socket.
	sockPath := filepath.Join(home, config.FileSupSocket)
	if st, err := os.Lstat(sockPath); err == nil && st.Mode()&os.ModeSocket != 0 {
		// Try to detect liveness by dialing is skipped (auth needed); rely on state file.
		stateAlive := false
		if b, err := os.ReadFile(filepath.Join(home, config.FileSupState)); err == nil {
			var s struct {
				ScriptPID int `json:"script_pid"`
			}
			if json.Unmarshal(b, &s) == nil && pidAlive(s.ScriptPID) {
				stateAlive = true
			}
		}
		if !stateAlive {
			if repair {
				_ = os.Remove(sockPath)
				r.add("runtime:socket", StatusFixed, "removed stale supervisor socket")
			} else {
				r.add("runtime:socket", StatusWarning, "supervisor socket present but no live supervisor")
			}
		} else {
			r.add("runtime:socket", StatusOK, "")
		}
	}
}

func doctorDatabase(home string, repair bool, r *Report) {
	dbPath := filepath.Join(home, config.FileDatabase)
	if _, err := os.Stat(dbPath); err != nil {
		r.add("database", StatusWarning, "database not initialized yet")
		return
	}
	db, err := database.Open(dbPath)
	if err != nil {
		r.add("database", StatusError, "cannot open database: "+err.Error())
		return
	}
	defer db.Close()

	if err := database.IntegrityCheck(db); err != nil {
		r.add("database:integrity", StatusError, err.Error())
	} else {
		r.add("database:integrity", StatusOK, "")
	}
	if repair {
		if err := database.Checkpoint(db); err != nil {
			r.add("database:wal", StatusWarning, "checkpoint failed: "+err.Error())
		} else {
			r.add("database:wal", StatusFixed, "WAL checkpointed")
		}
		if err := database.Migrate(db); err != nil {
			r.add("database:migrations", StatusError, err.Error())
		} else {
			r.add("database:migrations", StatusFixed, "migrations applied")
		}
	} else {
		v, _ := database.Version(db)
		r.add("database:version", StatusOK, fmt.Sprintf("schema version %d", v))
	}

	// Validate relative instance paths still resolve within home.
	rows, err := db.Query(`SELECT slug, server_directory FROM instances`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slug, rel string
			if rows.Scan(&slug, &rel) != nil {
				continue
			}
			abs := rel
			external := filepath.IsAbs(rel)
			if !external {
				abs = filepath.Join(home, rel)
			}
			if _, err := os.Stat(abs); err != nil {
				r.add("project:"+slug, StatusWarning, "server directory missing: "+abs)
			} else if external {
				r.add("project:"+slug, StatusWarning, "external linked project — not included in normal Bonghos migration: "+abs)
			}
		}
	}
}

func doctorSystemd(home string, repair bool, r *Report) {
	if _, err := exec.LookPath("systemctl"); err != nil {
		r.add("systemd", StatusWarning, "systemctl not found; services unavailable")
		return
	}
	out, err := exec.Command("systemctl", "--user", "is-system-running").CombinedOutput()
	state := strings.TrimSpace(string(out))
	if err != nil && state == "" {
		r.add("systemd", StatusWarning, "systemd user manager not reachable (no login session or lingering?)")
		return
	}
	r.add("systemd", StatusOK, "user manager: "+state)

	// Check unit files reference the current home.
	cfgDir, _ := os.UserHomeDir()
	unitDir := filepath.Join(cfgDir, ".config", "systemd", "user")
	for _, unit := range []string{"bonghos.service", "bonghos-minecraft.service"} {
		p := filepath.Join(unitDir, unit)
		b, err := os.ReadFile(p)
		if err != nil {
			r.add("unit:"+unit, StatusWarning, "not installed; run: bonghos service install")
			continue
		}
		if !strings.Contains(string(b), home) {
			r.add("unit:"+unit, StatusWarning, "unit file does not reference current home "+home+"; run: bonghos service repair")
		} else {
			r.add("unit:"+unit, StatusOK, "")
		}
	}
}

func doctorProjects(home string, r *Report) {
	root := filepath.Join(home, config.ServersJavaModded)
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	r.add("projects", StatusOK, fmt.Sprintf("%d modded Java project directories", count))
}

// FixPermissions restores expected modes inside BONGHOS_HOME only.
func FixPermissions(home string) (*Report, error) {
	r := &Report{Home: home, StartedAt: time.Now().UTC()}
	abs, err := filepath.Abs(home)
	if err != nil {
		return r, err
	}
	// Never operate outside home.
	fixes := []struct {
		rel  string
		mode os.FileMode
		dir  bool
	}{
		{"system/config", 0o700, true},
		{"system/data", 0o700, true},
		{"system/runtime", 0o700, true},
		{config.FileSecretKey, 0o600, false},
		{config.FileConfig, 0o600, false},
		{config.FileDatabase, 0o600, false},
	}
	for _, f := range fixes {
		p := filepath.Join(abs, f.rel)
		if !security.WithinRoot(abs, p) {
			continue
		}
		if st, err := os.Stat(p); err == nil {
			if st.Mode().Perm() != f.mode {
				if err := os.Chmod(p, f.mode); err == nil {
					r.add("perm:"+f.rel, StatusFixed, fmt.Sprintf("set %04o", f.mode))
				} else {
					r.add("perm:"+f.rel, StatusError, err.Error())
				}
			} else {
				r.add("perm:"+f.rel, StatusOK, "")
			}
		}
	}
	// Executables.
	binPath := filepath.Join(abs, config.DirBin, "bonghos")
	if st, err := os.Stat(binPath); err == nil && st.Mode().Perm()&0o100 == 0 {
		if err := os.Chmod(binPath, 0o755); err == nil {
			r.add("perm:bonghos", StatusFixed, "made executable")
		}
	}
	return r, nil
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func checkBinaryArch(path string) (bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return false, err.Error()
	}
	defer f.Close()
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return false, "unreadable executable"
	}
	if string(hdr[:4]) != "\x7fELF" {
		return false, "not an ELF executable"
	}
	machine := uint16(hdr[18]) | uint16(hdr[19])<<8
	var arch string
	switch machine {
	case 0x3e:
		arch = "amd64"
	case 0xb7:
		arch = "arm64"
	default:
		arch = fmt.Sprintf("machine 0x%x", machine)
	}
	if arch != runtime.GOARCH {
		return false, fmt.Sprintf("executable is %s but host is %s — reinstall or rebuild for this architecture", arch, runtime.GOARCH)
	}
	return true, "architecture " + arch + " matches host"
}

// ---------------------------------------------------------------------------
// Export / import
// ---------------------------------------------------------------------------

// ExportScope selects what an export contains.
type ExportScope string

const (
	ScopeConfigurationOnly ExportScope = "configuration_only"
	ScopeSystemData        ExportScope = "system_data"
	ScopeServers           ExportScope = "servers"
	ScopeBackups           ExportScope = "backups"
	ScopeComplete          ExportScope = "complete"
)

// Manifest describes a portable export archive.
type Manifest struct {
	Format             string            `json:"format"`
	FormatVersion      int               `json:"format_version"`
	BonghosVersion     string            `json:"bonghos_version"`
	CreatedAt          string            `json:"created_at"`
	SourceOS           string            `json:"source_os"`
	SourceArchitecture string            `json:"source_architecture"`
	IncludesServers    bool              `json:"includes_servers"`
	IncludesBackups    bool              `json:"includes_backups"`
	IncludesSecrets    bool              `json:"includes_secrets"`
	Checksums          map[string]string `json:"checksums,omitempty"`
}

// ExportOptions configures an export.
type ExportOptions struct {
	Scope          ExportScope
	IncludeSecrets bool
	Output         string
	Version        string
	DB             *sql.DB // optional: checkpointed before export
}

// excluded runtime paths never exported.
var exportExcludes = []string{
	"system/runtime",
	"system/temp",
	"system/logs/operations",
}

func exportIncluded(rel string, scope ExportScope, secrets bool) bool {
	for _, ex := range exportExcludes {
		if rel == ex || strings.HasPrefix(rel, ex+"/") {
			return false
		}
	}
	if !secrets && (rel == config.FileSecretKey ||
		rel == config.FileDatabase || strings.HasPrefix(rel, config.FileDatabase+"-")) {
		// Database contains accounts/encrypted secrets: only with secrets.
		return false
	}
	switch scope {
	case ScopeConfigurationOnly:
		return strings.HasPrefix(rel, "system/config")
	case ScopeSystemData:
		return strings.HasPrefix(rel, "system/")
	case ScopeServers:
		return strings.HasPrefix(rel, "servers/") || strings.HasPrefix(rel, "system/config")
	case ScopeBackups:
		return strings.HasPrefix(rel, "backups/") || strings.HasPrefix(rel, "system/config")
	default: // complete
		return true
	}
}

// Export writes a portable archive of the home.
func Export(home string, opts ExportOptions) (string, error) {
	if opts.Output == "" {
		opts.Output = fmt.Sprintf("bonghos-export-%s.tar.zst", time.Now().UTC().Format("2006-01-02"))
	}
	if opts.Scope == "" {
		opts.Scope = ScopeComplete
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return "", err
	}

	if opts.DB != nil {
		_ = database.Checkpoint(opts.DB)
	}

	manifest := Manifest{
		Format:             "bonghos-portable-export",
		FormatVersion:      1,
		BonghosVersion:     opts.Version,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SourceOS:           "linux",
		SourceArchitecture: runtime.GOARCH,
		IncludesServers:    opts.Scope == ScopeComplete || opts.Scope == ScopeServers,
		IncludesBackups:    opts.Scope == ScopeComplete || opts.Scope == ScopeBackups,
		IncludesSecrets:    opts.IncludeSecrets,
		Checksums:          map[string]string{},
	}

	tmp := opts.Output + ".partial"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer func() {
		out.Close()
		os.Remove(tmp)
	}()

	zw, err := zstd.NewWriter(out)
	if err != nil {
		return "", err
	}
	tw := tar.NewWriter(zw)

	var files []string
	err = filepath.WalkDir(absHome, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(absHome, p)
		if rerr != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			for _, ex := range exportExcludes {
				if rel == ex {
					return fs.SkipDir
				}
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			// Reject symlinks pointing outside home; skip otherwise.
			tgt, lerr := filepath.EvalSymlinks(p)
			if lerr != nil {
				return nil
			}
			if !security.WithinRoot(absHome, tgt) {
				return fmt.Errorf("unsafe symlink outside home: %s", rel)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // sockets, fifos, devices
		}
		if !exportIncluded(rel, opts.Scope, opts.IncludeSecrets) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}

	for _, rel := range files {
		p := filepath.Join(absHome, rel)
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		h := sha256.New()
		hdr := &tar.Header{
			Name:    "bonghos/" + rel,
			Mode:    int64(st.Mode().Perm()),
			Size:    st.Size(),
			ModTime: st.ModTime(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			f.Close()
			return "", err
		}
		if _, err := io.Copy(io.MultiWriter(tw, h), f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
		manifest.Checksums[rel] = hex.EncodeToString(h.Sum(nil))
	}

	mb, _ := json.MarshalIndent(manifest, "", "  ")
	if err := tw.WriteHeader(&tar.Header{
		Name: "bonghos-manifest.json", Mode: 0o600, Size: int64(len(mb)), ModTime: time.Now(),
	}); err != nil {
		return "", err
	}
	if _, err := tw.Write(mb); err != nil {
		return "", err
	}

	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := out.Sync(); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}

	// Verify: read manifest back.
	if _, err := ReadManifest(tmp); err != nil {
		return "", fmt.Errorf("verification failed: %w", err)
	}
	if err := os.Rename(tmp, opts.Output); err != nil {
		return "", err
	}
	return opts.Output, nil
}

// ReadManifest opens an export archive and returns its manifest.
func ReadManifest(archive string) (*Manifest, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == "bonghos-manifest.json" {
			b, err := io.ReadAll(io.LimitReader(tr, 32<<20))
			if err != nil {
				return nil, err
			}
			var m Manifest
			if err := json.Unmarshal(b, &m); err != nil {
				return nil, err
			}
			if m.Format != "bonghos-portable-export" {
				return nil, errors.New("not a Bonghos portable export")
			}
			return &m, nil
		}
	}
	return nil, errors.New("manifest missing from archive")
}

// Import extracts an export archive into a target home. It refuses to
// overwrite an existing live installation unless force is set, and always
// extracts through a staging directory first.
func Import(archive, targetHome string, force bool) (*Manifest, error) {
	m, err := ReadManifest(archive)
	if err != nil {
		return nil, err
	}
	if m.FormatVersion != 1 {
		return nil, fmt.Errorf("unsupported export format version %d", m.FormatVersion)
	}

	absTarget, err := filepath.Abs(targetHome)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(absTarget, config.FileDatabase)); err == nil && !force {
		return nil, errors.New("target already contains a Bonghos installation; use --force to merge over it (a safety copy is created)")
	}

	// Safety copy of existing config/database if present.
	if _, err := os.Stat(absTarget); err == nil {
		stamp := time.Now().UTC().Format("20060102-150405")
		for _, rel := range []string{config.FileDatabase, config.FileConfig} {
			src := filepath.Join(absTarget, rel)
			if _, err := os.Stat(src); err == nil {
				_ = copyFile(src, src+".pre-import-"+stamp)
			}
		}
	}

	staging, err := os.MkdirTemp(filepath.Dir(absTarget), ".bonghos-import-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)

	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == "bonghos-manifest.json" {
			continue
		}
		if !strings.HasPrefix(hdr.Name, "bonghos/") {
			return nil, fmt.Errorf("unexpected entry %q", hdr.Name)
		}
		rel := strings.TrimPrefix(hdr.Name, "bonghos/")
		if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("unsafe path %q", rel)
		}
		switch hdr.Typeflag {
		case tar.TypeReg:
			dest := filepath.Join(staging, filepath.FromSlash(rel))
			if !security.WithinRoot(staging, dest) {
				return nil, fmt.Errorf("path escape %q", rel)
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return nil, err
			}
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return nil, err
			}
			h := sha256.New()
			if _, err := io.Copy(io.MultiWriter(out, h), tr); err != nil {
				out.Close()
				return nil, err
			}
			out.Close()
			if want, ok := m.Checksums[rel]; ok && want != hex.EncodeToString(h.Sum(nil)) {
				return nil, fmt.Errorf("checksum mismatch for %s", rel)
			}
		case tar.TypeDir:
			// directories created lazily
		default:
			return nil, fmt.Errorf("unsupported entry type in export: %s", hdr.Name)
		}
	}

	// Move staged tree into target.
	if err := os.MkdirAll(absTarget, 0o755); err != nil {
		return nil, err
	}
	err = filepath.WalkDir(staging, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(staging, p)
		if rel == "." {
			return nil
		}
		dest := filepath.Join(absTarget, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		// Atomic per-file: rename within same filesystem (staging is a sibling).
		if err := os.Rename(p, dest); err != nil {
			return copyFile(p, dest)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Remove stale runtime state carried by no means, and re-init structure.
	_ = os.RemoveAll(filepath.Join(absTarget, config.DirRuntime))
	if err := config.InitHome(absTarget); err != nil {
		return nil, err
	}
	return m, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// ParseSizeMB is a small helper for CLI flags.
func ParseSizeMB(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n << 20, nil
}
