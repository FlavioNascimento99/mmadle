// Package domain holds the pure game logic for MMAdle.
//
// Design rules:
//   - No HTTP, no SQL, no I/O here: handlers call services which call this
//     package, so the comparison system can grow new attributes without
//     touching transport or persistence code.
//   - The frontend never computes comparisons; every attribute returns a
//     semantic Comparison value computed here.
//   - Derived values (age, last event, formatted record) are calculated from
//     canonical data, never stored.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Authentication rules (see issue #1). Passwords are hashed with argon2id;
// sessions are opaque random tokens whose SHA-256 hash is stored server-side.
// The raw token only ever travels in the session cookie and is never logged.
const (
	// MinPasswordLength is the minimum accepted password length.
	MinPasswordLength = 10
	// MaxPasswordLength bounds hashing work and request size.
	MaxPasswordLength = 256
	// MaxEmailLength bounds the email field (RFC 5321 path limit is 254).
	MaxEmailLength = 254
	// MaxDisplayNameLength bounds the optional display name.
	MaxDisplayNameLength = 30
	// DefaultSessionTTL is how long a session cookie stays valid.
	DefaultSessionTTL = 30 * 24 * time.Hour
	// sessionTokenBytes is the entropy of each opaque session token (256 bit).
	sessionTokenBytes = 32
)

// Auth validation errors.
var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrWeakPassword    = errors.New("password does not meet requirements")
	ErrInvalidDisplay  = errors.New("invalid display name")
	ErrInvalidPassword = errors.New("invalid credentials")
)

// commonPasswords is a small blocklist checked case-insensitively. Passwords
// must also meet the minimum length, so short entries here are belt-and-braces.
var commonPasswords = map[string]bool{
	"password": true, "password1": true, "password123": true, "password1234": true,
	"1234567890": true, "12345678910": true, "qwertyuiop": true, "qwerty12345": true,
	"letmein123": true, "letmein1234": true, "welcome123": true, "welcome1234": true,
	"changeme123": true, "football12": true, "baseball12": true, "superman12": true,
	"trustno123": true, "dragon1234": true, "master1234": true, "monkey1234": true,
	"mmadle1234": true, "ufc1234567": true, "mma1234567": true,
}

// NormalizeEmail trims whitespace and lowercases. Email uniqueness is enforced
// by a CITEXT column, so this is for validation/rate-limit keys, not security.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail rejects empty, overlong, or malformed addresses. It is
// intentionally simple: deliverability is not verified in v1 (no email
// provider yet; see the verification open question in issue #1).
func ValidateEmail(email string) error {
	email = NormalizeEmail(email)
	if email == "" || len(email) > MaxEmailLength {
		return ErrInvalidEmail
	}
	if strings.Contains(email, " ") || !strings.Contains(email, "@") {
		return ErrInvalidEmail
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ErrInvalidEmail
	}
	if !strings.Contains(parts[1], ".") || strings.HasPrefix(parts[1], ".") || strings.HasSuffix(parts[1], ".") {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword enforces minimum length and rejects common passwords.
// Comparison is case-insensitive so "Password123" cannot dodge the blocklist.
func ValidatePassword(password string) error {
	if !utf8.ValidString(password) {
		return ErrWeakPassword
	}
	if n := len([]rune(password)); n < MinPasswordLength || len(password) > MaxPasswordLength {
		return ErrWeakPassword
	}
	if commonPasswords[strings.ToLower(password)] {
		return ErrWeakPassword
	}
	return nil
}

// ValidateDisplayName allows empty (optional) and otherwise requires 1-30
// printable characters with no surrounding whitespace games.
func ValidateDisplayName(name string) error {
	if name == "" {
		return nil
	}
	if strings.TrimSpace(name) != name || name != strings.Join(strings.Fields(name), " ") {
		return ErrInvalidDisplay
	}
	if n := len([]rune(name)); n < 1 || n > MaxDisplayNameLength {
		return ErrInvalidDisplay
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return ErrInvalidDisplay
		}
	}
	return nil
}

// argon2id parameters (OWASP-recommended interactive profile shape).
const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

// HashPassword hashes with argon2id and encodes in PHC format:
// $argon2id$v=19$m=65536,t=3,p=4$<base64 salt>$<base64 hash>.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyPassword parses a PHC hash from HashPassword and compares in constant
// time. Unknown formats fail closed. A dummy hash keeps login timing similar
// when the email does not exist (callers pass it explicitly).
func VerifyPassword(encodedHash, password string) error {
	var m, t uint32
	var p uint8
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return ErrInvalidPassword
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return ErrInvalidPassword
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return ErrInvalidPassword
	}
	saltB64, hashB64 := parts[4], parts[5]
	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return ErrInvalidPassword
	}
	want, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return ErrInvalidPassword
	}
	if m == 0 || t == 0 || p == 0 || len(salt) == 0 || len(want) == 0 {
		return ErrInvalidPassword
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrInvalidPassword
	}
	return nil
}

// GenerateSessionToken returns a 256-bit opaque token (base64url, 43 chars)
// for the session cookie. Only its SHA-256 hash is stored server-side.
func GenerateSessionToken() (string, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashSessionToken hashes a token for storage/lookup. Hex-encoded SHA-256
// (64 chars) matches the sessions.token_hash CHECK constraint.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum)
}
