package header

import (
	"encoding/base64"
	"strings"

	authsvctypes "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/sanitize"
)

var (
	ErrHeaderPrefix = "Invalid Authorization header format."
	// #nosec G101 - This is an error message, not a hardcoded credential
	ErrAPIKeyInvalid    = "Invalid API Key provided."
	ErrEnvMisconfigured = "You might have forgotten to setup your API key in your environment."
)

type AuthHeaderResult struct {
	TokenString string
	Scheme      AuthScheme
}

func ValidateAndExtractAuthHeader(authHeader string) (*AuthHeaderResult, *contracts.APIError) {
	if authHeader == "" {
		return nil, contracts.NewAuthenticationError(ErrHeaderPrefix + " Authorization header is required.")
	}

	if err := validateAuthHeaderNotEmpty(authHeader); err != nil {
		return nil, err
	}

	var tokenString string
	var scheme AuthScheme
	var err *contracts.APIError

	if strings.HasPrefix(strings.ToLower(authHeader), "basic ") {
		tokenString, err = validateBasicAuthHeader(authHeader)
		scheme = AuthSchemeBasic
	} else if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		tokenString, err = validateBearerHeader(authHeader)
		scheme = AuthSchemeBearer
	} else {
		return nil, contracts.NewAuthenticationError(ErrHeaderPrefix + " Expected 'Basic <token>:' or 'Bearer <token>'.")
	}

	if err != nil {
		return nil, err
	}

	return &AuthHeaderResult{
		TokenString: tokenString,
		Scheme:      scheme,
	}, nil
}

func IsAPIKey(token string) bool {
	return strings.HasPrefix(token, string(authsvctypes.APIKeyPrefixSecretKey))
}

func cleanTokenForResponse(tokenString string) string {
	return sanitize.SanitizeString(tokenString, 12, 4)
}

func validateAuthHeaderNotEmpty(authHeader string) *contracts.APIError {
	if strings.TrimSpace(authHeader) == "Bearer" {
		return contracts.NewAuthenticationError(ErrHeaderPrefix + " You provided 'Bearer' but no token. " + ErrEnvMisconfigured)
	}

	if strings.TrimSpace(authHeader) == "Basic" {
		return contracts.NewAuthenticationError(ErrHeaderPrefix + " You provided 'Basic' but no token. " + ErrEnvMisconfigured)
	}

	return nil
}

func validateBasicAuthHeader(authHeader string) (string, *contracts.APIError) {
	base64Creds := authHeader[6:] // Remove the "basic " portion
	decoded, err := base64.StdEncoding.DecodeString(base64Creds)
	if err != nil {
		return "", contracts.NewAuthenticationError("The Basic auth header must be base64 encoded. You provided: " + cleanTokenForResponse(base64Creds) + ".")
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", contracts.NewAuthenticationError(ErrAPIKeyInvalid + cleanTokenForResponse(string(decoded)) + ". We expected 'Basic <token>:'. Did you forget the ':'?")
	}

	tokenString := parts[0]
	if err := validateTokenString(tokenString, ErrAPIKeyInvalid+cleanTokenForResponse(tokenString)); err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateBearerHeader(authHeader string) (string, *contracts.APIError) {
	tokenString := authHeader[7:] // Remove the "bearer " portion
	if err := validateTokenString(tokenString, ErrAPIKeyInvalid+cleanTokenForResponse(tokenString)); err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateTokenString(tokenString string, context string) *contracts.APIError {
	if tokenString == "" {
		return contracts.NewAuthenticationError(context + " You provided the header, but no token was provided. " + ErrEnvMisconfigured)
	}

	if tokenString == "undefined" {
		return contracts.NewAuthenticationError(context + " If you are sending the request with JavaScript, check if your API key is setup properly in your environment.")
	}

	if tokenString == "None" {
		return contracts.NewAuthenticationError(context + " If you are sending the request with Python, check if your API key is setup properly in your environment.")
	}

	if tokenString == "null" {
		return contracts.NewAuthenticationError(context + " If you are sending the request with Java, check if your API key is setup properly in your environment.")
	}

	return nil
}
