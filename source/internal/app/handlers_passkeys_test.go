package app

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/go-webauthn/webauthn/webauthn"
)

func TestRequestWebAuthnOriginPolicy(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		wantRP string
		ok     bool
	}{
		{name: "https tunnel", host: "panel.example.com", origin: "https://panel.example.com", wantRP: "panel.example.com", ok: true},
		{name: "https nondefault port", host: "panel.example.com:8443", origin: "https://panel.example.com:8443", wantRP: "panel.example.com", ok: true},
		{name: "https default port normalized", host: "panel.example.com:443", origin: "https://panel.example.com", wantRP: "panel.example.com", ok: true},
		{name: "localhost development", host: "localhost:8080", origin: "http://localhost:8080", wantRP: "localhost", ok: true},
		{name: "reject LAN HTTP", host: "192.168.1.20:8080", origin: "http://192.168.1.20:8080", ok: false},
		{name: "reject host mismatch", host: "panel.example.com", origin: "https://evil.example.com", ok: false},
		{name: "reject missing origin", host: "panel.example.com", origin: "", ok: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "https://"+tc.host+"/api/auth/passkey/begin", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			_, _, rpID, err := requestWebAuthn(req)
			if (err == nil) != tc.ok {
				t.Fatalf("requestWebAuthn() error = %v, want success %v", err, tc.ok)
			}
			if rpID != tc.wantRP {
				t.Fatalf("RP ID = %q, want %q", rpID, tc.wantRP)
			}
		})
	}
}

func TestPasskeyFlowIsBoundAndOneTime(t *testing.T) {
	a := &App{passkeyFlows: make(map[string]passkeyFlow)}
	token, err := a.putPasskeyFlow(passkeyFlow{
		Kind: "register", UserID: 7, Origin: "https://panel.example.com", RPID: "panel.example.com",
		ExpiresAt: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.takePasskeyFlow(token, "register", "https://other.example.com", "other.example.com", 7); err == nil {
		t.Fatal("origin mismatch should fail")
	}
	if _, err := a.takePasskeyFlow(token, "register", "https://panel.example.com", "panel.example.com", 7); err == nil {
		t.Fatal("failed flow should already be consumed")
	}

	token, err = a.putPasskeyFlow(passkeyFlow{
		Kind: "login", Origin: "https://panel.example.com", RPID: "panel.example.com",
		ExpiresAt: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.takePasskeyFlow(token, "login", "https://panel.example.com", "panel.example.com", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := a.takePasskeyFlow(token, "login", "https://panel.example.com", "panel.example.com", 0); err == nil {
		t.Fatal("flow reuse should fail")
	}
}

func TestPasskeyRenameAPIIsUserScopedAndAudited(t *testing.T) {
	env := newTestEnv(t)
	ownerSecret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	memberSecret := env.createUser("member", "another correct horse", authorization.RoleMember)
	owner, err := env.app.Auth.UserByName("owner")
	if err != nil {
		t.Fatal(err)
	}
	created, err := env.app.Auth.AddPasskey(owner.ID, "panel.example.test", "Laptop", &webauthn.Credential{
		ID: []byte("rename-api-credential"), PublicKey: []byte("public-key"),
	})
	if err != nil {
		t.Fatal(err)
	}

	memberClient := env.newClient()
	memberClient.mustLogin("member", "another correct horse", memberSecret)
	path := "/api/passkeys/" + strconv.FormatInt(created.ID, 10)
	if status, body := memberClient.do(http.MethodPatch, path, map[string]string{"name": "Not mine"}, nil); status != http.StatusNotFound {
		t.Fatalf("cross-account rename = %d %s, want 404", status, body)
	}

	ownerClient := env.newClient()
	ownerClient.mustLogin("owner", "correct horse battery", ownerSecret)
	var response struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	status, body := ownerClient.do(http.MethodPatch, path, map[string]string{"name": "  Office key  "}, &response)
	if status != http.StatusOK || response.ID != created.ID || response.Name != "Office key" {
		t.Fatalf("rename passkey = %d %s %+v", status, body, response)
	}
	items, err := env.app.Auth.ListPasskeys(owner.ID)
	if err != nil || len(items) != 1 || items[0].Name != "Office key" {
		t.Fatalf("stored passkeys = %+v, %v", items, err)
	}
	var detail string
	if err := env.app.DB.QueryRow(`SELECT detail FROM audit_log
		WHERE action='passkey_renamed' AND target=? ORDER BY id DESC LIMIT 1`, strconv.FormatInt(created.ID, 10)).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "Laptop") || !strings.Contains(detail, "Office key") {
		t.Fatalf("rename audit detail = %q", detail)
	}
	if status, _ := ownerClient.do(http.MethodPatch, path, map[string]string{"name": "   "}, nil); status != http.StatusBadRequest {
		t.Fatalf("blank passkey rename = %d, want 400", status)
	}
}
