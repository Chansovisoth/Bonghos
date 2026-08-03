package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/instance"
)

// newServerProject creates a project on disk with a small, recognisable set of
// files so restores can be checked by reading the filesystem afterwards.
func (e *testEnv) newServerProject(t *testing.T, slug string) *instance.Instance {
	t.Helper()
	inst := &instance.Instance{
		Slug: slug, DisplayName: slug,
		ServerType:      "minecraft-java-modded",
		SourceType:      "existing-directory",
		ServerDirectory: instance.RelativeDirFor(slug),
		StartupScript:   "start.sh",
	}
	if err := e.app.Instances.Create(inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	dir := inst.AbsoluteDir(e.app.Home)
	for path, body := range map[string]string{
		"start.sh":          "#!/bin/bash\njava -Xms1G -Xmx2G -jar server.jar nogui\n",
		"server.properties": "motd=Test\nlevel-name=world\n",
		"eula.txt":          "eula=true\n",
		"world/level.dat":   "ORIGINAL WORLD",
		"mods/example.jar":  "ORIGINAL MOD",
	} {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.app.Instances.SetActive(inst.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}
	return inst
}

// Restoring through the Web API must create an emergency safety copy of the
// current state first. Losing the present world in order to recover an older
// one is never an acceptable trade.
func TestWebRestoreCreatesEmergencyPreRestoreBackup(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "restore-safety")

	rec, err := env.app.RunBackup(context.Background(), inst, backup.TypeFull, "offline", "manual", 0)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}

	// Change the world so the restore has something to roll back.
	worldFile := filepath.Join(inst.AbsoluteDir(env.app.Home), "world", "level.dat")
	if err := os.WriteFile(worldFile, []byte("CHANGED WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	before, err := env.app.Backups.List(inst.ID)
	if err != nil {
		t.Fatal(err)
	}

	status, body := c.do("POST", "/api/backups/"+rec.BackupID+"/restore",
		map[string]any{"scope": "full_server", "confirm": true}, nil)
	if status != 200 {
		t.Fatalf("restore failed: %d %s", status, body)
	}

	after, err := env.app.Backups.List(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("expected one new emergency backup, went from %d to %d", len(before), len(after))
	}

	var found bool
	for _, r := range after {
		if r.TriggerType == "emergency-pre-restore" {
			found = true
			if r.VerificationStatus != "verified" {
				t.Errorf("emergency backup was not verified: %s", r.VerificationStatus)
			}
		}
	}
	if !found {
		t.Error("no backup recorded with trigger type emergency-pre-restore")
	}

	// And the restore itself worked.
	got, err := os.ReadFile(worldFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL WORLD" {
		t.Errorf("world was not restored, contains %q", got)
	}
}

// The UI once sent "world" while the backend compared against "world_only",
// so a world-only restore silently became a full restore. Scope names must be
// normalized rather than falling through.
func TestRestoreScopeAliasesAreNormalized(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", backup.ScopeFull},
		{"full", backup.ScopeFull},
		{"full_server", backup.ScopeFull},
		{"world", backup.ScopeWorld},
		{"world_only", backup.ScopeWorld},
		{"world_and_player_data", backup.ScopeWorld},
		{"configuration", backup.ScopeConfig},
		{"configuration_only", backup.ScopeConfig},
		{"  Full_Server  ", backup.ScopeFull},
	} {
		got, err := backup.NormalizeScope(tc.in)
		if err != nil {
			t.Errorf("NormalizeScope(%q) returned error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeScope(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// An unrecognised scope must be rejected, never treated as a full restore.
	for _, bad := range []string{"everything", "wrld", "world only", "../etc"} {
		if got, err := backup.NormalizeScope(bad); err == nil {
			t.Errorf("NormalizeScope(%q) accepted, returning %q", bad, got)
		}
	}
}

// A world-only restore must not replace mods or startup scripts.
func TestWorldOnlyRestoreLeavesOtherFilesAlone(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "scope-check")
	dir := inst.AbsoluteDir(env.app.Home)

	rec, err := env.app.RunBackup(context.Background(), inst, backup.TypeFull, "offline", "manual", 0)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}

	// Change both a world file and a non-world file after the backup.
	if err := os.WriteFile(filepath.Join(dir, "world", "level.dat"), []byte("CHANGED WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mods", "example.jar"), []byte("UPDATED MOD"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	status, body := c.do("POST", "/api/backups/"+rec.BackupID+"/restore",
		map[string]any{"scope": "world_only", "confirm": true}, nil)
	if status != 200 {
		t.Fatalf("restore failed: %d %s", status, body)
	}

	world, err := os.ReadFile(filepath.Join(dir, "world", "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if string(world) != "ORIGINAL WORLD" {
		t.Errorf("world should have been restored, got %q", world)
	}

	mod, err := os.ReadFile(filepath.Join(dir, "mods", "example.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mod) != "UPDATED MOD" {
		t.Errorf("world-only restore overwrote a mod file, got %q", mod)
	}
}

func TestRestoreRequiresExplicitConfirmation(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "confirm-check")

	rec, err := env.app.RunBackup(context.Background(), inst, backup.TypeFull, "offline", "manual", 0)
	if err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	if status, _ := c.do("POST", "/api/backups/"+rec.BackupID+"/restore",
		map[string]any{"scope": "full_server"}, nil); status != 400 {
		t.Errorf("restore without confirmation returned %d, want 400", status)
	}
	if status, _ := c.do("POST", "/api/backups/"+rec.BackupID+"/restore",
		map[string]any{"scope": "nonsense", "confirm": true}, nil); status != 400 {
		t.Errorf("restore with an unknown scope returned %d, want 400", status)
	}
}

// A world-only restore must find the world the archive actually contains, not
// whatever level-name the server happens to use now. Reading only the current
// server.properties meant that renaming the world after a backup made the
// restore match nothing and silently succeed while changing nothing.
func TestWorldOnlyRestoreAfterLevelNameChanged(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "renamed-world")
	dir := inst.AbsoluteDir(env.app.Home)

	// Neither name is the Minecraft default, so a "world" fallback cannot
	// accidentally rescue a restore that looks in the wrong place.
	if err := os.WriteFile(filepath.Join(dir, "server.properties"),
		[]byte("motd=Test\nlevel-name=alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha", "level.dat"),
		[]byte("ORIGINAL WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "world")); err != nil {
		t.Fatal(err)
	}

	// The backup captures the world while it is called "alpha".
	rec, err := env.app.RunBackup(context.Background(), inst, backup.TypeFull, "offline", "manual", 0)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}

	// Afterwards the operator renames the world to "beta" and it diverges.
	if err := os.WriteFile(filepath.Join(dir, "server.properties"),
		[]byte("motd=Test\nlevel-name=beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha", "level.dat"),
		[]byte("CHANGED WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	status, body := c.do("POST", "/api/backups/"+rec.BackupID+"/restore",
		map[string]any{"scope": "world_only", "confirm": true}, nil)
	if status != 200 {
		t.Fatalf("restore failed: %d %s", status, body)
	}

	got, err := os.ReadFile(filepath.Join(dir, "alpha", "level.dat"))
	if err != nil {
		t.Fatalf("archived world was not restored: %v", err)
	}
	if string(got) != "ORIGINAL WORLD" {
		t.Errorf("world contains %q, want the archived contents", got)
	}
}

// A scoped restore that matches nothing must report that rather than
// returning success after changing no files.
func TestScopedRestoreMatchingNothingReportsFailure(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "empty-scope")
	dir := inst.AbsoluteDir(env.app.Home)

	// Back up configuration only, then ask for a world-only restore of it.
	rec, err := env.app.RunBackup(context.Background(), inst, backup.TypeConfig, "offline", "manual", 0)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "world", "level.dat"),
		[]byte("CURRENT WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	status, body := c.do("POST", "/api/backups/"+rec.BackupID+"/restore",
		map[string]any{"scope": "world_only", "confirm": true}, nil)
	if status == 200 {
		t.Errorf("restore reported success despite matching nothing: %s", body)
	}

	// And it really did leave the current world alone.
	got, err := os.ReadFile(filepath.Join(dir, "world", "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "CURRENT WORLD" {
		t.Errorf("current world was modified: %q", got)
	}
}
