package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/instance"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/runtime/console"
	"github.com/Chansovisoth/Bonghos/internal/runtime/systemd"
	"github.com/Chansovisoth/Bonghos/internal/supervisor"
)

// Runner bridges the control plane to the systemd-managed supervisor process.
// It never owns the Minecraft process itself; it starts/stops the
// bonghos-minecraft user service (or a detached supervisor as fallback) and
// talks to the running supervisor over its protected Unix socket.
type Runner struct {
	app *App

	mu        sync.Mutex
	busy      string // active lifecycle operation name ("" when idle)
	console   *console.Client
	consoleWG sync.WaitGroup
}

var ErrBusy = errors.New("another lifecycle operation is in progress")

func newRunner(a *App) *Runner { return &Runner{app: a} }

func (r *Runner) acquire(op string) (func(), error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.busy != "" {
		return nil, fmt.Errorf("%w (%s)", ErrBusy, r.busy)
	}
	r.busy = op
	return func() { r.mu.Lock(); r.busy = ""; r.mu.Unlock() }, nil
}

// State reads the persisted supervisor state, verifying process liveness.
func (r *Runner) State() (string, *supervisor.PersistedState) {
	ps, err := supervisor.LoadPersisted(r.app.Home)
	if err != nil || ps == nil {
		return "stopped", nil
	}
	stale, _ := supervisor.StaleCheck(ps)
	if stale {
		return "stopped", ps
	}
	return string(ps.State), ps
}

// Online reports whether a live Minecraft process is running.
func (r *Runner) Online() bool {
	st, _ := r.State()
	return st == "running" || st == "starting" || st == "stopping" || st == "restarting"
}

// Start launches the supervisor for the active instance.
func (r *Runner) Start(ctx context.Context) error {
	release, err := r.acquire("start")
	if err != nil {
		return err
	}
	defer release()
	return r.startLocked(ctx)
}

func (r *Runner) startLocked(ctx context.Context) error {
	if r.Online() {
		return errors.New("server is already running")
	}

	inst, err := r.app.activeInstance()
	if err != nil {
		return err
	}
	if err := r.validateStartable(inst); err != nil {
		return err
	}

	// Clear verified stale state before starting.
	if ps, _ := supervisor.LoadPersisted(r.app.Home); ps != nil {
		if stale, _ := supervisor.StaleCheck(ps); stale {
			_ = os.Remove(filepath.Join(r.app.Home, config.FileSupState))
			_ = os.Remove(filepath.Join(r.app.Home, config.FileSupSocket))
		}
	}

	if systemd.Available() {
		if err := systemd.Start(systemd.ServiceMinecraft); err != nil {
			return fmt.Errorf("starting %s: %w", systemd.ServiceMinecraft, err)
		}
	} else {
		if err := r.spawnDetachedSupervisor(); err != nil {
			return err
		}
	}
	r.app.Instances.TouchStarted(inst.ID)
	r.app.resetPhases()
	r.app.recordEvent(inst.ID, CatLifecycle, "starting", SevInfo,
		"Starting "+inst.DisplayName, nil)
	r.app.broadcastStatus()

	// Wait briefly for the supervisor to come alive.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if st, _ := r.State(); st != "stopped" {
			go r.attachConsole()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return errors.New("supervisor did not report startup within 20s; check logs")
}

func (r *Runner) validateStartable(inst *instance.Instance) error {
	dir := inst.AbsoluteDir(r.app.Home)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("server directory missing: %s", dir)
	}
	if inst.StartupScript == "" {
		return errors.New("no startup script selected for this project")
	}
	if _, err := os.Stat(filepath.Join(dir, inst.StartupScript)); err != nil {
		return fmt.Errorf("startup script not found: %s", inst.StartupScript)
	}
	if _, err := minecraft.ResolveJava(inst.JavaSelection); err != nil {
		return fmt.Errorf("java selection %q: %w", inst.JavaSelection, err)
	}
	eula := filepath.Join(dir, "eula.txt")
	if b, err := os.ReadFile(eula); err != nil || !containsEulaTrue(string(b)) {
		return errors.New("the Minecraft EULA has not been accepted (eula.txt must contain eula=true)")
	}
	return nil
}

func containsEulaTrue(s string) bool {
	for _, line := range splitLines(s) {
		line = trimSpace(line)
		if line == "eula=true" || line == "eula = true" {
			return true
		}
	}
	return false
}

// spawnDetachedSupervisor is the non-systemd fallback: a detached child
// running `bonghos supervisor` in its own session.
func (r *Runner) spawnDetachedSupervisor() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "--home", r.app.Home, "supervisor")
	cmd.Dir = r.app.Home
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	logPath := filepath.Join(r.app.Home, config.DirLogs, "supervisor-spawn.log")
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
		cmd.Stdout, cmd.Stderr = f, f
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // reap
	return nil
}

// Stop performs a graceful stop through the supervisor.
// noteStopped records a clean stop once the process has actually exited.
func (r *Runner) noteStopped(reason string) {
	if inst, err := r.app.activeInstance(); err == nil {
		r.app.recordEvent(inst.ID, CatLifecycle, "stopped", SevInfo, reason, nil)
	}
}

func (r *Runner) Stop(ctx context.Context) error {
	release, err := r.acquire("stop")
	if err != nil {
		return err
	}
	defer release()
	return r.stopLocked(ctx)
}

func (r *Runner) stopLocked(ctx context.Context) error {
	if !r.Online() {
		return errors.New("server is not running")
	}
	if c, err := console.Dial(r.app.Home); err == nil {
		defer c.Close()
		_ = c.Control("stop")
	} else if systemd.Available() {
		// systemd stop → SIGTERM → supervisor graceful stop
		_ = systemd.Stop(systemd.ServiceMinecraft)
	}
	// Wait for exit.
	timeout := time.Duration(r.app.Cfg.GracefulStopSeconds+30) * time.Second
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if st, _ := r.State(); st == "stopped" || st == "crashed" {
			if inst, err := r.app.activeInstance(); err == nil {
				r.app.Instances.TouchStopped(inst.ID)
			}
			// Also stop the unit so systemd does not restart the supervisor.
			if systemd.Available() {
				_ = systemd.Stop(systemd.ServiceMinecraft)
			}
			r.app.broadcastStatus()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return errors.New("graceful stop timed out; use Force Stop if required")
}

// Restart performs a complete stop before starting a fresh supervisor. Keeping
// the lifecycle lock across both phases prevents another operation from
// slipping in while the server is stopped between them.
func (r *Runner) Restart(ctx context.Context) error {
	release, err := r.acquire("restart")
	if err != nil {
		return err
	}
	defer release()

	if r.Online() {
		if err := r.stopLocked(ctx); err != nil {
			return fmt.Errorf("stopping server for restart: %w", err)
		}
	}
	if err := r.startLocked(ctx); err != nil {
		return fmt.Errorf("server stopped but could not start again: %w", err)
	}
	return nil
}

// ForceStop kills the process group after confirmation upstream.
func (r *Runner) ForceStop() error {
	release, err := r.acquire("force-stop")
	if err != nil {
		return err
	}
	defer release()

	if c, err := console.Dial(r.app.Home); err == nil {
		_ = c.Control("force-stop")
		c.Close()
	}
	// Escalate: kill by persisted PGID if still alive.
	if ps, _ := supervisor.LoadPersisted(r.app.Home); ps != nil && ps.PGID > 0 {
		_ = syscall.Kill(-ps.PGID, syscall.SIGKILL)
	}
	if systemd.Available() {
		_ = systemd.Stop(systemd.ServiceMinecraft)
	}
	if inst, err := r.app.activeInstance(); err == nil {
		r.app.Instances.TouchStopped(inst.ID)
		r.app.recordEvent(inst.ID, CatLifecycle, "force_stopped", SevWarning,
			"Force stopped; recent world changes may be lost", nil)
	}
	r.app.broadcastStatus()
	return nil
}

// SendCommand forwards a Minecraft console command through the supervisor.
// SendInternalCommand issues a bookkeeping command (player polling) whose echo
// and reply are hidden from the console stream but still parsed and logged.
func (r *Runner) SendInternalCommand(cmd string) error {
	r.mu.Lock()
	c := r.console
	r.mu.Unlock()
	if c != nil {
		if err := c.SendInternal(cmd); err == nil {
			return nil
		}
	}
	nc, err := console.Dial(r.app.Home)
	if err != nil {
		return errors.New("server console is not available (is the server running?)")
	}
	defer nc.Close()
	return nc.SendInternal(cmd)
}

func (r *Runner) SendCommand(cmd string) error {
	r.mu.Lock()
	c := r.console
	r.mu.Unlock()
	if c != nil {
		if err := c.Send(cmd); err == nil {
			return nil
		}
	}
	nc, err := console.Dial(r.app.Home)
	if err != nil {
		return errors.New("server console is not available (is the server running?)")
	}
	defer nc.Close()
	return nc.Send(cmd)
}

// attachConsole maintains a persistent console client that fans supervisor
// output into the websocket hub and log parser.
func (r *Runner) attachConsole() {
	r.mu.Lock()
	if r.console != nil {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	for attempt := 0; attempt < 40; attempt++ {
		c, err := console.Dial(r.app.Home)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		r.mu.Lock()
		r.console = c
		r.mu.Unlock()
		r.app.resetConsoleHistory()

		r.consoleWG.Add(1)
		go func() {
			defer r.consoleWG.Done()
			defer func() {
				r.mu.Lock()
				r.console = nil
				r.mu.Unlock()
				c.Close()
				r.app.broadcastStatus()
			}()
			for {
				m, err := c.Read()
				if err != nil {
					return
				}
				switch m.Type {
				case "history":
					for _, ln := range m.Lines {
						r.app.handleConsoleLine(ln, false)
					}
				case "line":
					r.app.handleConsoleLine(m.Line, true)
				case "status":
					r.app.broadcastStatus()
				}
			}
		}()
		return
	}
}

// ---------------------------------------------------------------------------
// Scheduler executor
// ---------------------------------------------------------------------------

// executor adapts the runner + backup manager to scheduler.Executor.
type executor struct{ app *App }

func (e *executor) ServerOnline() bool { st, _ := e.app.Runner.State(); return st == "running" }

func (e *executor) StartServer(ctx context.Context) error { return e.app.Runner.Start(ctx) }
func (e *executor) StopServer(ctx context.Context) error  { return e.app.Runner.Stop(ctx) }
func (e *executor) RestartServer(ctx context.Context) error {
	return e.app.Runner.Restart(ctx)
}
func (e *executor) SendCommand(cmd string) error { return e.app.Runner.SendCommand(cmd) }
func (e *executor) Broadcast(msg string) error {
	return e.app.Runner.SendCommand("say " + msg)
}
func (e *executor) SaveAll() error { return e.app.Runner.SendCommand("save-all flush") }

func (e *executor) CreateBackup(ctx context.Context, instanceID int64, backupType string) error {
	inst, err := e.app.Instances.ByID(instanceID)
	if err != nil {
		return err
	}
	mode := "offline"
	if e.ServerOnline() {
		mode = "online"
	}
	_, err = e.app.runBackup(ctx, inst, backup.Type(backupType), mode, "scheduled", 0)
	return err
}

func (e *executor) OperationInProgress() bool {
	e.app.Runner.mu.Lock()
	busy := e.app.Runner.busy != ""
	e.app.Runner.mu.Unlock()
	if busy {
		return true
	}
	ops, _ := e.app.Operations.List(true)
	return len(ops) > 0
}

// handleConsoleLine parses a console line for player events and fans it out.
func (a *App) handleConsoleLine(line string, live bool) {
	a.appendConsoleHistory(line)
	if live {
		a.Hub.Broadcast("console", "line", map[string]any{
			"line": line,
			"at":   time.Now().UTC().Format(time.RFC3339),
		})
	}
	// Classify startup progress and common failures into timeline events, so
	// the interface can say what is happening instead of leaving the operator
	// to read raw console output and guess.
	if live {
		if event, msg, ok := startupPhase(line); ok {
			a.notePhase(event, msg)
		}
	}

	ev := minecraft.ParseLogLine(line)
	if ev == nil {
		return
	}
	instID := a.activeInstanceIDQuiet()
	switch ev.Kind {
	case "joined":
		a.recordPlayerJoin(instID, ev.Player)
		a.Hub.Broadcast("players", "joined", map[string]any{"username": ev.Player})
	case "left":
		a.recordPlayerLeave(instID, ev.Player)
		a.Hub.Broadcast("players", "left", map[string]any{"username": ev.Player})
	case "list":
		a.reconcileOnline(instID, ev.Online)
		a.Hub.Broadcast("players", "list", map[string]any{"players": ev.Online})
	case "done":
		a.recordEvent(instID, CatLifecycle, "ready", SevInfo,
			"Server is ready and accepting players", nil)
		a.Hub.Broadcast("overview", "started", nil)
	}
}

func (a *App) activeInstanceIDQuiet() int64 {
	id, err := a.Instances.ActiveID()
	if err != nil {
		return 0
	}
	return id
}

// ---------------------------------------------------------------------------

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

var _ = json.Marshal // placeholder to keep import if unused later
