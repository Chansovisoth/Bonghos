package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/download"
	"github.com/Chansovisoth/Bonghos/internal/files"
	"github.com/Chansovisoth/Bonghos/internal/image"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/operations"
	"github.com/Chansovisoth/Bonghos/internal/security"
)

// createProjectForImport validates name+slug and creates the instance row.
func (a *App) createProjectForImport(displayName, slug, sourceType, sourceHost string) (*instance.Instance, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, errors.New("a project display name is required")
	}
	if slug == "" {
		slug = instance.GenerateSlug(displayName)
	}
	if err := instance.ValidateSlug(slug); err != nil {
		return nil, err
	}
	inst := &instance.Instance{
		Slug: slug, DisplayName: displayName,
		ServerType: "minecraft-java-modded",
		SourceType: sourceType, SourceURLHost: sourceHost,
		ServerDirectory: instance.RelativeDirFor(slug),
	}
	if err := a.Instances.Create(inst); err != nil {
		return nil, err
	}
	return inst, nil
}

// handleImportUpload streams a multipart archive upload to temp storage and
// runs the import pipeline in the background.
func (a *App) handleImportUpload(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	r.Body = http.MaxBytesReader(w, r.Body, a.Cfg.MaxUploadBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		writeErr(w, 400, errors.New("multipart upload expected"))
		return
	}

	var displayName, slug, filename string
	var op *operations.Operation
	var uploadDir, archivePath string
	var written int64
	total := r.ContentLength

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			if op != nil {
				a.Operations.Finish(op.ID, "failed", "upload interrupted: "+err.Error())
				_ = os.RemoveAll(uploadDir)
			}
			writeErr(w, 400, errors.New("upload failed"))
			return
		}
		switch part.FormName() {
		case "display_name":
			b, _ := io.ReadAll(io.LimitReader(part, 1<<12))
			displayName = string(b)
		case "slug":
			b, _ := io.ReadAll(io.LimitReader(part, 1<<12))
			slug = string(b)
		case "archive":
			filename = filepath.Base(part.FileName())
			if displayName == "" {
				writeErr(w, 400, errors.New("display_name must be sent before the archive"))
				return
			}
			op, err = a.Operations.Create("import-upload", 0, u.ID, map[string]string{
				"filename": filename, "display_name": displayName,
			})
			if err != nil {
				writeErr(w, 500, err)
				return
			}
			uploadDir = filepath.Join(a.Home, config.DirUploads, op.ID)
			if err := os.MkdirAll(uploadDir, 0o700); err != nil {
				writeErr(w, 500, err)
				return
			}
			// Never trust original filename as a path.
			archivePath = filepath.Join(uploadDir, "upload.archive")
			a.Operations.SetStage(op.ID, "receiving_upload", "Receiving "+filename)

			dst, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				a.Operations.Finish(op.ID, "failed", err.Error())
				writeErr(w, 500, err)
				return
			}
			h := sha256.New()
			buf := make([]byte, 1<<20)
			lastProg := time.Now()
			for {
				n, rerr := part.Read(buf)
				if n > 0 {
					if _, werr := dst.Write(buf[:n]); werr != nil {
						dst.Close()
						a.Operations.Finish(op.ID, "failed", werr.Error())
						_ = os.RemoveAll(uploadDir)
						writeErr(w, 500, werr)
						return
					}
					h.Write(buf[:n])
					written += int64(n)
					if time.Since(lastProg) > 500*time.Millisecond {
						a.Operations.Progress(op.ID, written, total)
						lastProg = time.Now()
					}
				}
				if rerr == io.EOF {
					break
				}
				if rerr != nil {
					dst.Close()
					a.Operations.Finish(op.ID, "failed", "upload interrupted")
					_ = os.RemoveAll(uploadDir)
					writeErr(w, 400, errors.New("upload interrupted"))
					return
				}
			}
			if err := dst.Sync(); err == nil {
				_ = dst.Close()
			} else {
				dst.Close()
			}
			a.Operations.Progress(op.ID, written, written)
			a.Logf("upload complete %s (%d bytes, sha256 %s)", filename, written, hex.EncodeToString(h.Sum(nil)))
		}
	}

	if op == nil || archivePath == "" {
		writeErr(w, 400, errors.New("no archive received"))
		return
	}

	inst, err := a.createProjectForImport(displayName, slug, "archive-upload", "")
	if err != nil {
		a.Operations.Finish(op.ID, "failed", err.Error())
		_ = os.RemoveAll(uploadDir)
		writeErr(w, 409, err)
		return
	}
	a.audit(u.ID, u.Username, "server_upload", inst.Slug, filename, remoteIP(r))

	go a.installArchive(op.ID, inst, archivePath, uploadDir)
	writeJSON(w, 200, map[string]any{"operation_id": op.ID, "server": inst})
}

// handleImportURL starts a server-side download; the Linux host performs it.
func (a *App) handleImportURL(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		URL         string `json:"url"`
		DisplayName string `json:"display_name"`
		Slug        string `json:"slug"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	opts := download.DefaultOptions()
	opts.AllowInsecureHTTP = a.Cfg.AllowInsecureHTTPURL
	opts.TrustedHosts = a.Cfg.TrustedDownloadHosts
	opts.MaxBytes = a.Cfg.MaxArchiveBytes
	// Without this the downloader falls back to its own default and the
	// operator's configured reserve is ignored, so a large pack can fill the
	// filesystem the server itself needs.
	opts.FreeSpaceReserve = a.Cfg.FreeSpaceReserveMB << 20
	pu, err := download.ValidateURL(req.URL, opts)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	host := pu.Hostname()
	inst, err := a.createProjectForImport(req.DisplayName, req.Slug, "direct-url", host)
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	op, err := a.Operations.Create("import-url", inst.ID, u.ID, map[string]string{
		"url": download.RedactURL(req.URL), "display_name": inst.DisplayName,
	})
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	a.audit(u.ID, u.Username, "url_download_started", inst.Slug, download.RedactURL(req.URL), remoteIP(r))

	// Background: continues after browser disconnect.
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		unregister := a.Operations.RegisterCancel(op.ID, cancel)
		defer unregister()
		defer cancel()

		uploadDir := filepath.Join(a.Home, config.DirUploads, op.ID)
		if err := os.MkdirAll(uploadDir, 0o700); err != nil {
			a.Operations.Finish(op.ID, "failed", err.Error())
			return
		}
		a.Operations.SetStage(op.ID, "connecting", "Connecting to "+host)
		lastProg := time.Now()
		res, err := download.Fetch(ctx, req.URL, uploadDir, opts, func(done, tot int64) {
			if time.Since(lastProg) > 500*time.Millisecond {
				a.Operations.SetStage(op.ID, "downloading", "")
				a.Operations.Progress(op.ID, done, tot)
				lastProg = time.Now()
			}
		})
		if err != nil {
			a.Operations.Finish(op.ID, "failed", err.Error())
			_ = os.RemoveAll(uploadDir)
			return
		}
		a.Operations.SetStage(op.ID, "verifying_download", "Verifying archive")
		a.installArchive(op.ID, inst, res.Path, uploadDir)
	}()
	writeJSON(w, 200, map[string]any{"operation_id": op.ID, "server": inst})
}

// handleImportLocal imports an archive already on the Linux machine.
func (a *App) handleImportLocal(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Path        string `json:"path"`
		DisplayName string `json:"display_name"`
		Slug        string `json:"slug"`
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	src := filepath.Clean(req.Path)
	st, err := os.Stat(src)
	if err != nil || !st.Mode().IsRegular() {
		writeErr(w, 400, errors.New("path is not a readable regular file"))
		return
	}
	if st.Size() > a.Cfg.MaxArchiveBytes {
		writeErr(w, 400, errors.New("archive exceeds the configured maximum size"))
		return
	}
	inst, err := a.createProjectForImport(req.DisplayName, req.Slug, "local-archive", "")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	op, err := a.Operations.Create("import-local", inst.ID, u.ID, map[string]string{
		"display_name": inst.DisplayName,
	})
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	// Original path recorded only in protected operation logs.
	a.opLog(op.ID, "local archive source: %s", src)
	a.audit(u.ID, u.Username, "server_import_local", inst.Slug, "", remoteIP(r))

	go func() {
		uploadDir := filepath.Join(a.Home, config.DirUploads, op.ID)
		if err := os.MkdirAll(uploadDir, 0o700); err != nil {
			a.Operations.Finish(op.ID, "failed", err.Error())
			return
		}
		a.Operations.SetStage(op.ID, "receiving_upload", "Copying archive")
		dst := filepath.Join(uploadDir, "upload.archive")
		if err := copyFileProgress(src, dst, func(done, tot int64) {
			a.Operations.Progress(op.ID, done, tot)
		}); err != nil {
			a.Operations.Finish(op.ID, "failed", err.Error())
			_ = os.RemoveAll(uploadDir)
			return
		}
		a.installArchive(op.ID, inst, dst, uploadDir)
	}()
	writeJSON(w, 200, map[string]any{"operation_id": op.ID, "server": inst})
}

// handleImportExisting imports (copy/move/link) an existing directory.
func (a *App) handleImportExisting(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Path        string `json:"path"`
		DisplayName string `json:"display_name"`
		Slug        string `json:"slug"`
		Mode        string `json:"mode"`         // copy | move | link
		ConfirmLink bool   `json:"confirm_link"` // explicit confirmation for external link
	}
	if err := readJSON(r, &req, 1<<16); err != nil {
		writeErr(w, 400, errors.New("invalid request"))
		return
	}
	src, err := filepath.Abs(filepath.Clean(req.Path))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		writeErr(w, 400, errors.New("directory not found or unreadable"))
		return
	}
	st, err := os.Stat(resolved)
	if err != nil || !st.IsDir() {
		writeErr(w, 400, errors.New("path is not a directory"))
		return
	}
	mode := req.Mode
	if mode == "" {
		mode = "copy"
	}
	if mode == "link" && !req.ConfirmLink {
		writeErr(w, 400, errors.New("linking an external directory requires explicit confirmation"))
		return
	}
	// Refuse importing from inside the Bonghos system tree.
	if security.WithinRoot(filepath.Join(a.Home, "system"), resolved) {
		writeErr(w, 400, errors.New("cannot import from inside the Bonghos system directory"))
		return
	}

	// A live server writing into this directory while it is copied, moved or
	// adopted would produce a corrupt world. Refuse rather than adopting or
	// killing a process Bonghos did not start.
	if procs := minecraft.FindRunningJavaIn(resolved); len(procs) > 0 {
		pids := make([]string, 0, len(procs))
		for _, p := range procs {
			pids = append(pids, strconv.Itoa(p.PID))
		}
		writeErr(w, 409, fmt.Errorf(
			"a Java process is already running in this directory (PID %s); stop that server before importing it",
			strings.Join(pids, ", ")))
		return
	}

	sourceType := "existing-directory"
	if mode == "link" {
		sourceType = "external-directory-link"
	}
	inst, err := a.createProjectForImport(req.DisplayName, req.Slug, sourceType, "")
	if err != nil {
		writeErr(w, 409, err)
		return
	}
	op, err := a.Operations.Create("import-directory", inst.ID, u.ID, map[string]string{
		"mode": mode, "display_name": inst.DisplayName,
	})
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	a.opLog(op.ID, "existing directory source: %s (mode %s)", resolved, mode)
	a.audit(u.ID, u.Username, "server_import_directory", inst.Slug, mode, remoteIP(r))

	go func() {
		release, err := a.OpLock.Acquire("import")
		if err != nil {
			a.Operations.Finish(op.ID, "failed", err.Error())
			return
		}
		defer release()

		dest := inst.AbsoluteDir(a.Home)
		switch mode {
		case "link":
			inst.ServerDirectory = resolved
			inst.ExternalDirectory = true
			if err := a.Instances.Update(inst); err != nil {
				a.Operations.Finish(op.ID, "failed", err.Error())
				return
			}
		case "move":
			a.Operations.SetStage(op.ID, "moving_to_destination", "Moving directory")
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				a.Operations.Finish(op.ID, "failed", err.Error())
				return
			}
			if err := os.Rename(resolved, dest); err != nil {
				// Cross-device: fall back to copy, then leave source intact
				// (never silently delete external data).
				if err := files.CopyTree(resolved, dest); err != nil {
					a.Operations.Finish(op.ID, "failed", err.Error())
					return
				}
				a.opLog(op.ID, "cross-filesystem move: copied; original left at %s", resolved)
			}
		default: // copy
			a.Operations.SetStage(op.ID, "moving_to_destination", "Copying directory")
			if err := files.CopyTree(resolved, dest); err != nil {
				a.Operations.Finish(op.ID, "failed", err.Error())
				return
			}
		}
		a.finishInstall(op.ID, inst)
	}()
	writeJSON(w, 200, map[string]any{"operation_id": op.ID, "server": inst})
}

// installArchive validates, extracts and installs an archive into the project.
func (a *App) installArchive(opID string, inst *instance.Instance, archivePath, cleanupDir string) {
	release, err := a.OpLock.Acquire("import")
	if err != nil {
		a.Operations.Finish(opID, "failed", err.Error())
		return
	}
	defer release()
	defer os.RemoveAll(cleanupDir)

	fail := func(err error) {
		a.Operations.Finish(opID, "failed", err.Error())
		a.opLog(opID, "import failed: %v", err)
	}

	a.Operations.SetStage(opID, "verifying_download", "Detecting archive type")
	format, err := files.DetectFormat(archivePath)
	if err != nil {
		fail(fmt.Errorf("unsupported or corrupted archive: %w", err))
		return
	}
	a.opLog(opID, "archive format: %s", format)
	_ = format

	staging := filepath.Join(a.Home, config.DirStaging, opID)
	if err := os.MkdirAll(staging, 0o700); err != nil {
		fail(err)
		return
	}
	defer os.RemoveAll(staging)

	a.Operations.SetStage(opID, "extracting", "Extracting archive")
	limits := files.Limits{
		MaxBytes:         a.Cfg.MaxArchiveBytes,
		MaxFiles:         a.Cfg.MaxArchiveFiles,
		FreeSpaceReserve: a.Cfg.FreeSpaceReserveMB << 20,
	}
	var extracted int64
	lastProg := time.Now()
	if err := files.Extract(archivePath, staging, limits, func(n int64) {
		extracted += n
		if time.Since(lastProg) > time.Second {
			a.Operations.Progress(opID, extracted, 0)
			lastProg = time.Now()
		}
	}); err != nil {
		fail(fmt.Errorf("extraction failed: %w", err))
		return
	}

	a.Operations.SetStage(opID, "detecting_server_root", "Locating server root")
	root, err := files.FindServerRoot(staging)
	if err != nil {
		root = staging
	}

	a.Operations.SetStage(opID, "detecting_startup_script", "Detecting startup script")
	candidates, _ := minecraft.DetectStartupScripts(root, 3)
	if len(candidates) > 0 {
		inst.StartupScript = candidates[0].Path
		inst.Modloader = candidates[0].Modloader
	}

	a.Operations.SetStage(opID, "detecting_jvm_configuration", "Detecting JVM configuration")
	if inst.StartupScript != "" {
		if jvm, err := minecraft.DetectJVMConfig(root, inst.StartupScript); err == nil && jvm != nil {
			inst.JVMConfigurationSource = jvm.SourceFile
			inst.JVMXms = jvm.Xms
			inst.JVMXmx = jvm.Xmx
		}
	}

	a.Operations.SetStage(opID, "validating_installation", "Validating destination")
	dest := inst.AbsoluteDir(a.Home)
	if _, err := os.Stat(dest); err == nil {
		// The project shell may exist (created empty); allow only if empty.
		entries, _ := os.ReadDir(dest)
		if len(entries) > 0 {
			fail(errors.New("destination already contains files; refusing to overwrite"))
			return
		}
		_ = os.Remove(dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		fail(err)
		return
	}

	a.Operations.SetStage(opID, "moving_to_destination", "Installing server files")
	if err := os.Rename(root, dest); err != nil {
		if err := files.CopyTree(root, dest); err != nil {
			fail(err)
			return
		}
	}
	a.finishInstall(opID, inst)
}

func (a *App) finishInstall(opID string, inst *instance.Instance) {
	dir := inst.AbsoluteDir(a.Home)
	// Re-run detection against final location.
	if inst.StartupScript == "" {
		if cands, _ := minecraft.DetectStartupScripts(dir, 3); len(cands) > 0 {
			inst.StartupScript = cands[0].Path
			inst.Modloader = cands[0].Modloader
		}
	}
	if props, err := minecraft.ReadProperties(dir); err == nil {
		_ = props // versions unavailable from properties; left for detection
	}
	if err := a.Instances.Update(inst); err != nil {
		a.Operations.Finish(opID, "failed", err.Error())
		return
	}
	// Select as active if nothing is active yet.
	if activeID, _ := a.Instances.ActiveID(); activeID == 0 {
		_ = a.Instances.SetActive(inst.ID)
	}
	a.Operations.SetStage(opID, "completed", "Ready")
	a.Operations.Finish(opID, "completed", "")
	a.Hub.Broadcast("servers", "installed", inst)
}

// opLog appends to a protected per-operation log file.
func (a *App) opLog(opID, format string, args ...any) {
	p := filepath.Join(a.Home, config.DirOpLogs, opID+".log")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] ", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(f, format+"\n", args...)
}

func copyFileProgress(src, dst string, progress func(done, total int64)) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	buf := make([]byte, 1<<20)
	var done int64
	last := time.Now()
	for {
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				return werr
			}
			done += int64(n)
			if progress != nil && time.Since(last) > 500*time.Millisecond {
				progress(done, st.Size())
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return rerr
		}
	}
	return out.Close()
}

// ---------------------------------------------------------------------------
// operations
// ---------------------------------------------------------------------------

func (a *App) handleOperationList(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"
	ops, err := a.Operations.List(activeOnly)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, ops)
}

func (a *App) handleOperationCancel(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	id := r.PathValue("id")
	if err := a.Operations.Cancel(id); err != nil {
		writeErr(w, 404, err)
		return
	}
	a.audit(u.ID, u.Username, "import_cancelled", id, "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------------------------------------------------------------------------
// icons
// ---------------------------------------------------------------------------

func (a *App) handleIconGet(w http.ResponseWriter, r *http.Request) {
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	p := filepath.Join(inst.AbsoluteDir(a.Home), "server-icon.png")
	if !security.WithinRoot(inst.AbsoluteDir(a.Home), p) {
		writeErr(w, 400, errors.New("invalid icon path"))
		return
	}
	st, err := os.Stat(p)
	if err != nil {
		http.Error(w, "no icon", 404)
		return
	}
	etag := fmt.Sprintf(`"%d-%d-%d"`, st.Size(), st.ModTime().Unix(), inst.IconRevision)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeFile(w, r, p)
}

func (a *App) handleIconUpload(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, image.MaxUploadBytes+1<<16)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeErr(w, 400, errors.New("multipart upload expected"))
		return
	}
	f, _, err := r.FormFile("icon")
	if err != nil {
		writeErr(w, 400, errors.New("icon file missing"))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, image.MaxUploadBytes+1))
	if err != nil || int64(len(data)) > image.MaxUploadBytes {
		writeErr(w, 400, errors.New("icon too large"))
		return
	}
	var crop image.CropSpec
	if cs := r.FormValue("crop"); cs != "" {
		fmt.Sscanf(cs, "%d,%d,%d", &crop.X, &crop.Y, &crop.Size)
	}
	png, err := image.Process(data, crop)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := image.InstallIcon(inst.AbsoluteDir(a.Home), png); err != nil {
		writeErr(w, 500, err)
		return
	}
	inst.IconRevision++
	_ = a.Instances.Update(inst)
	a.audit(u.ID, u.Username, "icon_changed", inst.Slug, "", remoteIP(r))
	writeJSON(w, 200, map[string]any{"icon_revision": inst.IconRevision})
}

func (a *App) handleIconDelete(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	inst, err := a.pathInstance(r)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	p := filepath.Join(inst.AbsoluteDir(a.Home), "server-icon.png")
	_ = os.Remove(p)
	inst.IconRevision++
	_ = a.Instances.Update(inst)
	a.audit(u.ID, u.Username, "icon_removed", inst.Slug, "", remoteIP(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ImportDirectoryCLI copies an existing server directory into Bonghos from the
// command line. It reuses the same validation, locking and detection path as
// the Web UI import, but runs synchronously so the CLI can report the result.
func (a *App) ImportDirectoryCLI(ctx context.Context, displayName, slug, dir string) (*instance.Instance, error) {
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve %s: %w", dir, err)
	}
	st, err := os.Stat(resolved)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}
	if security.WithinRoot(filepath.Join(a.Home, "system"), resolved) {
		return nil, errors.New("cannot import from inside the Bonghos system directory")
	}

	inst, err := a.createProjectForImport(displayName, slug, "existing-directory", "")
	if err != nil {
		return nil, err
	}
	op, err := a.Operations.Create("import-directory", inst.ID, 0, map[string]string{
		"mode": "copy", "display_name": inst.DisplayName,
	})
	if err != nil {
		return nil, err
	}
	a.opLog(op.ID, "existing directory source: %s (mode copy, via CLI)", resolved)

	release, err := a.OpLock.Acquire("import")
	if err != nil {
		a.Operations.Finish(op.ID, "failed", err.Error())
		return nil, err
	}
	defer release()

	dest := inst.AbsoluteDir(a.Home)
	if entries, _ := os.ReadDir(dest); len(entries) > 0 {
		a.Operations.Finish(op.ID, "failed", "destination already contains files")
		return nil, errors.New("destination already contains files; refusing to overwrite")
	}
	a.Operations.SetStage(op.ID, "moving_to_destination", "Copying directory")
	if err := files.CopyTree(resolved, dest); err != nil {
		a.Operations.Finish(op.ID, "failed", err.Error())
		return nil, err
	}
	a.finishInstall(op.ID, inst)
	return a.Instances.ByID(inst.ID)
}
