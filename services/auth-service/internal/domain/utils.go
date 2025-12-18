package domain

import (
	"context"
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"

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
	Encode(ctx context.Context, userID string, expiresIn time.Duration, tokenType JWTType) (string, *contracts.APIError)
	Decode(ctx context.Context, tokenString string, expectedType JWTType) (*JWTClaims, *contracts.APIError)
}

type OpaqueTokenUtils interface {
	Gen(ctx context.Context) (string, *contracts.APIError)
}

type APIKeyUtils interface {
	Gen(ctx context.Context, appMode constants.AccountMode) (*ParsedAPIKey, *contracts.APIError)
	Parse(ctx context.Context, key string) (*ParsedAPIKey, *contracts.APIError)
	GenSecretHMAC(ctx context.Context, secret string) ([]byte, *contracts.APIError)
	VerifySecretHMAC(ctx context.Context, secret string, expectedHMAC []byte) (bool, *contracts.APIError)
	SanitizeForDisplay(apiKey string) string
}
