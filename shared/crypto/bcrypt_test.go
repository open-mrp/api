package crypto

import (
	"testing"
)

func TestHashBcrypt_RoundTrip(t *testing.T) {
	password := "correctPassword123!"

	hash, err := HashBcrypt(password, 4) // low cost for fast tests
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash(password, hash)
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}

	if !match {
		t.Error("CompareBcryptHash should return true for correct password")
	}
}

func TestCompareBcryptHash_WrongPassword(t *testing.T) {
	hash, err := HashBcrypt("correctPassword", 4)
	if err != nil {
		t.Fatalf("HashBcrypt() unexpected error: %v", err)
	}

	match, err := CompareBcryptHash("wrongPassword", hash)
	if err != nil {
		t.Fatalf("CompareBcryptHash() unexpected error: %v", err)
	}

	if match {
		t.Error("CompareBcryptHash should return false for wrong password")
	}
}

func TestCompareBcryptHash_InvalidHash(t *testing.T) {
	match, err := CompareBcryptHash("password", "invalidhash")

	if err == nil {
		t.Error("CompareBcryptHash should return error for invalid hash")
	}

	if match {
		t.Error("CompareBcryptHash should return false for invalid hash")
	}
}
