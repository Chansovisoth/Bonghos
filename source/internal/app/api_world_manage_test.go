package app

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/operations"
)

func TestDuplicateServerCopiesFilesAndDisablesAutostart(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	source := env.newServerProject(t, "duplicate-source")
	source.AutostartEnabled = true
	if err := env.app.Instances.Update(source); err != nil {
		t.Fatal(err)
	}
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	var response struct {
		OperationID string `json:"operation_id"`
		Server      struct {
			ID int64 `json:"id"`
		} `json:"server"`
	}
	status, body := c.do("POST", "/api/servers/"+itoa(source.ID)+"/duplicate",
		map[string]any{"display_name": "Duplicate Copy"}, &response)
	if status != http.StatusAccepted {
		t.Fatalf("duplicate failed: %d %s", status, body)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		op, err := env.app.Operations.Get(response.OperationID)
		if err != nil {
			t.Fatal(err)
		}
		if operations.Terminal(op.Stage) {
			if op.Stage != operations.StageCompleted {
				t.Fatalf("duplicate operation ended in %s: %s", op.Stage, op.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("duplicate operation did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}

	clone, err := env.app.Instances.ByID(response.Server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if clone.AutostartEnabled {
		t.Error("duplicate must not inherit autostart")
	}
	got, err := os.ReadFile(filepath.Join(clone.AbsoluteDir(env.app.Home), "world", "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL WORLD" {
		t.Fatalf("copied world contains %q", got)
	}
}

func TestWorldResetRequiresConfirmationAndCreatesSafetyBackup(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "reset-world")
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	path := "/api/servers/" + itoa(inst.ID) + "/world/reset"

	if status, _ := c.do("POST", path, map[string]bool{"confirm": false}, nil); status != http.StatusBadRequest {
		t.Fatalf("unconfirmed reset returned %d", status)
	}
	world := filepath.Join(inst.AbsoluteDir(env.app.Home), "world")
	if _, err := os.Stat(world); err != nil {
		t.Fatalf("unconfirmed reset touched world: %v", err)
	}

	if status, body := c.do("POST", path, map[string]bool{"confirm": true}, nil); status != http.StatusOK {
		t.Fatalf("confirmed reset failed: %d %s", status, body)
	}
	if _, err := os.Stat(world); !os.IsNotExist(err) {
		t.Fatalf("world still exists after reset: %v", err)
	}
	records, err := env.app.Backups.List(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].TriggerType != "emergency-pre-world-reset" || records[0].VerificationStatus != "verified" {
		t.Fatalf("unexpected reset safety backups: %+v", records)
	}
}

func TestWorldDownloadReturnsZIPWithWorldRoot(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "download-world")
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/servers/"+itoa(inst.ID)+"/world.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("download failed: %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(resp.Header.Get("Content-Disposition"), "download-world-world.zip") {
		t.Fatalf("unexpected content disposition: %q", resp.Header.Get("Content-Disposition"))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("invalid ZIP: %v", err)
	}
	for _, file := range zr.File {
		if file.Name != "world/level.dat" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(reader)
		reader.Close()
		if string(got) != "ORIGINAL WORLD" {
			t.Fatalf("ZIP world contains %q", got)
		}
		return
	}
	t.Fatal("world/level.dat missing from ZIP")
}
