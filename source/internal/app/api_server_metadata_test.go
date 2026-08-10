package app

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/instance"
)

func TestServerListBackfillsDetectedVersions(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	inst := env.newServerProject(t, "metadata-pack")
	variables := filepath.Join(inst.AbsoluteDir(env.app.Home), "variables.txt")
	if err := os.WriteFile(variables, []byte("MINECRAFT_VERSION=1.21.1\nMODLOADER=NeoForge\nMODLOADER_VERSION=21.1.228\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var response struct {
		Servers []*instance.Instance `json:"servers"`
	}
	if status, body := client.do(http.MethodGet, "/api/servers", nil, &response); status != http.StatusOK {
		t.Fatalf("GET /api/servers: %d %s", status, body)
	}
	if len(response.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(response.Servers))
	}
	got := response.Servers[0]
	if got.MinecraftVersion != "1.21.1" || got.Modloader != "neoforge" || got.ModloaderVersion != "21.1.228" {
		t.Fatalf("metadata = minecraft %q loader %q version %q", got.MinecraftVersion, got.Modloader, got.ModloaderVersion)
	}

	persisted, err := env.app.Instances.ByID(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.MinecraftVersion != got.MinecraftVersion || persisted.ModloaderVersion != got.ModloaderVersion {
		t.Fatalf("detected metadata was not persisted: %+v", persisted)
	}
}
