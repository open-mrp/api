package crypto

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashBcrypt_LengthLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"71 bytes", strings.Repeat("a", 71), false},
		{"72 bytes", strings.Repeat("a", 72), false},
		{"73 bytes", strings.Repeat("a", 73), true},
		{"255 bytes", strings.Repeat("a", 255), true},
		// The login endpoint's validate:"max=72" counts runes, so a 72-rune password of 4-byte runes reaches bcrypt as 288 bytes.
		{"72 multi-byte runes", strings.Repeat("😀", 72), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash, err := HashBcrypt(tt.password)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("HashBcrypt() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("HashBcrypt() expected error, got hash %q", hash)
			}
			if !errors.Is(err, bcrypt.ErrPasswordTooLong) {
				t.Fatalf("HashBcrypt() error = %v, want wrapped bcrypt.ErrPasswordTooLong", err)
			}
			if hash != "" {
				t.Errorf("HashBcrypt() returned a hash alongside an error: %q", hash)
			}
		})
	}
}

// TestCompareBcryptHash_IgnoresBytesPast72 pins bcrypt's truncation: only the first 72 bytes of the candidate are keyed, so a longer candidate sharing that prefix verifies. HashBcrypt's >72-byte rejection is the only thing keeping such a password out of the database — any caller that stores hashes by another path inherits this.
func TestCompareBcryptHash_IgnoresBytesPast72(t *testing.T) {
	t.Parallel()
	stored := strings.Repeat("a", 72)

	hash, err := HashBcrypt(stored)
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash(hash, stored+"completely-different-suffix")
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}
	if !match {
		t.Error("CompareBcryptHash returned false; bcrypt truncation behavior changed, review callers that rely on HashBcrypt rejecting >72 bytes")
	}

	// A difference inside the first 72 bytes must still be rejected.
	differing := "b" + strings.Repeat("a", 71) + "completely-different-suffix"
	match, err = CompareBcryptHash(hash, differing)
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}
	if match {
		t.Error("CompareBcryptHash should return false when the first 72 bytes differ")
	}
}

func TestCompareBcryptHash_ErrorPaths(t *testing.T) {
	t.Parallel()
	valid, err := HashBcrypt("correctPassword")
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	tests := []struct {
		name    string
		hash    string
		wantErr error
	}{
		{"empty hash", "", bcrypt.ErrHashTooShort},
		{"truncated hash", valid[:20], bcrypt.ErrHashTooShort},
		{"missing dollar prefix", "2a$10$" + valid[7:], bcrypt.InvalidHashPrefixError('2')},
		{"cost out of range", "$2a$99$" + valid[7:], bcrypt.InvalidCostError(99)},
		{"newer major version", "$3a$" + valid[4:], bcrypt.HashVersionTooNewError('3')},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			match, err := CompareBcryptHash(tt.hash, "correctPassword")
			if err == nil {
				t.Fatalf("CompareBcryptHash() expected error for %q", tt.name)
			}
			if match {
				t.Error("CompareBcryptHash() returned true alongside an error")
			}
			// Only ErrMismatchedHashAndPassword may collapse to (false, nil); every other failure must stay distinguishable so a corrupt hash column can never authenticate.
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CompareBcryptHash() error = %v, want wrapped %v", err, tt.wantErr)
			}
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				t.Fatal("CompareBcryptHash() reported a malformed hash as a plain mismatch")
			}
		})
	}
}

func TestCompareBcryptHash_EmptyPasswordAgainstRealHash(t *testing.T) {
	t.Parallel()
	hash, err := HashBcrypt("correctPassword")
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash(hash, "")
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}
	if match {
		t.Error("CompareBcryptHash should return false for an empty password")
	}
}

func TestHashBcrypt_EmptyPasswordRoundTrips(t *testing.T) {
	t.Parallel()
	hash, err := HashBcrypt("")
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash(hash, "")
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}
	if !match {
		t.Error("CompareBcryptHash should return true for the empty password it hashed")
	}

	match, err = CompareBcryptHash(hash, "x")
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}
	if match {
		t.Error("a hash of the empty password must not accept a non-empty password")
	}
}
