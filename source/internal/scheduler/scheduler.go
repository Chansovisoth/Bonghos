// Package scheduler runs persistent schedules entirely in the backend —
// never in browser timers. Schedules survive restarts, use database leases
// for duplicate-run protection, and honor timezone, missed-run, offline and
// conflict policies.
package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Schedule struct {
	ID                 int64           `json:"id"`
	InstanceID         int64           `json:"instance_id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Enabled            bool            `json:"enabled"`
	Timezone           string          `json:"timezone"`
	ScheduleType       string          `json:"schedule_type"`
	ScheduleExpression string          `json:"schedule_expression"`
	Action             string          `json:"action"`
	ActionPayload      json.RawMessage `json:"action_payload"`
	OfflinePolicy      string          `json:"offline_policy"`
	MissedRunPolicy    string          `json:"missed_run_policy"`
	ConflictPolicy     string          `json:"conflict_policy"`
	CreatedBy          int64           `json:"created_by,omitempty"`
	NextRunAt          string          `json:"next_run_at,omitempty"`
	LastRunAt          string          `json:"last_run_at,omitempty"`
	LastResult         string          `json:"last_result,omitempty"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

// SequenceStep is one step in a multi-step schedule (action "sequence").
type SequenceStep struct {
	OffsetSeconds int    `json:"offset_seconds"` // relative to the scheduled time (negative = before)
	Action        string `json:"action"`         // send_console_command | broadcast_message | save_all | restart_server | ...
	Payload       string `json:"payload"`
}

// Executor performs schedule actions against the running system.
type Executor interface {
	ServerOnline() bool
	StartServer(ctx context.Context) error
	StopServer(ctx context.Context) error
	RestartServer(ctx context.Context) error
	SendCommand(cmd string) error
	Broadcast(msg string) error
	SaveAll() error
	CreateBackup(ctx context.Context, instanceID int64, backupType string) error
	OperationInProgress() bool
}

type Scheduler struct {
	DB   *sql.DB
	Exec Executor
	Log  func(format string, args ...any)
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// ----- next-run computation --------------------------------------------------

// NextRun computes the next execution after 'after' for a schedule.
func NextRun(s *Schedule, after time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		loc = time.UTC
	}
	a := after.In(loc)
	expr := strings.TrimSpace(s.ScheduleExpression)
	switch s.ScheduleType {
	case "once":
		var t time.Time
		for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
			t, err = time.ParseInLocation(layout, expr, loc)
			if err == nil {
				break
			}
		}
		if err != nil {
			return time.Time{}, errors.New("once expression must be 'YYYY-MM-DD HH:MM' or 'YYYY-MM-DD HH:MM:SS'")
		}
		if t.After(a) {
			return t, nil
		}
		return time.Time{}, nil // no more runs
	case "hourly":
		// Legacy expr: "MM". Second-aware expr: "MM:SS".
		m, sec, err := parseHourly(expr)
		if err != nil {
			return time.Time{}, err
		}
		t := time.Date(a.Year(), a.Month(), a.Day(), a.Hour(), m, sec, 0, loc)
		for !t.After(a) {
			t = t.Add(time.Hour)
		}
		return t, nil
	case "daily":
		// expr: "HH:MM" or "HH:MM:SS"
		hh, mm, sec, err := parseHMS(expr)
		if err != nil {
			return time.Time{}, err
		}
		t := time.Date(a.Year(), a.Month(), a.Day(), hh, mm, sec, 0, loc)
		for !t.After(a) {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	case "weekly":
		// expr: "MON 04:00" or "MON 04:00:30" (3-letter weekday)
		parts := strings.Fields(expr)
		if len(parts) != 2 {
			return time.Time{}, errors.New("weekly expression must be 'DAY HH:MM' or 'DAY HH:MM:SS'")
		}
		wd, ok := weekdays[strings.ToUpper(parts[0])]
		if !ok {
			return time.Time{}, errors.New("unknown weekday")
		}
		hh, mm, sec, err := parseHMS(parts[1])
		if err != nil {
			return time.Time{}, err
		}
		t := time.Date(a.Year(), a.Month(), a.Day(), hh, mm, sec, 0, loc)
		for t.Weekday() != wd || !t.After(a) {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	case "monthly":
		// expr: "DD HH:MM" or "DD HH:MM:SS"
		parts := strings.Fields(expr)
		if len(parts) != 2 {
			return time.Time{}, errors.New("monthly expression must be 'DD HH:MM' or 'DD HH:MM:SS'")
		}
		dd, err := strconv.Atoi(parts[0])
		if err != nil || dd < 1 || dd > 31 {
			return time.Time{}, errors.New("day of month must be 1-31")
		}
		hh, mm, sec, err := parseHMS(parts[1])
		if err != nil {
			return time.Time{}, err
		}
		t := time.Date(a.Year(), a.Month(), dd, hh, mm, sec, 0, loc)
		for !t.After(a) || t.Day() != dd {
			t = time.Date(t.Year(), t.Month()+1, dd, hh, mm, sec, 0, loc)
		}
		return t, nil
	case "fixed_interval":
		// expr: seconds
		sec, err := strconv.Atoi(expr)
		if err != nil || sec < 60 {
			return time.Time{}, errors.New("fixed interval must be seconds >= 60")
		}
		return after.Add(time.Duration(sec) * time.Second), nil
	case "advanced_cron":
		return nextCron(expr, a)
	}
	return time.Time{}, fmt.Errorf("unknown schedule type %q", s.ScheduleType)
}

var weekdays = map[string]time.Weekday{
	"SUN": time.Sunday, "MON": time.Monday, "TUE": time.Tuesday, "WED": time.Wednesday,
	"THU": time.Thursday, "FRI": time.Friday, "SAT": time.Saturday,
}

func parseHMS(value string) (int, int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, 0, 0, errors.New("time must be HH:MM or HH:MM:SS")
	}
	hh, hourErr := strconv.Atoi(parts[0])
	mm, minuteErr := strconv.Atoi(parts[1])
	sec := 0
	var secondErr error
	if len(parts) == 3 {
		sec, secondErr = strconv.Atoi(parts[2])
	}
	if hourErr != nil || minuteErr != nil || secondErr != nil ||
		hh < 0 || hh > 23 || mm < 0 || mm > 59 || sec < 0 || sec > 59 {
		return 0, 0, 0, errors.New("time must be HH:MM or HH:MM:SS")
	}
	return hh, mm, sec, nil
}

func parseHourly(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) == 1 {
		minute, err := strconv.Atoi(parts[0])
		if err != nil || minute < 0 || minute > 59 {
			return 0, 0, errors.New("hourly expression must be MM or MM:SS")
		}
		return minute, 0, nil
	}
	if len(parts) != 2 {
		return 0, 0, errors.New("hourly expression must be MM or MM:SS")
	}
	minute, minuteErr := strconv.Atoi(parts[0])
	second, secondErr := strconv.Atoi(parts[1])
	if minuteErr != nil || secondErr != nil || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return 0, 0, errors.New("hourly expression must be MM or MM:SS")
	}
	return minute, second, nil
}

// nextCron supports the classic 5-field cron subset:
// minute hour day-of-month month day-of-week with *, lists, ranges, steps.
func nextCron(expr string, after time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, errors.New("cron expression needs 5 fields: m h dom mon dow")
	}
	mins, err := cronField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("minute: %w", err)
	}
	hours, err := cronField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("hour: %w", err)
	}
	doms, err := cronField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("day-of-month: %w", err)
	}
	mons, err := cronField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("month: %w", err)
	}
	dows, err := cronField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("day-of-week: %w", err)
	}
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := after.AddDate(2, 0, 0)
	for t.Before(limit) {
		if mons[int(t.Month())] && doms[t.Day()] && dows[int(t.Weekday())] &&
			hours[t.Hour()] && mins[t.Minute()] {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, errors.New("no matching time within two years")
}

func cronField(f string, lo, hi int) (map[int]bool, error) {
	out := map[int]bool{}
	for _, part := range strings.Split(f, ",") {
		step := 1
		if base, st, ok := strings.Cut(part, "/"); ok {
			var err error
			step, err = strconv.Atoi(st)
			if err != nil || step < 1 {
				return nil, errors.New("invalid step")
			}
			part = base
		}
		start, end := lo, hi
		if part != "*" {
			if a, b, ok := strings.Cut(part, "-"); ok {
				var err error
				start, err = strconv.Atoi(a)
				if err != nil {
					return nil, errors.New("invalid range start")
				}
				end, err = strconv.Atoi(b)
				if err != nil {
					return nil, errors.New("invalid range end")
				}
			} else {
				v, err := strconv.Atoi(part)
				if err != nil {
					return nil, errors.New("invalid value")
				}
				start, end = v, v
			}
		}
		if start < lo || end > hi || start > end {
			return nil, fmt.Errorf("value out of range %d-%d", lo, hi)
		}
		for v := start; v <= end; v += step {
			out[v] = true
		}
	}
	return out, nil
}

// ----- store -----------------------------------------------------------------

const schedCols = `id, instance_id, name, description, enabled, timezone, schedule_type,
 schedule_expression, action, action_payload_json, offline_policy, missed_run_policy,
 conflict_policy, COALESCE(created_by,0), COALESCE(next_run_at,''), COALESCE(last_run_at,''),
 last_result, created_at, updated_at`

func scanSched(row interface{ Scan(...any) error }) (*Schedule, error) {
	var s Schedule
	var enabled int
	var payload string
	err := row.Scan(&s.ID, &s.InstanceID, &s.Name, &s.Description, &enabled, &s.Timezone,
		&s.ScheduleType, &s.ScheduleExpression, &s.Action, &payload,
		&s.OfflinePolicy, &s.MissedRunPolicy, &s.ConflictPolicy, &s.CreatedBy,
		&s.NextRunAt, &s.LastRunAt, &s.LastResult, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.Enabled = enabled != 0
	s.ActionPayload = json.RawMessage(payload)
	return &s, nil
}

func (sc *Scheduler) Create(s *Schedule) error {
	next, err := NextRun(s, time.Now())
	if err != nil {
		return err
	}
	nextStr := ""
	if !next.IsZero() {
		nextStr = next.UTC().Format(time.RFC3339)
	}
	if len(s.ActionPayload) == 0 {
		s.ActionPayload = json.RawMessage("{}")
	}
	res, err := sc.DB.Exec(`INSERT INTO schedules (instance_id, name, description, enabled,
		timezone, schedule_type, schedule_expression, action, action_payload_json,
		offline_policy, missed_run_policy, conflict_policy, created_by,
		created_at, updated_at, next_run_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.InstanceID, s.Name, s.Description, b2i(s.Enabled), s.Timezone,
		s.ScheduleType, s.ScheduleExpression, s.Action, string(s.ActionPayload),
		defStr(s.OfflinePolicy, "skip_when_offline"),
		defStr(s.MissedRunPolicy, "skip_missed_run"),
		defStr(s.ConflictPolicy, "skip"),
		nullID(s.CreatedBy), now(), now(), nullStr(nextStr))
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	s.NextRunAt = nextStr
	return nil
}

func (sc *Scheduler) Update(s *Schedule) error {
	next, err := NextRun(s, time.Now())
	if err != nil {
		return err
	}
	nextStr := ""
	if !next.IsZero() {
		nextStr = next.UTC().Format(time.RFC3339)
	}
	_, err = sc.DB.Exec(`UPDATE schedules SET name=?, description=?, enabled=?, timezone=?,
		schedule_type=?, schedule_expression=?, action=?, action_payload_json=?,
		offline_policy=?, missed_run_policy=?, conflict_policy=?, updated_at=?, next_run_at=?
		WHERE id=?`,
		s.Name, s.Description, b2i(s.Enabled), s.Timezone, s.ScheduleType,
		s.ScheduleExpression, s.Action, string(s.ActionPayload),
		s.OfflinePolicy, s.MissedRunPolicy, s.ConflictPolicy, now(), nullStr(nextStr), s.ID)
	return err
}

func (sc *Scheduler) Delete(id int64) error {
	_, err := sc.DB.Exec(`DELETE FROM schedules WHERE id=?`, id)
	return err
}

func (sc *Scheduler) Get(id int64) (*Schedule, error) {
	return scanSched(sc.DB.QueryRow(`SELECT `+schedCols+` FROM schedules WHERE id=?`, id))
}

func (sc *Scheduler) List(instanceID int64) ([]*Schedule, error) {
	q := `SELECT ` + schedCols + ` FROM schedules`
	var args []any
	if instanceID > 0 {
		q += ` WHERE instance_id=?`
		args = append(args, instanceID)
	}
	q += ` ORDER BY name`
	rows, err := sc.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Schedule
	for rows.Next() {
		s, err := scanSched(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ----- run loop --------------------------------------------------------------

// Run polls due schedules once per tick until ctx is cancelled. Browser
// connections are irrelevant: this runs inside the control plane.
func (sc *Scheduler) Run(ctx context.Context) {
	sc.handleMissed()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.tick(ctx)
		}
	}
}

// handleMissed applies missed-run policies at startup.
func (sc *Scheduler) handleMissed() {
	all, err := sc.List(0)
	if err != nil {
		return
	}
	nowT := time.Now()
	for _, s := range all {
		if !s.Enabled || s.NextRunAt == "" {
			continue
		}
		nr, err := time.Parse(time.RFC3339, s.NextRunAt)
		if err != nil || nr.After(nowT) {
			continue
		}
		switch s.MissedRunPolicy {
		case "run_once_after_startup":
			go sc.execute(context.Background(), s, nr)
		default: // skip_missed_run
			sc.recordRun(s, nr, "skipped", "missed while Bonghos was not running")
		}
		sc.advance(s, nowT)
	}
}

func (sc *Scheduler) tick(ctx context.Context) {
	all, err := sc.List(0)
	if err != nil {
		return
	}
	nowT := time.Now()
	for _, s := range all {
		if !s.Enabled || s.NextRunAt == "" {
			continue
		}
		nr, err := time.Parse(time.RFC3339, s.NextRunAt)
		if err != nil || nr.After(nowT) {
			continue
		}
		s := s
		if !sc.lease(s, nr) {
			sc.advance(s, nowT)
			continue // another process/run already took it
		}
		go sc.execute(ctx, s, nr)
		sc.advance(s, nowT)
	}
}

// lease inserts a unique (schedule, planned time) row: duplicate-run protection.
func (sc *Scheduler) lease(s *Schedule, planned time.Time) bool {
	key := fmt.Sprintf("%d@%s", s.ID, planned.UTC().Format(time.RFC3339))
	_, err := sc.DB.Exec(`INSERT INTO schedule_runs (schedule_id, lease_key, planned_at, status)
		VALUES (?,?,?,'queued')`, s.ID, key, planned.UTC().Format(time.RFC3339))
	return err == nil
}

func (sc *Scheduler) advance(s *Schedule, after time.Time) {
	next, err := NextRun(s, after)
	nextStr := ""
	if err == nil && !next.IsZero() {
		nextStr = next.UTC().Format(time.RFC3339)
	}
	sc.DB.Exec(`UPDATE schedules SET next_run_at=? WHERE id=?`, nullStr(nextStr), s.ID)
}

func (sc *Scheduler) recordRun(s *Schedule, planned time.Time, status, detail string) {
	key := fmt.Sprintf("%d@%s", s.ID, planned.UTC().Format(time.RFC3339))
	sc.DB.Exec(`INSERT OR IGNORE INTO schedule_runs (schedule_id, lease_key, planned_at, status, detail)
		VALUES (?,?,?,?,?)`, s.ID, key, planned.UTC().Format(time.RFC3339), status, detail)
	sc.DB.Exec(`UPDATE schedule_runs SET status=?, detail=?, finished_at=? WHERE lease_key=?`,
		status, detail, now(), key)
	sc.DB.Exec(`UPDATE schedules SET last_run_at=?, last_result=? WHERE id=?`, now(), status, s.ID)
}

// execute performs the schedule action honoring offline and conflict policies.
// RunNow executes a schedule immediately (manual trigger), recording the run
// but leaving next_run_at untouched.
func (sc *Scheduler) RunNow(ctx context.Context, s *Schedule) {
	now := time.Now()
	if err := sc.runAction(ctx, s, now); err != nil {
		sc.recordRun(s, now, "failed", err.Error())
		return
	}
	sc.recordRun(s, now, "success", "manual run")
}

func (sc *Scheduler) execute(ctx context.Context, s *Schedule, planned time.Time) {
	logf := sc.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	key := fmt.Sprintf("%d@%s", s.ID, planned.UTC().Format(time.RFC3339))
	sc.DB.Exec(`UPDATE schedule_runs SET status='running', started_at=? WHERE lease_key=?`, now(), key)

	if sc.Exec.OperationInProgress() && s.Action != "create_backup" {
		if s.ConflictPolicy == "retry_later" {
			sc.recordRun(s, planned, "queued", "conflict: will retry")
			time.AfterFunc(2*time.Minute, func() { sc.execute(ctx, s, planned) })
			return
		}
		sc.recordRun(s, planned, "skipped", "conflicting operation in progress")
		return
	}

	needsOnline := map[string]bool{
		"send_console_command": true, "broadcast_message": true, "save_all": true,
	}
	if needsOnline[s.Action] && !sc.Exec.ServerOnline() {
		switch s.OfflinePolicy {
		case "start_then_execute":
			if err := sc.Exec.StartServer(ctx); err != nil {
				sc.recordRun(s, planned, "failed", "start before execute failed: "+err.Error())
				return
			}
		case "wait_until_online":
			deadline := time.Now().Add(10 * time.Minute)
			for !sc.Exec.ServerOnline() && time.Now().Before(deadline) {
				time.Sleep(10 * time.Second)
			}
			if !sc.Exec.ServerOnline() {
				sc.recordRun(s, planned, "skipped", "server did not come online in time")
				return
			}
		default:
			sc.recordRun(s, planned, "skipped", "server offline")
			return
		}
	}

	err := sc.runAction(ctx, s, planned)
	if err != nil {
		logf("schedule %q failed: %v", s.Name, err)
		sc.recordRun(s, planned, "failed", err.Error())
		return
	}
	sc.recordRun(s, planned, "succeeded", "")
}

func (sc *Scheduler) runAction(ctx context.Context, s *Schedule, planned time.Time) error {
	var payload struct {
		Command string         `json:"command"`
		Message string         `json:"message"`
		Type    string         `json:"backup_type"`
		Steps   []SequenceStep `json:"steps"`
	}
	json.Unmarshal(s.ActionPayload, &payload)

	switch s.Action {
	case "start_server":
		return sc.Exec.StartServer(ctx)
	case "stop_server":
		return sc.Exec.StopServer(ctx)
	case "restart_server":
		return sc.Exec.RestartServer(ctx)
	case "send_console_command":
		if !safeConsoleCommand(payload.Command) {
			return errors.New("command rejected by policy")
		}
		return sc.Exec.SendCommand(payload.Command)
	case "broadcast_message":
		return sc.Exec.Broadcast(payload.Message)
	case "save_all":
		return sc.Exec.SaveAll()
	case "create_backup":
		bt := payload.Type
		if bt == "" {
			bt = "full_server"
		}
		return sc.Exec.CreateBackup(ctx, s.InstanceID, bt)
	case "sequence":
		// steps ordered by offset relative to the scheduled time
		steps := payload.Steps
		for _, step := range steps {
			target := planned.Add(time.Duration(step.OffsetSeconds) * time.Second)
			if wait := time.Until(target); wait > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
				}
			}
			sub := &Schedule{InstanceID: s.InstanceID, Action: step.Action,
				ActionPayload: mustJSON(map[string]string{
					"command": step.Payload, "message": step.Payload, "backup_type": step.Payload,
				})}
			if err := sc.runAction(ctx, sub, planned); err != nil {
				return fmt.Errorf("step %s: %w", step.Action, err)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown action %q", s.Action)
}

// safeConsoleCommand blocks obviously dangerous console strings. Scheduled
// actions are Minecraft commands only, never Linux shell commands.
func safeConsoleCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || strings.ContainsAny(cmd, "\n\r") {
		return false
	}
	return true
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func defStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func nullStr(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// RunHistory returns recent executions of a schedule.
func (sc *Scheduler) RunHistory(scheduleID int64, limit int) ([]map[string]any, error) {
	rows, err := sc.DB.Query(`SELECT planned_at, COALESCE(started_at,''), COALESCE(finished_at,''),
		status, detail FROM schedule_runs WHERE schedule_id=? ORDER BY planned_at DESC LIMIT ?`,
		scheduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var planned, started, finished, status, detail string
		if err := rows.Scan(&planned, &started, &finished, &status, &detail); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"planned_at": planned, "started_at": started, "finished_at": finished,
			"status": status, "detail": detail,
		})
	}
	return out, rows.Err()
}
