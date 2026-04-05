package token

import (
	"context"
	"testing"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

func TestEncodeJWT_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	secret := "test-secret-key-123"
	userID := "user123"

	token, err := EncodeJWT(ctx, secret, userID, time.Hour, JWTTypeAccess)
	if err != nil {
		t.Fatalf("EncodeJWT() unexpected error = %v", err)
	}
	if token == "" {
		t.Fatal("EncodeJWT() returned empty token")
	}

	// Verify the token can be decoded
	claims, err := DecodeJWT(ctx, secret, token, JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to decode generated token: %v", err)
	}
	if claims.Subject != userID {
		t.Errorf("Token subject mismatch: expected %s, got %s", userID, claims.Subject)
	}
	if claims.Issuer != defaultJWTIssuer {
		t.Errorf("Token issuer mismatch: expected %s, got %s", defaultJWTIssuer, claims.Issuer)
	}
	if claims.TokenType != JWTTypeAccess {
		t.Errorf("Token type mismatch: expected %s, got %s", JWTTypeAccess, claims.TokenType)
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now().UTC()) {
		t.Error("Generated token is already expired")
	}
}

func TestEncodeJWT_EmptyUserID(t *testing.T) {
	t.Parallel()
	_, err := EncodeJWT(context.Background(), "test-secret", "", time.Hour, JWTTypeAccess)
	if err == nil {
		t.Fatal("EncodeJWT() expected error for empty userID but got none")
	}
	if err.Type != apierror.ErrorTypeAPI {
		t.Errorf("Expected error type to be %s, got %s", apierror.ErrorTypeAPI, err.Type)
	}
}

func TestEncodeJWT_EmptyTokenType(t *testing.T) {
	t.Parallel()
	_, err := EncodeJWT(context.Background(), "test-secret", "user123", time.Hour, "")
	if err == nil {
		t.Fatal("EncodeJWT() expected error for empty token type but got none")
	}
	if err.Type != apierror.ErrorTypeAPI {
		t.Errorf("Expected error type to be %s, got %s", apierror.ErrorTypeAPI, err.Type)
	}
}

func TestDecodeJWT_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	secret := "test-secret-key-123"
	userID := "user123"

	token, err := EncodeJWT(ctx, secret, userID, time.Hour, JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	claims, err := DecodeJWT(ctx, secret, token, JWTTypeAccess)
	if err != nil {
		t.Fatalf("DecodeJWT() unexpected error = %v", err)
	}
	if claims == nil {
		t.Fatal("DecodeJWT() returned nil claims")
	}
	if claims.Subject != userID {
		t.Errorf("Decoded subject mismatch: expected %s, got %s", userID, claims.Subject)
	}
	if claims.TokenType != JWTTypeAccess {
		t.Errorf("Decoded token type mismatch: expected %s, got %s", JWTTypeAccess, claims.TokenType)
	}
}

func TestDecodeJWT_InvalidToken(t *testing.T) {
	t.Parallel()
	secret := "test-secret-key-123"
	testCases := []struct {
		name  string
		token string
	}{
		{name: "empty token", token: ""},
		{name: "malformed token", token: "not.a.valid.token"},
		{name: "token with wrong signature", token: "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeJWT(context.Background(), secret, tc.token, JWTTypeAccess)
			if err == nil {
				t.Fatal("DecodeJWT() expected error for invalid token but got none")
			}
			if err.Type != apierror.ErrorTypeInvalidRequest {
				t.Errorf("Expected error type to be %s, got %s", apierror.ErrorTypeInvalidRequest, err.Type)
			}
			if err.PublicMessage != ErrInvalidJWT {
				t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, err.PublicMessage)
			}
		})
	}
}

func TestDecodeJWT_WrongTokenType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	secret := "test-secret-key-123"

	token, err := EncodeJWT(ctx, secret, "user123", time.Hour, JWTTypePasswordReset)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	_, decodeErr := DecodeJWT(ctx, secret, token, JWTTypeAccess)
	if decodeErr == nil {
		t.Fatal("DecodeJWT() expected error for mismatched token type but got none")
	}
	if decodeErr.PublicMessage != ErrInvalidJWT {
		t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, decodeErr.PublicMessage)
	}
}

func TestDecodeJWT_WrongSecret(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	token, err := EncodeJWT(ctx, "secret-one", "user123", time.Hour, JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	_, decodeErr := DecodeJWT(ctx, "secret-two", token, JWTTypeAccess)
	if decodeErr == nil {
		t.Fatal("DecodeJWT() expected error for wrong secret but got none")
	}
	if decodeErr.PublicMessage != ErrInvalidJWT {
		t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, decodeErr.PublicMessage)
	}
}

func TestDecodeJWT_ExpiredToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	secret := "test-secret-key-123"

	token, err := EncodeJWT(ctx, secret, "user123", -time.Hour, JWTTypeAccess)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	_, decodeErr := DecodeJWT(ctx, secret, token, JWTTypeAccess)
	if decodeErr == nil {
		t.Fatal("DecodeJWT() expected error for expired token but got none")
	}
	if decodeErr.PublicMessage != ErrInvalidJWT {
		t.Errorf("Expected error message to be '%s', got: %s", ErrInvalidJWT, decodeErr.PublicMessage)
	}
}

func TestEncodeJWT_DifferentExpirationTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	secret := "test-secret-key-123"
	userID := "user123"

	testCases := []time.Duration{
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour,
	}

	for _, expiresIn := range testCases {
		t.Run(expiresIn.String(), func(t *testing.T) {
			token, err := EncodeJWT(ctx, secret, userID, expiresIn, JWTTypeAccess)
			if err != nil {
				t.Fatalf("EncodeJWT() unexpected error = %v", err)
			}

			claims, err := DecodeJWT(ctx, secret, token, JWTTypeAccess)
			if err != nil {
				t.Fatalf("Failed to decode generated token: %v", err)
			}
			if claims.ExpiresAt == nil {
				t.Fatal("Token expiration time is nil")
			}

			expectedExpiry := time.Now().UTC().Add(expiresIn)
			diff := claims.ExpiresAt.Time.Sub(expectedExpiry)
			if diff < -2*time.Second || diff > 2*time.Second {
				t.Errorf("Token expiration time mismatch: expected around %v, got %v (diff: %v)",
					expectedExpiry, claims.ExpiresAt.Time, diff)
			}
		})
	}
}

func TestJWT_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	secret := "integration-test-secret-456"
	userID := "integration-user-789"
	expiresIn := 30 * time.Minute

	token, err := EncodeJWT(ctx, secret, userID, expiresIn, JWTTypeAccess)
	if err != nil {
		t.Fatalf("Integration test failed at encode step: %v", err)
	}

	claims, err := DecodeJWT(ctx, secret, token, JWTTypeAccess)
	if err != nil {
		t.Fatalf("Integration test failed at decode step: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("subject mismatch, expected %s, got %s", userID, claims.Subject)
	}
	if claims.Issuer != defaultJWTIssuer {
		t.Errorf("issuer mismatch, expected %s, got %s", defaultJWTIssuer, claims.Issuer)
	}
	if claims.IssuedAt == nil {
		t.Error("issued at time is nil")
	}
	if claims.ExpiresAt == nil {
		t.Error("expiration time is nil")
	}

	now := time.Now().UTC()
	if claims.IssuedAt.Time.After(now) {
		t.Errorf("issued time is in the future: %v", claims.IssuedAt.Time)
	}
	if claims.ExpiresAt.Time.Before(now) {
		t.Errorf("token is already expired: %v", claims.ExpiresAt.Time)
	}
}

func TestSanitizeJWT(t *testing.T) {
	t.Parallel()
	token := "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbl90eXBlIjoiYWNjZXNzIn0.abc123"
	sanitized := SanitizeJWT(token)
	if sanitized == token {
		t.Error("SanitizeJWT should modify the token for display")
	}
	if sanitized == "" {
		t.Error("SanitizeJWT returned empty string")
	}
}
