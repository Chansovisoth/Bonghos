// Package app is the Bonghos control plane: HTTP + WebSocket API, embedded
// Web UI, and coordination of imports, lifecycle, schedules, backups and
// monitoring. Minecraft itself runs under the separate supervisor process.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/bot"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/monitoring"
	"github.com/Chansovisoth/Bonghos/internal/operations"
	"github.com/Chansovisoth/Bonghos/internal/scheduler"
	"github.com/Chansovisoth/Bonghos/internal/security"
	"github.com/Chansovisoth/Bonghos/internal/supervisor"
	"github.com/Chansovisoth/Bonghos/internal/turnstile"
	"github.com/Chansovisoth/Bonghos/internal/websocket"
)

// Version is stamped at build time via -ldflags.
var Version = "0.2.0-dev"

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
	Bots       *bot.Store
	Turnstile  *turnstile.Service
	BotNotify  *bot.Dispatcher
	Sched      *scheduler.Scheduler
	Hub        *websocket.Hub
	Runner     *Runner

	// Startup phases already reported for the current server run.
	phaseMu                 sync.Mutex
	seenPhases              map[string]bool
	consoleMu               sync.Mutex
	consoleHistory          []string
	Collector               *monitoring.Collector
	InternetTester          *monitoring.InternetTester
	InternetMonitor         *monitoring.InternetMonitor
	storageMu               sync.RWMutex
	storageSnapshot         monitoring.StorageSnapshot
	internetSpeedMu         sync.Mutex
	uploadMu                sync.Mutex
	botLifecycleMu          sync.Mutex
	botSawOnline            bool
	botReadySent            bool
	botStoppedSent          bool
	serverStateMu           sync.Mutex
	lastServerState         string
	lastServerStateInstance int64
	passkeyMu               sync.Mutex
	passkeyFlows            map[string]passkeyFlow
	accountMu               sync.Mutex
	accountActions          map[string]accountAction
	totpEnrollments         map[string]totpEnrollment

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
	backupRoot := config.BackupRoot(home, cfg)
	absHome, absErr := filepath.Abs(home)
	absBackupRoot, rootErr := filepath.Abs(backupRoot)
	absDefaultRoot, defaultErr := filepath.Abs(config.DefaultBackupRoot(home))
	if absErr != nil || rootErr != nil || defaultErr != nil {
		return nil, errors.Join(absErr, rootErr, defaultErr)
	}
	if filepath.Clean(absBackupRoot) != filepath.Clean(absDefaultRoot) && security.WithinRoot(absHome, absBackupRoot) {
		return nil, fmt.Errorf("custom backup directory must be outside BONGHOS_HOME; use %s for internal backups", absDefaultRoot)
	}
	backupRoot = absBackupRoot
	if st, err := os.Lstat(backupRoot); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("configured backup directory cannot be a symbolic link: %s", backupRoot)
	}
	if st, err := os.Stat(backupRoot); err != nil || !st.IsDir() {
		if strings.TrimSpace(cfg.BackupDirectory) != "" {
			return nil, fmt.Errorf("configured backup directory is unavailable: %s", backupRoot)
		}
		if err := os.MkdirAll(backupRoot, 0o755); err != nil {
			return nil, fmt.Errorf("creating backup directory: %w", err)
		}
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
		Home:            home,
		Cfg:             cfg,
		DB:              db,
		SecretKey:       key,
		WebFS:           webFS,
		passkeyFlows:    make(map[string]passkeyFlow),
		accountActions:  make(map[string]accountAction),
		totpEnrollments: make(map[string]totpEnrollment),
		Logf: func(format string, args ...any) {
			logger.Printf(format, args...)
		},
	}
	a.Auth = &auth.Store{DB: db, SecretKey: key, Sessions: time.Duration(cfg.SessionHours) * time.Hour}
	a.Instances = &instance.Store{DB: db}
	a.Operations = operations.NewStore(db)
	a.OpLock = operations.NewLock(home)
	a.Backups = &backup.Manager{
		Home: home, Root: backupRoot, DB: db,
		FreeSpaceReserve: cfg.FreeSpaceReserveMB << 20,
	}
	a.Bots = &bot.Store{DB: db, SecretKey: key}
	a.Turnstile = &turnstile.Service{Store: &turnstile.Store{DB: db, SecretKey: key}}
	a.BotNotify = &bot.Dispatcher{Store: a.Bots, Sender: bot.NewSender(), Logf: a.Logf}
	a.Hub = websocket.NewHub()
	a.Runner = newRunner(a)
	a.Collector = monitoring.NewCollector()
	a.InternetTester = monitoring.NewInternetTester()
	a.InternetMonitor = monitoring.NewInternetMonitor(a.InternetTester)
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
	state, processState := a.Runner.State()
	a.observeServerState(state, processState)

	go a.Sched.Run(ctx)
	go a.metricsLoop(ctx)
	go a.playerPollLoop(ctx)
	go a.BotNotify.RunTelegramCommands(ctx)
	go a.BotNotify.RunDiscordCommands(ctx)
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
	historyInterval := time.Duration(a.Cfg.MetricsIntervalSec) * time.Second
	if historyInterval < 2*time.Second {
		historyInterval = 10 * time.Second
	}
	pulse := time.NewTicker(1 * time.Second)
	defer pulse.Stop()
	pruneT := time.NewTicker(1 * time.Hour)
	defer pruneT.Stop()
	var lastSampleAt time.Time
	var lastHistoryAt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-pruneT.C:
			monitoring.Prune(a.DB, time.Duration(a.Cfg.MetricsRetentionDays)*24*time.Hour)
		case now := <-pulse.C:
			// Temporarily collect faster when an authorized live Overview/sidebar
			// or open Performance view asks for a more responsive feed. Persisted history
			// stays on the configured cadence so live dashboards do not multiply
			// database writes.
			effective := historyInterval
			for _, topic := range []string{"overview_performance", "performance"} {
				if a.Hub.SubscriberCount(topic) == 0 {
					continue
				}
				if requested := a.Hub.MinimumInterval(topic, historyInterval); requested < effective {
					effective = requested
				}
			}
			if !lastSampleAt.IsZero() && now.Sub(lastSampleAt)+250*time.Millisecond < effective {
				continue
			}
			state, processState := a.Runner.State()
			a.observeServerState(state, processState)
			s := a.collectSample()
			if lastHistoryAt.IsZero() || now.Sub(lastHistoryAt)+250*time.Millisecond >= historyInterval {
				instID := a.activeInstanceIDQuiet()
				monitoring.StoreSample(a.DB, instID, s)
				lastHistoryAt = now
			}
			lastSampleAt = now
			a.Hub.BroadcastDue("performance", "sample", s, historyInterval)
			a.Hub.BroadcastDue("overview_performance", "sample", s, historyInterval)
		}
	}
}

func (a *App) collectSample() *monitoring.Sample {
	s := &monitoring.Sample{CollectedAt: time.Now().UTC().Format(time.RFC3339)}
	s.HostCPUPercent, s.CPUCores = a.Collector.HostCPU()
	var coreTemperatures map[int]float64
	s.CPUTempCelsius, coreTemperatures = monitoring.CPUTemperatures()
	for index := range s.CPUCores {
		if temperature, ok := coreTemperatures[s.CPUCores[index].Index]; ok {
			temperatureCopy := temperature
			s.CPUCores[index].TempCelsius = &temperatureCopy
		}
	}
	state, ps := a.Runner.State()
	processActive := state == "starting" || state == "running" || state == "stopping"
	if processActive && ps != nil && ps.JavaPID > 0 {
		s.JavaPID = ps.JavaPID
		s.CPUPercent = a.Collector.ProcessCPU(ps.JavaPID)
		s.RSSBytes = monitoring.ProcessRSS(ps.JavaPID)
		if t, err := time.Parse(time.RFC3339, ps.StartedAt); err == nil {
			s.UptimeSeconds = int64(time.Since(t).Seconds())
		}
	}
	s.HostMemTotal, s.HostMemAvail = monitoring.HostMemory()
	s.Load1 = monitoring.LoadAvg()
	storage := a.cachedStorageSnapshot()
	s.DiskTotal = storage.DiskTotal
	s.DiskFree = storage.DiskFree
	s.BonghosDirBytes = storage.BonghosDirBytes
	s.ServerDirBytes = storage.ServerDirBytes
	s.BackupDirBytes = storage.BackupDirBytes
	s.SystemDirBytes = storage.SystemDirBytes
	var online int
	if inst, err := a.activeInstance(); err == nil {
		if processActive {
			_ = a.DB.QueryRow(`SELECT COUNT(*) FROM players WHERE instance_id=? AND is_online=1`, inst.ID).Scan(&online)
		}
		s.JVMXmsBytes, _ = minecraft.ParseMemoryBytes(inst.JVMXms)
		s.JVMXmxBytes, _ = minecraft.ParseMemoryBytes(inst.JVMXmx)
	}
	s.OnlinePlayers = online
	return s
}

func (a *App) cachedStorageSnapshot() monitoring.StorageSnapshot {
	a.storageMu.RLock()
	defer a.storageMu.RUnlock()
	return a.storageSnapshot
}

func (a *App) collectStorageSnapshot() monitoring.StorageSnapshot {
	diskTotal, diskFree := monitoring.DiskUsage(a.Home)
	homeSize, directories := monitoring.DirectoryBreakdown(a.Home)
	snapshot := monitoring.StorageSnapshot{
		CollectedAt:     time.Now().UTC().Format(time.RFC3339),
		DiskTotal:       diskTotal,
		DiskFree:        diskFree,
		BonghosDirBytes: homeSize,
		ServerDirBytes:  directories[config.DirServers],
		BackupDirBytes:  directories[config.DirBackups],
		SystemDirBytes:  directories[config.DirSystem],
	}
	a.storageMu.Lock()
	a.storageSnapshot = snapshot
	a.storageMu.Unlock()
	return snapshot
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
			// Internal bookkeeping: the reply is parsed for the player list
			// but kept out of the operator's console.
			_ = a.Runner.SendInternalCommand("list")
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

func (a *App) markAllPlayersOffline(instID int64) int64 {
	if instID == 0 {
		return 0
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := a.DB.Exec(`
		UPDATE players SET
			is_online=0,
			last_left_at=?,
			last_seen_at=?,
			observed_playtime_seconds = observed_playtime_seconds + CAST(MAX(0,
				strftime('%s', ?) - strftime('%s', COALESCE(current_session_started_at, ?))) AS INTEGER),
			current_session_started_at=NULL
		WHERE instance_id=? AND is_online=1`,
		now, now, now, now, instID)
	if err != nil {
		a.Logf("marking players offline: %v", err)
		return 0
	}
	changed, _ := result.RowsAffected()
	return changed
}

func (a *App) reconcileOnline(instID int64, online []string) bool {
	if instID == 0 {
		return false
	}
	changed := false
	set := map[string]bool{}
	for _, n := range online {
		set[n] = true
		if a.recordPlayerJoinIfNew(instID, n) {
			changed = true
		}
	}
	rows, err := a.DB.Query(`SELECT username FROM players WHERE instance_id=? AND is_online=1`, instID)
	if err != nil {
		return changed
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
		changed = true
	}
	return changed
}

func (a *App) recordPlayerJoinIfNew(instID int64, name string) bool {
	var online int
	err := a.DB.QueryRow(`SELECT is_online FROM players WHERE instance_id=? AND username=?`,
		instID, name).Scan(&online)
	if err == sql.ErrNoRows || (err == nil && online == 0) {
		a.recordPlayerJoin(instID, name)
		return true
	}
	return false
}

func (a *App) broadcastPlayerChange(typ string, data any, overview bool) {
	a.Hub.Broadcast("players", typ, data)
	if overview {
		a.Hub.Broadcast("overview", "players", nil)
	}
}

func (a *App) observeServerState(state string, ps *supervisor.PersistedState) {
	instID := a.activeInstanceIDQuiet()
	a.serverStateMu.Lock()
	previous := a.lastServerState
	previousInstance := a.lastServerStateInstance
	a.lastServerState = state
	a.lastServerStateInstance = instID
	a.serverStateMu.Unlock()
	transition := previous != state || previousInstance != instID
	if transition && (state == "stopped" || state == "crashed" || state == "restarting") {
		if changed := a.markAllPlayersOffline(instID); changed > 0 {
			a.broadcastPlayerChange("list", map[string]any{"players": []string{}}, true)
		}
	}
	if previous == "" || !transition || previousInstance != instID || instID == 0 {
		return
	}

	switch state {
	case "stopped":
		a.recordEvent(instID, CatLifecycle, "stopped", SevInfo, "Server stopped", nil)
	case "crashed":
		detail := map[string]any{}
		if ps != nil {
			if ps.State == supervisor.StateCrashed || ps.LastExitCode != 0 {
				detail["exit_code"] = ps.LastExitCode
			}
			if ps.LastSignal != "" {
				detail["signal"] = ps.LastSignal
			}
		}
		a.recordEvent(instID, CatLifecycle, "crashed", SevError, "Server stopped unexpectedly", detail)
	}
}

func (a *App) rememberServerState(state string) {
	instID := a.activeInstanceIDQuiet()
	a.serverStateMu.Lock()
	a.lastServerState = state
	a.lastServerStateInstance = instID
	a.serverStateMu.Unlock()
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
	a.observeServerState(st, ps)
	payload := map[string]any{"state": st}
	if ps != nil {
		payload["detail"] = ps
	}
	a.Hub.Broadcast("overview", "status", payload)
	a.Hub.Broadcast("console", "status", payload)
	a.observeBotLifecycle(st)
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
