package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
