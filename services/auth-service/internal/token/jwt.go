package token

import (
	"context"
	"errors"
	"time"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/sanitize"
	"github.com/augno/api/shared/tracing"

	"github.com/golang-jwt/jwt/v5"
)

// JWTType represents the type of JWT token.
type JWTType string

const (
	// JWTTypeAccess represents an access token.
	JWTTypeAccess JWTType = "access"
	// JWTTypePasswordReset represents a password reset token.
	JWTTypePasswordReset JWTType = "password_reset"
)

type JWTClaims struct {
	jwt.RegisteredClaims
	TokenType JWTType `json:"token_type"`
}

const (
	ErrInvalidJWT = "Invalid JWT."
)

var (
	ErrJWT = errors.New("JWT error")
)

const defaultJWTIssuer = "https://augno.com"

var jwtTracer = tracing.GetTracer("auth-service.jwt")

// EncodeJWT encodes a new JWT for the given user ID, expiration, and token type.
func EncodeJWT(ctx context.Context, jwtSecret, userID string, expiresIn time.Duration, tokenType JWTType) (string, *apierror.APIError) {
	_, span := jwtTracer.Start(ctx, "token.jwt.encode")
	defer span.End()

	if userID == "" {
		return "", tracing.Trace(span, apierror.NewInternalError(ErrJWT, "Attempted to encode token with no user ID."))
	}
	if tokenType == "" {
		return "", tracing.Trace(span, apierror.NewInternalError(ErrJWT, "Attempted to encode token with no token type."))
	}

	now := time.Now().UTC()
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    defaultJWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
		TokenType: tokenType,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signingKey := []byte(jwtSecret)

	signedToken, err := t.SignedString(signingKey)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to sign JWT token."))
	}

	return signedToken, nil
}

// DecodeJWT decodes a given JWT string into claims.
func DecodeJWT(ctx context.Context, jwtSecret, tokenString string, expectedType JWTType) (*JWTClaims, *apierror.APIError) {
	_, span := jwtTracer.Start(ctx, "token.jwt.decode")
	defer span.End()

	if expectedType == "" {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}
	if tokenString == "" {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	t, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
		}
		return []byte(jwtSecret), nil
	}, jwt.WithIssuer(defaultJWTIssuer))

	if err != nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	claims, ok := t.Claims.(*JWTClaims)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	if !t.Valid {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	if claims.TokenType != expectedType {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	return claims, nil
}

// SanitizeJWT sanitizes a given JWT for display.
func SanitizeJWT(tokenString string) string {
	return sanitize.SanitizeString(tokenString, 12, 4)
}
