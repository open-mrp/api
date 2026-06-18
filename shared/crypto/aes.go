package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// AESGCMEncodedVersion is the current envelope version.
	//
	// Format:
	//   enc_v1_k<keyID>_<b64url(nonce || ciphertext || tag)>
	AESGCMEncodedVersion = "enc_v1"

	// AESGCMKeyBytes is the required key length for AES-256-GCM.
	AESGCMKeyBytes = 32
)

// AESGCMEnvelope represents a parsed AES-GCM envelope.
type AESGCMEnvelope struct {
	// Envelope version string (e.g. "enc_v1")
	Version string

	// Identifier used to select the correct encryption key for decryption
	KeyID string

	// Base64 URL-safe (no padding) encoding of (nonce || ciphertext || tag)
	Payload string
}

// EncryptAESGCM encrypts plaintext using AES-256-GCM and returns a fully encoded envelope string suitable for storage in a database.
//
// Params:
//   - plaintext: data to encrypt
//   - key: 32-byte AES-256 key
//   - aad: associated data (optional); must be present to decrypt if provided
//   - keyID: identifier for key rotation; included in the envelope
func EncryptAESGCM(plaintext, key, aad []byte, keyID string) (string, error) {
	if len(key) != AESGCMKeyBytes {
		return "", fmt.Errorf("encryption key must be %d bytes, got %d", AESGCMKeyBytes, len(key))
	}
	if keyID == "" {
		return "", errors.New("keyID must be non-empty")
	}
	if strings.Contains(keyID, "_") {
		return "", errors.New("keyID must not contain '_'")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	blob := gcm.Seal(nonce, nonce, plaintext, aad)
	payload := base64.RawURLEncoding.EncodeToString(blob)

	return fmt.Sprintf("%s_k%s_%s", AESGCMEncodedVersion, keyID, payload), nil
}

// DecryptAESGCM decrypts a previously generated envelope string.
//
// Params:
//   - encoded: string returned by EncryptAESGCM.
//   - key: the AES-256 key corresponding to envelope.KeyID.
//   - aad: must match the aad used at encryption time.
//
// Returns:
//   - plaintext on success.
//   - error if parsing fails, key is wrong, aad mismatches, or ciphertext is invalid.
func DecryptAESGCM(encoded string, key, aad []byte) ([]byte, error) {
	env, err := ParseAESGCMEnvelope(encoded)
	if err != nil {
		return nil, err
	}

	if len(key) != AESGCMKeyBytes {
		return nil, fmt.Errorf("encryption key must be %d bytes, got %d", AESGCMKeyBytes, len(key))
	}

	data, err := base64.RawURLEncoding.DecodeString(env.Payload)
	if err != nil {
		return nil, errors.New("invalid envelope payload encoding")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("invalid envelope payload")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, errors.New("invalid ciphertext or authentication failed")
	}

	return plaintext, nil
}

// ParseAESGCMEnvelope parses an encoded AES-GCM envelope string into a struct.
//
// Expected format:
//
//	enc_v1_k<keyID>_<payload>
//
// Returns:
//   - AESGCMEnvelope struct containing Version, KeyID, and Payload.
//   - error if the format or version is invalid.
//
// Callers typically:
//  1. Parse envelope
//  2. Use KeyID to select key
//  3. Call DecryptAESGCM with that key
func ParseAESGCMEnvelope(encoded string) (AESGCMEnvelope, error) {
	var env AESGCMEnvelope

	parts := strings.SplitN(encoded, "_", 4)
	if len(parts) != 4 {
		return env, errors.New("invalid envelope format")
	}

	version := parts[0] + "_" + parts[1]
	if version != AESGCMEncodedVersion {
		return env, fmt.Errorf("unsupported envelope version: %s", version)
	}

	kpart := parts[2]
	if len(kpart) < 2 || kpart[0] != 'k' {
		return env, errors.New("invalid envelope format: missing keyID")
	}

	keyID := kpart[1:]
	if keyID == "" {
		return env, errors.New("invalid envelope format: empty keyID")
	}

	payload := parts[3]
	if payload == "" {
		return env, errors.New("invalid envelope format: empty payload")
	}

	env = AESGCMEnvelope{
		Version: version,
		KeyID:   keyID,
		Payload: payload,
	}

	return env, nil
}

// DecodeHexKey256 decodes a hex-encoded AES-256 key.
//
// The hex string must decode to exactly 32 bytes (64 hex characters).
func DecodeHexKey256(hexKey string) ([]byte, error) {
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	if len(b) != AESGCMKeyBytes {
		return nil, fmt.Errorf("key must decode to %d bytes, got %d", AESGCMKeyBytes, len(b))
	}
	return b, nil
}
