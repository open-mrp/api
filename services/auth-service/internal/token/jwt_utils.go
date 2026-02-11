package token

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	sanitize "github.com/augno/api/shared/sanitize"
	"github.com/augno/api/shared/tracing"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrJWT = errors.New("JWT error")
)

var jwtUtilsTracer = tracing.GetTracer("auth-service.jwt_utils")

const defaultJWTIssuer = "https://augno.com"

// JWTConfig holds the configuration for JWT token encoding and decoding.
type JWTConfig struct {
	// Secret (required) is the secret used to sign and verify JWT tokens.
	Secret string // #nosec G117 - Struct field, not a hardcoded credential

	// Issuer (optional; default: "https://augno.com") is the issuer claim set on generated tokens.
	Issuer string
}

// WithDefaults returns a new JWTConfig with zero-value fields replaced by production defaults.
func (c *JWTConfig) WithDefaults() *JWTConfig {
	if c == nil {
		c = &JWTConfig{}
	}

	return &JWTConfig{
		Secret: c.Secret,
		Issuer: cmp.Or(c.Issuer, defaultJWTIssuer),
	}
}

// validate checks that all required JWTConfig fields are set.
func (c *JWTConfig) validate() error {
	if c.Secret == "" {
		return fmt.Errorf("jwt: secret is required")
	}
	return nil
}

type jwtUtilsImpl struct {
	config JWTConfig
}

// NewJWTUtils creates a new JWT utility with the given configuration.
func NewJWTUtils(config *JWTConfig) domain.JWTUtils {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &jwtUtilsImpl{config: *config}
}

// Encode encodes a new access token (a JWT) for the given user ID, expires in, and token type.
func (atu *jwtUtilsImpl) Encode(ctx context.Context, userID string, expiresIn time.Duration, tokenType domain.JWTType) (string, *apierror.APIError) {
	_, span := jwtUtilsTracer.Start(ctx, "utils.jwt.encode")
	defer span.End()

	// Validate inputs
	if userID == "" {
		return "", tracing.Trace(span, apierror.NewInternalError(ErrJWT, "Attempted to encode token with no user ID."))
	}
	if tokenType == "" {
		return "", tracing.Trace(span, apierror.NewInternalError(ErrJWT, "Attempted to encode token with no token type."))
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
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to sign JWT token."))
	}

	return signedToken, nil
}

// Decode decodes a given access token string into claims
func (atu *jwtUtilsImpl) Decode(ctx context.Context, tokenString string, expectedType domain.JWTType) (*domain.JWTClaims, *apierror.APIError) {
	_, span := jwtUtilsTracer.Start(ctx, "utils.jwt.decode")
	defer span.End()

	// Validate inputs
	if expectedType == "" {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}
	if tokenString == "" {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &domain.JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
		}
		return []byte(atu.config.Secret), nil
	}, jwt.WithIssuer(atu.config.Issuer))

	if err != nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	// Validate the token
	claims, ok := token.Claims.(*domain.JWTClaims)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	if !token.Valid {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	if claims.TokenType != expectedType {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidJWT))
	}

	return claims, nil
}

// SanitizeForDisplay sanitizes a given access token for display.
//
// Ex: eyJhbGci0iJI****sw5c
func (atu *jwtUtilsImpl) SanitizeForDisplay(tokenString string) string {
	return sanitize.SanitizeString(tokenString, 12, 4)
}
