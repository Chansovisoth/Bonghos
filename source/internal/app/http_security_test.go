package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestWebThemeBootstrapCompliesWithScriptCSP(t *testing.T) {
	webSource := filepath.Join("..", "..", "web", "src")
	index, err := os.ReadFile(filepath.Join(webSource, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	scripts := regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`).FindAllSubmatch(index, -1)
	for _, script := range scripts {
		attributes := strings.ToLower(string(script[1]))
		body := strings.TrimSpace(string(script[2]))
		if body != "" && !strings.Contains(attributes, "src=") {
			t.Fatalf("index.html contains inline JavaScript blocked by script-src 'self': %q", body)
		}
	}
	if !strings.Contains(string(index), `<script src="/theme-init.js"></script>`) {
		t.Fatal("index.html does not load the external theme bootstrap")
	}
	theme, err := os.ReadFile(filepath.Join(webSource, "theme-init.js"))
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(theme))) == 0 {
		t.Fatal("theme-init.js is empty")
	}
}

func TestRequestUsesHTTPS(t *testing.T) {
	direct := httptest.NewRequest("GET", "https://panel.example/api", nil)
	if !requestUsesHTTPS(direct) {
		t.Fatal("direct TLS request was not recognized as HTTPS")
	}

	proxied := httptest.NewRequest("POST", "http://panel.example/api", nil)
	proxied.Header.Set("Origin", "https://panel.example")
	if !requestUsesHTTPS(proxied) {
		t.Fatal("same-origin HTTPS request through a proxy was not recognized")
	}

	forged := httptest.NewRequest("POST", "http://panel.example/api", nil)
	forged.Header.Set("Origin", "https://attacker.example")
	if requestUsesHTTPS(forged) {
		t.Fatal("cross-origin header was trusted as HTTPS")
	}
}

func TestSecurityHeadersAllowOnlyTurnstileChallengeOrigin(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodGet, "http://panel.example/", nil)
	response := httptest.NewRecorder()
	a.secureHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(response, req)
	csp := response.Header().Get("Content-Security-Policy")
	for _, directive := range []string{
		"script-src 'self' https://challenges.cloudflare.com",
		"frame-src https://challenges.cloudflare.com",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP missing %q: %s", directive, csp)
		}
	}
	if strings.Contains(csp, "static.cloudflareinsights.com") {
		t.Fatalf("CSP unexpectedly permits Cloudflare analytics: %s", csp)
	}
}

func TestReadJSONRejectsTrailingValues(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"ok":true} {"extra":true}`))
	var dst struct {
		OK bool `json:"ok"`
	}
	if err := readJSON(req, &dst, 1024); err == nil {
		t.Fatal("readJSON accepted multiple JSON values")
	}
}

func TestIssueCSRFCannotDowngradeExistingCookie(t *testing.T) {
	token := strings.Repeat("a", 64)
	a := &App{}

	proxiedGET := httptest.NewRequest(http.MethodGet, "http://panel.example/api/auth/csrf", nil)
	proxiedGET.Host = "panel.example"
	proxiedGET.AddCookie(&http.Cookie{Name: csrfCookie, Value: token, Secure: true})
	response := httptest.NewRecorder()
	if got := a.issueCSRF(response, proxiedGET); got != token {
		t.Fatalf("CSRF token = %q, want existing token", got)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("origin-less GET reissued existing cookie: %+v", cookies)
	}

	httpsPOST := httptest.NewRequest(http.MethodPost, "http://panel.example/api/login", nil)
	httpsPOST.Host = "panel.example"
	httpsPOST.Header.Set("Origin", "https://panel.example")
	httpsPOST.AddCookie(&http.Cookie{Name: csrfCookie, Value: token})
	response = httptest.NewRecorder()
	a.issueCSRF(response, httpsPOST)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("HTTPS request did not secure existing cookie: %+v", cookies)
	}
}
