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
	Encode(ctx context.Context, userID string, expiresIn time.Duration, tokenType JWTType) (string, *apierror.APIError)
	Decode(ctx context.Context, tokenString string, expectedType JWTType) (*JWTClaims, *apierror.APIError)
}

type APIKeyUtils interface {
	Gen(ctx context.Context, appMode constants.AccountMode) (*ParsedAPIKey, *apierror.APIError)
	Parse(ctx context.Context, key string) (*ParsedAPIKey, *apierror.APIError)
	GenSecretHMAC(ctx context.Context, secret string) ([]byte, *apierror.APIError)
	VerifySecretHMAC(ctx context.Context, secret string, expectedHMAC []byte) (bool, *apierror.APIError)
	SanitizeForDisplay(apiKey string) string
}
