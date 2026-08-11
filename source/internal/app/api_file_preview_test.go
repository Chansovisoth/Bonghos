package app

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestFileImagePreviewIsInlineAndLimitedToRasterImages(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	server := env.newServerProject(t, "preview-project")
	if err := env.app.Instances.SetActive(server.ID); err != nil {
		t.Fatal(err)
	}

	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	dir := server.AbsoluteDir(env.app.Home)
	if err := os.WriteFile(filepath.Join(dir, "server-icon.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	resp, err := c.http.Get(env.server.URL + "/api/files/preview?path=" + url.QueryEscape("server-icon.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("image preview = %d %s, want 200", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
	if got := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(got, "inline;") {
		t.Fatalf("Content-Disposition = %q, want inline", got)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if string(body) != string(png) {
		t.Fatal("preview response did not preserve image bytes")
	}

	if status, _ := c.do(http.MethodGet, "/api/files/preview?path=notes.txt", nil, nil); status != http.StatusBadRequest {
		t.Fatalf("text preview = %d, want 400", status)
	}
}
