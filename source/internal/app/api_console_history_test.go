package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/instance"
)

func TestConsoleHistoryReturnsLastThousandLinesFromActiveServerLog(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
	c := env.newClient()
	c.mustLogin("viewer", "correct horse battery", secret)

	inst := &instance.Instance{
		Slug:            "history-log",
		DisplayName:     "History Log",
		ServerDirectory: instance.RelativeDirFor("history-log"),
		StartupScript:   "run.sh",
	}
	if err := env.app.Instances.Create(inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := env.app.Instances.SetActive(inst.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}

	logDir := filepath.Join(inst.AbsoluteDir(env.home), "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	var b strings.Builder
	for i := 1; i <= 1050; i++ {
		fmt.Fprintf(&b, "line-%04d\n", i)
	}
	if err := os.WriteFile(filepath.Join(logDir, "bonghos-console.log"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var out struct {
		Lines  []string `json:"lines"`
		Limit  int      `json:"limit"`
		Source string   `json:"source"`
	}
	if status, body := c.do("GET", "/api/server/console/history?limit=2000", nil, &out); status != 200 {
		t.Fatalf("history status=%d body=%s", status, body)
	}
	if out.Limit != 1000 {
		t.Fatalf("limit = %d, want 1000", out.Limit)
	}
	if out.Source != "log" {
		t.Fatalf("source = %q, want log", out.Source)
	}
	if len(out.Lines) != 1000 {
		t.Fatalf("returned %d lines, want 1000", len(out.Lines))
	}
	if out.Lines[0] != "line-0051" || out.Lines[len(out.Lines)-1] != "line-1050" {
		t.Fatalf("unexpected log tail: first=%q last=%q", out.Lines[0], out.Lines[len(out.Lines)-1])
	}
}

func TestConsoleHistoryFallsBackToMemoryCache(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
	c := env.newClient()
	c.mustLogin("viewer", "correct horse battery", secret)

	for i := 1; i <= 1005; i++ {
		env.app.handleConsoleLine(fmt.Sprintf("cached-%04d", i), false)
	}

	var out struct {
		Lines  []string `json:"lines"`
		Source string   `json:"source"`
	}
	if status, body := c.do("GET", "/api/server/console/history", nil, &out); status != 200 {
		t.Fatalf("history status=%d body=%s", status, body)
	}
	if out.Source != "cache" {
		t.Fatalf("source = %q, want cache", out.Source)
	}
	if len(out.Lines) != 1000 {
		t.Fatalf("returned %d lines, want 1000", len(out.Lines))
	}
	if out.Lines[0] != "cached-0006" || out.Lines[len(out.Lines)-1] != "cached-1005" {
		t.Fatalf("unexpected cached tail: first=%q last=%q", out.Lines[0], out.Lines[len(out.Lines)-1])
	}
}
