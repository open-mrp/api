package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

// ---------------------------------------------------------------------------
// EncryptAESGCM
// ---------------------------------------------------------------------------

func TestEncryptAESGCM_RoundTrip(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("this is a secret API key value")
	aad := []byte("apke_abc123")

	encrypted, err := EncryptAESGCM(plaintext, key, aad, "k1")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptAESGCM(encrypted, key, aad)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptAESGCM_RoundTripNilAAD(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("secret")

	encrypted, err := EncryptAESGCM(plaintext, key, nil, "k1")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptAESGCM(encrypted, key, nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptAESGCM_EmptyPlaintext(t *testing.T) {
	key := testKey(t)

	encrypted, err := EncryptAESGCM([]byte{}, key, nil, "k1")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptAESGCM(encrypted, key, nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty plaintext, got %q", decrypted)
	}
}

func TestEncryptAESGCM_LargePlaintext(t *testing.T) {
	key := testKey(t)
	plaintext := make([]byte, 64*1024)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatal(err)
	}

	encrypted, err := EncryptAESGCM(plaintext, key, nil, "k1")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptAESGCM(encrypted, key, nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatal("large plaintext round-trip mismatch")
	}
}

func TestEncryptAESGCM_UniqueNonces(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("same plaintext")

	enc1, err := EncryptAESGCM(plaintext, key, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := EncryptAESGCM(plaintext, key, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}

	if enc1 == enc2 {
		t.Fatal("two encryptions produced identical output; nonce should differ")
	}
}

func TestEncryptAESGCM_EnvelopeFormat(t *testing.T) {
	key := testKey(t)

	tests := []struct {
		keyID          string
		expectedPrefix string
	}{
		{"k1", "enc_v1_kk1_"},
		{"abc", "enc_v1_kabc_"},
		{"20260101", "enc_v1_k20260101_"},
	}

	for _, tt := range tests {
		t.Run("keyID="+tt.keyID, func(t *testing.T) {
			encrypted, err := EncryptAESGCM([]byte("test"), key, nil, tt.keyID)
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			if !strings.HasPrefix(encrypted, tt.expectedPrefix) {
				t.Fatalf("got %q, want prefix %q", encrypted, tt.expectedPrefix)
			}
		})
	}
}

func TestEncryptAESGCM_KeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"16 bytes", 16, true},
		{"31 bytes", 31, true},
		{"32 bytes", 32, false},
		{"33 bytes", 33, true},
		{"0 bytes", 0, true},
		{"64 bytes", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := EncryptAESGCM([]byte("test"), key, nil, "k1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("keyLen=%d: err=%v, wantErr=%v", tt.keyLen, err, tt.wantErr)
			}
		})
	}
}

func TestEncryptAESGCM_KeyIDValidation(t *testing.T) {
	key := testKey(t)

	tests := []struct {
		name    string
		keyID   string
		wantErr bool
	}{
		{"valid single char", "k", false},
		{"valid alphanumeric", "abc123", false},
		{"empty", "", true},
		{"contains underscore", "key_1", true},
		{"multiple underscores", "a_b_c", true},
		{"leading underscore", "_k1", true},
		{"trailing underscore", "k1_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptAESGCM([]byte("test"), key, nil, tt.keyID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("keyID=%q: err=%v, wantErr=%v", tt.keyID, err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DecryptAESGCM
// ---------------------------------------------------------------------------

func TestDecryptAESGCM_WrongKey(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)

	encrypted, err := EncryptAESGCM([]byte("secret"), key1, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptAESGCM(encrypted, key2, nil)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestDecryptAESGCM_AADMismatch(t *testing.T) {
	key := testKey(t)

	tests := []struct {
		name       string
		encryptAAD []byte
		decryptAAD []byte
	}{
		{"different values", []byte("aad1"), []byte("aad2")},
		{"nil vs present", nil, []byte("aad")},
		{"present vs nil", []byte("aad"), nil},
		{"empty vs present", []byte{}, []byte("aad")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptAESGCM([]byte("secret"), key, tt.encryptAAD, "k1")
			if err != nil {
				t.Fatal(err)
			}
			_, err = DecryptAESGCM(encrypted, key, tt.decryptAAD)
			if err == nil {
				t.Fatal("expected error with mismatched AAD")
			}
		})
	}
}

func TestDecryptAESGCM_TamperedPayload(t *testing.T) {
	key := testKey(t)

	encrypted, err := EncryptAESGCM([]byte("secret"), key, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}

	// Flip a character in the payload portion
	parts := strings.SplitN(encrypted, "_", 4)
	payload := []byte(parts[3])
	if payload[len(payload)-2] == 'A' {
		payload[len(payload)-2] = 'B'
	} else {
		payload[len(payload)-2] = 'A'
	}
	tampered := parts[0] + "_" + parts[1] + "_" + parts[2] + "_" + string(payload)

	_, err = DecryptAESGCM(tampered, key, nil)
	if err == nil {
		t.Fatal("expected error decrypting tampered ciphertext")
	}
}

func TestDecryptAESGCM_TruncatedPayload(t *testing.T) {
	key := testKey(t)

	encrypted, err := EncryptAESGCM([]byte("secret"), key, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}

	// Truncate the payload to be shorter than a nonce
	parts := strings.SplitN(encrypted, "_", 4)
	truncated := parts[0] + "_" + parts[1] + "_" + parts[2] + "_" + base64.RawURLEncoding.EncodeToString([]byte("short"))

	_, err = DecryptAESGCM(truncated, key, nil)
	if err == nil {
		t.Fatal("expected error decrypting truncated payload")
	}
}

func TestDecryptAESGCM_InvalidEnvelope(t *testing.T) {
	key := testKey(t)

	tests := []struct {
		name    string
		encoded string
	}{
		{"empty string", ""},
		{"garbage", "not-an-envelope"},
		{"wrong version", "enc_v2_k1_AAAA"},
		{"missing key prefix", "enc_v1_1_AAAA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptAESGCM(tt.encoded, key, nil)
			if err == nil {
				t.Fatalf("expected error for %q", tt.encoded)
			}
		})
	}
}

func TestDecryptAESGCM_KeyValidation(t *testing.T) {
	key := testKey(t)

	encrypted, err := EncryptAESGCM([]byte("test"), key, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}

	shortKey := make([]byte, 16)
	_, err = DecryptAESGCM(encrypted, shortKey, nil)
	if err == nil {
		t.Fatal("expected error with short key on decrypt")
	}

	longKey := make([]byte, 64)
	_, err = DecryptAESGCM(encrypted, longKey, nil)
	if err == nil {
		t.Fatal("expected error with long key on decrypt")
	}
}

func TestDecryptAESGCM_InvalidBase64Payload(t *testing.T) {
	key := testKey(t)

	// Valid envelope structure but payload is not valid base64url
	_, err := DecryptAESGCM("enc_v1_k1_!!!invalid-base64!!!", key, nil)
	if err == nil {
		t.Fatal("expected error with invalid base64 payload")
	}
}

// ---------------------------------------------------------------------------
// ParseAESGCMEnvelope
// ---------------------------------------------------------------------------

func TestParseAESGCMEnvelope_Valid(t *testing.T) {
	key := testKey(t)

	tests := []struct {
		name  string
		keyID string
	}{
		{"single char", "k"},
		{"numeric", "1"},
		{"alphanumeric", "abc123"},
		{"long keyID", "key20260101rotation2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptAESGCM([]byte("test"), key, nil, tt.keyID)
			if err != nil {
				t.Fatal(err)
			}

			env, err := ParseAESGCMEnvelope(encrypted)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if env.Version != AESGCMEncodedVersion {
				t.Fatalf("version: got %q, want %q", env.Version, AESGCMEncodedVersion)
			}
			if env.KeyID != tt.keyID {
				t.Fatalf("keyID: got %q, want %q", env.KeyID, tt.keyID)
			}
			if env.Payload == "" {
				t.Fatal("payload is empty")
			}
		})
	}
}

func TestParseAESGCMEnvelope_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"empty", ""},
		{"no underscores", "foobar"},
		{"one underscore", "enc_v1"},
		{"two underscores", "enc_v1_payload"},
		{"wrong version prefix", "enc_v2_k1_payload"},
		{"wrong version major", "foo_v1_k1_payload"},
		{"missing k prefix", "enc_v1_1_payload"},
		{"empty keyID after k", "enc_v1_k_payload"},
		{"empty payload", "enc_v1_k1_"},
		{"only version and underscores", "enc_v1__"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAESGCMEnvelope(tt.encoded)
			if err == nil {
				t.Fatalf("expected error for %q", tt.encoded)
			}
		})
	}
}

func TestParseAESGCMEnvelope_PayloadPreservesUnderscores(t *testing.T) {
	// Payload section may contain underscores (base64url doesn't use them,
	// but SplitN with limit 4 should capture everything after the third _).
	env, err := ParseAESGCMEnvelope("enc_v1_k1_payload_with_extra_underscores")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if env.Payload != "payload_with_extra_underscores" {
		t.Fatalf("payload: got %q, want %q", env.Payload, "payload_with_extra_underscores")
	}
}

// ---------------------------------------------------------------------------
// DecodeHexKey256
// ---------------------------------------------------------------------------

func TestDecodeHexKey256_Valid(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key, err := DecodeHexKey256(hexKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}
}

func TestDecodeHexKey256_UppercaseHex(t *testing.T) {
	hexKey := "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"
	key, err := DecodeHexKey256(hexKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}
}

func TestDecodeHexKey256_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		hexKey string
	}{
		{"empty", ""},
		{"too short (16 bytes)", "0123456789abcdef0123456789abcdef"},
		{"too long (48 bytes)", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		{"odd length", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"},
		{"invalid hex chars", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
		{"spaces", "01234567 89abcdef 01234567 89abcdef 01234567 89abcdef 01234567 89abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeHexKey256(tt.hexKey)
			if err == nil {
				t.Fatalf("expected error for %q", tt.name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: encrypt → parse → decrypt
// ---------------------------------------------------------------------------

func TestFullFlow_EncryptParseDecrypt(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("aug_sk_test_abc123_secret")
	aad := []byte("apke_test123")

	// Encrypt
	envelope, err := EncryptAESGCM(plaintext, key, aad, "k1")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Parse to inspect
	parsed, err := ParseAESGCMEnvelope(envelope)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Version != "enc_v1" {
		t.Fatalf("version: got %q", parsed.Version)
	}
	if parsed.KeyID != "k1" {
		t.Fatalf("keyID: got %q", parsed.KeyID)
	}

	// Decrypt
	decrypted, err := DecryptAESGCM(envelope, key, aad)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestFullFlow_KeyIDSelectsContext(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)

	// Encrypt with key1, tagged as "k1"
	enc1, err := EncryptAESGCM([]byte("secret1"), key1, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}

	// Encrypt with key2, tagged as "k2"
	enc2, err := EncryptAESGCM([]byte("secret2"), key2, nil, "k2")
	if err != nil {
		t.Fatal(err)
	}

	// Parse envelopes and verify keyIDs
	env1, _ := ParseAESGCMEnvelope(enc1)
	env2, _ := ParseAESGCMEnvelope(enc2)

	if env1.KeyID != "k1" {
		t.Fatalf("env1.KeyID: got %q", env1.KeyID)
	}
	if env2.KeyID != "k2" {
		t.Fatalf("env2.KeyID: got %q", env2.KeyID)
	}

	// Decrypt each with its own key
	dec1, err := DecryptAESGCM(enc1, key1, nil)
	if err != nil {
		t.Fatalf("decrypt enc1: %v", err)
	}
	if string(dec1) != "secret1" {
		t.Fatalf("dec1: got %q", dec1)
	}

	dec2, err := DecryptAESGCM(enc2, key2, nil)
	if err != nil {
		t.Fatalf("decrypt enc2: %v", err)
	}
	if string(dec2) != "secret2" {
		t.Fatalf("dec2: got %q", dec2)
	}

	// Cross-key decrypt must fail
	_, err = DecryptAESGCM(enc1, key2, nil)
	if err == nil {
		t.Fatal("expected error decrypting enc1 with key2")
	}
	_, err = DecryptAESGCM(enc2, key1, nil)
	if err == nil {
		t.Fatal("expected error decrypting enc2 with key1")
	}
}

func TestFullFlow_AADPreventsRowSwap(t *testing.T) {
	key := testKey(t)

	enc1, err := EncryptAESGCM([]byte("secret1"), key, []byte("row1"), "k1")
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := EncryptAESGCM([]byte("secret2"), key, []byte("row2"), "k1")
	if err != nil {
		t.Fatal(err)
	}

	// Each decrypts with its own AAD
	if _, err := DecryptAESGCM(enc1, key, []byte("row1")); err != nil {
		t.Fatalf("decrypt enc1 with row1 AAD: %v", err)
	}
	if _, err := DecryptAESGCM(enc2, key, []byte("row2")); err != nil {
		t.Fatalf("decrypt enc2 with row2 AAD: %v", err)
	}

	// Swapped AAD must fail
	if _, err := DecryptAESGCM(enc1, key, []byte("row2")); err == nil {
		t.Fatal("expected error decrypting enc1 with row2 AAD")
	}
	if _, err := DecryptAESGCM(enc2, key, []byte("row1")); err == nil {
		t.Fatal("expected error decrypting enc2 with row1 AAD")
	}
}

func TestFullFlow_DecodeHexKeyAndEncrypt(t *testing.T) {
	hexKey := fmt.Sprintf("%064x", 42)
	key, err := DecodeHexKey256(hexKey)
	if err != nil {
		t.Fatal(err)
	}

	encrypted, err := EncryptAESGCM([]byte("test"), key, nil, "k1")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptAESGCM(encrypted, key, nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != "test" {
		t.Fatalf("got %q, want %q", decrypted, "test")
	}
}
