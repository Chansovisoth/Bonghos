package security

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWithinRoot(t *testing.T) {
	root := t.TempDir()
	if !WithinRoot(root, filepath.Join(root, "a", "b")) {
		t.Error("child should be within root")
	}
	if WithinRoot(root, filepath.Join(root, "..", "escape")) {
		t.Error("../ escape must not be within root")
	}
	// String-prefix trap: /tmp/rootX must not count as inside /tmp/root.
	if WithinRoot(root, root+"x/file") {
		t.Error("sibling with shared prefix must not be within root")
	}
	if !WithinRoot(root, root) {
		t.Error("root itself should be within root")
	}
}

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	if _, err := SafeJoin(root, "ok", "path.txt"); err != nil {
		t.Errorf("SafeJoin plain: %v", err)
	}
	for _, parts := range [][]string{{".."}, {"a", "..", "..", "b"}, {"/abs"}} {
		if _, err := SafeJoin(root, parts...); err == nil {
			t.Errorf("SafeJoin(%v) expected error", parts)
		}
	}
}

func TestSecretKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "secret.key")
	if err := GenerateSecretKey(p); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("secret.key mode = %v, want 0600", st.Mode().Perm())
	}
	key, err := LoadSecretKey(p)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("totp-secret-material")
	ct, err := Encrypt(key, msg)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, msg) {
		t.Error("ciphertext contains plaintext")
	}
	pt, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, msg) {
		t.Error("decrypt mismatch")
	}
	// Tampering must fail (authenticated encryption).
	ct[len(ct)-1] ^= 0xff
	if _, err := Decrypt(key, ct); err == nil {
		t.Error("tampered ciphertext decrypted without error")
	}
}
