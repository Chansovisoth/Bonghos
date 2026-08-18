package playit

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/database"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "bonghos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Store{DB: db, SecretKey: []byte("0123456789abcdef0123456789abcdef")}
}

func TestStoreDefaultsPreserveManualNetworking(t *testing.T) {
	store := testStore(t)
	config, err := store.Config()
	if err != nil {
		t.Fatal(err)
	}
	if config.Enabled || config.ManagementMode != ManagementNone || config.SecretConfigured {
		t.Fatalf("unexpected default config: %+v", config)
	}
}

func TestStoreEncryptsSecretAndClearsOldTunnelOnRelink(t *testing.T) {
	store := testStore(t)
	const firstSecret = "first-playit-secret"
	if _, err := store.CompleteClaim(firstSecret, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAgent("old-agent"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTunnel("old-tunnel", "old.example:25565", 25565); err != nil {
		t.Fatal(err)
	}
	const secondSecret = "second-playit-secret"
	config, err := store.CompleteClaim(secondSecret, 0)
	if err != nil {
		t.Fatal(err)
	}
	if config.AgentID != "" || config.TunnelID != "" || config.PublicAddress != "" {
		t.Fatalf("relink retained old agent state: %+v", config)
	}
	var encrypted []byte
	if err := store.DB.QueryRow(`SELECT agent_secret_enc FROM playit_settings WHERE id=1`).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte(secondSecret)) {
		t.Fatal("Playit secret was stored in plaintext")
	}
	secret, err := store.Secret()
	if err != nil || secret != secondSecret {
		t.Fatalf("secret round trip = %q, %v", secret, err)
	}
}

func TestExternalAddressMustBeExplicitAndSafe(t *testing.T) {
	store := testStore(t)
	for _, address := range []string{"", "https://example.com", "bad host:25565"} {
		if _, err := store.SaveExternalAddress(address, 25565, 0); err == nil {
			t.Fatalf("accepted unsafe external address %q", address)
		}
	}
	config, err := store.SaveExternalAddress("example.gl.joinmc.link:25565", 25565, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !config.Enabled || config.ManagementMode != ManagementExternal {
		t.Fatalf("unexpected external config: %+v", config)
	}
}

func TestDisablingPreservesSelectedManagementMode(t *testing.T) {
	store := testStore(t)
	if _, err := store.SetPreference(false, AccountModeAccount, ManagementBonghos, 0); err != nil {
		t.Fatal(err)
	}
	config, err := store.Config()
	if err != nil {
		t.Fatal(err)
	}
	if config.Enabled || config.ManagementMode != ManagementBonghos {
		t.Fatalf("disabled config forgot mode: %+v", config)
	}
}

func TestConfigDoesNotDecryptCredential(t *testing.T) {
	store := testStore(t)
	if _, err := store.DB.Exec(`UPDATE playit_settings SET agent_secret_enc=? WHERE id=1`, []byte("not encrypted")); err != nil {
		t.Fatal(err)
	}
	config, err := store.Config()
	if err != nil || !config.SecretConfigured {
		t.Fatalf("safe config = %+v, %v", config, err)
	}
	if _, err := store.Secret(); err == nil {
		t.Fatal("expected corrupt credential decryption to fail")
	}
}

func TestClearTunnelPreservesAgentAndPort(t *testing.T) {
	store := testStore(t)
	if _, err := store.CompleteClaim("agent-secret", 0); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTunnel("tunnel-id", "example.gl.joinmc.link", 25566); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearTunnel(); err != nil {
		t.Fatal(err)
	}
	config, err := store.Config()
	if err != nil {
		t.Fatal(err)
	}
	if config.TunnelID != "" || config.PublicAddress != "" || config.LocalPort != 25566 || !config.SecretConfigured {
		t.Fatalf("clear tunnel changed unrelated state: %+v", config)
	}
}
