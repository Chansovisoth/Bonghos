// Package app is the Bonghos control plane: HTTP + WebSocket API, embedded
// Web UI, and coordination of imports, lifecycle, schedules, backups and
// monitoring. Minecraft itself runs under the separate supervisor process.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/monitoring"
	"github.com/Chansovisoth/Bonghos/internal/operations"
	"github.com/Chansovisoth/Bonghos/internal/scheduler"
	"github.com/Chansovisoth/Bonghos/internal/security"
	"github.com/Chansovisoth/Bonghos/internal/supervisor"
	"github.com/Chansovisoth/Bonghos/internal/websocket"
)

// Version is stamped at build time via -ldflags.
var Version = "0.1.0-dev"

// App is the control plane.
type App struct {
	Home string
	Cfg  *config.Config
	DB   *sql.DB

	SecretKey  []byte
	Auth       *auth.Store
	Instances  *instance.Store
	Operations *operations.Store
	OpLock     *operations.Lock
	Backups    *backup.Manager
	Sched      *scheduler.Scheduler
	Hub        *websocket.Hub
	Runner     *Runner
	Collector  *monitoring.Collector

	WebFS fs.FS // embedded frontend (dist), may be nil in dev

	Logf func(format string, args ...any)
}

// New wires the application against an initialized home.
func New(home string, webFS fs.FS) (*App, error) {
	if err := config.InitHome(home); err != nil {
		return nil, err
	}
	cfg, err := config.Load(home)
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(home, config.FileSecretKey)
	if _, err := os.Stat(keyPath); err != nil {
		if err := security.GenerateSecretKey(keyPath); err != nil {
			return nil, err
		}
	}
	key, err := security.LoadSecretKey(keyPath)
	if err != nil {
		return nil, err
	}
	db, err := database.Open(filepath.Join(home, config.FileDatabase))
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	logFile, _ := os.OpenFile(filepath.Join(home, config.DirLogs, "bonghos.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	logger := log.New(logFile, "", log.LstdFlags|log.LUTC)

	a := &App{
		Home:      home,
		Cfg:       cfg,
		DB:        db,
		SecretKey: key,
		WebFS:     webFS,
		Logf: func(format string, args ...any) {
			logger.Printf(format, args...)
		},
	}
	a.Auth = &auth.Store{DB: db, SecretKey: key, Sessions: time.Duration(cfg.SessionHours) * time.Hour}
	a.Instances = &instance.Store{DB: db}
	a.Operations = operations.NewStore(db)
	a.OpLock = operations.NewLock(home)
	a.Backups = &backup.Manager{Home: home, DB: db}
	a.Hub = websocket.NewHub()
	a.Runner = newRunner(a)
	a.Collector = monitoring.NewCollector()
	a.Sched = &scheduler.Scheduler{DB: db, Exec: &executor{app: a}, Log: a.Logf}

	a.Operations.OnUpdate(func(op *operations.Operation) {
		a.Hub.Broadcast("servers", "operation", op)
	})
	return a, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}

func (a *App) activeInstance() (*instance.Instance, error) {
	id, err := a.Instances.ActiveID()
	if err != nil || id == 0 {
		return nil, fmt.Errorf("no active server project selected")
	}
	return a.Instances.ByID(id)
}

// Serve runs the HTTP server and background loops until ctx is cancelled.
func (a *App) Serve(ctx context.Context) error {
	a.Operations.CleanStale()

	go a.Sched.Run(ctx)
	go a.metricsLoop(ctx)
	go a.playerPollLoop(ctx)
	go a.bootAutostart(ctx)
	go a.Runner.attachConsole()

	addr := net.JoinHostPort(a.Cfg.BindAddress, fmt.Sprintf("%d", a.Cfg.Port))
	srv := &http.Server{
		Addr:              addr,
		Handler:           a.routes(),
		ReadHeaderTimeout: 15 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	a.Logf("bonghos %s serving on http://%s", Version, addr)
	fmt.Printf("Bonghos %s — Web UI: http://%s\n", Version, addr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

// bootAutostart starts the active project once after control-plane startup
// when autostart is enabled, honoring boot delay, conflicts and recovery.
func (a *App) bootAutostart(ctx context.Context) {
	inst, err := a.activeInstance()
	if err != nil || !inst.AutostartEnabled {
		return
	}
	// Unclean shutdown classification.
	unclean := false
	if ps, _ := supervisor.LoadPersisted(a.Home); ps != nil {
		stale, wasUnclean := supervisor.StaleCheck(ps)
		if !stale {
			return // already running
		}
		unclean = wasUnclean
	}
	if unclean && !inst.RecoverAfterUncleanShutdown {
		a.Logf("autostart: previous shutdown was unclean and recovery is disabled for %s", inst.Slug)
		a.audit(0, "system", "autostart_skipped", inst.Slug, "unclean shutdown, recovery disabled", "")
		return
	}

	delay := time.Duration(inst.BootDelaySeconds) * time.Second
	if delay < 0 {
		delay = 0
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}
	if a.Runner.Online() {
		return // duplicate-start prevention
	}
	if err := a.Runner.Start(ctx); err != nil {
		a.Logf("autostart failed: %v", err)
		a.audit(0, "system", "autostart_failed", inst.Slug, err.Error(), "")
		return
	}
	kind := "autostart"
	if unclean {
		kind = "boot_recovery"
	}
	a.audit(0, "system", kind, inst.Slug, "", "")
}

func (a *App) metricsLoop(ctx context.Context) {
	interval := time.Duration(a.Cfg.MetricsIntervalSec) * time.Second
	if interval < 2*time.Second {
		interval = 10 * time.Second
	}
	slow := time.NewTicker(interval)
	defer slow.Stop()
	pruneT := time.NewTicker(1 * time.Hour)
	defer pruneT.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pruneT.C:
			monitoring.Prune(a.DB, time.Duration(a.Cfg.MetricsRetentionDays)*24*time.Hour)
		case <-slow.C:
			// Sample at a lower rate when nobody is watching.
			watching := a.Hub.SubscriberCount("performance") > 0 || a.Hub.SubscriberCount("overview") > 0
			s := a.collectSample()
			instID := a.activeInstanceIDQuiet()
			monitoring.StoreSample(a.DB, instID, s)
			if watching {
				a.Hub.Broadcast("performance", "sample", s)
				a.Hub.Broadcast("overview", "sample", s)
			}
		}
	}
}

func (a *App) collectSample() *monitoring.Sample {
	s := &monitoring.Sample{CollectedAt: time.Now().UTC().Format(time.RFC3339)}
	_, ps := a.Runner.State()
	if ps != nil && ps.JavaPID > 0 {
		s.JavaPID = ps.JavaPID
		s.CPUPercent = a.Collector.ProcessCPU(ps.JavaPID)
		s.RSSBytes = monitoring.ProcessRSS(ps.JavaPID)
		if t, err := time.Parse(time.RFC3339, ps.StartedAt); err == nil {
			s.UptimeSeconds = int64(time.Since(t).Seconds())
		}
	}
	s.HostMemTotal, s.HostMemAvail = monitoring.HostMemory()
	s.Load1 = monitoring.LoadAvg()
	s.DiskTotal, s.DiskFree = monitoring.DiskUsage(a.Home)
	var online int
	if id := a.activeInstanceIDQuiet(); id != 0 {
		_ = a.DB.QueryRow(`SELECT COUNT(*) FROM players WHERE instance_id=? AND is_online=1`, id).Scan(&online)
	}
	s.OnlinePlayers = online
	return s
}

// playerPollLoop issues `list` periodically while relevant pages are open.
func (a *App) playerPollLoop(ctx context.Context) {
	t := time.NewTicker(12 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if a.Hub.SubscriberCount("players") == 0 && a.Hub.SubscriberCount("overview") == 0 {
				continue
			}
			if st, _ := a.Runner.State(); st != "running" {
				continue
			}
			_ = a.Runner.SendCommand("list")
		}
	}
}

// ---------------------------------------------------------------------------
// Player tracking helpers
// ---------------------------------------------------------------------------

func (a *App) recordPlayerJoin(instID int64, name string) {
	if instID == 0 || name == "" {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := a.DB.Exec(`
		INSERT INTO players (instance_id, username, first_seen_at, last_seen_at, last_joined_at, current_session_started_at, is_online)
		VALUES (?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(instance_id, username) DO UPDATE SET
			last_seen_at=excluded.last_seen_at,
			last_joined_at=excluded.last_joined_at,
			current_session_started_at=excluded.current_session_started_at,
			is_online=1`,
		instID, name, now, now, now, now)
	if err != nil {
		a.Logf("player join record: %v", err)
	}
}

func (a *App) recordPlayerLeave(instID int64, name string) {
	if instID == 0 || name == "" {
		return
	}
	now := time.Now().UTC()
	_, _ = a.DB.Exec(`
		UPDATE players SET
			is_online=0,
			last_left_at=?,
			last_seen_at=?,
			observed_playtime_seconds = observed_playtime_seconds + CAST(MAX(0,
				strftime('%s', ?) - strftime('%s', COALESCE(current_session_started_at, ?))) AS INTEGER),
			current_session_started_at=NULL
		WHERE instance_id=? AND username=?`,
		now.Format(time.RFC3339), now.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339),
		instID, name)
}

func (a *App) reconcileOnline(instID int64, online []string) {
	if instID == 0 {
		return
	}
	set := map[string]bool{}
	for _, n := range online {
		set[n] = true
		a.recordPlayerJoinIfNew(instID, n)
	}
	rows, err := a.DB.Query(`SELECT username FROM players WHERE instance_id=? AND is_online=1`, instID)
	if err != nil {
		return
	}
	var toLeave []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil && !set[n] {
			toLeave = append(toLeave, n)
		}
	}
	rows.Close()
	for _, n := range toLeave {
		a.recordPlayerLeave(instID, n)
	}
}

func (a *App) recordPlayerJoinIfNew(instID int64, name string) {
	var online int
	err := a.DB.QueryRow(`SELECT is_online FROM players WHERE instance_id=? AND username=?`,
		instID, name).Scan(&online)
	if err == sql.ErrNoRows || (err == nil && online == 0) {
		a.recordPlayerJoin(instID, name)
	}
}

func (a *App) audit(userID int64, username, action, target, detail, remote string) {
	a.Auth.Audit(userID, username, action, target, detail, remote)
	a.Hub.Broadcast("activity", "event", map[string]any{
		"username": username, "action": action, "target": target,
		"at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (a *App) broadcastStatus() {
	st, ps := a.Runner.State()
	payload := map[string]any{"state": st}
	if ps != nil {
		payload["detail"] = ps
	}
	a.Hub.Broadcast("overview", "status", payload)
	a.Hub.Broadcast("console", "status", payload)
}

// ActiveInstance is the exported accessor used by the CLI.
func (a *App) ActiveInstance() (*instance.Instance, error) { return a.activeInstance() }

// RunBackup is the exported backup workflow used by the CLI.
func (a *App) RunBackup(ctx context.Context, inst *instance.Instance, t backup.Type,
	mode, trigger string, by int64) (*backup.Record, error) {
	return a.runBackup(ctx, inst, t, mode, trigger, by)
}

// Handler exposes the fully wired HTTP handler (middleware, routes, SPA) so
// tests can drive the real API through httptest without opening a socket.
func (a *App) Handler() http.Handler { return a.routes() }
