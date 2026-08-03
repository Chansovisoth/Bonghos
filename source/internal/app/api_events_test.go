package app

import (
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestEventsTimelineRecordsAndReturnsEntries(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "timeline")

	env.app.recordEvent(inst.ID, CatLifecycle, "starting", SevInfo, "Starting timeline", nil)
	env.app.recordEvent(inst.ID, CatProgress, "loading_mods", SevInfo, "Loading mods", nil)
	env.app.recordEvent(inst.ID, CatLifecycle, "ready", SevInfo, "Server is ready", map[string]any{"seconds": 18.4})

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	var out struct {
		Events []ServerEvent `json:"events"`
	}
	status, body := c.do("GET", "/api/events", nil, &out)
	if status != 200 {
		t.Fatalf("GET /api/events returned %d: %s", status, body)
	}
	if len(out.Events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(out.Events))
	}
	// Newest first, so the interface can show the current situation on top.
	if out.Events[0].Event != "ready" {
		t.Errorf("first event is %q, want the most recent (ready)", out.Events[0].Event)
	}
	if out.Events[0].Detail == nil || out.Events[0].Detail["seconds"] == nil {
		t.Error("structured detail was not round-tripped")
	}
}

// A Member may see the dashboard, so they may see what the server is doing.
func TestEventsAreVisibleToAnyoneWhoCanSeeTheServer(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("member", "correct horse battery", authorization.RoleMember)
	c := env.newClient()
	c.mustLogin("member", "correct horse battery", secret)

	if status, body := c.do("GET", "/api/events", nil, nil); status != 200 {
		t.Errorf("GET /api/events as Member returned %d (%s), want 200", status, body)
	}
}

func TestEventsRequireAuthentication(t *testing.T) {
	env := newTestEnv(t)
	c := env.newClient()
	if status, _ := c.do("GET", "/api/events", nil, nil); status != 401 {
		t.Errorf("unauthenticated /api/events returned %d, want 401", status)
	}
}

// Mod loaders repeat progress lines constantly; a timeline that says "Loading
// mods" hundreds of times is no better than the raw log.
func TestStartupPhasesAreRecordedOncePerRun(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "phases")

	for i := 0; i < 50; i++ {
		env.app.notePhase("loading_mods", "Loading mods")
	}
	events, err := env.app.listEvents(0, 100)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range events {
		if e.Event == "loading_mods" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("recorded loading_mods %d times, want once per run", count)
	}

	// A new server start reports its progress again.
	env.app.resetPhases()
	env.app.notePhase("loading_mods", "Loading mods")
	events, _ = env.app.listEvents(0, 100)
	count = 0
	for _, e := range events {
		if e.Event == "loading_mods" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("after a restart loading_mods appears %d times, want 2", count)
	}
}

func TestStartupPhaseClassification(t *testing.T) {
	cases := map[string]string{
		"[12:00:00] [main/INFO]: Loading 412 mods":                                   "loading_mods",
		"Preparing level \"world\"":                                                  "preparing_world",
		"Starting minecraft server version 1.20.1":                                   "java_started",
		"You need to agree to the EULA in order to run the server":                   "eula_required",
		"[Server thread/WARN]: **** FAILED TO BIND TO PORT!":                         "port_in_use",
		"java.lang.UnsupportedClassVersionError: net/minecraft/Main":                 "wrong_java",
		"Error occurred during initialization of VM\nCould not reserve enough space": "insufficient_memory",
	}
	for line, want := range cases {
		got, msg, ok := startupPhase(line)
		if !ok {
			t.Errorf("startupPhase(%q) matched nothing, want %q", line, want)
			continue
		}
		if got != want {
			t.Errorf("startupPhase(%q) = %q, want %q", line, got, want)
		}
		if msg == "" {
			t.Errorf("startupPhase(%q) produced no message for the operator", line)
		}
	}

	// Ordinary chatter must not produce timeline noise.
	for _, line := range []string{
		"[12:00:00] [Server thread/INFO]: Steve joined the game",
		"There are 0 of a max of 10 players online",
		"random unmatched text",
	} {
		if _, _, ok := startupPhase(line); ok {
			t.Errorf("startupPhase(%q) produced an event for ordinary output", line)
		}
	}
}

// Failures must be surfaced as errors, not buried as progress.
func TestStartupFailuresAreRecordedAsErrors(t *testing.T) {
	env := newTestEnv(t)
	env.newServerProject(t, "failures")
	env.app.notePhase("eula_required", "The server stopped: the EULA has not been accepted")

	events, err := env.app.listEvents(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("no event recorded")
	}
	e := events[0]
	if e.Severity != SevError {
		t.Errorf("severity = %q, want %q", e.Severity, SevError)
	}
	if e.Category != CatError {
		t.Errorf("category = %q, want %q", e.Category, CatError)
	}
	if !strings.Contains(strings.ToLower(e.Message), "eula") {
		t.Errorf("message does not explain the problem: %q", e.Message)
	}
}
