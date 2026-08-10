package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestPlayerListIncludesOperatorState(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	inst := env.newServerProject(t, "player-ops")
	now := time.Now().UTC().Format(time.RFC3339)

	ops := `[{"uuid":"11111111-2222-3333-4444-555555555555","name":"iKlaude","level":4}]`
	if err := os.WriteFile(filepath.Join(inst.AbsoluteDir(env.app.Home), "ops.json"), []byte(ops), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, player := range []struct {
		name string
		uuid string
	}{
		{"iKlaude", "11111111222233334444555555555555"},
		{"Alex", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
	} {
		if _, err := env.app.DB.Exec(`INSERT INTO players
			(instance_id, username, uuid, first_seen_at, last_seen_at, is_online)
			VALUES (?, ?, ?, ?, ?, 1)`, inst.ID, player.name, player.uuid, now, now); err != nil {
			t.Fatal(err)
		}
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	var response struct {
		Players []struct {
			Username string `json:"username"`
			OP       bool   `json:"op"`
		} `json:"players"`
	}
	if status, body := c.do(http.MethodGet, "/api/players", nil, &response); status != http.StatusOK {
		t.Fatalf("player list failed: %d %s", status, body)
	}

	got := map[string]bool{}
	for _, player := range response.Players {
		got[player.Username] = player.OP
	}
	if !got["iKlaude"] {
		t.Error("iKlaude should be marked as an operator")
	}
	if got["Alex"] {
		t.Error("Alex should not be marked as an operator")
	}
}

func TestPlayerAvatarProxiesSameOriginImage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/helm/Alex/64.png" {
			t.Fatalf("unexpected upstream path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer upstream.Close()

	oldBase := playerAvatarBaseURL
	oldClient := playerAvatarClient
	playerAvatarBaseURL = upstream.URL + "/helm"
	playerAvatarClient = upstream.Client()
	t.Cleanup(func() {
		playerAvatarBaseURL = oldBase
		playerAvatarClient = oldClient
	})

	env := newTestEnv(t)
	secret := env.createUser("viewer", "correct horse battery", authorization.RoleViewer)
	c := env.newClient()
	c.mustLogin("viewer", "correct horse battery", secret)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/players/avatar?username=Alex&size=64", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("avatar status = %d, body = %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
	if string(body) != "png-bytes" {
		t.Fatalf("avatar body = %q", body)
	}
}
