package crypto

import (
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("this is a secret API key value")

	encrypted, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := DecryptAESGCM(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("same plaintext")

	enc1, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	enc2, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if enc1 == enc2 {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatal(err)
	}

	encrypted, err := EncryptAESGCM([]byte("secret"), key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptAESGCM(encrypted, key2)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	encrypted, err := EncryptAESGCM([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the ciphertext by modifying a character
	tampered := []byte(encrypted)
	if tampered[len(tampered)-2] == 'A' {
		tampered[len(tampered)-2] = 'B'
	} else {
		tampered[len(tampered)-2] = 'A'
	}

	_, err = DecryptAESGCM(string(tampered), key)
	if err == nil {
		t.Fatal("expected error decrypting tampered ciphertext")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	shortKey := make([]byte, 16)

	_, err := EncryptAESGCM([]byte("test"), shortKey)
	if err == nil {
		t.Fatal("expected error with short key")
	}

	_, err = DecryptAESGCM("dGVzdA==", shortKey)
	if err == nil {
		t.Fatal("expected error with short key")
	}
}
