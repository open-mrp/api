package domain

import (
	"context"
	"time"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/golang-jwt/jwt/v5"
)

type JWTType string

const (
	JWTTypeAccess        JWTType = "access"
	JWTTypePasswordReset JWTType = "password_reset"
)

type JWTClaims struct {
	jwt.RegisteredClaims
	TokenType JWTType `json:"token_type"`
}

type JWTUtils interface {
	// Encode encodes a new access token (a JWT) for the given user ID, expires in, and token type.
	//
	//  1. Creates the claims.
	//  2. Creates and signs the token.
	//  3. Returns the signed token.
	Encode(ctx context.Context, userID string, expiresIn time.Duration, tokenType JWTType) (string, *apierror.APIError)

	// Decode decodes a given access token string into claims.
	//
	//  1. Parses the token.
	//  2. Validates the token.
	//  3. Returns the claims.
	Decode(ctx context.Context, tokenString string, expectedType JWTType) (*JWTClaims, *apierror.APIError)

	// SanitizeForDisplay sanitizes a given access token for display.
	//
	//  1. Trim the sensitive parts of the access token for display.
	//  2. Replace these with asterisks.
	//  3. Returns the sanitized access token.
	SanitizeForDisplay(tokenString string) string
}

type APIKeyUtils interface {
	// Gen generates a new API key for the given account mode.
	//
	//  1. Generates a random secret key of the given strength.
	//  2. Generates a random id key of the given strength.
	//  3. Generates a checksum for the key using the id and secret.
	//  4. Returns the parsed API key.
	Gen(ctx context.Context, appMode constants.AccountMode) (*ParsedAPIKey, *apierror.APIError)

	// Parse parses a given API key string into its components.
	//
	//  1. Splits the API key string into its components.
	//  2. Validates the API key string.
	//  3. Returns the parsed API key.
	Parse(ctx context.Context, key string) (*ParsedAPIKey, *apierror.APIError)

	// GenSecretHMAC generates a HMAC for the given secret.
	//
	//  1. Generates a HMAC for the given secret and pepper.
	//  2. Returns the HMAC.
	GenSecretHMAC(ctx context.Context, secret string) ([]byte, *apierror.APIError)

	// VerifySecretHMAC verifies a given secret against a expected HMAC.
	//
	//  1. Generates a HMAC for the given secret and pepper.
	//  2. Validates the HMAC against the expected HMAC.
	VerifySecretHMAC(ctx context.Context, secret string, expectedHMAC []byte) (bool, *apierror.APIError)

	// SanitizeForDisplay sanitizes a given API key for display.
	//
	//  1. Trim the sensitive parts of the API key for display.
	//  2. Replace these with asterisks.
	//  3. Returns the sanitized API key.
	SanitizeForDisplay(apiKey string) string
}
