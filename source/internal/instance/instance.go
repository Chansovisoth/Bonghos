// Package instance models Bonghos server projects. The filesystem remains the
// source of truth for Minecraft files; this store holds metadata only.
package instance

import (
	"database/sql"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/config"
)

var (
	ErrSlugInvalid = errors.New("invalid project slug")
	ErrSlugTaken   = errors.New("a project with this slug already exists")
)

type Instance struct {
	ID                          int64  `json:"id"`
	Slug                        string `json:"slug"`
	DisplayName                 string `json:"display_name"`
	ServerType                  string `json:"server_type"`
	SourceType                  string `json:"source_type"`
	SourceURLHost               string `json:"source_url_host"`
	MinecraftVersion            string `json:"minecraft_version"`
	Modloader                   string `json:"modloader"`
	ModloaderVersion            string `json:"modloader_version"`
	ServerDirectory             string `json:"server_directory"` // relative unless external
	ExternalDirectory           bool   `json:"external_directory"`
	StartupScript               string `json:"startup_script"`
	JavaSelection               string `json:"java_selection"`
	JVMConfigurationSource      string `json:"jvm_configuration_source"`
	JVMXms                      string `json:"jvm_xms"`
	JVMXmx                      string `json:"jvm_xmx"`
	JVMExtraArgs                string `json:"jvm_extra_args"`
	IconRevision                int64  `json:"icon_revision"`
	AutostartEnabled            bool   `json:"autostart_enabled"`
	BootDelaySeconds            int    `json:"boot_delay_seconds"`
	RecoverAfterUncleanShutdown bool   `json:"recover_after_unclean_shutdown"`
	RestartPolicy               string `json:"restart_policy"`
	RestartDelaySeconds         int    `json:"restart_delay_seconds"`
	CreatedAt                   string `json:"created_at"`
	UpdatedAt                   string `json:"updated_at"`
	LastStartedAt               string `json:"last_started_at,omitempty"`
	LastStoppedAt               string `json:"last_stopped_at,omitempty"`
}

// AbsoluteDir returns the absolute server directory for a given home.
func (i *Instance) AbsoluteDir(home string) string {
	if i.ExternalDirectory {
		return i.ServerDirectory
	}
	return path.Join(home, i.ServerDirectory)
}

// ----- slugs -----------------------------------------------------------------

var (
	slugStrip    = regexp.MustCompile(`[^a-z0-9\s-]+`)
	slugHyphens  = regexp.MustCompile(`[\s-]+`)
	slugValidRE  = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	maxSlugChars = 64
)

// GenerateSlug derives a directory-safe slug from a display name:
// lowercase ASCII letters, digits and hyphens; whitespace becomes hyphens;
// repeated hyphens collapse; unsupported characters are removed.
func GenerateSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugStrip.ReplaceAllString(s, "")
	s = slugHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > maxSlugChars {
		s = strings.Trim(s[:maxSlugChars], "-")
	}
	return s
}

// ValidateSlug enforces the complete slug policy.
func ValidateSlug(s string) error {
	if s == "" || s == "." || s == ".." {
		return ErrSlugInvalid
	}
	if len(s) > maxSlugChars {
		return ErrSlugInvalid
	}
	if strings.ContainsAny(s, "/\\") || strings.HasPrefix(s, ".") {
		return ErrSlugInvalid
	}
	if !slugValidRE.MatchString(s) {
		return ErrSlugInvalid
	}
	return nil
}

// RelativeDirFor returns the canonical relative directory for a v1 project.
func RelativeDirFor(slug string) string {
	return path.Join(config.ServersJavaModded, slug)
}

// BackupDirFor returns the relative backup directory for a project slug.
func BackupDirFor(slug string) string {
	return path.Join(config.BackupsJavaModded, slug)
}

// ----- store -----------------------------------------------------------------

type Store struct{ DB *sql.DB }

func now() string { return time.Now().UTC().Format(time.RFC3339) }

const cols = `id, slug, display_name, server_type, source_type, source_url_host,
 minecraft_version, modloader, modloader_version, server_directory, external_directory,
 startup_script, java_selection, jvm_configuration_source, jvm_xms, jvm_xmx, jvm_extra_args,
 icon_revision, autostart_enabled, boot_delay_seconds, recover_after_unclean_shutdown,
 restart_policy, restart_delay_seconds, created_at, updated_at,
 COALESCE(last_started_at,''), COALESCE(last_stopped_at,'')`

func scan(row interface{ Scan(...any) error }) (*Instance, error) {
	var i Instance
	var ext, auto, recover int
	err := row.Scan(&i.ID, &i.Slug, &i.DisplayName, &i.ServerType, &i.SourceType,
		&i.SourceURLHost, &i.MinecraftVersion, &i.Modloader, &i.ModloaderVersion,
		&i.ServerDirectory, &ext, &i.StartupScript, &i.JavaSelection,
		&i.JVMConfigurationSource, &i.JVMXms, &i.JVMXmx, &i.JVMExtraArgs,
		&i.IconRevision, &auto, &i.BootDelaySeconds, &recover,
		&i.RestartPolicy, &i.RestartDelaySeconds, &i.CreatedAt, &i.UpdatedAt,
		&i.LastStartedAt, &i.LastStoppedAt)
	if err != nil {
		return nil, err
	}
	i.ExternalDirectory = ext != 0
	i.AutostartEnabled = auto != 0
	i.RecoverAfterUncleanShutdown = recover != 0
	return &i, nil
}

func (s *Store) Create(i *Instance) error {
	if err := ValidateSlug(i.Slug); err != nil {
		return err
	}
	if strings.TrimSpace(i.DisplayName) == "" {
		return errors.New("display name is required")
	}
	if i.ServerType == "" {
		i.ServerType = "minecraft-java-modded"
	}
	if i.RestartPolicy == "" {
		i.RestartPolicy = "on-failure"
	}
	if i.JavaSelection == "" {
		i.JavaSelection = "auto"
	}
	if i.BootDelaySeconds == 0 {
		i.BootDelaySeconds = 30
	}
	if i.RestartDelaySeconds == 0 {
		i.RestartDelaySeconds = 10
	}
	// Recommended defaults for a new project: autostart off, 30s boot delay,
	// recovery after an unclean shutdown on. Callers that want recovery
	// disabled turn it off through Update once the project exists.
	i.RecoverAfterUncleanShutdown = true
	res, err := s.DB.Exec(`INSERT INTO instances
		(slug, display_name, server_type, source_type, source_url_host, minecraft_version,
		 modloader, modloader_version, server_directory, external_directory, startup_script,
		 java_selection, jvm_configuration_source, jvm_xms, jvm_xmx, jvm_extra_args,
		 autostart_enabled, boot_delay_seconds, recover_after_unclean_shutdown,
		 restart_policy, restart_delay_seconds, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		i.Slug, i.DisplayName, i.ServerType, i.SourceType, i.SourceURLHost,
		i.MinecraftVersion, i.Modloader, i.ModloaderVersion, i.ServerDirectory,
		b2i(i.ExternalDirectory), i.StartupScript, i.JavaSelection,
		i.JVMConfigurationSource, i.JVMXms, i.JVMXmx, i.JVMExtraArgs,
		b2i(i.AutostartEnabled), i.BootDelaySeconds, b2i(i.RecoverAfterUncleanShutdown),
		i.RestartPolicy, i.RestartDelaySeconds, now(), now())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return ErrSlugTaken
		}
		return err
	}
	i.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) ByID(id int64) (*Instance, error) {
	return scan(s.DB.QueryRow(`SELECT `+cols+` FROM instances WHERE id=?`, id))
}

func (s *Store) BySlug(serverType, slug string) (*Instance, error) {
	return scan(s.DB.QueryRow(`SELECT `+cols+` FROM instances WHERE server_type=? AND slug=?`, serverType, slug))
}

func (s *Store) List() ([]*Instance, error) {
	rows, err := s.DB.Query(`SELECT ` + cols + ` FROM instances ORDER BY display_name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Instance
	for rows.Next() {
		i, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// Update persists mutable metadata fields.
func (s *Store) Update(i *Instance) error {
	_, err := s.DB.Exec(`UPDATE instances SET display_name=?, minecraft_version=?, modloader=?,
		modloader_version=?, startup_script=?, java_selection=?, jvm_configuration_source=?,
		jvm_xms=?, jvm_xmx=?, jvm_extra_args=?, icon_revision=?, autostart_enabled=?,
		boot_delay_seconds=?, recover_after_unclean_shutdown=?, restart_policy=?,
		restart_delay_seconds=?, updated_at=? WHERE id=?`,
		i.DisplayName, i.MinecraftVersion, i.Modloader, i.ModloaderVersion,
		i.StartupScript, i.JavaSelection, i.JVMConfigurationSource,
		i.JVMXms, i.JVMXmx, i.JVMExtraArgs, i.IconRevision,
		b2i(i.AutostartEnabled), i.BootDelaySeconds, b2i(i.RecoverAfterUncleanShutdown),
		i.RestartPolicy, i.RestartDelaySeconds, now(), i.ID)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM instances WHERE id=?`, id)
	return err
}

func (s *Store) TouchStarted(id int64) {
	s.DB.Exec(`UPDATE instances SET last_started_at=? WHERE id=?`, now(), id)
}

func (s *Store) TouchStopped(id int64) {
	s.DB.Exec(`UPDATE instances SET last_stopped_at=? WHERE id=?`, now(), id)
}

// ActiveID returns the currently selected instance id (0 when none).
func (s *Store) ActiveID() (int64, error) {
	var v sql.NullInt64
	err := s.DB.QueryRow(`SELECT instance_id FROM active_instance WHERE id=1`).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return v.Int64, nil
}

func (s *Store) SetActive(id int64) error {
	if id == 0 {
		_, err := s.DB.Exec(`UPDATE active_instance SET instance_id=NULL WHERE id=1`)
		return err
	}
	if _, err := s.ByID(id); err != nil {
		return fmt.Errorf("instance %d not found", id)
	}
	_, err := s.DB.Exec(`UPDATE active_instance SET instance_id=? WHERE id=1`, id)
	return err
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
