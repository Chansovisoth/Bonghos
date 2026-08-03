package app

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

// Event categories and severities.
const (
	CatLifecycle = "lifecycle"
	CatProgress  = "progress"
	CatBackup    = "backup"
	CatSchedule  = "schedule"
	CatRecovery  = "recovery"
	CatError     = "error"

	SevInfo    = "info"
	SevWarning = "warning"
	SevError   = "error"
)

// maxStoredEvents bounds the timeline. It is a narrative for humans, not an
// archive: the log files remain the complete record.
const maxStoredEvents = 2000

// ServerEvent is one entry in the durable timeline.
type ServerEvent struct {
	ID         int64          `json:"id"`
	OccurredAt string         `json:"occurred_at"`
	InstanceID int64          `json:"instance_id,omitempty"`
	Category   string         `json:"category"`
	Event      string         `json:"event"`
	Severity   string         `json:"severity"`
	Message    string         `json:"message"`
	Detail     map[string]any `json:"detail,omitempty"`
}

// recordEvent appends to the timeline and broadcasts it to any open interface.
// Failures are swallowed: losing a timeline entry must never disrupt whatever
// the server was actually doing.
func (a *App) recordEvent(instanceID int64, category, event, severity, message string, detail map[string]any) {
	if severity == "" {
		severity = SevInfo
	}
	var encoded string
	if len(detail) > 0 {
		if b, err := json.Marshal(detail); err == nil {
			encoded = string(b)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)

	var instArg any
	if instanceID > 0 {
		instArg = instanceID
	}
	res, err := a.DB.Exec(`INSERT INTO server_events
		(occurred_at, instance_id, category, event, severity, message, detail)
		VALUES (?,?,?,?,?,?,?)`,
		now, instArg, category, event, severity, message, encoded)
	if err != nil {
		a.Logf("could not record event %s/%s: %v", category, event, err)
		return
	}
	id, _ := res.LastInsertId()

	// Keep the table bounded. Doing this inline is cheap because the delete
	// only matches once the table is over the limit.
	if id%100 == 0 {
		a.DB.Exec(`DELETE FROM server_events WHERE id <= (
			SELECT MAX(id) - ? FROM server_events)`, maxStoredEvents)
	}

	a.Hub.Broadcast("overview", "event", map[string]any{
		"id": id, "occurred_at": now, "category": category, "event": event,
		"severity": severity, "message": message, "detail": detail,
	})
}

// listEvents returns the most recent timeline entries, newest first.
func (a *App) listEvents(instanceID int64, limit int) ([]ServerEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var (
		rows *sql.Rows
		err  error
	)
	if instanceID > 0 {
		rows, err = a.DB.Query(`SELECT id, occurred_at, COALESCE(instance_id,0),
			category, event, severity, message, detail
			FROM server_events WHERE instance_id = ? ORDER BY id DESC LIMIT ?`,
			instanceID, limit)
	} else {
		rows, err = a.DB.Query(`SELECT id, occurred_at, COALESCE(instance_id,0),
			category, event, severity, message, detail
			FROM server_events ORDER BY id DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ServerEvent{}
	for rows.Next() {
		var e ServerEvent
		var detail string
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.InstanceID, &e.Category,
			&e.Event, &e.Severity, &e.Message, &detail); err != nil {
			continue
		}
		if detail != "" {
			_ = json.Unmarshal([]byte(detail), &e.Detail)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// notePhase records a startup phase once per server start. Mod loaders repeat
// these lines many times, and a timeline that says "Loading mods" four hundred
// times is no more useful than the raw log.
func (a *App) notePhase(event, message string) {
	a.phaseMu.Lock()
	if a.seenPhases == nil {
		a.seenPhases = map[string]bool{}
	}
	if a.seenPhases[event] {
		a.phaseMu.Unlock()
		return
	}
	a.seenPhases[event] = true
	a.phaseMu.Unlock()

	severity := SevInfo
	category := CatProgress
	switch event {
	case "eula_required", "port_in_use", "wrong_java", "insufficient_memory":
		severity, category = SevError, CatError
	}
	a.recordEvent(a.activeInstanceIDQuiet(), category, event, severity, message, nil)
}

// resetPhases clears the once-per-start phase memory. Called when a server
// starts so the next boot reports its progress again.
func (a *App) resetPhases() {
	a.phaseMu.Lock()
	a.seenPhases = map[string]bool{}
	a.phaseMu.Unlock()
}

// startupPhase classifies a console line into a coarse startup phase, so the
// interface can show "installing Forge" or "loading mods" instead of leaving
// the operator to watch raw output and guess whether anything is wrong.
//
// Matching is deliberately loose: mod loaders word these differently and an
// unrecognised line simply produces no event.
func startupPhase(line string) (event, message string, ok bool) {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "installing forge") || strings.Contains(l, "installing neoforge"),
		strings.Contains(l, "forge installer") || strings.Contains(l, "neoforge installer"):
		return "installing_loader", "Installing the mod loader", true
	case strings.Contains(l, "downloading") && strings.Contains(l, "librar"):
		return "downloading_libraries", "Downloading libraries", true
	case strings.Contains(l, "loading ") && strings.Contains(l, "mods"):
		return "loading_mods", "Loading mods", true
	case strings.Contains(l, "preparing level") || strings.Contains(l, "preparing start region"):
		return "preparing_world", "Preparing the world", true
	case strings.Contains(l, "starting minecraft server"):
		return "java_started", "Minecraft server process started", true
	case strings.Contains(l, "agree to the eula"):
		return "eula_required", "The server stopped: the Minecraft EULA has not been accepted", true
	case strings.Contains(l, "failed to bind to port") || strings.Contains(l, "address already in use"):
		return "port_in_use", "The server port is already in use", true
	case strings.Contains(l, "unsupportedclassversionerror"):
		return "wrong_java", "The selected Java version is not compatible with this pack", true
	case strings.Contains(l, "could not reserve enough space"):
		return "insufficient_memory", "Java could not reserve the configured heap; lower Xmx", true
	}
	return "", "", false
}
