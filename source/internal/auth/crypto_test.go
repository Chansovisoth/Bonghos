package auth

import (
	"testing"
	"time"
)

func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("correct horse battery staple", h) {
		t.Error("valid password rejected")
	}
	if VerifyPassword("wrong", h) {
		t.Error("wrong password accepted")
	}
}

// RFC 6238 test vector: secret "12345678901234567890" base32 GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ
// SHA-1, 30s step, 8 digits truncated to 6 for comparison of our 6-digit output.
func TestTOTPKnownVector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	// At time 59 (counter 1) the RFC 8-digit value is 94287082 → 6-digit 287082.
	code, err := TOTPCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	if code != "287082" {
		t.Errorf("TOTP at t=59 = %s, want 287082", code)
	}
}

func TestVerifyTOTPWindow(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	code, _ := TOTPCode(secret, now)
	if !VerifyTOTP(secret, code, now) {
		t.Error("current code rejected")
	}
	prev, _ := TOTPCode(secret, now.Add(-30*time.Second))
	if !VerifyTOTP(secret, prev, now) {
		t.Error("previous-step code rejected (drift window)")
	}
	old, _ := TOTPCode(secret, now.Add(-5*time.Minute))
	if VerifyTOTP(secret, old, now) {
		t.Error("stale code accepted")
	}
	if VerifyTOTP(secret, "000000", now) {
		// Extremely unlikely to be valid; treat as failure if it is.
		c1, _ := TOTPCode(secret, now)
		c2, _ := TOTPCode(secret, now.Add(-30*time.Second))
		c3, _ := TOTPCode(secret, now.Add(30*time.Second))
		if c1 != "000000" && c2 != "000000" && c3 != "000000" {
			t.Error("invalid code accepted")
		}
	}
}

func TestValidateUsername(t *testing.T) {
	for _, ok := range []string{"alice", "bob_2", "k-9", "User01"} {
		if err := ValidateUsername(ok); err != nil {
			t.Errorf("ValidateUsername(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"", "a b", "root!", "x", "waaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaay-too-long-username-over-limit"} {
		if err := ValidateUsername(bad); err == nil {
			t.Errorf("ValidateUsername(%q) expected error", bad)
		}
	}
}
