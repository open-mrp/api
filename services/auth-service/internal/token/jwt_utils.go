package token

import (
	"context"
	"errors"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	sanitize "github.com/augno/api/shared/sanitize"
	"github.com/augno/api/shared/tracing"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrJWT = errors.New("JWT error")
)

var jwtUtilsTracer = tracing.GetTracer("auth-service.jwt_utils")

type JWTConfig struct {
	Secret string
	Issuer string
}

func DefaultJWTConfig(secret string) JWTConfig {
	return JWTConfig{
		Secret: secret,
		Issuer: "https://augno.com",
	}
}

type jwtUtilsImpl struct {
	config JWTConfig
}

func NewJWTUtils(config JWTConfig) domain.JWTUtils {
	if config.Secret == "" {
		panic("JWT secret is not set in the config.")
	}

	return &jwtUtilsImpl{config: config}
}

// Encode encodes a new access token (a JWT) for the given user ID, expires in, and token type.
func (atu *jwtUtilsImpl) Encode(ctx context.Context, userID string, expiresIn time.Duration, tokenType domain.JWTType) (string, *contracts.APIError) {
	_, span := jwtUtilsTracer.Start(ctx, "utils.jwt.encode")
	defer span.End()

	// Validate inputs
	if userID == "" {
		return "", tracing.Trace(span, contracts.NewInternalError(ErrJWT, "Attempted to encode token with no user ID."))
	}
	if tokenType == "" {
		return "", tracing.Trace(span, contracts.NewInternalError(ErrJWT, "Attempted to encode token with no token type."))
	}

	// Create the claims
	now := time.Now().UTC()
	claims := domain.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    atu.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
		TokenType: tokenType,
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signingKey := []byte(atu.config.Secret)

	signedToken, err := token.SignedString(signingKey)
	if err != nil {
		return "", tracing.Trace(span, contracts.NewInternalError(err, "Failed to sign JWT token."))
	}

	return signedToken, nil
}

// Decode decodes a given access token string into claims
func (atu *jwtUtilsImpl) Decode(ctx context.Context, tokenString string, expectedType domain.JWTType) (*domain.JWTClaims, *contracts.APIError) {
	_, span := jwtUtilsTracer.Start(ctx, "utils.jwt.decode")
	defer span.End()

	// Validate inputs
	if expectedType == "" {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
	}
	if tokenString == "" {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &domain.JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
		}
		return []byte(atu.config.Secret), nil
	}, jwt.WithIssuer(atu.config.Issuer))

	if err != nil {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
	}

	// Validate the token
	claims, ok := token.Claims.(*domain.JWTClaims)
	if !ok {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
	}

	if !token.Valid {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
	}

	if claims.TokenType != expectedType {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidJWT))
	}

	return claims, nil
}

// SanitizeForDisplay sanitizes a given access token for display.
//
// Ex: eyJhbGci0iJI****sw5c
func (atu *jwtUtilsImpl) SanitizeForDisplay(tokenString string) string {
	return sanitize.SanitizeString(tokenString, 12, 4)
}
