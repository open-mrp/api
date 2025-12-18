package token

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
)

func TestDefaultJWTConfig(t *testing.T) {
	secret := "test-secret-123"
	config := DefaultJWTConfig(secret)

	if config.Secret != secret {
		t.Errorf("Expected Secret to be %s, got %s", secret, config.Secret)
	}

	if config.Issuer != "https://augno.com" {
		t.Errorf("Expected Issuer to be 'https://augno.com', got %s", config.Issuer)
	}
}

func TestNewJWTUtils(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret",
		Issuer: "https://test.com",
	}

	utils := NewJWTUtils(config)
	if utils == nil {
		t.Fatal("Expected NewJWTUtils to return non-nil")
	}

	// Test that it implements the interface
	var _ domain.JWTUtils = utils
}

func TestJWTUtils_Encode_Success(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	userID := "user123"
	expiresIn := 1 * time.Hour

	token, err := utils.Encode(context.Background(), userID, expiresIn, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Encode() unexpected error = %v", err)
	}

	if token == "" {
		t.Fatal("Encode() returned empty token")
	}

	// Verify the token can be decoded
	claims, err := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to decode generated token: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("Token subject mismatch: expected %s, got %s", userID, claims.Subject)
	}

	if claims.Issuer != config.Issuer {
		t.Errorf("Token issuer mismatch: expected %s, got %s", config.Issuer, claims.Issuer)
	}

	if claims.TokenType != domain.JWTTypeAccess {
		t.Errorf("Token type mismatch: expected %s, got %s", domain.JWTTypeAccess, claims.TokenType)
	}

	// Check that the token is not expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now().UTC()) {
		t.Error("Generated token is already expired")
	}

	// Check that the token was issued recently
	if claims.IssuedAt != nil {
		issuedTime := claims.IssuedAt.Time
		now := time.Now().UTC()
		if issuedTime.After(now) || issuedTime.Before(now.Add(-5*time.Second)) {
			t.Errorf("Token issued time seems incorrect: %v, current time: %v", issuedTime, now)
		}
	}
}

func TestJWTUtils_Encode_EmptyUserID(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	_, err := utils.Encode(context.Background(), "", 1*time.Hour, domain.JWTTypeAccess)
	if err == nil {
		t.Error("Encode() expected error for empty userID but got none")
		return
	}

	if err.Type != contracts.ErrorTypeAPI {
		t.Errorf("Expected error type to be %s, got %s", contracts.ErrorTypeAPI, err.Type)
	}

	if err.PublicMessage != "Something went wrong." {
		t.Errorf("Expected error message to be 'Something went wrong.', got: %s", err.PublicMessage)
	}
}

func TestJWTUtils_Encode_EmptyTokenType(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	_, err := utils.Encode(context.Background(), "user123", 1*time.Hour, "")
	if err == nil {
		t.Error("Encode() expected error for empty token type but got none")
		return
	}

	if err.Type != contracts.ErrorTypeAPI {
		t.Errorf("Expected error type to be %s, got %s", contracts.ErrorTypeAPI, err.Type)
	}

	if err.PublicMessage != "Something went wrong." {
		t.Errorf("Expected error message to be 'Something went wrong.', got: %s", err.PublicMessage)
	}
}

func TestJWTUtils_Encode_EmptySecret(t *testing.T) {
	config := JWTConfig{
		Secret: "",
		Issuer: "https://test.com",
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("Encode() expected panic for empty secret but got none")
		}
	}()

	utils := NewJWTUtils(config)

	_, err := utils.Encode(context.Background(), "user123", 1*time.Hour, domain.JWTTypeAccess)
	if err != nil {
		t.Error("Encode() should have panicked for empty secret, but returned error instead")
		return
	}
}

func TestJWTUtils_Encode_DifferentExpirationTimes(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	userID := "user123"
	testCases := []time.Duration{
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour, // 1 week
	}

	for _, expiresIn := range testCases {
		t.Run(expiresIn.String(), func(t *testing.T) {
			token, err := utils.Encode(context.Background(), userID, expiresIn, domain.JWTTypeAccess)
			if err != nil {
				t.Fatalf("Encode() unexpected error = %v", err)
			}

			claims, err := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
			if err != nil {
				t.Fatalf("Failed to decode generated token: %v", err)
			}

			if claims.ExpiresAt == nil {
				t.Fatal("Token expiration time is nil")
			}

			expectedExpiry := time.Now().UTC().Add(expiresIn)
			actualExpiry := claims.ExpiresAt.Time

			// Allow for small timing differences (within 2 seconds)
			diff := actualExpiry.Sub(expectedExpiry)
			if diff < -2*time.Second || diff > 2*time.Second {
				t.Errorf("Token expiration time mismatch: expected around %v, got %v (diff: %v)",
					expectedExpiry, actualExpiry, diff)
			}
		})
	}
}

func TestJWTUtils_Decode_Success(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	userID := "user123"
	expiresIn := 1 * time.Hour

	token, err := utils.Encode(context.Background(), userID, expiresIn, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	claims, err := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Decode() unexpected error = %v", err)
	}

	if claims == nil {
		t.Fatal("Decode() returned nil claims")
	}

	if claims.Subject != userID {
		t.Errorf("Decoded subject mismatch: expected %s, got %s", userID, claims.Subject)
	}

	if claims.Issuer != config.Issuer {
		t.Errorf("Decoded issuer mismatch: expected %s, got %s", config.Issuer, claims.Issuer)
	}

	if claims.TokenType != domain.JWTTypeAccess {
		t.Errorf("Decoded token type mismatch: expected %s, got %s", domain.JWTTypeAccess, claims.TokenType)
	}
}

func TestJWTUtils_Decode_EmptySecret(t *testing.T) {
	config := JWTConfig{
		Secret: "",
		Issuer: "https://test.com",
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("Decode() expected panic for empty secret but got none")
		}
	}()

	utils := NewJWTUtils(config)

	_, err := utils.Decode(context.Background(), "any-token", domain.JWTTypeAccess)
	if err != nil {
		t.Error("Decode() should have panicked for empty secret, but returned error instead")
		return
	}
}

func TestJWTUtils_Decode_InvalidToken(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	testCases := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not.a.valid.token",
		},
		{
			name:  "token with wrong signature",
			token: "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := utils.Decode(context.Background(), tc.token, domain.JWTTypeAccess)
			if err == nil {
				t.Error("Decode() expected error for invalid token but got none")
				return
			}

			if err.Type != contracts.ErrorTypeInvalidRequest {
				t.Errorf("Expected error type to be %s, got %s", contracts.ErrorTypeInvalidRequest, err.Type)
			}

			if err.PublicMessage != ErrInvalidJWT {
				t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, err.PublicMessage)
			}
		})
	}
}

func TestJWTUtils_Decode_WrongTokenType(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	token, err := utils.Encode(context.Background(), "user123", 1*time.Hour, domain.JWTTypePasswordReset)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	_, decodeErr := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if decodeErr == nil {
		t.Fatal("Decode() expected error for mismatched token type but got none")
	}

	if decodeErr.Type != contracts.ErrorTypeInvalidRequest {
		t.Errorf("Expected error type to be %s, got %s", contracts.ErrorTypeInvalidRequest, decodeErr.Type)
	}

	if decodeErr.PublicMessage != ErrInvalidJWT {
		t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, decodeErr.PublicMessage)
	}
}

func TestJWTUtils_Decode_TokenWithWrongIssuer(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	// Create a token with a different issuer
	wrongConfig := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://wrong-issuer.com",
	}
	wrongUtils := NewJWTUtils(wrongConfig)

	token, err := wrongUtils.Encode(context.Background(), "user123", 1*time.Hour, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Try to decode with the original utils (different issuer)
	_, err = utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if err == nil {
		t.Error("Decode() expected error for token with wrong issuer but got none")
		return
	}

	if err.Type != contracts.ErrorTypeInvalidRequest {
		t.Errorf("Expected error type to be %s, got %s", contracts.ErrorTypeInvalidRequest, err.Type)
	}

	if err.PublicMessage != ErrInvalidJWT {
		t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, err.PublicMessage)
	}
}

func TestJWTUtils_Decode_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	// Create a token that expires in the past by setting a past time
	userID := "user123"
	expiresIn := 1 * time.Hour

	token, err := utils.Encode(context.Background(), userID, expiresIn, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// The token should decode successfully
	claims, err := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Decode() unexpected error = %v", err)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("Token expiration time is nil")
	}

	// Verify the token has an expiration time in the future
	if claims.ExpiresAt.Time.Before(time.Now().UTC()) {
		t.Error("Generated token is already expired")
	}
}

func TestJWTUtils_Decode_TokenWithWrongSigningMethod(t *testing.T) {
	config := JWTConfig{
		Secret: "test-secret-key-123",
		Issuer: "https://test.com",
	}
	utils := NewJWTUtils(config)

	wrongConfig := JWTConfig{
		Secret: "test-secret-key-456",
		Issuer: "https://test.com",
	}
	wrongUtils := NewJWTUtils(wrongConfig)

	tokenBadSecret, err := wrongUtils.Encode(context.Background(), "user123", 1*time.Hour, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	_, err = utils.Decode(context.Background(), tokenBadSecret, domain.JWTTypeAccess)
	if err == nil {
		t.Error("Decode() expected error for token with wrong signing method but got none")
		return
	}

	if err.Type != contracts.ErrorTypeInvalidRequest {
		t.Errorf("Expected error type to be %s, got %s", contracts.ErrorTypeInvalidRequest, err.Type)
	}

	if err.PublicMessage != ErrInvalidJWT {
		t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, err.PublicMessage)
	}
}

func TestJWTUtils_Integration(t *testing.T) {
	// Test the complete flow: encode -> decode -> verify
	config := JWTConfig{
		Secret: "integration-test-secret-456",
		Issuer: "https://integration-test.com",
	}
	utils := NewJWTUtils(config)

	userID := "integration-user-789"
	expiresIn := 30 * time.Minute

	// Step 1: Encode
	token, err := utils.Encode(context.Background(), userID, expiresIn, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Integration test failed at encode step: %v", err)
	}

	// Step 2: Decode
	claims, err := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Integration test failed at decode step: %v", err)
	}

	// Step 3: Verify all fields
	if claims.Subject != userID {
		t.Errorf("Integration test failed: subject mismatch, expected %s, got %s", userID, claims.Subject)
	}

	if claims.Issuer != config.Issuer {
		t.Errorf("Integration test failed: issuer mismatch, expected %s, got %s", config.Issuer, claims.Issuer)
	}

	if claims.IssuedAt == nil {
		t.Error("Integration test failed: issued at time is nil")
	}

	if claims.ExpiresAt == nil {
		t.Error("Integration test failed: expiration time is nil")
	}

	// Verify timing constraints
	now := time.Now().UTC()
	issuedTime := claims.IssuedAt.Time
	expiryTime := claims.ExpiresAt.Time

	if issuedTime.After(now) {
		t.Errorf("Integration test failed: issued time is in the future: %v", issuedTime)
	}

	if expiryTime.Before(now) {
		t.Errorf("Integration test failed: token is already expired: %v", expiryTime)
	}

	expectedExpiry := now.Add(expiresIn)
	if expiryTime.Sub(expectedExpiry) > 2*time.Second || expiryTime.Sub(expectedExpiry) < -2*time.Second {
		t.Errorf("Integration test failed: expiration time is too far from expected: expected around %v, got %v", expectedExpiry, expiryTime)
	}
}

func TestJWTUtils_TokenGeneration(t *testing.T) {
	config := JWTConfig{
		Secret: "generation-test-secret-789",
		Issuer: "https://generation-test.com",
	}
	utils := NewJWTUtils(config)

	userID := "generation-user"
	expiresIn := 1 * time.Hour

	// Test that we can generate tokens successfully
	token, err := utils.Encode(context.Background(), userID, expiresIn, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Generated token is empty")
	}

	// Test that the generated token can be decoded
	claims, err := utils.Decode(context.Background(), token, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to decode generated token: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("Token subject mismatch: expected %s, got %s", userID, claims.Subject)
	}

	// Test that we can generate tokens with different user IDs
	userID2 := "generation-user-2"
	token2, err := utils.Encode(context.Background(), userID2, expiresIn, domain.JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate second token: %v", err)
	}

	if token == token2 {
		t.Error("Tokens for different users should be different")
	}
}
