package domain

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{"a@b.co", "Fighter@UFC.COM", "  fan@mmadle.gg  ", "first.last+tag@example.org"}
	for _, e := range valid {
		if err := ValidateEmail(e); err != nil {
			t.Fatalf("ValidateEmail(%q) = %v, want nil", e, err)
		}
	}
	invalid := []string{"", "no-at-sign", "a@b", "a@.com", "a@b.", "a @b.co", "a@@b.co", "@b.co", "a@"}
	for _, e := range invalid {
		if err := ValidateEmail(e); err == nil {
			t.Fatalf("ValidateEmail(%q) = nil, want error", e)
		}
	}
	if got := NormalizeEmail("  FAN@Mmadle.GG  "); got != "fan@mmadle.gg" {
		t.Fatalf("NormalizeEmail = %q", got)
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

func TestValidateDisplayName(t *testing.T) {
	if err := ValidateDisplayName(""); err != nil {
		t.Fatalf("empty display name is optional: %v", err)
	}
	if err := ValidateDisplayName("Conor"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
	for _, bad := range []string{" padded ", "two  spaces", "has\nnewline", strings.Repeat("x", 31)} {
		if err := ValidateDisplayName(bad); err == nil {
			t.Fatalf("display name %q must be rejected", bad)
		}
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
