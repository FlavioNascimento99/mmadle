package domain

import (
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	valid := []string{"poatan", "Fan_99", "  MMAFan  ", "a_b_c"}
	for _, u := range valid {
		if err := ValidateUsername(u); err != nil {
			t.Fatalf("ValidateUsername(%q) = %v, want nil", u, err)
		}
	}
	invalid := []string{"", "ab", "fan@mmadle.gg", "has space", "dash-name", "dots.name", strings.Repeat("x", 21), "ünïcode"}
	for _, u := range invalid {
		if err := ValidateUsername(u); err == nil {
			t.Fatalf("ValidateUsername(%q) = nil, want error", u)
		}
	}
	if got := NormalizeUsername("  PoaTan_99  "); got != "poatan_99" {
		t.Fatalf("NormalizeUsername = %q", got)
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short9"); err == nil {
		t.Fatal("short password must be rejected")
	}
	if err := ValidatePassword("a-correct-horse-battery9"); err != nil {
		t.Fatalf("long password rejected: %v", err)
	}
	for _, common := range []string{"password123", "Password123", "  PASSWORD123  "} {
		// Surrounding spaces are part of the password; the blocklist is
		// case-insensitive on the exact value.
		err := ValidatePassword(strings.TrimSpace(common))
		if err == nil {
			t.Fatalf("common password %q must be rejected", common)
		}
	}
	// Exactly 10 chars is the minimum; blocklisted values are still rejected.
	if err := ValidatePassword("1234567890"); err == nil {
		t.Fatal("blocklisted 10-char password must be rejected")
	}
	if err := ValidatePassword("uFc-faN-09!x"); err != nil {
		t.Fatalf("10-char unique password rejected: %v", err)
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("a-correct-horse-battery9")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash must be PHC argon2id, got %q", hash[:20])
	}
	if err := VerifyPassword(hash, "a-correct-horse-battery9"); err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}
	if err := VerifyPassword(hash, "wrong-password-1"); err == nil {
		t.Fatal("wrong password must be rejected")
	}
	if err := VerifyPassword("not-a-hash", "whatever-password"); err == nil {
		t.Fatal("malformed hash must fail closed")
	}
	// Hashes embed a random salt: two hashes of the same password differ,
	// but both verify.
	other, err := HashPassword("a-correct-horse-battery9")
	if err != nil {
		t.Fatal(err)
	}
	if other == hash {
		t.Fatal("salts must randomize hashes")
	}
	if err := VerifyPassword(other, "a-correct-horse-battery9"); err != nil {
		t.Fatalf("second hash rejected: %v", err)
	}
}

func TestSessionToken(t *testing.T) {
	tok, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 43 {
		t.Fatalf("32-byte token base64url = 43 chars, got %d", len(tok))
	}
	other, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if tok == other {
		t.Fatal("tokens must be unique")
	}
	hashed := HashSessionToken(tok)
	if len(hashed) != 64 {
		t.Fatalf("SHA-256 hex = 64 chars, got %d", len(hashed))
	}
	if hashed == tok || HashSessionToken(tok) != hashed {
		t.Fatal("token hashing must be deterministic and one-way in shape")
	}
}
