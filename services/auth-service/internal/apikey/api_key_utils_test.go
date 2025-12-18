package apikey

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
)

func TestDefaultAPIKeyConfig(t *testing.T) {
	pepper := []byte("test-pepper")
	apiKeyConfig := DefaultAPIKeyConfig(pepper)

	if apiKeyConfig.SecretKeyStrength != KeyStrengthHigh {
		t.Errorf("Expected SecretKeyStrength to be %s, got %s", KeyStrengthHigh, apiKeyConfig.SecretKeyStrength)
	}

	if apiKeyConfig.IDKeyStrength != KeyStrengthLow {
		t.Errorf("Expected IDKeyStrength to be %s, got %s", KeyStrengthLow, apiKeyConfig.IDKeyStrength)
	}

	if string(apiKeyConfig.Pepper) != string(pepper) {
		t.Errorf("Expected Pepper to be %v, got %v", pepper, apiKeyConfig.Pepper)
	}
}

func TestNewAPIKeyUtils(t *testing.T) {
	apiKeyConfig := APIKeyConfig{
		SecretKeyStrength: KeyStrengthMedium,
		IDKeyStrength:     KeyStrengthLow,
		Pepper:            []byte("test-pepper"),
	}

	utils := NewAPIKeyUtils(apiKeyConfig)
	if utils == nil {
		t.Fatal("Expected NewAPIKeyUtils to return non-nil")
	}

	// Test that it implements the interface
	var _ domain.APIKeyUtils = utils
}

func TestAPIKeyUtils_Generate(t *testing.T) {
	apiKeyConfig := DefaultAPIKeyConfig([]byte("test-pepper"))
	utils := NewAPIKeyUtils(apiKeyConfig)

	tests := []struct {
		name     string
		appMode  constants.AccountMode
		validate func(*domain.ParsedAPIKey) bool
	}{
		{
			name:    "generate production key",
			appMode: constants.AccountModeProduction,
			validate: func(key *domain.ParsedAPIKey) bool {
				return key.AccountMode == constants.AccountModeProduction &&
					len(key.ID) == 22 && // KeyStrengthLow = 22
					len(key.Secret) == 44 && // KeyStrengthHigh = 44
					len(key.Checksum) == 6
			},
		},
		{
			name:    "generate sandbox key",
			appMode: constants.AccountModeSandbox,
			validate: func(key *domain.ParsedAPIKey) bool {
				return key.AccountMode == constants.AccountModeSandbox &&
					len(key.ID) == 22 &&
					len(key.Secret) == 44 &&
					len(key.Checksum) == 6
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := utils.Gen(context.Background(), tt.appMode)

			if err != nil {
				t.Errorf("Gen() unexpected error = %v", err)
				return
			}

			if key == nil {
				t.Fatal("Gen() returned nil key")
			}

			if !tt.validate(key) {
				t.Errorf("Gen() returned invalid key: %+v", key)
			}

			// Test that the checksum is correct
			expectedChecksum := genKeyChecksum(key.ID, key.Secret)
			if key.Checksum != expectedChecksum {
				t.Errorf("Gen() checksum mismatch: expected %s, got %s", expectedChecksum, key.Checksum)
			}
		})
	}
}

func TestAPIKeyUtils_Generate_Uniqueness(t *testing.T) {
	apiKeyConfig := DefaultAPIKeyConfig([]byte("test-pepper"))
	utils := NewAPIKeyUtils(apiKeyConfig)

	// Generate multiple keys and ensure they're unique
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key, err := utils.Gen(context.Background(), constants.AccountModeProduction)
		if err != nil {
			t.Fatalf("Gen() unexpected error = %v", err)
		}

		keyString := key.String()
		if keys[keyString] {
			t.Errorf("Gen() produced duplicate key: %s", keyString)
		}
		keys[keyString] = true
	}
}

func TestAPIKeyUtils_Parse_ValidKeys(t *testing.T) {
	apiKeyConfig := DefaultAPIKeyConfig([]byte("test-pepper"))
	utils := NewAPIKeyUtils(apiKeyConfig)

	// Generate a valid key first
	originalKey, err := utils.Gen(context.Background(), constants.AccountModeProduction)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	keyString := originalKey.String()

	// Test parsing the generated key
	parsedKey, err := utils.Parse(context.Background(), keyString)
	if err != nil {
		t.Errorf("Parse() failed to parse valid key: %v", err)
		return
	}

	if parsedKey.AccountMode != originalKey.AccountMode {
		t.Errorf("Parse() AccountMode mismatch: expected %s, got %s", originalKey.AccountMode, parsedKey.AccountMode)
	}

	if parsedKey.ID != originalKey.ID {
		t.Errorf("Parse() ID mismatch: expected %s, got %s", originalKey.ID, parsedKey.ID)
	}

	if parsedKey.Secret != originalKey.Secret {
		t.Errorf("Parse() Secret mismatch: expected %s, got %s", originalKey.Secret, parsedKey.Secret)
	}

	if parsedKey.Checksum != originalKey.Checksum {
		t.Errorf("Parse() Checksum mismatch: expected %s, got %s", originalKey.Checksum, parsedKey.Checksum)
	}
}

func TestAPIKeyUtils_Parse_InvalidKeys(t *testing.T) {
	apiKeyConfig := DefaultAPIKeyConfig([]byte("test-pepper"))
	utils := NewAPIKeyUtils(apiKeyConfig)

	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{
			name:        "empty key",
			key:         "",
			expectError: true,
		},
		{
			name:        "wrong prefix",
			key:         "invalid_prefix_prod_123_456_789",
			expectError: true,
		},
		{
			name:        "too few parts",
			key:         "aug_sk_prod_123",
			expectError: true,
		},
		{
			name:        "too many parts",
			key:         "aug_sk_prod_123_456_789_extra",
			expectError: true,
		},
		{
			name:        "invalid app mode",
			key:         "aug_sk_invalid_123_456_789abc",
			expectError: true,
		},
		{
			name:        "secret too short",
			key:         "aug_sk_prod_123_45_789abc",
			expectError: true,
		},
		{
			name:        "invalid checksum",
			key:         "aug_sk_prod_123_456789_000000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := utils.Parse(context.Background(), tt.key)

			if tt.expectError && err == nil {
				t.Errorf("Parse() expected error but got none for key: %s", tt.key)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Parse() unexpected error: %v for key: %s", err, tt.key)
			}

			if tt.expectError && err != nil {
				if err.Type != contracts.ErrorTypeInvalidRequest {
					t.Errorf("Parse() expected invalid request error, got: %s", err.Type)
				}
			}
		})
	}
}

func TestAPIKeyUtils_GenerateSecretHMAC(t *testing.T) {
	pepper := []byte("test-pepper-123")
	apiKeyConfig := APIKeyConfig{Pepper: pepper}
	utils := NewAPIKeyUtils(apiKeyConfig)

	secret := "test-secret"
	hmac, err := utils.GenSecretHMAC(context.Background(), secret)

	if err != nil {
		t.Errorf("GenerateSecretHMAC() unexpected error = %v", err)
		return
	}

	if hmac == nil {
		t.Fatal("GenerateSecretHMAC() returned nil HMAC")
	}

	if len(hmac) != 32 { // SHA256 produces 32 bytes
		t.Errorf("GenerateSecretHMAC() returned HMAC of wrong length: expected 32, got %d", len(hmac))
	}

	// Test that the same secret produces the same HMAC
	hmac2, err := utils.GenSecretHMAC(context.Background(), secret)
	if err != nil {
		t.Errorf("GenerateSecretHMAC() unexpected error on second call: %v", err)
		return
	}

	if string(hmac) != string(hmac2) {
		t.Error("GenerateSecretHMAC() returned different HMACs for the same secret")
	}
}

func TestAPIKeyUtils_VerifySecretHMAC(t *testing.T) {
	pepper := []byte("test-pepper-123")
	apiKeyConfig := APIKeyConfig{Pepper: pepper}
	utils := NewAPIKeyUtils(apiKeyConfig)

	secret := "test-secret"
	expectedHMAC, err := utils.GenSecretHMAC(context.Background(), secret)
	if err != nil {
		t.Fatalf("Failed to generate HMAC for test: %v", err)
	}

	// Test valid verification
	valid, err := utils.VerifySecretHMAC(context.Background(), secret, expectedHMAC)
	if err != nil {
		t.Errorf("VerifySecretHMAC() unexpected error: %v", err)
		return
	}

	if !valid {
		t.Error("VerifySecretHMAC() failed to verify valid secret")
	}

	// Test invalid verification
	invalid, err := utils.VerifySecretHMAC(context.Background(), "wrong-secret", expectedHMAC)
	if err != nil {
		t.Errorf("VerifySecretHMAC() unexpected error: %v", err)
		return
	}

	if invalid {
		t.Error("VerifySecretHMAC() incorrectly verified invalid secret")
	}

	// Test with different HMAC
	differentHMAC := []byte("different-hmac-123456789012345678901234")
	valid, err = utils.VerifySecretHMAC(context.Background(), secret, differentHMAC)
	if err != nil {
		t.Errorf("VerifySecretHMAC() unexpected error: %v", err)
		return
	}

	if valid {
		t.Error("VerifySecretHMAC() incorrectly verified secret with different HMAC")
	}
}

func TestAPIKeyUtils_LengthForKeyStrength(t *testing.T) {
	// Test through the public interface by generating keys with different strengths
	// and checking their lengths

	// Test low strength
	lowConfig := APIKeyConfig{
		SecretKeyStrength: KeyStrengthLow,
		IDKeyStrength:     KeyStrengthLow,
		Pepper:            []byte("test-pepper"),
	}
	lowUtils := NewAPIKeyUtils(lowConfig)
	lowKey, err := lowUtils.Gen(context.Background(), constants.AccountModeProduction)
	if err != nil {
		t.Fatalf("Failed to generate low strength key: %v", err)
	}

	if len(lowKey.Secret) != 22 {
		t.Errorf("Low strength secret should be 22 chars, got %d", len(lowKey.Secret))
	}

	if len(lowKey.ID) != 22 {
		t.Errorf("Low strength ID should be 22 chars, got %d", len(lowKey.ID))
	}

	// Test medium strength
	mediumConfig := APIKeyConfig{
		SecretKeyStrength: KeyStrengthMedium,
		IDKeyStrength:     KeyStrengthMedium,
		Pepper:            []byte("test-pepper"),
	}
	mediumUtils := NewAPIKeyUtils(mediumConfig)
	mediumKey, err := mediumUtils.Gen(context.Background(), constants.AccountModeProduction)
	if err != nil {
		t.Fatalf("Failed to generate medium strength key: %v", err)
	}

	if len(mediumKey.Secret) != 33 {
		t.Errorf("Medium strength secret should be 33 chars, got %d", len(mediumKey.Secret))
	}

	if len(mediumKey.ID) != 33 {
		t.Errorf("Medium strength ID should be 33 chars, got %d", len(mediumKey.ID))
	}

	// Test high strength
	highConfig := APIKeyConfig{
		SecretKeyStrength: KeyStrengthHigh,
		IDKeyStrength:     KeyStrengthHigh,
		Pepper:            []byte("test-pepper"),
	}
	highUtils := NewAPIKeyUtils(highConfig)
	highKey, err := highUtils.Gen(context.Background(), constants.AccountModeProduction)
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

func TestParsedAPIKey_String(t *testing.T) {
	key := &domain.ParsedAPIKey{
		AccountMode: constants.AccountModeProduction,
		ID:          "testid123",
		Secret:      "testsecret456",
		Checksum:    "abc123",
	}

	expected := "aug_sk_prod_testid123_testsecret456abc123"
	result := key.String()

	if result != expected {
		t.Errorf("ParsedAPIKey.String() = %s, want %s", result, expected)
	}
}

func TestAPIKeyUtils_Integration(t *testing.T) {
	// Test the full flow: generate -> string -> parse -> verify
	apiKeyConfig := DefaultAPIKeyConfig([]byte("test-pepper"))
	utils := NewAPIKeyUtils(apiKeyConfig)

	// Generate a key
	originalKey, err := utils.Gen(context.Background(), constants.AccountModeProduction)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Convert to string
	keyString := originalKey.String()

	// Parse back
	parsedKey, err := utils.Parse(context.Background(), keyString)
	if err != nil {
		t.Fatalf("Failed to parse generated key: %v", err)
	}

	// Verify all fields match
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

	// Test HMAC generation and verification
	hmac, err := utils.GenSecretHMAC(context.Background(), parsedKey.Secret)
	if err != nil {
		t.Fatalf("Failed to generate HMAC: %v", err)
	}

	valid, err := utils.VerifySecretHMAC(context.Background(), parsedKey.Secret, hmac)
	if err != nil {
		t.Fatalf("Failed to verify HMAC: %v", err)
	}

	if !valid {
		t.Error("HMAC verification failed")
	}
}
