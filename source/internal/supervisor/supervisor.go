// Package supervisor owns the Minecraft child process: pseudo-terminal,
// console stream, process group, runtime state and restart policy. It runs
// inside bonghos-minecraft.service; tmux is never part of this lifecycle.
package supervisor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/Chansovisoth/Bonghos/internal/config"
)

// State is the supervisor's view of the Minecraft lifecycle.
type State string

const (
	StateStopped    State = "stopped"
	StateStarting   State = "starting"
	StateRunning    State = "running"
	StateStopping   State = "stopping"
	StateRestarting State = "restarting"
	StateCrashed    State = "crashed"
)

// Classification of a terminated session.
const (
	ClassCleanStop        = "clean_stop"
	ClassRequestedRestart = "requested_restart"
	ClassCrash            = "crash"
	ClassUnclean          = "unclean"
)

// Config for one supervised run.
type Config struct {
	Home                string
	InstanceID          int64
	ServerDir           string // absolute
	StartupScript       string // relative to ServerDir
	JavaPath            string // resolved absolute java (exported as JAVA for scripts honoring it)
	RestartPolicy       string // never | on-failure | always
	RestartDelaySeconds int
	GracefulStopSeconds int
	MaxCrashLoop        int // crashes within window triggering loop protection
}

// PersistedState is written atomically to system/runtime/supervisor-state.json.
type PersistedState struct {
	InstanceID   int64  `json:"instance_id"`
	State        State  `json:"state"`
	ScriptPID    int    `json:"script_pid"`
	JavaPID      int    `json:"java_pid"`
	PGID         int    `json:"pgid"`
	StartedAt    string `json:"started_at"`
	LastExitCode int    `json:"last_exit_code"`
	LastSignal   string `json:"last_signal"`
	LastClass    string `json:"last_classification"`
	RestartCount int    `json:"restart_count"`
	UpdatedAt    string `json:"updated_at"`
}

// Supervisor manages one Minecraft process at a time.
type Supervisor struct {
	cfg Config

	mu            sync.Mutex
	state         State
	cmd           *exec.Cmd
	tty           *os.File
	scriptPID     int
	pgid          int
	startedAt     time.Time
	lastExitCode  int
	lastSignal    string
	lastClass     string
	restartCount  int
	stopIntent    bool // requested shutdown overrides restart policy
	restartIntent bool

	crashTimes []time.Time

	// console fan-out
	subMu       sync.Mutex
	subs        map[chan string]bool
	history     []string // bounded recent console history
	historySize int      // approximate bytes held in history

	// Commands Bonghos issues for its own bookkeeping (player polling) are
	// hidden from the user-facing console: their echo and reply are still
	// parsed and logged, but repeating "There are 0 of a max of 10 players
	// online" every twelve seconds buries whatever the operator was reading.
	suppressMu    sync.Mutex
	suppressUntil time.Time
	suppressRe    *regexp.Regexp
	suppressEcho  string

	onLine  func(line string) // log-parser hook
	onState func(s State, ps PersistedState)
}

const (
	historyLines = 500
	// historyBytes bounds the replay buffer in bytes as well as lines. The
	// console protocol caps a frame at 64 KiB, and 500 lines of modded boot
	// output comfortably exceeds that: the frame was rejected and connecting
	// clients silently received no backlog at all. Staying under the frame
	// limit keeps replay working; clients wanting more read the log file.
	historyBytes = 48 * 1024
)

func New(cfg Config) *Supervisor {
	if cfg.GracefulStopSeconds <= 0 {
		cfg.GracefulStopSeconds = 120
	}
	if cfg.RestartDelaySeconds <= 0 {
		cfg.RestartDelaySeconds = 10
	}
	if cfg.MaxCrashLoop <= 0 {
		cfg.MaxCrashLoop = 5
	}
	return &Supervisor{cfg: cfg, state: StateStopped, subs: map[chan string]bool{}}
}

func (s *Supervisor) OnLine(fn func(string))                 { s.onLine = fn }
func (s *Supervisor) OnState(fn func(State, PersistedState)) { s.onState = fn }
func (s *Supervisor) State() State                           { s.mu.Lock(); defer s.mu.Unlock(); return s.state }
func (s *Supervisor) Snapshot() PersistedState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Supervisor) snapshotLocked() PersistedState {
	ps := PersistedState{
		InstanceID: s.cfg.InstanceID, State: s.state,
		ScriptPID: s.scriptPID, JavaPID: s.findJavaPID(), PGID: s.pgid,
		LastExitCode: s.lastExitCode, LastSignal: s.lastSignal,
		LastClass: s.lastClass, RestartCount: s.restartCount,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if !s.startedAt.IsZero() {
		ps.StartedAt = s.startedAt.UTC().Format(time.RFC3339)
	}
	return ps
}

func (s *Supervisor) persist() {
	ps := s.Snapshot()
	data, _ := json.MarshalIndent(ps, "", "  ")
	config.AtomicWrite(filepath.Join(s.cfg.Home, config.FileSupState), data, 0o600)
	if s.onState != nil {
		s.onState(ps.State, ps)
	}
}

// Run is the supervisor main loop: start the script, supervise, and apply the
// restart policy until ctx is cancelled (systemd stop) or a terminal state.
func (s *Supervisor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		err := s.startOnce(ctx)
		if err != nil {
			s.mu.Lock()
			s.state = StateCrashed
			s.lastClass = ClassCrash
			s.mu.Unlock()
			s.persist()
			s.broadcast(fmt.Sprintf("[bonghos] start failed: %v", err))
		}
		exitCode, sig := s.wait(ctx)

		s.mu.Lock()
		s.lastExitCode = exitCode
		s.lastSignal = sig
		stopReq := s.stopIntent
		restartReq := s.restartIntent
		s.stopIntent = false
		s.restartIntent = false
		policy := s.cfg.RestartPolicy

		switch {
		case restartReq:
			s.lastClass = ClassRequestedRestart
			s.state = StateRestarting
		case stopReq || ctx.Err() != nil:
			s.lastClass = ClassCleanStop
			s.state = StateStopped
		case exitCode == 0:
			s.lastClass = ClassCleanStop
			s.state = StateStopped
		default:
			s.lastClass = ClassCrash
			s.state = StateCrashed
			s.crashTimes = append(s.crashTimes, time.Now())
		}
		state := s.state
		s.mu.Unlock()
		s.persist()

		// decide restart
		switch {
		case ctx.Err() != nil:
			return nil
		case state == StateRestarting:
			// fall through to restart after delay
		case state == StateStopped:
			// requested stop or clean exit: never restart, regardless of policy
			return nil
		case state == StateCrashed:
			if policy == "never" {
				return nil
			}
			if policy == "on-failure" || policy == "always" {
				if s.crashLooping() {
					s.broadcast("[bonghos] crash-loop protection engaged; automatic restarts paused")
					return errors.New("crash loop detected")
				}
			}
		}
		s.mu.Lock()
		s.restartCount++
		delay := time.Duration(s.cfg.RestartDelaySeconds) * time.Second
		// exponential backoff on repeated crashes
		recent := len(s.recentCrashes(10 * time.Minute))
		for i := 1; i < recent && i < 5; i++ {
			delay *= 2
		}
		s.mu.Unlock()
		s.broadcast(fmt.Sprintf("[bonghos] restarting in %s", delay))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
}

func (s *Supervisor) recentCrashes(window time.Duration) []time.Time {
	cutoff := time.Now().Add(-window)
	var out []time.Time
	for _, t := range s.crashTimes {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}

func (s *Supervisor) crashLooping() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.recentCrashes(10*time.Minute)) >= s.cfg.MaxCrashLoop
}

// startOnce launches the startup script on a PTY in its own process group.
func (s *Supervisor) startOnce(ctx context.Context) error {
	s.mu.Lock()
	if s.state == StateRunning || s.state == StateStarting {
		s.mu.Unlock()
		return errors.New("already running")
	}
	s.state = StateStarting
	s.startedAt = time.Now()
	s.mu.Unlock()
	s.persist()

	script := filepath.Join(s.cfg.ServerDir, s.cfg.StartupScript)
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("startup script missing: %w", err)
	}
	cmd := exec.Command("/bin/bash", script)
	cmd.Dir = s.cfg.ServerDir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if s.cfg.JavaPath != "" {
		javaDir := filepath.Dir(s.cfg.JavaPath)
		cmd.Env = append(cmd.Env,
			"JAVA="+s.cfg.JavaPath,
			"JAVA_HOME="+filepath.Dir(javaDir),
			"PATH="+javaDir+":"+os.Getenv("PATH"),
		)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // own session+pgid; pty gives controlling terminal

	tty, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("starting under pty: %w", err)
	}
	pty.Setsize(tty, &pty.Winsize{Rows: 40, Cols: 160})

	s.mu.Lock()
	s.cmd = cmd
	s.tty = tty
	s.scriptPID = cmd.Process.Pid
	s.pgid, _ = syscall.Getpgid(cmd.Process.Pid)
	s.state = StateStarting
	s.mu.Unlock()
	s.persist()

	go s.readConsole(tty)
	return nil
}

// readConsole streams PTY output to subscribers, history and log files.
func (s *Supervisor) readConsole(tty *os.File) {
	logPath := filepath.Join(s.cfg.ServerDir, "logs", "bonghos-console.log")
	os.MkdirAll(filepath.Dir(logPath), 0o755)
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if logFile != nil {
		defer logFile.Close()
	}
	sc := bufio.NewScanner(tty)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		// The raw PTY text goes to the log file so nothing is lost for
		// debugging, but everything the UI sees is sanitized: escape
		// sequences are meaningless outside a terminal and would otherwise
		// be stored in the history buffer and rendered in the browser.
		raw := sc.Text()
		if logFile != nil {
			fmt.Fprintln(logFile, SanitizeLine(raw))
		}
		line := SanitizeLine(raw)
		if line == "" && strings.TrimSpace(raw) != "" {
			continue // the line was entirely control sequences
		}
		// Parsing happens for every line; only the console stream is filtered.
		if s.onLine != nil {
			s.onLine(line)
		}
		if !s.suppressed(line) {
			s.broadcast(line)
		}
		// running detection: "Done (12.3s)!" from the server
		if strings.Contains(line, "Done (") && strings.Contains(line, ")!") {
			s.mu.Lock()
			if s.state == StateStarting {
				s.state = StateRunning
			}
			s.mu.Unlock()
			s.persist()
		}
	}
}

func (s *Supervisor) broadcast(line string) {
	s.subMu.Lock()
	s.history = append(s.history, line)
	s.historySize += len(line) + 1
	for len(s.history) > historyLines || (s.historySize > historyBytes && len(s.history) > 1) {
		s.historySize -= len(s.history[0]) + 1
		s.history = s.history[1:]
	}
	for ch := range s.subs {
		select {
		case ch <- line:
		default: // slow client: drop rather than block Minecraft
		}
	}
	s.subMu.Unlock()
}

// Subscribe returns recent history plus a live channel; call the cancel func
// when the client disconnects.
func (s *Supervisor) Subscribe() (history []string, ch chan string, cancel func()) {
	ch = make(chan string, 256)
	s.subMu.Lock()
	history = append([]string(nil), s.history...)
	s.subs[ch] = true
	s.subMu.Unlock()
	return history, ch, func() {
		s.subMu.Lock()
		delete(s.subs, ch)
		s.subMu.Unlock()
	}
}

// SendInternalCommand issues a command Bonghos needs for its own bookkeeping
// and hides its echo and reply from the user-facing console for a short
// window. The output is still written to the log file and still passed to the
// log parser, so player tracking is unaffected.
func (s *Supervisor) SendInternalCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	reply, ok := internalReplies[cmd]
	if !ok {
		// Only known bookkeeping commands may be hidden. Anything else is
		// treated as an ordinary command so nothing can be issued invisibly.
		return s.SendCommand(cmd)
	}
	s.suppressMu.Lock()
	s.suppressUntil = time.Now().Add(5 * time.Second)
	s.suppressRe = reply
	s.suppressEcho = cmd
	s.suppressMu.Unlock()
	return s.SendCommand(cmd)
}

// internalReplies maps a bookkeeping command to the reply it produces. Keeping
// this list closed means an operator's own commands can never be suppressed.
var internalReplies = map[string]*regexp.Regexp{
	"list": regexp.MustCompile(`players online|There are \d+ of a max`),
}

// suppressed reports whether a console line came from an internal command and
// should be kept out of the console stream and history.
func (s *Supervisor) suppressed(line string) bool {
	s.suppressMu.Lock()
	defer s.suppressMu.Unlock()
	if s.suppressUntil.IsZero() || time.Now().After(s.suppressUntil) {
		return false
	}
	trimmed := strings.TrimSpace(line)
	// The terminal echoes the command itself before the server replies.
	if s.suppressEcho != "" && (trimmed == s.suppressEcho || strings.HasSuffix(trimmed, "> "+s.suppressEcho)) {
		return true
	}
	if s.suppressRe != nil && s.suppressRe.MatchString(line) {
		// The reply has arrived; stop suppressing so a command the operator
		// types themselves is never hidden.
		s.suppressUntil = time.Time{}
		return true
	}
	return false
}

// SendCommand forwards a Minecraft console command. Input is rejected while
// the server is not running/starting; a Linux shell is never reachable.
func (s *Supervisor) SendCommand(cmd string) error {
	cmd = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, cmd))
	if cmd == "" {
		return errors.New("empty command")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateRunning && s.state != StateStarting && s.state != StateStopping {
		return errors.New("server is not running; command rejected")
	}
	if s.tty == nil {
		return errors.New("console not attached")
	}
	_, err := s.tty.WriteString(cmd + "\n")
	return err
}

// wait blocks until the child exits, returning (exitCode, signalName).
func (s *Supervisor) wait(ctx context.Context) (int, string) {
	s.mu.Lock()
	cmd := s.cmd
	tty := s.tty
	s.mu.Unlock()
	if cmd == nil {
		return -1, ""
	}
	err := cmd.Wait()
	if tty != nil {
		tty.Close()
	}
	s.mu.Lock()
	s.cmd = nil
	s.tty = nil
	s.mu.Unlock()

	// ensure no orphans: kill the whole process group if anything survived
	if s.pgid > 0 {
		syscall.Kill(-s.pgid, syscall.SIGKILL)
	}

	if err == nil {
		return 0, ""
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				return 128 + int(ws.Signal()), ws.Signal().String()
			}
			return ws.ExitStatus(), ""
		}
	}
	return -1, ""
}

// Stop performs a graceful stop: save-all flush, stop, wait, escalate.
func (s *Supervisor) Stop(graceful bool) error {
	s.mu.Lock()
	if s.state == StateStopped || s.state == StateCrashed {
		s.mu.Unlock()
		return nil
	}
	s.stopIntent = true
	s.state = StateStopping
	tty := s.tty
	pgid := s.pgid
	pid := s.scriptPID
	s.mu.Unlock()
	s.persist()

	if graceful && tty != nil {
		tty.WriteString("save-all flush\n")
		time.Sleep(2 * time.Second)
		tty.WriteString("stop\n")
		deadline := time.Now().Add(time.Duration(s.cfg.GracefulStopSeconds) * time.Second)
		for time.Now().Before(deadline) {
			if !processAlive(pid) {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	// escalate: TERM then KILL to the process group
	if pgid > 0 {
		syscall.Kill(-pgid, syscall.SIGTERM)
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if !processAlive(pid) {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
	return nil
}

// Restart flags a requested restart then performs a graceful stop; the Run
// loop starts the server again.
func (s *Supervisor) Restart() error {
	s.mu.Lock()
	if s.state != StateRunning && s.state != StateStarting {
		s.mu.Unlock()
		return errors.New("server is not running")
	}
	s.restartIntent = true
	s.mu.Unlock()
	return s.Stop(true)
}

// ForceStop kills the process group immediately.
func (s *Supervisor) ForceStop() error {
	s.mu.Lock()
	s.stopIntent = true
	pgid := s.pgid
	s.mu.Unlock()
	if pgid > 0 {
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
	return nil
}

// findJavaPID scans /proc for a java child of the script's process group.
func (s *Supervisor) findJavaPID() int {
	if s.pgid <= 0 {
		return 0
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	for _, e := range entries {
		pid := 0
		if _, err := fmt.Sscanf(e.Name(), "%d", &pid); err != nil || pid <= 0 {
			continue
		}
		commData, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil || strings.TrimSpace(string(commData)) != "java" {
			continue
		}
		if pgid, err := syscall.Getpgid(pid); err == nil && pgid == s.pgid {
			return pid
		}
	}
	return 0
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

// LoadPersisted reads the last supervisor-state.json (may be stale).
func LoadPersisted(home string) (*PersistedState, error) {
	data, err := os.ReadFile(filepath.Join(home, config.FileSupState))
	if err != nil {
		return nil, err
	}
	var ps PersistedState
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}

// StaleCheck classifies persisted state: if the recorded PIDs are dead the
// state is stale; a prior "running" record with dead PIDs marks unclean
// shutdown (used by boot recovery).
func StaleCheck(ps *PersistedState) (stale bool, unclean bool) {
	if ps == nil {
		return true, false
	}
	alive := processAlive(ps.ScriptPID) || processAlive(ps.JavaPID)
	if alive {
		return false, false
	}
	wasRunning := ps.State == StateRunning || ps.State == StateStarting || ps.State == StateStopping
	return true, wasRunning
}
