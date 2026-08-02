// Package security provides canonical path validation, the portable secret
// key, and authenticated encryption used for TOTP secrets and future
// provider credentials.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathEscape   = errors.New("path escapes its allowed root")
	ErrUnsafeLink   = errors.New("unsafe symlink detected")
	ErrKeyMissing   = errors.New("secret key missing")
	ErrCiphertext   = errors.New("invalid ciphertext")
	ErrKeyMalformed = errors.New("secret key malformed")
)

// WithinRoot reports whether candidate, after lexical cleaning, stays inside
// root. Both paths must be absolute. This is a lexical check; combine with
// EvalWithinRoot when symlinks may be present.
func WithinRoot(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	if candidate == root {
		return true
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// EvalWithinRoot resolves symlinks in the deepest existing ancestor of
// candidate and verifies the resolved location stays inside root. It returns
// the cleaned absolute candidate path on success.
func EvalWithinRoot(root, candidate string) (string, error) {
	root, err := filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("resolving root: %w", err)
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	// Resolve the deepest existing ancestor so a not-yet-created leaf is
	// still validated against the real (symlink-resolved) parent.
	probe := abs
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			joined := filepath.Join(append([]string{resolved}, reverse(tail)...)...)
			if !WithinRoot(root, joined) {
				return "", ErrPathEscape
			}
			return abs, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", ErrPathEscape
		}
		tail = append(tail, filepath.Base(probe))
		probe = parent
	}
}

func reverse(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

// SafeJoin joins root with untrusted relative parts and validates containment.
func SafeJoin(root string, parts ...string) (string, error) {
	for _, p := range parts {
		if filepath.IsAbs(p) || strings.Contains(p, "\x00") {
			return "", ErrPathEscape
		}
	}
	joined := filepath.Join(append([]string{root}, parts...)...)
	if !WithinRoot(root, joined) {
		return "", ErrPathEscape
	}
	return joined, nil
}

// ----- secret key ------------------------------------------------------------

const keyBytes = 32

// GenerateSecretKey creates a random 256-bit key at path with mode 0600.
// It refuses to overwrite an existing key.
func GenerateSecretKey(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("secret key already exists at %s", path)
	}
	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(hex.EncodeToString(key) + "\n"); err != nil {
		return err
	}
	return f.Sync()
}

// LoadSecretKey reads the key from disk and enforces strict permissions.
func LoadSecretKey(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrKeyMissing
	}
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("secret key %s has permissive mode %o; run: chmod 600 %s", path, info.Mode().Perm(), path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(key) != keyBytes {
		return nil, ErrKeyMalformed
	}
	return key, nil
}

// ----- authenticated encryption (AES-256-GCM) --------------------------------

// Encrypt seals plaintext with AES-256-GCM. Output: nonce || ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens data produced by Encrypt.
func Decrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, ErrCiphertext
	}
	pt, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return nil, ErrCiphertext
	}
	return pt, nil
}

// RandomToken returns a hex token with n random bytes.
func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ConstantTimeEqual compares two strings without leaking length-adjusted timing.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
