package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/supervisor"
)

func insertOnlinePlayer(t *testing.T, a *App, instanceID int64, username string, sessionStart time.Time) {
	t.Helper()
	started := sessionStart.UTC().Format(time.RFC3339)
	if _, err := a.DB.Exec(`INSERT INTO players
		(instance_id, username, first_seen_at, last_seen_at, last_joined_at,
		 current_session_started_at, observed_playtime_seconds, is_online)
		VALUES (?, ?, ?, ?, ?, ?, 10, 1)`,
		instanceID, username, started, started, started, started); err != nil {
		t.Fatal(err)
	}
}

func TestMarkAllPlayersOfflineClosesActiveSessions(t *testing.T) {
	env := newTestEnv(t)
	inst := env.newServerProject(t, "offline-players")
	insertOnlinePlayer(t, env.app, inst.ID, "Alex", time.Now().Add(-2*time.Minute))
	insertOnlinePlayer(t, env.app, inst.ID, "Steve", time.Now().Add(-time.Minute))

	if changed := env.app.markAllPlayersOffline(inst.ID); changed != 2 {
		t.Fatalf("markAllPlayersOffline changed %d rows, want 2", changed)
	}
	if changed := env.app.markAllPlayersOffline(inst.ID); changed != 0 {
		t.Fatalf("second markAllPlayersOffline changed %d rows, want 0", changed)
	}

	rows, err := env.app.DB.Query(`SELECT is_online, last_left_at,
		current_session_started_at, observed_playtime_seconds
		FROM players WHERE instance_id=?`, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var online int
		var left, session *string
		var playtime int64
		if err := rows.Scan(&online, &left, &session, &playtime); err != nil {
			t.Fatal(err)
		}
		if online != 0 || left == nil || session != nil {
			t.Fatalf("player session was not closed: online=%d left=%v session=%v", online, left, session)
		}
		if playtime <= 10 {
			t.Fatalf("observed playtime was not finalized: %d", playtime)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("checked %d players, want 2", count)
	}
}

func TestObserveServerStateRecordsTerminalTransitionsOnce(t *testing.T) {
	env := newTestEnv(t)
	inst := env.newServerProject(t, "terminal-events")
	insertOnlinePlayer(t, env.app, inst.ID, "Alex", time.Now().Add(-time.Minute))

	// Establishing the initial stopped state cleans stale sessions without
	// inventing a shutdown event every time the control plane starts.
	env.app.observeServerState("stopped", nil)
	events, err := env.app.listEvents(inst.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("initial state recorded %d events, want 0", len(events))
	}

	env.app.observeServerState("running", &supervisor.PersistedState{State: supervisor.StateRunning})
	env.app.observeServerState("stopped", &supervisor.PersistedState{State: supervisor.StateStopped})
	env.app.observeServerState("stopped", &supervisor.PersistedState{State: supervisor.StateStopped})
	env.app.observeServerState("starting", &supervisor.PersistedState{State: supervisor.StateStarting})
	env.app.observeServerState("crashed", &supervisor.PersistedState{
		State: supervisor.StateCrashed, LastExitCode: 137, LastSignal: "killed",
	})
	env.app.observeServerState("crashed", &supervisor.PersistedState{State: supervisor.StateCrashed})

	events, err = env.app.listEvents(inst.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("terminal transitions recorded %d events, want 2", len(events))
	}
	if events[0].Event != "crashed" || events[0].Severity != SevError {
		t.Fatalf("latest event = %#v, want crashed error", events[0])
	}
	if events[0].Detail["exit_code"] != float64(137) || events[0].Detail["signal"] != "killed" {
		t.Fatalf("crash detail = %#v", events[0].Detail)
	}
	if events[1].Event != "stopped" {
		t.Fatalf("older event = %q, want stopped", events[1].Event)
	}
}

func TestCollectSampleRejectsStaleSupervisorProcessState(t *testing.T) {
	env := newTestEnv(t)
	inst := env.newServerProject(t, "stale-process")
	insertOnlinePlayer(t, env.app, inst.ID, "Alex", time.Now().Add(-time.Minute))

	state := supervisor.PersistedState{
		InstanceID: inst.ID,
		State:      supervisor.StateRunning,
		ScriptPID:  2147483000,
		JavaPID:    2147483001,
		StartedAt:  time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(env.app.Home, config.FileSupState)
	if err := os.WriteFile(statePath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := env.app.Runner.State(); got != "crashed" {
		t.Fatalf("stale running state classified as %q, want crashed", got)
	}

	sample := env.app.collectSample()
	if sample.JavaPID != 0 || sample.UptimeSeconds != 0 {
		t.Fatalf("stale process leaked into sample: pid=%d uptime=%d", sample.JavaPID, sample.UptimeSeconds)
	}
	if sample.OnlinePlayers != 0 {
		t.Fatalf("offline sample reports %d online players, want 0", sample.OnlinePlayers)
	}
}
