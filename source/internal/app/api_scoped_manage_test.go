package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestFilesAndConfigurationCanTargetInactiveProject(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	active := env.newServerProject(t, "active-project")
	target := env.newServerProject(t, "inactive-project")
	if err := env.app.Instances.SetActive(active.ID); err != nil {
		t.Fatal(err)
	}

	activeProperties := filepath.Join(active.AbsoluteDir(env.app.Home), "server.properties")
	targetProperties := filepath.Join(target.AbsoluteDir(env.app.Home), "server.properties")
	if err := os.WriteFile(activeProperties, []byte("motd=Active project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetProperties, []byte("motd=Inactive project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	scope := "server_id=" + itoa(target.ID)

	var content struct {
		Content string `json:"content"`
	}
	if status, body := c.do(http.MethodGet, "/api/files/content?"+scope+"&path=server.properties", nil, &content); status != http.StatusOK {
		t.Fatalf("read inactive project file: %d %s", status, body)
	}
	if !strings.Contains(content.Content, "Inactive project") {
		t.Fatalf("scoped file read returned %q", content.Content)
	}
	if status, body := c.do(http.MethodGet, "/api/files/content?path=server.properties", nil, &content); status != http.StatusOK {
		t.Fatalf("read active project file: %d %s", status, body)
	}
	if !strings.Contains(content.Content, "Active project") {
		t.Fatalf("unscoped file read returned %q", content.Content)
	}

	if status, body := c.do(http.MethodPost, "/api/files/content?"+scope,
		map[string]string{"path": "server.properties", "content": "motd=Scoped file edit\n"}, nil); status != http.StatusOK {
		t.Fatalf("write inactive project file: %d %s", status, body)
	}
	assertFileContains(t, targetProperties, "Scoped file edit")
	assertFileContains(t, activeProperties, "Active project")

	var configuration struct {
		Instance struct {
			ID int64 `json:"id"`
		} `json:"instance"`
	}
	if status, body := c.do(http.MethodGet, "/api/configuration?"+scope, nil, &configuration); status != http.StatusOK {
		t.Fatalf("read inactive project configuration: %d %s", status, body)
	}
	if configuration.Instance.ID != target.ID {
		t.Fatalf("configuration returned instance %d, want %d", configuration.Instance.ID, target.ID)
	}

	if status, body := c.do(http.MethodPost, "/api/configuration/property?"+scope,
		map[string]string{"key": "motd", "value": "Scoped configuration edit"}, nil); status != http.StatusOK {
		t.Fatalf("write inactive project configuration: %d %s", status, body)
	}
	assertFileContains(t, targetProperties, "motd=Scoped configuration edit")
	assertFileContains(t, activeProperties, "motd=Active project")
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s contains %q, want substring %q", path, string(data), want)
	}
}
