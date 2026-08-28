package crypto

import (
	"strings"
	"testing"
)

// TestDecryptAESGCM_RotationIsIndistinguishableFromTampering pins the fact that decrypting a k2 envelope with the k1 key fails exactly like a corrupted ciphertext. Call sites turn that error into "reconnect your integration", so the keyID in the envelope — not the decrypt error — is the only thing that can tell a rotation apart from tampering.
func TestDecryptAESGCM_RotationIsIndistinguishableFromTampering(t *testing.T) {
	t.Parallel()
	k1 := testKey(t)
	k2 := testKey(t)

	rotated, err := EncryptAESGCM([]byte("secret"), k2, nil, "k2")
	if err != nil {
		t.Fatal(err)
	}

	_, rotationErr := DecryptAESGCM(rotated, k1, nil)
	if rotationErr == nil {
		t.Fatal("expected error decrypting a k2 envelope with the k1 key")
	}

	sameKey, err := EncryptAESGCM([]byte("secret"), k1, nil, "k1")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(sameKey, "_", 4)
	payload := []byte(parts[3])
	if payload[len(payload)-2] == 'A' {
		payload[len(payload)-2] = 'B'
	} else {
		payload[len(payload)-2] = 'A'
	}
	tampered := strings.Join([]string{parts[0], parts[1], parts[2], string(payload)}, "_")

	_, tamperErr := DecryptAESGCM(tampered, k1, nil)
	if tamperErr == nil {
		t.Fatal("expected error decrypting tampered ciphertext")
	}

	if rotationErr.Error() != tamperErr.Error() {
		t.Fatalf("rotation error %q now differs from tampering error %q; callers may depend on them being the same class", rotationErr, tamperErr)
	}

	// The keyID is the distinguishing signal, and it survives the failed decrypt.
	env, err := ParseAESGCMEnvelope(rotated)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if env.KeyID != "k2" {
		t.Fatalf("keyID: got %q, want k2", env.KeyID)
	}
}

// TestDecryptAESGCM_KeyringSelectionByKeyID exercises the rotation path a caller is supposed to take: parse first, pick the key the envelope names, then decrypt.
func TestDecryptAESGCM_KeyringSelectionByKeyID(t *testing.T) {
	t.Parallel()
	keyring := map[string][]byte{
		"k1": testKey(t),
		"k2": testKey(t),
	}

	old, err := EncryptAESGCM([]byte("old secret"), keyring["k1"], nil, "k1")
	if err != nil {
		t.Fatal(err)
	}
	current, err := EncryptAESGCM([]byte("new secret"), keyring["k2"], nil, "k2")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		envelope string
		want     string
	}{
		{"pre-rotation envelope", old, "old secret"},
		{"post-rotation envelope", current, "new secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env, err := ParseAESGCMEnvelope(tt.envelope)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			key, ok := keyring[env.KeyID]
			if !ok {
				t.Fatalf("keyID %q not in keyring", env.KeyID)
			}
			got, err := DecryptAESGCM(tt.envelope, key, nil)
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAESGCMEnvelope_UnknownKeyIDIsReportableBeforeDecrypt(t *testing.T) {
	t.Parallel()
	key := testKey(t)

	encrypted, err := EncryptAESGCM([]byte("secret"), key, nil, "k9")
	if err != nil {
		t.Fatal(err)
	}

	env, err := ParseAESGCMEnvelope(encrypted)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// A retired or not-yet-deployed keyID must be detectable without attempting a decrypt, so an operator sees "unknown key" rather than every customer seeing "reconnect your integration".
	if _, ok := map[string][]byte{"k1": key}[env.KeyID]; ok {
		t.Fatalf("keyID %q unexpectedly resolved", env.KeyID)
	}
}

// TestDecryptAESGCM_LegacyPlaintextValue covers a column written before encryption was introduced: whatever its shape, it must fail rather than decode, and must never come back as plaintext.
func TestDecryptAESGCM_LegacyPlaintextValue(t *testing.T) {
	t.Parallel()
	key := testKey(t)

	tests := []struct {
		name  string
		value string
	}{
		{"no underscores", "shpat0123456789abcdef"},
		{"two segments", "Bearer_token"},
		{"three segments", "sk-live-token_account_1"},
		{"four segments", "sk-live-token_account_1_extra"},
		{"envelope-like prefix", "enc_v0_k1_notreallyencrypted"},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecryptAESGCM(tt.value, key, nil)
			if err == nil {
				t.Fatalf("expected error for legacy value %q, got %q", tt.value, got)
			}
			if got != nil {
				t.Fatalf("expected nil plaintext for legacy value %q, got %q", tt.value, got)
			}
		})
	}
}
