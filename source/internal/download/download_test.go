package download

import (
	"net"
	"strings"
	"testing"
)

func blockedIPString(s string) bool { return blockedIP(net.ParseIP(s)) }

func TestValidateURLSchemes(t *testing.T) {
	o := DefaultOptions()
	if _, err := ValidateURL("https://example.com/pack.zip", o); err != nil {
		t.Errorf("https rejected: %v", err)
	}
	for _, bad := range []string{
		"http://example.com/pack.zip", // insecure by default
		"ftp://example.com/x",
		"file:///etc/passwd",
		"ssh://host/x",
		"data:text/plain,hi",
		"javascript:alert(1)",
		"https://user:pass@example.com/x", // embedded credentials
	} {
		if _, err := ValidateURL(bad, o); err == nil {
			t.Errorf("ValidateURL(%q) expected rejection", bad)
		}
	}
	o.AllowInsecureHTTP = true
	if _, err := ValidateURL("http://example.com/pack.zip", o); err != nil {
		t.Errorf("http with explicit opt-in rejected: %v", err)
	}
}

func TestBlockedIPs(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.0.0.5", "192.168.1.1", "172.16.0.9",
		"169.254.169.254", // cloud metadata / link-local
		"0.0.0.0", "224.0.0.1", "::1", "fe80::1", "fd00::1",
	}
	for _, s := range blocked {
		if !blockedIPString(s) {
			t.Errorf("IP %s should be blocked", s)
		}
	}
	for _, s := range []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"} {
		if blockedIPString(s) {
			t.Errorf("public IP %s wrongly blocked", s)
		}
	}
}

func TestRedactURL(t *testing.T) {
	r := RedactURL("https://example.com/pack.zip?token=SECRET123&sig=abc")
	if strings.Contains(r, "SECRET123") || strings.Contains(r, "abc") {
		t.Errorf("RedactURL leaked query values: %s", r)
	}
	if !strings.Contains(r, "example.com") {
		t.Errorf("RedactURL lost host: %s", r)
	}
}

func TestTrustedHosts(t *testing.T) {
	o := DefaultOptions()
	o.TrustedHosts = []string{"downloads.example.com"}
	if _, err := ValidateURL("https://downloads.example.com/p.zip", o); err != nil {
		t.Errorf("trusted host rejected: %v", err)
	}
	if _, err := ValidateURL("https://evil.example.net/p.zip", o); err == nil {
		t.Error("untrusted host accepted while allowlist active")
	}
}
