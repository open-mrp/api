package apikey

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
)

func TestGenParsedAPIKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mode     constants.AccountMode
		validate func(*ParsedAPIKey) bool
	}{
		{
			name: "generate production key",
			mode: constants.AccountModeProduction,
			validate: func(key *ParsedAPIKey) bool {
				return key.AccountMode == constants.AccountModeProduction &&
					len(key.ID) == 22 &&
					len(key.Secret) == 44 &&
					len(key.Checksum) == 6
			},
		},
		{
			name: "generate sandbox key",
			mode: constants.AccountModeSandbox,
			validate: func(key *ParsedAPIKey) bool {
				return key.AccountMode == constants.AccountModeSandbox &&
					len(key.ID) == 22 &&
					len(key.Secret) == 44 &&
					len(key.Checksum) == 6
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenParsedAPIKey(tt.mode, nil)

			if err != nil {
				t.Errorf("GenParsedAPIKey() unexpected error = %v", err)
				return
			}

			if key == nil {
				t.Fatal("GenParsedAPIKey() returned nil key")
			}

			if !tt.validate(key) {
				t.Errorf("GenParsedAPIKey() returned invalid key: %+v", key)
			}

			expectedChecksum := genAPIKeyChecksum(key.ID, key.Secret)
			if key.Checksum != expectedChecksum {
				t.Errorf("GenParsedAPIKey() checksum mismatch: expected %s, got %s", expectedChecksum, key.Checksum)
			}
		})
	}
}

func TestGenParsedAPIKey_Uniqueness(t *testing.T) {
	t.Parallel()
	keys := make(map[string]bool)
	for range 10 {
		key, err := GenParsedAPIKey(constants.AccountModeProduction, nil)
		if err != nil {
			t.Fatalf("GenParsedAPIKey() unexpected error = %v", err)
		}

		keyString := key.String()
		if keys[keyString] {
			t.Errorf("GenParsedAPIKey() produced duplicate key: %s", keyString)
		}
		keys[keyString] = true
	}
}

func TestGenParsedAPIKey_KeyStrengths(t *testing.T) {
	t.Parallel()
	lowConfig := &APIKeyGenConfig{
		SecretKeyStrength: KeyStrengthLow,
		IDKeyStrength:     KeyStrengthLow,
	}
	lowKey, err := GenParsedAPIKey(constants.AccountModeProduction, lowConfig)
	if err != nil {
		t.Fatalf("Failed to generate low strength key: %v", err)
	}
	if len(lowKey.Secret) != 22 {
		t.Errorf("Low strength secret should be 22 chars, got %d", len(lowKey.Secret))
	}
	if len(lowKey.ID) != 22 {
		t.Errorf("Low strength ID should be 22 chars, got %d", len(lowKey.ID))
	}

	mediumConfig := &APIKeyGenConfig{
		SecretKeyStrength: KeyStrengthMedium,
		IDKeyStrength:     KeyStrengthMedium,
	}
	mediumKey, err := GenParsedAPIKey(constants.AccountModeProduction, mediumConfig)
	if err != nil {
		t.Fatalf("Failed to generate medium strength key: %v", err)
	}
	if len(mediumKey.Secret) != 33 {
		t.Errorf("Medium strength secret should be 33 chars, got %d", len(mediumKey.Secret))
	}
	if len(mediumKey.ID) != 33 {
		t.Errorf("Medium strength ID should be 33 chars, got %d", len(mediumKey.ID))
	}

	highConfig := &APIKeyGenConfig{
		SecretKeyStrength: KeyStrengthHigh,
		IDKeyStrength:     KeyStrengthHigh,
	}
	highKey, err := GenParsedAPIKey(constants.AccountModeProduction, highConfig)
	if err != nil {
		t.Fatalf("Failed to generate high strength key: %v", err)
	}
	if len(highKey.Secret) != 44 {
		t.Errorf("High strength secret should be 44 chars, got %d", len(highKey.Secret))
	}
	if len(highKey.ID) != 44 {
		t.Errorf("High strength ID should be 44 chars, got %d", len(highKey.ID))
	}
}

func TestParseAPIKey_ValidKeys(t *testing.T) {
	t.Parallel()
	originalKey, err := GenParsedAPIKey(constants.AccountModeProduction, nil)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	keyString := originalKey.String()
	parsedKey, err := ParseAPIKey(keyString)
	if err != nil {
		t.Errorf("ParseAPIKey() failed to parse valid key: %v", err)
		return
	}

	if parsedKey.AccountMode != originalKey.AccountMode {
		t.Errorf("ParseAPIKey() AccountMode mismatch: expected %s, got %s", originalKey.AccountMode, parsedKey.AccountMode)
	}
	if parsedKey.ID != originalKey.ID {
		t.Errorf("ParseAPIKey() ID mismatch: expected %s, got %s", originalKey.ID, parsedKey.ID)
	}
	if parsedKey.Secret != originalKey.Secret {
		t.Errorf("ParseAPIKey() Secret mismatch: expected %s, got %s", originalKey.Secret, parsedKey.Secret)
	}
	if parsedKey.Checksum != originalKey.Checksum {
		t.Errorf("ParseAPIKey() Checksum mismatch: expected %s, got %s", originalKey.Checksum, parsedKey.Checksum)
	}
}

func TestParseAPIKey_InvalidKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
	}{
		{"empty key", ""},
		{"wrong prefix", "invalid_prefix_prod_123_456_789"},
		{"too few parts", "mrp_sk_prod_123"},
		{"too many parts", "mrp_sk_prod_123_456_789_extra"},
		{"invalid app mode", "mrp_sk_invalid_123_456_789abc"},
		{"secret too short", "mrp_sk_prod_123_45_789abc"},
		{"invalid checksum", "mrp_sk_prod_123_456789_000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAPIKey(tt.key)
			if err == nil {
				t.Errorf("ParseAPIKey() expected error but got none for key: %s", tt.key)
			}
			if err != nil && err.Type != apierror.ErrorTypeInvalidRequest {
				t.Errorf("ParseAPIKey() expected invalid request error, got: %s", err.Type)
			}
		})
	}
}

func TestParsedAPIKey_GenSecretHMAC(t *testing.T) {
	t.Parallel()
	pepper := []byte("test-pepper-123")

	key, err := GenParsedAPIKey(constants.AccountModeProduction, nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	hmac := key.GenSecretHMAC(pepper)
	if hmac == nil {
		t.Fatal("GenSecretHMAC() returned nil HMAC")
	}
	if len(hmac) != 32 {
		t.Errorf("GenSecretHMAC() returned HMAC of wrong length: expected 32, got %d", len(hmac))
	}

	// Same secret produces same HMAC
	hmac2 := key.GenSecretHMAC(pepper)
	if string(hmac) != string(hmac2) {
		t.Error("GenSecretHMAC() returned different HMACs for the same secret")
	}
}

func TestParsedAPIKey_VerifySecretHMAC(t *testing.T) {
	t.Parallel()
	pepper := []byte("test-pepper-123")

	key, err := GenParsedAPIKey(constants.AccountModeProduction, nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	expectedHMAC := key.GenSecretHMAC(pepper)

	if !key.VerifySecretHMAC(pepper, expectedHMAC) {
		t.Error("VerifySecretHMAC() failed to verify valid secret")
	}

	// Different HMAC should fail
	differentHMAC := []byte("different-hmac-123456789012345678901234")
	if key.VerifySecretHMAC(pepper, differentHMAC) {
		t.Error("VerifySecretHMAC() incorrectly verified secret with different HMAC")
	}

	// Different key should fail
	otherKey, err := GenParsedAPIKey(constants.AccountModeProduction, nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	if otherKey.VerifySecretHMAC(pepper, expectedHMAC) {
		t.Error("VerifySecretHMAC() incorrectly verified different key's secret")
	}
}

func TestSanitizeAPIKey(t *testing.T) {
	t.Parallel()
	key := "mrp_sk_prod_testid123_testsecret456abc123"
	sanitized := SanitizeAPIKey(key)

	if !strings.HasPrefix(sanitized, "mrp_sk_prod_te") {
		t.Errorf("SanitizeAPIKey() should preserve prefix, got: %s", sanitized)
	}
	if !strings.Contains(sanitized, "****") {
		t.Errorf("SanitizeAPIKey() should contain asterisks, got: %s", sanitized)
	}
}

func TestToProto_NilReturnsNil(t *testing.T) {
	t.Parallel()
	var apiKey *APIKey
	proto := apiKey.ToProto()
	if proto != nil {
		t.Error("ToProto() on nil should return nil")
	}
}

func TestToProto_OptionalTimestamps(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	lastUsed := now.Add(-time.Hour)
	expires := now.Add(24 * time.Hour)
	revoked := now.Add(-30 * time.Minute)

	apiKey := &APIKey{
		TypeID:     "apikey_test123",
		Name:       "Key",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: &lastUsed,
		ExpiresAt:  &expires,
		RevokedAt:  &revoked,
	}

	proto := apiKey.ToProto()
	if proto.LastUsedAt == nil {
		t.Error("ToProto() should set LastUsedAt when present")
	}
	if proto.ExpiresAt == nil {
		t.Error("ToProto() should set ExpiresAt when present")
	}
	if proto.RevokedAt == nil {
		t.Error("ToProto() should set RevokedAt when present")
	}
}

func TestAPIKey_IsRevoked(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name      string
		revokedAt *time.Time
		want      bool
	}{
		{"nil revoked_at is not revoked", nil, false},
		{"past revoked_at is revoked", &past, true},
		{"future revoked_at is scheduled, not yet revoked", &future, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key := &APIKey{RevokedAt: tc.revokedAt}
			if got := key.IsRevoked(); got != tc.want {
				t.Errorf("IsRevoked() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAPIKey_IsExpired(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"nil expires_at is not expired", nil, false},
		{"past expires_at is expired", &past, true},
		{"future expires_at is not expired", &future, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key := &APIKey{ExpiresAt: tc.expiresAt}
			if got := key.IsExpired(); got != tc.want {
				t.Errorf("IsExpired() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParsedAPIKey_RedactedValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mode     constants.AccountMode
		wantMode string
	}{
		{"production", constants.AccountModeProduction, "prod"},
		{"sandbox", constants.AccountModeSandbox, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenParsedAPIKey(tt.mode, nil)
			if err != nil {
				t.Fatalf("GenParsedAPIKey() unexpected error = %v", err)
			}

			redacted := key.RedactedValue()

			fullKey := key.String()
			expectedSuffix := fullKey[len(fullKey)-4:]
			expectedPrefix := "mrp_sk_" + tt.wantMode + "_****"

			if !strings.HasPrefix(redacted, expectedPrefix) {
				t.Errorf("RedactedValue() = %s, want prefix %s", redacted, expectedPrefix)
			}
			if !strings.HasSuffix(redacted, expectedSuffix) {
				t.Errorf("RedactedValue() = %s, want suffix %s", redacted, expectedSuffix)
			}
			if !strings.Contains(redacted, "****") {
				t.Errorf("RedactedValue() = %s, want to contain ****", redacted)
			}
		})
	}
}

func TestParsedAPIKey_String(t *testing.T) {
	t.Parallel()
	key := &ParsedAPIKey{
		AccountMode: constants.AccountModeProduction,
		ID:          "testid123",
		Secret:      "testsecret456",
		Checksum:    "abc123",
	}

	expected := "mrp_sk_prod_testid123_testsecret456abc123"
	result := key.String()

	if result != expected {
		t.Errorf("ParsedAPIKey.String() = %s, want %s", result, expected)
	}
}

func TestAPIKey_Integration(t *testing.T) {
	t.Parallel()
	pepper := []byte("test-pepper")

	// Generate a key
	originalKey, err := GenParsedAPIKey(constants.AccountModeProduction, nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Convert to string
	keyString := originalKey.String()

	// Parse back
	parsedKey, err := ParseAPIKey(keyString)
	if err != nil {
		t.Fatalf("Failed to parse generated key: %v", err)
	}

	if parsedKey.AccountMode != originalKey.AccountMode {
		t.Errorf("AccountMode mismatch: expected %s, got %s", originalKey.AccountMode, parsedKey.AccountMode)
	}
	if parsedKey.ID != originalKey.ID {
		t.Errorf("ID mismatch: expected %s, got %s", originalKey.ID, parsedKey.ID)
	}
	if parsedKey.Secret != originalKey.Secret {
		t.Errorf("Secret mismatch: expected %s, got %s", originalKey.Secret, parsedKey.Secret)
	}
	if parsedKey.Checksum != originalKey.Checksum {
		t.Errorf("Checksum mismatch: expected %s, got %s", originalKey.Checksum, parsedKey.Checksum)
	}

	// HMAC round-trip
	hmac := parsedKey.GenSecretHMAC(pepper)
	if !parsedKey.VerifySecretHMAC(pepper, hmac) {
		t.Error("HMAC verification failed")
	}

	// Cross-verify: HMAC from crypto directly should match
	directHMAC := crypto.HMACSHA256(pepper, []byte(parsedKey.Secret))
	if string(hmac) != string(directHMAC) {
		t.Error("GenSecretHMAC should produce same result as crypto.HMACSHA256")
	}
}
