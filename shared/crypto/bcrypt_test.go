package crypto

import (
	"testing"
)

func TestHashBcrypt_RoundTrip(t *testing.T) {
	t.Parallel()
	password := "correctPassword123!"

	hash, err := HashBcrypt(password)
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash(hash, password)
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}

	if !match {
		t.Error("CompareBcryptHash should return true for correct password")
	}
}

func TestCompareBcryptHash_WrongPassword(t *testing.T) {
	t.Parallel()
	hash, err := HashBcrypt("correctPassword")
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash(hash, "wrongPassword")
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}

	if match {
		t.Error("CompareBcryptHash should return false for wrong password")
	}
}

func TestCompareBcryptHash_InvalidHash(t *testing.T) {
	t.Parallel()
	match, err := CompareBcryptHash("invalidhash", "password")

	if err == nil {
		t.Error("CompareBcryptHash should return error for invalid hash")
	}

	if match {
		t.Error("CompareBcryptHash should return false for invalid hash")
	}
}
