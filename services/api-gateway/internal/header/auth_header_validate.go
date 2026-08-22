package header

import (
	"encoding/base64"
	"fmt"
	"strings"

	authsvctypes "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/sanitize"
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

func ValidateAndExtractAuthHeader(authHeader string) (*AuthHeaderResult, *apierror.APIError) {
	if authHeader == "" {
		return nil, apierror.NewAuthenticationError(ErrHeaderPrefix + " Authorization header is required.")
	}

	if err := validateAuthHeaderNotEmpty(authHeader); err != nil {
		return nil, err
	}

	var tokenString string
	var scheme AuthScheme
	var err *apierror.APIError

	if strings.HasPrefix(strings.ToLower(authHeader), fmt.Sprintf("%s ", string(AuthSchemeBasic))) {
		tokenString, err = validateBasicAuthHeader(authHeader)
		scheme = AuthSchemeBasic
	} else if strings.HasPrefix(strings.ToLower(authHeader), fmt.Sprintf("%s ", string(AuthSchemeBearer))) {
		tokenString, err = validateBearerHeader(authHeader)
		scheme = AuthSchemeBearer
	} else {
		return nil, apierror.NewAuthenticationError(fmt.Sprintf("%s Expected 'Basic <token>:' or 'Bearer <token>'.", ErrHeaderPrefix))
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

// validateAuthHeaderNotEmpty validates that the auth header is not empty.
func validateAuthHeaderNotEmpty(authHeader string) *apierror.APIError {
	trimmed := strings.TrimSpace(authHeader)
	switch strings.ToLower(trimmed) {
	case string(AuthSchemeBearer), string(AuthSchemeBasic):
		return apierror.NewAuthenticationError(fmt.Sprintf("%s You provided '%s' but no token. %s", ErrHeaderPrefix, trimmed, ErrEnvMisconfigured))
	}

	return nil
}

func validateBasicAuthHeader(authHeader string) (string, *apierror.APIError) {
	base64Creds := authHeader[6:] // Remove the "basic " portion
	decoded, err := base64.StdEncoding.DecodeString(base64Creds)
	if err != nil {
		return "", apierror.NewAuthenticationError(fmt.Sprintf("The %s auth header must be base64 encoded. You provided: %s.", AuthSchemeBasic, cleanTokenForResponse(base64Creds)))
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", apierror.NewAuthenticationError(fmt.Sprintf("%s %s. We expected '%s <token>:'. Did you forget the ':'?", ErrAPIKeyInvalid, cleanTokenForResponse(string(decoded)), AuthSchemeBasic))
	}

	tokenString := parts[0]
	if err := validateTokenString(tokenString, fmt.Sprintf("%s %s", ErrAPIKeyInvalid, cleanTokenForResponse(tokenString))); err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateBearerHeader(authHeader string) (string, *apierror.APIError) {
	tokenString := authHeader[7:] // Remove the "bearer " portion
	if err := validateTokenString(tokenString, ErrAPIKeyInvalid+cleanTokenForResponse(tokenString)); err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateTokenString(tokenString string, context string) *apierror.APIError {
	if tokenString == "" {
		return apierror.NewAuthenticationError(context + " You provided the header, but no token was provided. " + ErrEnvMisconfigured)
	}

	if tokenString == "undefined" {
		return apierror.NewAuthenticationError(context + " If you are sending the request with JavaScript, ensure your API key is setup properly in your environment.")
	}

	if tokenString == "None" {
		return apierror.NewAuthenticationError(context + " If you are sending the request with Python, ensure your API key is setup properly in your environment.")
	}

	if tokenString == "null" {
		return apierror.NewAuthenticationError(context + " If you are sending the request with Java, ensure your API key is setup properly in your environment.")
	}

	return nil
}
