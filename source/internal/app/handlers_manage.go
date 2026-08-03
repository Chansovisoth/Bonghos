package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/files"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/monitoring"
	"github.com/Chansovisoth/Bonghos/internal/scheduler"
)

// maxTextEditBytes caps in-browser text editing. Larger files are still
// listed and downloadable, they just cannot be opened in the editor.
const maxTextEditBytes = 2 << 20 // 2 MiB

// activeFiles returns a path-jailed file manager for the active project.
func (a *App) activeFiles() (*files.Manager, *instance.Instance, error) {
	inst, err := a.activeInstance()
	if err != nil {
		return nil, nil, err
	}
	// MaxEditBytes keeps the text editor away from huge or binary files;
	// browsing, download and upload are unaffected.
	return &files.Manager{
		Root:         inst.AbsoluteDir(a.Home),
		MaxEditBytes: maxTextEditBytes,
	}, inst, nil
}

// ---------------------------------------------------------------------------
// files
// ---------------------------------------------------------------------------

func (a *App) handleFileList(w http.ResponseWriter, r *http.Request) {
	fm, _, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	if q := r.URL.Query().Get("q"); q != "" {
		entries, err := fm.Search(q)
		if err != nil {
			writeErr(w, 400, err)
			return
		}
		writeJSON(w, 200, entries)
		return
	}
	entries, err := fm.List(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, entries)
}

func (a *App) handleFileRead(w http.ResponseWriter, r *http.Request) {
	fm, _, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	content, err := fm.ReadText(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]string{"content": content})
}

func (a *App) handleFileWrite(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	fm, inst, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := readJSON(r, &req, 16<<20); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if err := fm.WriteText(req.Path, req.Content); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "file_edited", inst.Slug, req.Path, remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleFileMkdir(w http.ResponseWriter, r *http.Request) {
	fm, _, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if err := fm.Mkdir(req.Path); err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleFileRename(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	fm, inst, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if err := fm.Rename(req.From, req.To); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "file_renamed", inst.Slug, req.From+" -> "+req.To, remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	fm, inst, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Path    string `json:"path"`
		Confirm bool   `json:"confirm"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil || !req.Confirm {
		writeErr(w, 400, errors.New("deletion requires confirmation"))
		return
	}
	if err := fm.Delete(req.Path); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "file_deleted", inst.Slug, req.Path, remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	fm, inst, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, a.Cfg.MaxUploadBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		writeErr(w, 400, errors.New("multipart upload expected"))
		return
	}
	dir := r.URL.Query().Get("path")
	var saved []string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		if part.FormName() != "file" || part.FileName() == "" {
			continue
		}
		name := filepath.Base(part.FileName())
		rel := filepath.ToSlash(filepath.Join(dir, name))
		if _, err := fm.SaveUpload(rel, part); err != nil {
			writeErr(w, 400, fmt.Errorf("saving %s: %w", name, err))
			return
		}
		saved = append(saved, rel)
	}
	a.audit(u.ID, u.Username, "files_uploaded", inst.Slug, strings.Join(saved, ", "), remoteIP(r))
	writeJSON(w, 200, map[string]any{"saved": saved})
}

func (a *App) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	fm, _, err := a.activeFiles()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	f, st, err := fm.Open(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", filepath.Base(st.Name())))
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

// ---------------------------------------------------------------------------
// configuration
// ---------------------------------------------------------------------------

func (a *App) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	dir := inst.AbsoluteDir(a.Home)
	props, _ := minecraft.ReadProperties(dir)
	var jvm *minecraft.JVMConfig
	if inst.StartupScript != "" {
		jvm, _ = minecraft.DetectJVMConfig(dir, inst.StartupScript)
	}
	scripts, _ := minecraft.DetectStartupScripts(dir, 3)
	eula := false
	if fm := (&files.Manager{Root: dir}); fm != nil {
		if s, err := fm.ReadText("eula.txt"); err == nil {
			eula = containsEulaTrue(s)
		}
	}
	writeJSON(w, 200, map[string]any{
		"instance":   inst,
		"properties": props,
		"jvm":        jvm,
		"scripts":    scripts,
		"java":       minecraft.DiscoverJava(),
		"eula":       eula,
	})
}

func (a *App) handleConfigJVM(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Xms string `json:"xms"`
		Xmx string `json:"xmx"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	memT, _ := monitoring.HostMemory()
	if err := minecraft.ValidateMemory(req.Xms, req.Xmx, memT); err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := a.applyJVMMemory(inst, req.Xms, req.Xmx); err != nil {
		writeErr(w, 400, err)
		return
	}
	inst.JVMXms, inst.JVMXmx = req.Xms, req.Xmx
	_ = a.Instances.Update(inst)
	a.audit(u.ID, u.Username, "jvm_updated", inst.Slug, fmt.Sprintf("Xms=%s Xmx=%s", req.Xms, req.Xmx), remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true, "restart_required": true})
}

func (a *App) handleConfigStartup(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Script string `json:"script"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	rel := filepath.ToSlash(filepath.Clean(req.Script))
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		writeErr(w, 400, errors.New("startup script must be inside the project"))
		return
	}
	inst.StartupScript = rel
	if err := a.Instances.Update(inst); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "startup_script_selected", inst.Slug, rel, remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleConfigProperty(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if err := minecraft.WriteProperty(inst.AbsoluteDir(a.Home), req.Key, req.Value); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "property_updated", inst.Slug, req.Key, remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true, "restart_required": true})
}

func (a *App) handleConfigEULA(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Accept bool `json:"accept"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil || !req.Accept {
		writeErr(w, 400, errors.New("EULA acceptance must be explicit"))
		return
	}
	fm := &files.Manager{Root: inst.AbsoluteDir(a.Home)}
	content := "# Accepted via Bonghos by " + u.Username + " at " + time.Now().UTC().Format(time.RFC3339) + "\neula=true\n"
	if err := fm.WriteText("eula.txt", content); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "eula_accepted", inst.Slug, "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleJavaList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, minecraft.DiscoverJava())
}

// ---------------------------------------------------------------------------
// backups
// ---------------------------------------------------------------------------

func (a *App) handleBackupList(w http.ResponseWriter, r *http.Request) {
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	list, err := a.Backups.List(inst.ID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, list)
}

// runBackup executes the online/offline backup workflow.
func (a *App) runBackup(ctx context.Context, inst *instance.Instance, t backup.Type, mode, trigger string, by int64) (*backup.Record, error) {
	release, err := a.OpLock.Acquire("backup")
	if err != nil {
		return nil, err
	}
	defer release()
	return a.runBackupLocked(ctx, inst, t, mode, trigger, by)
}

// runBackupLocked performs a backup assuming the caller already holds the
// operation lock. Restore uses this for its emergency pre-restore copy, which
// happens inside the restore lock and must not try to take a second one.
func (a *App) runBackupLocked(ctx context.Context, inst *instance.Instance, t backup.Type, mode, trigger string, by int64) (*backup.Record, error) {
	online := mode == "online" && a.Runner.Online()
	if online {
		_ = a.Runner.SendCommand("say Backup starting…")
		_ = a.Runner.SendCommand("save-off")
		_ = a.Runner.SendCommand("save-all flush")
		time.Sleep(3 * time.Second) // allow flush to settle
		defer func() {
			_ = a.Runner.SendCommand("save-on")
			_ = a.Runner.SendCommand("say Backup finished.")
		}()
	}

	rec, err := a.Backups.Create(inst, t, mode, trigger, by, nil, nil,
		func(stage string, done, total int64) {
			a.Hub.Broadcast("backups", "progress", map[string]any{
				"stage": stage, "bytes_processed": done, "total_bytes": total,
			})
		})
	if err != nil {
		return nil, err
	}
	a.Hub.Broadcast("backups", "created", rec)
	return rec, nil
}

func (a *App) handleBackupCreate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var req struct {
		Type string `json:"type"` // world | full | configuration
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	var t backup.Type
	switch req.Type {
	case "world", string(backup.TypeWorld):
		t = backup.TypeWorld
	case "full", string(backup.TypeFull):
		t = backup.TypeFull
	case "configuration", string(backup.TypeConfig):
		t = backup.TypeConfig
	default:
		writeErr(w, 400, errors.New("type must be world, full or configuration"))
		return
	}
	mode := "offline"
	if a.Runner.Online() {
		mode = "online"
	}
	go func() {
		if _, err := a.runBackup(context.Background(), inst, t, mode, "manual", u.ID); err != nil {
			a.Logf("manual backup failed: %v", err)
			a.Hub.Broadcast("backups", "failed", map[string]string{"error": err.Error()})
		}
	}()
	a.audit(u.ID, u.Username, "backup_started", inst.Slug, string(t), remoteIP(r))
	writeJSON(w, 202, map[string]bool{"started": true})
}

func (a *App) handleBackupVerify(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.Backups.Verify(id); err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleBackupRestore(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	if a.Runner.Online() {
		writeErr(w, 409, errors.New("stop the server before restoring a backup"))
		return
	}
	var req struct {
		Scope string `json:"scope"` // full_server | world_only | configuration_only
		// SkipEmergencyBackup disables the automatic pre-restore safety copy.
		// It is deliberately awkward to set and is audited when used.
		SkipEmergencyBackup bool `json:"skip_emergency_backup"`
		Confirm             bool `json:"confirm"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil || !req.Confirm {
		writeErr(w, 400, errors.New("restore requires explicit confirmation"))
		return
	}
	scope, err := backup.NormalizeScope(req.Scope)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	rec, err := a.Backups.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	release, err := a.OpLock.Acquire("restore")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	defer release()

	// A restore overwrites live server files, so take an emergency safety copy
	// of the current state first. If that fails, refuse to restore: losing the
	// present world to recover an older one is not an acceptable trade.
	if req.SkipEmergencyBackup {
		a.audit(u.ID, u.Username, "backup_restore_without_safety_copy", inst.Slug, rec.BackupID, remoteIP(r))
		a.Logf("restore of %s proceeding WITHOUT an emergency pre-restore backup (requested by %s)",
			rec.BackupID, u.Username)
	} else {
		a.Hub.Broadcast("backups", "progress", map[string]any{
			"stage": "emergency_pre_restore_backup", "backup_id": rec.BackupID,
		})
		safety, err := a.runBackupLocked(context.Background(), inst, backup.TypeFull,
			"offline", "emergency-pre-restore", u.ID)
		if err != nil {
			writeErr(w, 500, fmt.Errorf("emergency pre-restore backup failed, refusing to restore: %w", err))
			return
		}
		a.audit(u.ID, u.Username, "backup_emergency_pre_restore", inst.Slug, safety.BackupID, remoteIP(r))
	}

	a.Hub.Broadcast("backups", "progress", map[string]any{
		"stage": "restoring", "backup_id": rec.BackupID,
	})
	res, err := a.Backups.Restore(rec, inst.AbsoluteDir(a.Home), scope)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	detail := rec.BackupID + " scope=" + scope
	if res.LevelNameUpdated {
		detail += fmt.Sprintf(" level-name %s→%s", res.PreviousLevel, res.WorldName)
	}
	a.audit(u.ID, u.Username, "backup_restored", inst.Slug, detail, remoteIP(r))
	a.Hub.Broadcast("backups", "restored", map[string]any{"backup_id": rec.BackupID, "scope": scope})
	writeJSON(w, 200, map[string]any{
		"ok": true, "scope": scope,
		"world_name":         res.WorldName,
		"level_name_updated": res.LevelNameUpdated,
		"previous_level":     res.PreviousLevel,
	})
}

func (a *App) handleBackupProtect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protected bool `json:"protected"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if err := a.Backups.SetProtected(r.PathValue("id"), req.Protected); err != nil {
		writeErr(w, 404, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleBackupDelete(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id := r.PathValue("id")
	if err := a.Backups.Delete(id); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "backup_deleted", id, "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------------------------------------------------------------------------
// schedules
// ---------------------------------------------------------------------------

func (a *App) handleScheduleList(w http.ResponseWriter, r *http.Request) {
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	list, err := a.Sched.List(inst.ID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, list)
}

func (a *App) decodeSchedule(r *http.Request, s *scheduler.Schedule) error {
	if err := readJSON(r, s, 1<<18); err != nil {
		return errors.New("invalid request")
	}
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("a schedule name is required")
	}
	if _, err := time.LoadLocation(defString(s.Timezone, "UTC")); err != nil {
		return fmt.Errorf("unknown timezone %q", s.Timezone)
	}
	if _, err := scheduler.NextRun(s, time.Now()); err != nil {
		return err
	}
	return nil
}

func defString(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func (a *App) handleScheduleCreate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	var s scheduler.Schedule
	if err := a.decodeSchedule(r, &s); err != nil {
		writeErr(w, 400, err)
		return
	}
	s.InstanceID = inst.ID
	s.CreatedBy = u.ID
	if err := a.Sched.Create(&s); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "schedule_created", inst.Slug, s.Name, remoteIP(r))
	writeJSON(w, 200, s)
}

func (a *App) handleScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	existing, err := a.Sched.Get(id)
	if err != nil {
		writeErr(w, 404, errors.New("schedule not found"))
		return
	}
	var s scheduler.Schedule
	if err := a.decodeSchedule(r, &s); err != nil {
		writeErr(w, 400, err)
		return
	}
	s.ID = existing.ID
	s.InstanceID = existing.InstanceID
	if err := a.Sched.Update(&s); err != nil {
		writeErr(w, 400, err)
		return
	}
	a.audit(u.ID, u.Username, "schedule_updated", s.Name, "", remoteIP(r))
	writeJSON(w, 200, s)
}

func (a *App) handleScheduleDelete(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := a.Sched.Delete(id); err != nil {
		writeErr(w, 404, err)
		return
	}
	a.audit(u.ID, u.Username, "schedule_deleted", fmt.Sprint(id), "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleScheduleRunNow(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s, err := a.Sched.Get(id)
	if err != nil {
		writeErr(w, 404, errors.New("schedule not found"))
		return
	}
	go a.Sched.RunNow(context.Background(), s)
	a.audit(u.ID, u.Username, "schedule_run_now", s.Name, "", remoteIP(r))
	writeJSON(w, 202, map[string]bool{"started": true})
}

func (a *App) handleScheduleHistory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	hist, err := a.Sched.RunHistory(id, 50)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, hist)
}

// ---------------------------------------------------------------------------
// players
// ---------------------------------------------------------------------------

func (a *App) handlePlayerList(w http.ResponseWriter, r *http.Request) {
	inst, err := a.activeInstance()
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	admin := minecraft.ReadAdminFiles(inst.AbsoluteDir(a.Home))
	opNames := make(map[string]bool, len(admin.Ops))
	opUUIDs := make(map[string]bool, len(admin.Ops))
	for _, op := range admin.Ops {
		opNames[strings.ToLower(op.Name)] = true
		opUUIDs[strings.ToLower(strings.ReplaceAll(op.UUID, "-", ""))] = true
	}
	rows, err := a.DB.Query(`SELECT username, uuid, is_online, first_seen_at, last_seen_at,
		last_joined_at, last_left_at, observed_playtime_seconds, current_session_started_at
		FROM players WHERE instance_id=? ORDER BY is_online DESC, last_seen_at DESC`, inst.ID)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	defer rows.Close()
	type player struct {
		Username         string `json:"username"`
		UUID             string `json:"uuid,omitempty"`
		Online           bool   `json:"online"`
		OP               bool   `json:"op"`
		FirstSeenAt      string `json:"first_seen_at"`
		LastSeenAt       string `json:"last_seen_at"`
		LastJoinedAt     string `json:"last_joined_at,omitempty"`
		LastLeftAt       string `json:"last_left_at,omitempty"`
		ObservedPlaytime int64  `json:"observed_playtime_seconds"`
		SessionStart     string `json:"current_session_started_at,omitempty"`
	}
	var out []player
	for rows.Next() {
		var p player
		var uuid, joined, left, sess *string
		var online int
		if rows.Scan(&p.Username, &uuid, &online, &p.FirstSeenAt, &p.LastSeenAt,
			&joined, &left, &p.ObservedPlaytime, &sess) != nil {
			continue
		}
		p.Online = online != 0
		if uuid != nil {
			p.UUID = *uuid
		}
		p.OP = opNames[strings.ToLower(p.Username)] ||
			opUUIDs[strings.ToLower(strings.ReplaceAll(p.UUID, "-", ""))]
		if joined != nil {
			p.LastJoinedAt = *joined
		}
		if left != nil {
			p.LastLeftAt = *left
		}
		if sess != nil {
			p.SessionStart = *sess
		}
		out = append(out, p)
	}
	writeJSON(w, 200, map[string]any{
		"players": out,
		"note":    "Playtime reflects only sessions observed while Bonghos was running (observed playtime).",
	})
}

func (a *App) handlePlayerAction(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		// kick | ban | pardon | ban_ip | pardon_ip | op | deop |
		// whitelist_add | whitelist_remove | send_message
		Action string `json:"action"`
		// Username carries a player name, or an IP address for the *_ip
		// actions. "player" is accepted as an alias.
		Username string `json:"username"`
		Player   string `json:"player"`
		Reason   string `json:"reason"`
		Message  string `json:"message"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	target := strings.TrimSpace(req.Username)
	if target == "" {
		target = strings.TrimSpace(req.Player)
	}
	extra := req.Reason
	if req.Action == "send_message" {
		extra = req.Message
	}

	// Build the command from a fixed template. Validation lives in the
	// minecraft package so the CLI, scheduler and API cannot drift apart, and
	// so player-supplied text never reaches a shell.
	cmd, err := minecraft.PlayerCommand(req.Action, target, extra)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := a.Runner.SendCommand(cmd); err != nil {
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "player_"+req.Action, target, strings.TrimSpace(extra), remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// applyJVMMemory rewrites the detected editable JVM source with new values.
func (a *App) applyJVMMemory(inst *instance.Instance, xms, xmx string) error {
	dir := inst.AbsoluteDir(a.Home)
	jvm, err := minecraft.DetectJVMConfig(dir, inst.StartupScript)
	if err != nil || jvm == nil {
		return errors.New("could not locate a JVM configuration source for this project")
	}
	if !jvm.Editable {
		return fmt.Errorf("JVM memory is defined in %s which Bonghos cannot safely edit; adjust it in the file browser", jvm.SourceFile)
	}
	fm := &files.Manager{Root: dir}
	content, err := fm.ReadText(jvm.SourceFile)
	if err != nil {
		return err
	}
	updated := minecraft.UpdateJVMArgFile(content, xms, xmx)
	return fm.WriteText(jvm.SourceFile, updated)
}
