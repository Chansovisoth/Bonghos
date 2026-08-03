// Package backup implements the Bonghos backup subsystem: portable tar.zst
// archives created via staging, verified with checksums, retained safely and
// restored through staging — never directly over a running server.
package backup

import (
	"archive/tar"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/security"
)

type Type string

const (
	TypeFull   Type = "full_server"
	TypeWorld  Type = "world_and_player_data"
	TypeConfig Type = "configuration_only"
)

type Record struct {
	ID                 int64  `json:"id"`
	BackupID           string `json:"backup_id"`
	InstanceID         int64  `json:"instance_id"`
	InstanceSlug       string `json:"instance_slug"`
	DisplayName        string `json:"display_name"`
	BackupType         string `json:"backup_type"`
	ConsistencyMode    string `json:"consistency_mode"`
	TriggerType        string `json:"trigger_type"`
	TriggeredBy        int64  `json:"triggered_by,omitempty"`
	ArchivePath        string `json:"archive_path"`
	ArchiveFormat      string `json:"archive_format"`
	CompressedSize     int64  `json:"compressed_size"`
	UncompressedSize   int64  `json:"uncompressed_size"`
	FileCount          int64  `json:"file_count"`
	ChecksumAlgorithm  string `json:"checksum_algorithm"`
	Checksum           string `json:"checksum"`
	VerificationStatus string `json:"verification_status"`
	Protected          bool   `json:"protected"`
	CreatedAt          string `json:"created_at"`
	CompletedAt        string `json:"completed_at,omitempty"`
}

// Manager coordinates backup operations for a Bonghos home.
type Manager struct {
	Home string
	DB   *sql.DB
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// selectPaths chooses paths to include for the backup type. Include/exclude
// patterns extend the defaults; nothing assumes an identical modpack layout.
func selectPaths(serverDir string, t Type, includes, excludes []string) ([]string, error) {
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return nil, err
	}
	excluded := func(name string) bool {
		defaults := []string{"logs", "crash-reports", ".bonghos-tmp", "cache", "libraries/.cache"}
		for _, e := range append(defaults, excludes...) {
			if matched, _ := filepath.Match(e, name); matched || e == name {
				return true
			}
		}
		return false
	}
	var out []string
	switch t {
	case TypeFull:
		for _, e := range entries {
			if excluded(e.Name()) {
				continue
			}
			out = append(out, e.Name())
		}
	case TypeWorld:
		world := minecraft.WorldDir(serverDir)
		candidates := append([]string{world, "server.properties",
			"whitelist.json", "ops.json", "banned-players.json", "banned-ips.json",
			"usercache.json"}, includes...)
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(serverDir, c)); err == nil && !excluded(c) {
				out = append(out, c)
			}
		}
	case TypeConfig:
		candidates := append([]string{
			"server.properties", "eula.txt", "user_jvm_args.txt",
			"run.sh", "start.sh", "startserver.sh", "server-start.sh", "start-server.sh",
			"config", "defaultconfigs", "kubejs", "scripts",
			"whitelist.json", "ops.json", "banned-players.json", "banned-ips.json",
			"server-icon.png", "variables.txt",
		}, includes...)
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(serverDir, c)); err == nil && !excluded(c) {
				out = append(out, c)
			}
		}
	default:
		return nil, fmt.Errorf("unknown backup type %q", t)
	}
	if len(out) == 0 {
		return nil, errors.New("nothing to back up for this type")
	}
	sort.Strings(out)
	return out, nil
}

// Create builds a verified backup for inst. It stages the archive under
// system/temp/staging, verifies it, then moves it into final storage
// atomically. Pruning runs only after success.
func (m *Manager) Create(inst *instance.Instance, t Type, mode, trigger string,
	triggeredBy int64, includes, excludes []string,
	progress func(stage string, done, total int64)) (*Record, error) {

	serverDir := inst.AbsoluteDir(m.Home)
	if _, err := os.Stat(serverDir); err != nil {
		return nil, fmt.Errorf("server directory missing: %w", err)
	}
	paths, err := selectPaths(serverDir, t, includes, excludes)
	if err != nil {
		return nil, err
	}
	backupID, err := security.RandomToken(8)
	if err != nil {
		return nil, err
	}
	stamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	shortType := map[Type]string{TypeFull: "full", TypeWorld: "world", TypeConfig: "config"}[t]
	relDir := instance.BackupDirFor(inst.Slug)

	// The timestamp only has one-second resolution, so two backups of the same
	// type started in the same second would collide. That is not hypothetical:
	// a restore takes an emergency safety copy immediately before restoring,
	// and an overwritten archive silently invalidates the earlier backup's
	// checksum. Append the backup ID whenever the name is already taken.
	archiveName := fmt.Sprintf("%s_%s.tar.zst", stamp, shortType)
	relPath := filepath.Join(relDir, archiveName)
	finalPath := filepath.Join(m.Home, relPath)
	if _, err := os.Lstat(finalPath); err == nil {
		archiveName = fmt.Sprintf("%s_%s_%s.tar.zst", stamp, shortType, backupID)
		relPath = filepath.Join(relDir, archiveName)
		finalPath = filepath.Join(m.Home, relPath)
	}
	stagingDir := filepath.Join(m.Home, config.DirStaging, "backup-"+backupID)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(stagingDir)
	stagingArchive := filepath.Join(stagingDir, archiveName)
	if _, err := os.Lstat(finalPath); err == nil {
		return nil, fmt.Errorf("refusing to overwrite an existing backup archive at %s", relPath)
	}

	rec := &Record{
		BackupID: backupID, InstanceID: inst.ID, InstanceSlug: inst.Slug,
		DisplayName: inst.DisplayName, BackupType: string(t), ConsistencyMode: mode,
		TriggerType: trigger, TriggeredBy: triggeredBy,
		ArchivePath: filepath.ToSlash(relPath), ArchiveFormat: "tar.zst",
		ChecksumAlgorithm: "sha256", VerificationStatus: "unverified",
		CreatedAt: now(),
	}
	if _, err := m.DB.Exec(`INSERT INTO backups (backup_id, instance_id, instance_slug,
		display_name, backup_type, consistency_mode, trigger_type, triggered_by,
		archive_path, archive_format, checksum_algorithm, verification_status,
		minecraft_version, modloader, modloader_version, included_paths, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rec.BackupID, rec.InstanceID, rec.InstanceSlug, rec.DisplayName, rec.BackupType,
		rec.ConsistencyMode, rec.TriggerType, nullID(rec.TriggeredBy),
		rec.ArchivePath, rec.ArchiveFormat, "sha256", "unverified",
		inst.MinecraftVersion, inst.Modloader, inst.ModloaderVersion,
		jsonList(paths), rec.CreatedAt); err != nil {
		return nil, err
	}

	// create the archive in staging
	uncompressed, count, err := writeTarZst(stagingArchive, serverDir, paths, progress)
	if err != nil {
		m.DB.Exec(`UPDATE backups SET verification_status='failed' WHERE backup_id=?`, backupID)
		return nil, err
	}
	// checksum + verification
	sum, csize, err := fileChecksum(stagingArchive)
	if err != nil {
		return nil, err
	}
	if progress != nil {
		progress("verifying", 0, 0)
	}
	vCount, vBytes, err := verifyArchive(stagingArchive)
	if err != nil || vCount != count || vBytes != uncompressed {
		m.DB.Exec(`UPDATE backups SET verification_status='failed' WHERE backup_id=?`, backupID)
		if err == nil {
			err = fmt.Errorf("verification mismatch (files %d/%d, bytes %d/%d)", vCount, count, vBytes, uncompressed)
		}
		return nil, err
	}
	// move atomically into final storage
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(stagingArchive, finalPath); err != nil {
		// cross-device fallback
		if cerr := copyFile(stagingArchive, finalPath); cerr != nil {
			return nil, cerr
		}
		os.Remove(stagingArchive)
	}
	rec.CompressedSize = csize
	rec.UncompressedSize = uncompressed
	rec.FileCount = count
	rec.Checksum = sum
	rec.VerificationStatus = "verified"
	rec.CompletedAt = now()
	m.DB.Exec(`UPDATE backups SET compressed_size=?, uncompressed_size=?, file_count=?,
		checksum=?, verification_status='verified', completed_at=? WHERE backup_id=?`,
		csize, uncompressed, count, sum, rec.CompletedAt, backupID)

	// sidecar metadata JSON beside the archive
	meta, _ := json.MarshalIndent(rec, "", "  ")
	metaDir := filepath.Join(m.Home, relDir, "metadata")
	os.MkdirAll(metaDir, 0o755)
	os.WriteFile(filepath.Join(metaDir, archiveName+".json"), meta, 0o644)
	return rec, nil
}

func writeTarZst(dest, baseDir string, paths []string,
	progress func(string, int64, int64)) (uncompressed int64, count int64, err error) {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	zw, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return 0, 0, err
	}
	tw := tar.NewWriter(zw)
	for _, p := range paths {
		full := filepath.Join(baseDir, p)
		err := filepath.Walk(full, func(fp string, info os.FileInfo, werr error) error {
			if werr != nil {
				return werr
			}
			rel, err := filepath.Rel(baseDir, fp)
			if err != nil {
				return err
			}
			var link string
			if info.Mode()&os.ModeSymlink != 0 {
				link, err = os.Readlink(fp)
				if err != nil {
					return err
				}
				// reject symlinks escaping the server directory
				resolved := link
				if !filepath.IsAbs(resolved) {
					resolved = filepath.Join(filepath.Dir(fp), link)
				}
				if !strings.HasPrefix(filepath.Clean(resolved)+string(filepath.Separator),
					filepath.Clean(baseDir)+string(filepath.Separator)) {
					return nil // skip unsafe links rather than embedding them
				}
			}
			hdr, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}
			hdr.Name = filepath.ToSlash(rel)
			if info.IsDir() {
				hdr.Name += "/"
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				in, err := os.Open(fp)
				if err != nil {
					return err
				}
				n, err := io.Copy(tw, in)
				in.Close()
				uncompressed += n
				count++
				if progress != nil {
					progress("archiving", uncompressed, 0)
				}
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			tw.Close()
			zw.Close()
			return 0, 0, err
		}
	}
	if err := tw.Close(); err != nil {
		return 0, 0, err
	}
	if err := zw.Close(); err != nil {
		return 0, 0, err
	}
	return uncompressed, count, f.Sync()
}

// verifyArchive opens the archive and validates the complete file listing.
func verifyArchive(path string) (count int64, bytes int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	zr, err := zstd.NewReader(f)
	if err != nil {
		return 0, 0, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return count, bytes, nil
		}
		if err != nil {
			return count, bytes, err
		}
		if hdr.Typeflag == tar.TypeReg {
			n, err := io.Copy(io.Discard, tr)
			if err != nil {
				return count, bytes, err
			}
			bytes += n
			count++
		}
	}
}

func fileChecksum(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// Verify re-checks an existing backup's checksum and archive readability.
func (m *Manager) Verify(backupID string) error {
	rec, err := m.Get(backupID)
	if err != nil {
		return err
	}
	full := filepath.Join(m.Home, rec.ArchivePath)
	sum, size, err := fileChecksum(full)
	if err == nil && sum != rec.Checksum {
		err = fmt.Errorf("checksum mismatch: stored %s, computed %s", rec.Checksum[:12], sum[:12])
	}
	if err == nil {
		var count int64
		count, _, err = verifyArchive(full)
		if err == nil && count != rec.FileCount {
			err = fmt.Errorf("file count mismatch: stored %d, found %d", rec.FileCount, count)
		}
		_ = size
	}
	status := "verified"
	if err != nil {
		status = "failed"
	}
	m.DB.Exec(`UPDATE backups SET verification_status=? WHERE backup_id=?`, status, backupID)
	return err
}

// worldNames returns every directory name that could hold the world being
// restored. The archive's own server.properties is authoritative, because
// level-name may have been changed since the backup was taken: reading only
// the destination would look for the new name and silently restore nothing.
// The destination name is included too, so a rename in either direction still
// replaces the live world rather than leaving it beside a stale copy.
func worldNames(stagingDir, destServerDir string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	add(minecraft.WorldDir(stagingDir)) // what the archive calls it
	add(minecraft.WorldDir(destServerDir))
	add("world") // the Minecraft default, in case neither file was readable
	return out
}

// scopeWants reports whether a top-level entry belongs to the requested
// restore scope. worlds lists the acceptable world directory names.
func scopeWants(scope, name string, worlds []string) bool {
	switch scope {
	case ScopeWorld:
		// The world directory itself plus the sibling dimension folders
		// (world_nether, world_the_end) and the player data a world is
		// meaningless without.
		for _, w := range worlds {
			if name == w || strings.HasPrefix(name, w+"_") {
				return true
			}
		}
		switch name {
		case "playerdata", "stats", "advancements", "usercache.json":
			return true
		}
		return false
	case ScopeConfig:
		// Everything a world is not: settings, mod configuration and scripts.
		switch name {
		case "server.properties", "eula.txt", "ops.json", "whitelist.json",
			"banned-players.json", "banned-ips.json", "server-icon.png",
			"user_jvm_args.txt", "config", "defaultconfigs", "kubejs", "scripts":
			return true
		}
		// Startup scripts are configuration too.
		return strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".txt") ||
			strings.HasSuffix(name, ".cfg") || strings.HasSuffix(name, ".toml")
	default: // ScopeFull
		return true
	}
}

// Restore scopes. NormalizeScope accepts the shorthand the UI and CLI use as
// well as the canonical names, so a caller sending "world" cannot silently fall
// through to a full-server restore.
const (
	ScopeFull   = "full_server"
	ScopeWorld  = "world_only"
	ScopeConfig = "configuration_only"
)

// NormalizeScope maps accepted aliases onto canonical scope names. An empty
// scope defaults to a full restore. Unknown values are rejected rather than
// silently treated as "full_server", because guessing wrong here overwrites a
// live world.
func NormalizeScope(scope string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "full", "full_server", "all":
		return ScopeFull, nil
	case "world", "world_only", "world_and_player_data":
		return ScopeWorld, nil
	case "config", "configuration", "configuration_only":
		return ScopeConfig, nil
	default:
		return "", fmt.Errorf("unknown restore scope %q (use full_server, world_only or configuration_only)", scope)
	}
}

// RestoreResult describes what a restore actually did, so callers can report
// it rather than guessing.
type RestoreResult struct {
	Scope        string `json:"scope"`
	EntriesTaken int    `json:"entries_restored"`
	// WorldName is the world directory the server will use after the restore.
	WorldName string `json:"world_name,omitempty"`
	// LevelNameUpdated is true when server.properties was repointed at the
	// restored world because the live value named a different one.
	LevelNameUpdated bool   `json:"level_name_updated,omitempty"`
	PreviousLevel    string `json:"previous_level_name,omitempty"`
}

// Restore extracts a verified backup into staging, validates paths, then
// replaces the target. The caller guarantees the server is stopped and has
// created an emergency pre-restore backup unless explicitly disabled.
func (m *Manager) Restore(rec *Record, destServerDir string, scope string) (*RestoreResult, error) {
	scope, err := NormalizeScope(scope)
	if err != nil {
		return nil, err
	}
	full := filepath.Join(m.Home, rec.ArchivePath)
	if err := m.Verify(rec.BackupID); err != nil {
		return nil, fmt.Errorf("backup failed verification, refusing restore: %w", err)
	}
	stagingDir := filepath.Join(m.Home, config.DirStaging, "restore-"+rec.BackupID+"-"+fmt.Sprint(time.Now().Unix()))
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(stagingDir)
	if err := extractTarZstSafe(full, stagingDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destServerDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(stagingDir)
	if err != nil {
		return nil, err
	}
	worlds := worldNames(stagingDir, destServerDir)
	restored := 0
	for _, e := range entries {
		if !scopeWants(scope, e.Name(), worlds) {
			continue
		}
		restored++
		src := filepath.Join(stagingDir, e.Name())
		dst := filepath.Join(destServerDir, e.Name())
		// replace atomically where possible: move old aside, then rename in
		old := dst + ".bonghos-pre-restore"
		os.RemoveAll(old)
		if _, err := os.Lstat(dst); err == nil {
			if err := os.Rename(dst, old); err != nil {
				return nil, err
			}
		}
		if err := os.Rename(src, dst); err != nil {
			os.Rename(old, dst) // roll back this entry
			return nil, err
		}
		// The replacement succeeded, so the displaced copy is no longer needed.
		// The durable safety net is the emergency pre-restore backup, not this
		// sidecar directory.
		os.RemoveAll(old)
	}
	// A scoped restore that matched nothing has quietly done nothing at all,
	// which looks identical to success from the outside. Say so instead.
	if restored == 0 {
		return nil, fmt.Errorf("nothing in this backup matched the %s scope; no files were changed", scope)
	}

	res := &RestoreResult{Scope: scope, EntriesTaken: restored}

	// Restoring a world under its archived name is only half the job: the
	// server boots whatever level-name says. If the live value names a
	// different world, the restored one would sit on disk unused and the
	// operator would see an untouched world after a "successful" restore.
	// A full restore already replaces server.properties from the archive, so
	// this only applies to a world-only restore.
	if scope == ScopeWorld {
		archiveWorld := minecraft.WorldDir(stagingDir)
		current := minecraft.WorldDir(destServerDir)
		res.WorldName = archiveWorld
		if archiveWorld != current {
			if err := minecraft.WriteProperty(destServerDir, "level-name", archiveWorld); err != nil {
				return res, fmt.Errorf("restored world %q but could not point level-name at it (still %q): %w",
					archiveWorld, current, err)
			}
			res.LevelNameUpdated = true
			res.PreviousLevel = current
		}
	}
	return res, nil
}

// extractTarZstSafe unpacks with the same traversal protections as imports.
func extractTarZstSafe(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := zstd.NewReader(f)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(strings.ReplaceAll(hdr.Name, "\\", "/"))
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("unsafe path in backup archive: %s", hdr.Name)
		}
		out := filepath.Join(dest, name)
		rel, err := filepath.Rel(dest, out)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unsafe path in backup archive: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if filepath.IsAbs(hdr.Linkname) {
				continue
			}
			resolved := filepath.Join(filepath.Dir(out), hdr.Linkname)
			if r, err := filepath.Rel(dest, filepath.Clean(resolved)); err != nil || strings.HasPrefix(r, "..") {
				continue
			}
			os.MkdirAll(filepath.Dir(out), 0o755)
			os.Remove(out)
			os.Symlink(hdr.Linkname, out)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			w, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode).Perm()|0o200)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, tr); err != nil {
				w.Close()
				return err
			}
			w.Close()
		}
	}
}

// ----- retention -------------------------------------------------------------

type RetentionPolicy struct {
	MaxCount     int
	MaxAgeDays   int
	MaxStorageMB int64
}

// Prune applies the retention policy: never the last valid backup, never
// protected backups, only verified deletions, audited by the caller.
// Returns the deleted backup IDs.
func (m *Manager) Prune(instanceID int64, p RetentionPolicy) ([]string, error) {
	recs, err := m.List(instanceID)
	if err != nil {
		return nil, err
	}
	// oldest first
	sort.Slice(recs, func(i, j int) bool { return recs[i].CreatedAt < recs[j].CreatedAt })
	validCount := 0
	for _, r := range recs {
		if r.VerificationStatus == "verified" {
			validCount++
		}
	}
	deletable := func(r *Record) bool {
		if r.Protected || r.VerificationStatus != "verified" {
			return false
		}
		return validCount > 1 // never delete the last valid backup
	}
	var deleted []string
	var totalBytes int64
	for _, r := range recs {
		totalBytes += r.CompressedSize
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -p.MaxAgeDays)
	remaining := len(recs)
	for _, r := range recs {
		reason := ""
		switch {
		case p.MaxAgeDays > 0 && r.CreatedAt < cutoff.Format(time.RFC3339):
			reason = "age"
		case p.MaxCount > 0 && remaining > p.MaxCount:
			reason = "count"
		case p.MaxStorageMB > 0 && totalBytes > p.MaxStorageMB<<20:
			reason = "storage"
		}
		if reason == "" || !deletable(r) {
			continue
		}
		if err := m.Delete(r.BackupID); err != nil {
			continue
		}
		deleted = append(deleted, r.BackupID)
		validCount--
		remaining--
		totalBytes -= r.CompressedSize
	}
	return deleted, nil
}

// Delete removes an archive (path-validated) and its metadata.
func (m *Manager) Delete(backupID string) error {
	rec, err := m.Get(backupID)
	if err != nil {
		return err
	}
	if rec.Protected {
		return errors.New("backup is protected")
	}
	full, err := security.SafeJoin(m.Home, rec.ArchivePath)
	if err != nil {
		return err
	}
	if !strings.Contains(rec.ArchivePath, "backups/") {
		return errors.New("archive path outside backup storage")
	}
	os.Remove(full)
	os.Remove(filepath.Join(filepath.Dir(full), "metadata", filepath.Base(full)+".json"))
	_, err = m.DB.Exec(`DELETE FROM backups WHERE backup_id=?`, backupID)
	return err
}

func (m *Manager) Get(backupID string) (*Record, error) {
	return scanRec(m.DB.QueryRow(`SELECT `+recCols+` FROM backups WHERE backup_id=?`, backupID))
}

const recCols = `id, backup_id, COALESCE(instance_id,0), instance_slug, display_name,
 backup_type, consistency_mode, trigger_type, COALESCE(triggered_by,0), archive_path,
 archive_format, compressed_size, uncompressed_size, file_count, checksum_algorithm,
 checksum, verification_status, protected, created_at, COALESCE(completed_at,'')`

func scanRec(row interface{ Scan(...any) error }) (*Record, error) {
	var r Record
	var prot int
	err := row.Scan(&r.ID, &r.BackupID, &r.InstanceID, &r.InstanceSlug, &r.DisplayName,
		&r.BackupType, &r.ConsistencyMode, &r.TriggerType, &r.TriggeredBy, &r.ArchivePath,
		&r.ArchiveFormat, &r.CompressedSize, &r.UncompressedSize, &r.FileCount,
		&r.ChecksumAlgorithm, &r.Checksum, &r.VerificationStatus, &prot,
		&r.CreatedAt, &r.CompletedAt)
	if err != nil {
		return nil, err
	}
	r.Protected = prot != 0
	return &r, nil
}

func (m *Manager) List(instanceID int64) ([]*Record, error) {
	q := `SELECT ` + recCols + ` FROM backups`
	var args []any
	if instanceID > 0 {
		q += ` WHERE instance_id=?`
		args = append(args, instanceID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := m.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Record
	for rows.Next() {
		r, err := scanRec(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (m *Manager) SetProtected(backupID string, protected bool) error {
	_, err := m.DB.Exec(`UPDATE backups SET protected=? WHERE backup_id=?`, b2i(protected), backupID)
	return err
}

func jsonList(v []string) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
