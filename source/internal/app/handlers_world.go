package app

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/files"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/operations"
)

func (a *App) instanceIsActive(id int64) bool {
	activeID, err := a.Instances.ActiveID()
	return err == nil && activeID == id
}

func worldPath(inst *instance.Instance, home string) (string, string, error) {
	serverDir := inst.AbsoluteDir(home)
	worldName := filepath.Clean(minecraft.WorldDir(serverDir))
	if worldName == "." || worldName == ".." || filepath.IsAbs(worldName) {
		return "", "", errors.New("invalid world directory in server.properties")
	}
	world := filepath.Join(serverDir, worldName)
	rel, err := filepath.Rel(serverDir, world)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", errors.New("world directory escapes the server project")
	}
	info, err := os.Lstat(world)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", errors.New("world directory does not exist")
		}
		return "", "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", "", errors.New("world path must be a real directory")
	}
	return world, filepath.Base(worldName), nil
}

func (a *App) handleServerDuplicate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	source, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	if a.instanceIsActive(source.ID) && a.Runner.Online() {
		writeErr(w, 409, errors.New("stop this server before duplicating it"))
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		req.DisplayName = source.DisplayName + " Copy"
	}

	clone, err := a.reserveDuplicate(source, req.DisplayName)
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	op, err := a.Operations.Create("duplicate-server", clone.ID, u.ID, map[string]any{
		"source_instance_id": source.ID,
		"display_name":       clone.DisplayName,
	})
	if err != nil {
		_ = a.Instances.Delete(clone.ID)
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "server_duplicate_started", source.Slug, clone.Slug, remoteIP(r))

	go a.duplicateServer(op.ID, source, clone)
	writeJSON(w, 202, map[string]any{"operation_id": op.ID, "server": clone})
}

func (a *App) reserveDuplicate(source *instance.Instance, displayName string) (*instance.Instance, error) {
	base := instance.GenerateSlug(displayName)
	if base == "" {
		base = source.Slug + "-copy"
	}
	for n := 1; n <= 100; n++ {
		slug := base
		if n > 1 {
			suffix := fmt.Sprintf("-%d", n)
			prefix := strings.TrimRight(base[:min(len(base), 64-len(suffix))], "-")
			slug = prefix + suffix
		}
		clone := &instance.Instance{
			Slug: slug, DisplayName: strings.TrimSpace(displayName),
			ServerType: source.ServerType, SourceType: "duplicate",
			MinecraftVersion: source.MinecraftVersion, Modloader: source.Modloader,
			ModloaderVersion: source.ModloaderVersion, ServerDirectory: instance.RelativeDirFor(slug),
			StartupScript: source.StartupScript, JavaSelection: source.JavaSelection,
			JVMConfigurationSource: source.JVMConfigurationSource,
			JVMXms:                 source.JVMXms, JVMXmx: source.JVMXmx, JVMExtraArgs: source.JVMExtraArgs,
			IconRevision: source.IconRevision, AutostartEnabled: false,
			BootDelaySeconds:            source.BootDelaySeconds,
			RecoverAfterUncleanShutdown: source.RecoverAfterUncleanShutdown,
			RestartPolicy:               source.RestartPolicy, RestartDelaySeconds: source.RestartDelaySeconds,
		}
		if err := a.Instances.Create(clone); err != nil {
			if errors.Is(err, instance.ErrSlugTaken) {
				continue
			}
			return nil, err
		}
		// Create applies safety defaults; persist the source's selected runtime
		// settings while deliberately leaving autostart disabled on the copy.
		clone.AutostartEnabled = false
		if err := a.Instances.Update(clone); err != nil {
			_ = a.Instances.Delete(clone.ID)
			return nil, err
		}
		return clone, nil
	}
	return nil, errors.New("could not allocate a unique duplicate slug")
}

func (a *App) duplicateServer(opID string, source, clone *instance.Instance) {
	fail := func(err error) {
		_ = os.RemoveAll(clone.AbsoluteDir(a.Home))
		_ = a.Instances.Delete(clone.ID)
		a.Operations.Finish(opID, operations.StageFailed, err.Error())
	}
	release, err := a.OpLock.Acquire("duplicate-server")
	if err != nil {
		fail(err)
		return
	}
	defer release()
	releaseLifecycle, err := a.Runner.acquire("duplicate-server")
	if err != nil {
		fail(err)
		return
	}
	defer releaseLifecycle()
	if a.instanceIsActive(source.ID) && a.Runner.Online() {
		fail(errors.New("server started before duplication began; stop it and try again"))
		return
	}
	a.Operations.SetStage(opID, operations.StageMoving, "Copying server files")
	if err := files.CopyTree(source.AbsoluteDir(a.Home), clone.AbsoluteDir(a.Home)); err != nil {
		fail(err)
		return
	}
	a.finishInstall(opID, clone)
}

func (a *App) handleWorldReset(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := readJSON(r, &req, 1<<14); err != nil || !req.Confirm {
		writeErr(w, 400, errors.New("world reset requires explicit confirmation"))
		return
	}
	if a.instanceIsActive(inst.ID) && a.Runner.Online() {
		writeErr(w, 409, errors.New("stop this server before resetting its world"))
		return
	}
	world, _, err := worldPath(inst, a.Home)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	release, err := a.OpLock.Acquire("world-reset")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	defer release()
	releaseLifecycle, err := a.Runner.acquire("world-reset")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	defer releaseLifecycle()
	if a.instanceIsActive(inst.ID) && a.Runner.Online() {
		writeErr(w, 409, errors.New("server started before the reset began; stop it and try again"))
		return
	}
	rec, err := a.runBackupLocked(context.Background(), inst, backup.TypeWorld, "offline", "emergency-pre-world-reset", u.ID)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("safety backup failed; world was not reset: %w", err))
		return
	}
	if err := os.RemoveAll(world); err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "world_reset", inst.Slug, "safety_backup="+rec.BackupID, remoteIP(r))
	writeJSON(w, 200, map[string]any{"ok": true, "backup_id": rec.BackupID})
}

func (a *App) handleWorldDownload(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	world, worldName, err := worldPath(inst, a.Home)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	release, err := a.OpLock.Acquire("world-download")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	defer release()
	releaseLifecycle, err := a.Runner.acquire("world-download")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	defer releaseLifecycle()

	online := a.instanceIsActive(inst.ID) && a.Runner.Online()
	if online {
		if err := a.Runner.SendCommand("save-off"); err != nil {
			writeErr(w, 409, fmt.Errorf("could not pause world saves: %w", err))
			return
		}
		defer a.Runner.SendCommand("save-on")
		if err := a.Runner.SendCommand("save-all flush"); err != nil {
			writeErr(w, 409, fmt.Errorf("could not flush world saves: %w", err))
			return
		}
		time.Sleep(3 * time.Second)
	}

	tempDir := filepath.Join(a.Home, config.DirTemp)
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		writeErr(w, 500, err)
		return
	}
	temp, err := os.CreateTemp(tempDir, "world-download-*.zip")
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	defer func() {
		temp.Close()
		os.Remove(temp.Name())
	}()
	if err := writeWorldZIP(temp, world, worldName); err != nil {
		writeErr(w, 500, err)
		return
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		writeErr(w, 500, err)
		return
	}
	filename := inst.Slug + "-world.zip"
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Cache-Control", "no-store")
	a.audit(u.ID, u.Username, "world_downloaded", inst.Slug, worldName, remoteIP(r))
	http.ServeContent(w, r, filename, time.Now(), temp)
}

func writeWorldZIP(dst io.Writer, world, worldName string) error {
	zw := zip.NewWriter(dst)
	err := filepath.WalkDir(world, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(world, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(worldName, rel))
		if entry.IsDir() {
			if rel == "." {
				return nil
			}
			_, err := zw.Create(name + "/")
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate
		out, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}
