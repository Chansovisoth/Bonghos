package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/scheduler"
)

func getOverview(t *testing.T, c *client) map[string]json.RawMessage {
	t.Helper()
	var payload map[string]json.RawMessage
	if status, body := c.do("GET", "/api/overview", nil, &payload); status != 200 {
		t.Fatalf("GET overview = %d (%s)", status, body)
	}
	return payload
}

func TestOverviewFieldsFollowGranularPermissions(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	viewerSecret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
	inst := env.newServerProject(t, "overview-permissions")

	archivePath := filepath.ToSlash(filepath.Join("backups", "overview-permissions.tar.zst"))
	if err := os.MkdirAll(env.app.Backups.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.app.Backups.Root, "overview-permissions.tar.zst"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.DB.Exec(`INSERT INTO backups
		(backup_id, instance_id, instance_slug, display_name, backup_type, consistency_mode,
		 trigger_type, archive_path, archive_format, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, "overview-permissions", inst.ID, inst.Slug, inst.DisplayName,
		"configuration_only", "offline", "manual", archivePath, "tar.zst", time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	schedule := &scheduler.Schedule{
		InstanceID: inst.ID, Name: "Overview test", Enabled: true, Timezone: "UTC",
		ScheduleType: "once", ScheduleExpression: time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04"),
		Action: "start_server",
	}
	if err := env.app.Sched.Create(schedule); err != nil {
		t.Fatal(err)
	}

	ownerClient := env.newClient()
	ownerClient.mustLogin("owner", "correct horse battery", ownerSecret)
	ownerOverview := getOverview(t, ownerClient)
	for _, field := range []string{"sample", "last_backup", "next_schedule_at"} {
		if _, ok := ownerOverview[field]; !ok {
			t.Errorf("Owner overview is missing %q", field)
		}
	}

	viewerClient := env.newClient()
	viewerClient.mustLogin("viewer", "correct horse battery", viewerSecret)
	viewerOverview := getOverview(t, viewerClient)
	for _, field := range []string{"sample", "last_backup", "next_schedule_at"} {
		if _, ok := viewerOverview[field]; ok {
			t.Errorf("default Viewer overview exposed %q", field)
		}
	}
	if status, body := viewerClient.do("GET", "/api/metrics?hours=1", nil, nil); status != 403 {
		t.Fatalf("default Viewer metrics = %d (%s), want 403", status, body)
	}

	desired := []string{
		string(authorization.PermServerView),
		string(authorization.PermPerformanceView),
		string(authorization.PermBackupsView),
	}
	if status, body := putRolePermissions(ownerClient, "viewer", desired, 0, nil); status != 200 {
		t.Fatalf("granting Viewer view permissions = %d (%s)", status, body)
	}
	viewerOverview = getOverview(t, viewerClient)
	for _, field := range []string{"sample", "last_backup"} {
		if _, ok := viewerOverview[field]; !ok {
			t.Errorf("Viewer overview with permission is missing %q", field)
		}
	}
	if _, ok := viewerOverview["next_schedule_at"]; ok {
		t.Error("Viewer overview exposed schedule data without schedule management")
	}
	if status, body := viewerClient.do("GET", "/api/metrics?hours=1", nil, nil); status != 200 {
		t.Fatalf("Viewer metrics with permission = %d (%s), want 200", status, body)
	}
}
