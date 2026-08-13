package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/config"
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
	if status, body := c.do(http.MethodPost, "/api/files/create?"+scope,
		map[string]string{"path": "notes.txt"}, nil); status != http.StatusOK {
		t.Fatalf("create inactive project file: %d %s", status, body)
	}
	if status, body := c.do(http.MethodPost, "/api/files/copy?"+scope,
		map[string]any{"paths": []string{"notes.txt", "world"}, "destination": "copies"}, nil); status != http.StatusOK {
		t.Fatalf("copy inactive project files: %d %s", status, body)
	}
	if status, body := c.do(http.MethodPost, "/api/files/move?"+scope,
		map[string]any{"paths": []string{"copies/notes.txt"}, "destination": "moved"}, nil); status != http.StatusOK {
		t.Fatalf("move inactive project file: %d %s", status, body)
	}
	if _, err := os.Stat(filepath.Join(target.AbsoluteDir(env.app.Home), "moved", "notes.txt")); err != nil {
		t.Fatalf("moved inactive project file: %v", err)
	}

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

func TestFilesCanCopyAndMoveBetweenProjectsButNotOutsideThem(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	source := env.newServerProject(t, "transfer-source")
	destination := env.newServerProject(t, "transfer-destination")

	sourceRoot := source.AbsoluteDir(env.app.Home)
	destinationRoot := destination.AbsoluteDir(env.app.Home)
	if err := os.WriteFile(filepath.Join(sourceRoot, "copy.txt"), []byte("copy across projects"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "move.txt"), []byte("move across projects"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	scope := "server_id=" + itoa(source.ID)
	if status, body := c.do(http.MethodPost, "/api/files/copy?"+scope, map[string]any{
		"paths": []string{"copy.txt"}, "destination": "imports", "destination_server_id": destination.ID,
	}, nil); status != http.StatusOK {
		t.Fatalf("copy between projects: %d %s", status, body)
	}
	assertFileContains(t, filepath.Join(destinationRoot, "imports", "copy.txt"), "copy across projects")
	assertFileContains(t, filepath.Join(sourceRoot, "copy.txt"), "copy across projects")

	if status, body := c.do(http.MethodPost, "/api/files/move?"+scope, map[string]any{
		"paths": []string{"move.txt"}, "destination": "imports", "destination_server_id": destination.ID,
	}, nil); status != http.StatusOK {
		t.Fatalf("move between projects: %d %s", status, body)
	}
	assertFileContains(t, filepath.Join(destinationRoot, "imports", "move.txt"), "move across projects")
	if _, err := os.Stat(filepath.Join(sourceRoot, "move.txt")); !os.IsNotExist(err) {
		t.Fatalf("cross-project move left source behind: %v", err)
	}

	if status, _ := c.do(http.MethodPost, "/api/files/copy?"+scope, map[string]any{
		"paths": []string{"copy.txt"}, "destination": "", "destination_server_id": int64(999999),
	}, nil); status != http.StatusBadRequest {
		t.Fatalf("unknown destination project status = %d, want %d", status, http.StatusBadRequest)
	}
	if status, _ := c.do(http.MethodPost, "/api/files/copy?"+scope, map[string]any{
		"paths": []string{"copy.txt"}, "destination": "../outside", "destination_server_id": destination.ID,
	}, nil); status != http.StatusBadRequest {
		t.Fatalf("destination traversal status = %d, want %d", status, http.StatusBadRequest)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(destinationRoot), "outside", "copy.txt")); !os.IsNotExist(err) {
		t.Fatalf("copy escaped destination project: %v", err)
	}
}

func TestFilesCanBrowseServersRootWithoutEscapingOrMutatingIt(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	project := env.newServerProject(t, "browse-project")
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	var entries []struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
	}
	if status, body := c.do(http.MethodGet, "/api/files?root=servers&path=minecraft-java/modded", nil, &entries); status != http.StatusOK {
		t.Fatalf("browse servers root: %d %s", status, body)
	}
	found := false
	for _, entry := range entries {
		if entry.Name == project.Slug && entry.IsDir {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("servers-root listing did not include project %q: %#v", project.Slug, entries)
	}

	var content struct {
		Content string `json:"content"`
	}
	projectFile := "minecraft-java/modded/" + project.Slug + "/server.properties"
	if status, body := c.do(http.MethodGet, "/api/files/content?root=servers&path="+projectFile, nil, &content); status != http.StatusOK {
		t.Fatalf("read through servers root: %d %s", status, body)
	}
	if !strings.Contains(content.Content, "motd=Test") {
		t.Fatalf("servers-root read returned %q", content.Content)
	}

	if status, _ := c.do(http.MethodGet, "/api/files?root=servers&path=..", nil, nil); status != http.StatusBadRequest {
		t.Fatalf("servers-root traversal status = %d, want %d", status, http.StatusBadRequest)
	}
	if status, _ := c.do(http.MethodGet, "/api/files?root=home", nil, nil); status != http.StatusConflict {
		t.Fatalf("unknown files root status = %d, want %d", status, http.StatusConflict)
	}

	if status, body := c.do(http.MethodPost, "/api/files/content?root=servers",
		map[string]string{"path": projectFile, "content": "motd=Must not change\n"}, nil); status != http.StatusBadRequest {
		t.Fatalf("write through servers root: %d %s", status, body)
	}
	assertFileContains(t, filepath.Join(project.AbsoluteDir(env.app.Home), "server.properties"), "motd=Test")
}

func TestRequestFilesServersRootIsJailed(t *testing.T) {
	home := t.TempDir()
	serversRoot := filepath.Join(home, config.DirServers)
	if err := os.MkdirAll(serversRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serversRoot, "visible.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "outside.txt"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{Home: home}
	req := httptest.NewRequest(http.MethodGet, "/api/files?root=servers", nil)
	fm, inst, err := a.requestFiles(req)
	if err != nil {
		t.Fatalf("request servers files: %v", err)
	}
	if inst != nil {
		t.Fatalf("servers-root scope returned project %#v", inst)
	}
	if entries, err := fm.List(""); err != nil || len(entries) != 1 || entries[0].Name != "visible.txt" {
		t.Fatalf("servers-root list = %#v, %v", entries, err)
	}
	if _, err := fm.ReadText("../outside.txt"); err == nil {
		t.Fatal("servers-root manager allowed traversal into BONGHOS_HOME")
	}

	recorder := httptest.NewRecorder()
	mutation := httptest.NewRequest(http.MethodPost, "/api/files/content?root=servers", nil)
	if !rejectServersRootMutation(recorder, mutation) || recorder.Code != http.StatusBadRequest {
		t.Fatalf("servers-root mutation guard = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
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
