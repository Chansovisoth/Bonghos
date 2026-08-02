package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// ----- Argon2id password hashing ---------------------------------------------

const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword returns a PHC-formatted argon2id hash.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword checks password against a PHC argon2id hash in constant time.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash is a real argon2id hash of a random password, used to equalize
// work for nonexistent usernames (anti-enumeration).
var dummyHash string

func init() {
	dummyHash, _ = HashPassword("bonghos-dummy-anti-enumeration-password")
}

// DummyVerify performs password verification work without a real account.
func DummyVerify(password string) {
	VerifyPassword(password, dummyHash)
}

// ----- TOTP (RFC 6238, SHA-1, 6 digits, 30 s) --------------------------------

var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateTOTPSecret returns a new base32 secret (160 bits).
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return totpEncoding.EncodeToString(raw), nil
}

// TOTPCode computes the 6-digit code for the given secret and time.
func TOTPCode(secret string, at time.Time) (string, error) {
	key, err := totpEncoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", errors.New("invalid totp secret")
	}
	counter := uint64(at.Unix()) / 30
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff) % 1000000
	return fmt.Sprintf("%06d", code), nil
}

// VerifyTOTP accepts the current step and one step of clock skew either way.
func VerifyTOTP(secret, code string, at time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	ok := false
	for _, skew := range []time.Duration{0, -30 * time.Second, 30 * time.Second} {
		want, err := TOTPCode(secret, at.Add(skew))
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			ok = true // do not early-return: keep work uniform
		}
	}
	return ok
}

// DummyTOTPWork performs equivalent TOTP computation for anti-enumeration.
func DummyTOTPWork(code string) {
	VerifyTOTP("JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP", code, time.Now())
}

// TOTPProvisioningURI builds an otpauth:// URI for authenticator apps.
func TOTPProvisioningURI(username, secret string) string {
	return fmt.Sprintf("otpauth://totp/Bonghos:%s?secret=%s&issuer=Bonghos&digits=6&period=30", username, secret)
}

// ----- token helpers ---------------------------------------------------------

// NewToken returns (plaintextToken, sha256HexOfToken).
func NewToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	tok := hex.EncodeToString(raw)
	return tok, HashToken(tok), nil
}

// HashToken hashes a session/invitation/recovery token for storage.
func HashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

// GenerateRecoveryCodes returns n human-typable one-use codes.
func GenerateRecoveryCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := range codes {
		raw := make([]byte, 5)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		s := strings.ToLower(hex.EncodeToString(raw))
		codes[i] = s[:5] + "-" + s[5:]
	}
	return codes, nil
}
