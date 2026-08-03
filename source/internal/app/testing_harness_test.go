package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

// testEnv is a fully wired Bonghos control plane backed by a throwaway home
// directory, driven through the real HTTP handler. These tests exercise the
// same middleware, authorization and handlers a browser would hit.
type testEnv struct {
	t      *testing.T
	app    *App
	server *httptest.Server
	home   string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	home := t.TempDir()
	a, err := New(home, nil)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(func() {
		srv.Close()
		a.Close()
	})
	return &testEnv{t: t, app: a, server: srv, home: home}
}

// createUser makes an account directly through the store, returning the
// TOTP secret so a client can sign in.
func (e *testEnv) createUser(username, password string, role authorization.Role) string {
	e.t.Helper()
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		e.t.Fatalf("GenerateTOTPSecret: %v", err)
	}
	if _, _, err := e.app.Auth.CreateUser(username, password, secret, role); err != nil {
		e.t.Fatalf("CreateUser(%s): %v", username, err)
	}
	return secret
}

// client is an authenticated (or anonymous) HTTP client that maintains the
// session cookie and CSRF token exactly like the browser frontend does.
type client struct {
	t    *testing.T
	env  *testEnv
	http *http.Client
	csrf string
}

func (e *testEnv) newClient() *client {
	e.t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatalf("cookiejar: %v", err)
	}
	c := &client{t: e.t, env: e, http: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
	c.refreshCSRF()
	return c
}

func (c *client) refreshCSRF() {
	var out struct {
		CSRF string `json:"csrf"`
	}
	if status, _ := c.do("GET", "/api/auth/csrf", nil, &out); status == 200 {
		c.csrf = out.CSRF
	}
}

// do performs a request and decodes the JSON body into out (which may be nil).
// It returns the status code and the raw body for assertions on errors.
func (c *client) do(method, path string, body any, out any) (int, string) {
	c.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.env.server.URL+path, rdr)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != "GET" && c.csrf != "" {
		req.Header.Set("X-Bonghos-CSRF", c.csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, out)
	}
	return resp.StatusCode, strings.TrimSpace(string(raw))
}

// login performs the real login flow with a valid TOTP code.
func (c *client) login(username, password, secret string) (int, string) {
	c.t.Helper()
	code, err := auth.TOTPCode(secret, time.Now())
	if err != nil {
		c.t.Fatalf("TOTPCode: %v", err)
	}
	status, body := c.do("POST", "/api/auth/login", map[string]string{
		"username": username, "password": password, "code": code,
	}, nil)
	if status == 200 {
		c.refreshCSRF()
	}
	return status, body
}

// mustLogin fails the test if authentication does not succeed.
func (c *client) mustLogin(username, password, secret string) {
	c.t.Helper()
	if status, body := c.login(username, password, secret); status != 200 {
		c.t.Fatalf("login as %s failed: %d %s", username, status, body)
	}
}
