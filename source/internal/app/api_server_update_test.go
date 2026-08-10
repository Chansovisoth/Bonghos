package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestServerRenameChangesOnlyDisplayName(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "rename-target")
	originalSlug := inst.Slug
	originalDirectory := inst.ServerDirectory
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	var response struct {
		DisplayName     string `json:"display_name"`
		Slug            string `json:"slug"`
		ServerDirectory string `json:"server_directory"`
	}
	status, body := c.do(http.MethodPatch, "/api/servers/"+itoa(inst.ID),
		map[string]string{"display_name": "  Renamed Survival  "}, &response)
	if status != http.StatusOK {
		t.Fatalf("rename failed: %d %s", status, body)
	}
	if response.DisplayName != "Renamed Survival" {
		t.Fatalf("response display name = %q", response.DisplayName)
	}
	if response.Slug != originalSlug || response.ServerDirectory != originalDirectory {
		t.Fatalf("rename changed project identity: slug=%q directory=%q", response.Slug, response.ServerDirectory)
	}

	stored, err := env.app.Instances.ByID(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.DisplayName != "Renamed Survival" {
		t.Fatalf("stored display name = %q", stored.DisplayName)
	}
	if stored.Slug != originalSlug || stored.ServerDirectory != originalDirectory {
		t.Fatalf("stored identity changed: slug=%q directory=%q", stored.Slug, stored.ServerDirectory)
	}
	var detail string
	if err := env.app.DB.QueryRow(`SELECT detail FROM audit_log
		WHERE action='project_updated' AND target=? ORDER BY id DESC LIMIT 1`, originalSlug).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, inst.DisplayName) || !strings.Contains(detail, "Renamed Survival") {
		t.Fatalf("rename audit detail = %q", detail)
	}
}

func TestServerRenameRejectsInvalidNames(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "rename-validation")
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	path := "/api/servers/" + itoa(inst.ID)

	for _, name := range []string{"   ", strings.Repeat("x", 121)} {
		if status, body := c.do(http.MethodPatch, path,
			map[string]string{"display_name": name}, nil); status != http.StatusBadRequest {
			t.Errorf("rename to %q returned %d (%s), want 400", name, status, body)
		}
	}

	stored, err := env.app.Instances.ByID(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.DisplayName != inst.DisplayName {
		t.Fatalf("invalid rename changed display name to %q", stored.DisplayName)
	}
}

func TestServerRenameRequiresConfigurationPermission(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("member", "correct horse battery", authorization.RoleMember)
	inst := env.newServerProject(t, "rename-permission")
	c := env.newClient()
	c.mustLogin("member", "correct horse battery", secret)

	status, body := c.do(http.MethodPatch, "/api/servers/"+itoa(inst.ID),
		map[string]string{"display_name": "Not allowed"}, nil)
	if status != http.StatusForbidden {
		t.Fatalf("member rename returned %d (%s), want 403", status, body)
	}
}
