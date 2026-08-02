// Package operations persists long-running backend operations (imports,
// downloads, backups, restores) in SQLite so the UI can reconnect, and
// provides an on-disk operation lock serializing conflicting work.
package operations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/security"
)

// Stages an operation moves through.
const (
	StageQueued            = "queued"
	StageReceivingUpload   = "receiving_upload"
	StageResolvingSource   = "resolving_source"
	StageConnecting        = "connecting"
	StageDownloading       = "downloading"
	StageVerifyingDownload = "verifying_download"
	StageExtracting        = "extracting"
	StageDetectingRoot     = "detecting_server_root"
	StageDetectingStartup  = "detecting_startup_script"
	StageDetectingJava     = "detecting_java"
	StageDetectingJVM      = "detecting_jvm_configuration"
	StageValidating        = "validating_installation"
	StageMoving            = "moving_to_destination"
	StageCompleted         = "completed"
	StageFailed            = "failed"
	StageCancelled         = "cancelled"
)

// Terminal reports whether a stage is final.
func Terminal(stage string) bool {
	return stage == StageCompleted || stage == StageFailed || stage == StageCancelled
}

type Operation struct {
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	InstanceID      int64           `json:"instance_id,omitempty"`
	Stage           string          `json:"stage"`
	StatusMessage   string          `json:"status_message"`
	BytesProcessed  int64           `json:"bytes_processed"`
	TotalBytes      int64           `json:"total_bytes"`
	Percentage      float64         `json:"percentage"`
	CurrentSpeedBps float64         `json:"current_speed_bytes_per_second"`
	AverageSpeedBps float64         `json:"average_speed_bytes_per_second"`
	ETASeconds      int64           `json:"estimated_seconds_remaining"`
	CreatedBy       int64           `json:"created_by,omitempty"`
	CreatedAt       string          `json:"created_at"`
	StartedAt       string          `json:"started_at,omitempty"`
	LastProgressAt  string          `json:"last_progress_at,omitempty"`
	FinishedAt      string          `json:"finished_at,omitempty"`
	Error           string          `json:"error,omitempty"`
	Detail          json.RawMessage `json:"detail"`
}

type Store struct {
	DB *sql.DB

	mu       sync.Mutex
	cancels  map[string]func()
	speedLog map[string][2]any // opID -> [lastTime, lastBytes]
	notify   func(op *Operation)
}

func NewStore(db *sql.DB) *Store {
	return &Store{DB: db, cancels: map[string]func(){}, speedLog: map[string][2]any{}}
}

// OnUpdate registers a broadcast callback (WebSocket hub).
func (s *Store) OnUpdate(fn func(op *Operation)) { s.notify = fn }

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// Create registers a new operation and returns its generated ID.
func (s *Store) Create(kind string, instanceID, createdBy int64, detail any) (*Operation, error) {
	id, err := security.RandomToken(12)
	if err != nil {
		return nil, err
	}
	dj, _ := json.Marshal(detail)
	if detail == nil {
		dj = []byte("{}")
	}
	_, err = s.DB.Exec(`INSERT INTO operations (id, kind, instance_id, stage, created_by, created_at, detail_json)
		VALUES (?,?,?,?,?,?,?)`, id, kind, nullID(instanceID), StageQueued, nullID(createdBy), now(), string(dj))
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

// RegisterCancel makes an operation cancellable; the returned func removes it.
func (s *Store) RegisterCancel(id string, cancel func()) func() {
	s.mu.Lock()
	s.cancels[id] = cancel
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.cancels, id)
		s.mu.Unlock()
	}
}

// Cancel signals a running operation. Terminal marking is done by the worker.
func (s *Store) Cancel(id string) error {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()
	if !ok {
		return errors.New("operation is not running or not cancellable")
	}
	cancel()
	return nil
}

func (s *Store) SetStage(id, stage, msg string) {
	started := ""
	if stage != StageQueued {
		started = now()
	}
	s.DB.Exec(`UPDATE operations SET stage=?, status_message=?,
		started_at=COALESCE(started_at, NULLIF(?, '')), last_progress_at=? WHERE id=?`,
		stage, msg, started, now(), id)
	s.broadcast(id)
}

func (s *Store) Progress(id string, bytesProcessed, totalBytes int64) {
	s.DB.Exec(`UPDATE operations SET bytes_processed=?, total_bytes=?, last_progress_at=? WHERE id=?`,
		bytesProcessed, totalBytes, now(), id)
	s.broadcast(id)
}

func (s *Store) Finish(id string, stage string, errMsg string) {
	s.DB.Exec(`UPDATE operations SET stage=?, error=?, finished_at=? WHERE id=?`,
		stage, errMsg, now(), id)
	s.mu.Lock()
	delete(s.cancels, id)
	delete(s.speedLog, id)
	s.mu.Unlock()
	s.broadcast(id)
}

func (s *Store) broadcast(id string) {
	if s.notify == nil {
		return
	}
	if op, err := s.Get(id); err == nil {
		s.notify(op)
	}
}

func (s *Store) Get(id string) (*Operation, error) {
	row := s.DB.QueryRow(`SELECT id, kind, COALESCE(instance_id,0), stage, status_message,
		bytes_processed, total_bytes, COALESCE(created_by,0), created_at,
		COALESCE(started_at,''), COALESCE(last_progress_at,''), COALESCE(finished_at,''),
		error, detail_json FROM operations WHERE id=?`, id)
	return scanOp(row)
}

func (s *Store) List(activeOnly bool) ([]*Operation, error) {
	q := `SELECT id, kind, COALESCE(instance_id,0), stage, status_message,
		bytes_processed, total_bytes, COALESCE(created_by,0), created_at,
		COALESCE(started_at,''), COALESCE(last_progress_at,''), COALESCE(finished_at,''),
		error, detail_json FROM operations`
	if activeOnly {
		q += ` WHERE stage NOT IN ('completed','failed','cancelled')`
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := s.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Operation
	for rows.Next() {
		op, err := scanOp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, rows.Err()
}

// CleanStale marks non-terminal operations from before process start as failed
// (called on control-plane startup after a crash).
func (s *Store) CleanStale() {
	s.DB.Exec(`UPDATE operations SET stage='failed',
		error='interrupted by Bonghos restart', finished_at=?
		WHERE stage NOT IN ('completed','failed','cancelled')`, now())
}

func scanOp(row interface{ Scan(...any) error }) (*Operation, error) {
	var op Operation
	var detail string
	err := row.Scan(&op.ID, &op.Kind, &op.InstanceID, &op.Stage, &op.StatusMessage,
		&op.BytesProcessed, &op.TotalBytes, &op.CreatedBy, &op.CreatedAt,
		&op.StartedAt, &op.LastProgressAt, &op.FinishedAt, &op.Error, &detail)
	if err != nil {
		return nil, err
	}
	op.Detail = json.RawMessage(detail)
	if op.TotalBytes > 0 {
		op.Percentage = float64(op.BytesProcessed) / float64(op.TotalBytes) * 100
	}
	if op.StartedAt != "" && op.BytesProcessed > 0 {
		if start, err := time.Parse(time.RFC3339, op.StartedAt); err == nil {
			elapsed := time.Since(start).Seconds()
			if elapsed > 0.5 {
				op.AverageSpeedBps = float64(op.BytesProcessed) / elapsed
				op.CurrentSpeedBps = op.AverageSpeedBps
				if op.TotalBytes > op.BytesProcessed && op.AverageSpeedBps > 0 {
					op.ETASeconds = int64(float64(op.TotalBytes-op.BytesProcessed) / op.AverageSpeedBps)
				}
			}
		}
	}
	return &op, nil
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ----- operation lock --------------------------------------------------------

// Lock is a cross-process advisory lock at system/runtime/operation.lock.
// The file records holder PID + purpose; stale locks (dead PID) are reclaimed.
type Lock struct {
	path string
}

func NewLock(home string) *Lock {
	return &Lock{path: filepath.Join(home, config.FileOpLock)}
}

type lockInfo struct {
	PID     int    `json:"pid"`
	Purpose string `json:"purpose"`
	Since   string `json:"since"`
}

// Acquire takes the lock or fails describing the current holder.
func (l *Lock) Acquire(purpose string) (release func(), err error) {
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			info := lockInfo{PID: os.Getpid(), Purpose: purpose, Since: now()}
			b, _ := json.Marshal(info)
			f.Write(b)
			f.Close()
			return func() { os.Remove(l.path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		// Existing lock: stale?
		data, rerr := os.ReadFile(l.path)
		if rerr != nil {
			return nil, fmt.Errorf("operation lock unreadable: %w", rerr)
		}
		var info lockInfo
		if json.Unmarshal(data, &info) == nil && info.PID > 0 && !processAlive(info.PID) {
			os.Remove(l.path) // verified stale
			continue
		}
		return nil, fmt.Errorf("another operation holds the lock (%s since %s, pid %d)",
			info.Purpose, info.Since, info.PID)
	}
	return nil, errors.New("could not acquire operation lock")
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// On Linux, check /proc/<pid>.
	if _, err := os.Stat("/proc/" + strconv.Itoa(pid)); err == nil {
		return true
	}
	return false
}

var _ = strings.TrimSpace
