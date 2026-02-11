package header

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/testutil"
	apierror "github.com/augno/api/shared/errors"
)

func TestValidateAndExtractAuthHeader(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedResult *AuthHeaderResult
		expectedError  *apierror.APIError
		hasError       bool
	}{
		// Valid Basic auth headers
		{
			name:       "valid basic auth header",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("test-token:")),
			expectedResult: &AuthHeaderResult{
				TokenString: "test-token",
				Scheme:      AuthSchemeBasic,
			},
			hasError: false,
		},
		{
			name:       "valid basic auth header with uppercase",
			authHeader: "BASIC " + base64.StdEncoding.EncodeToString([]byte("api-key:")),
			expectedResult: &AuthHeaderResult{
				TokenString: "api-key",
				Scheme:      AuthSchemeBasic,
			},
			hasError: false,
		},
		{
			name:       "valid basic auth header with mixed case",
			authHeader: "BaSiC " + base64.StdEncoding.EncodeToString([]byte("my-token:")),
			expectedResult: &AuthHeaderResult{
				TokenString: "my-token",
				Scheme:      AuthSchemeBasic,
			},
			hasError: false,
		},

		// Valid Bearer auth headers
		{
			name:       "valid bearer auth header",
			authHeader: "Bearer " + testutil.APIKeyValidSandboxMode,
			expectedResult: &AuthHeaderResult{
				TokenString: testutil.APIKeyValidSandboxMode,
				Scheme:      AuthSchemeBearer,
			},
			hasError: false,
		},
		{
			name:       "valid bearer auth header with uppercase",
			authHeader: "BEARER " + testutil.APIKeyValidProdMode,
			expectedResult: &AuthHeaderResult{
				TokenString: testutil.APIKeyValidProdMode,
				Scheme:      AuthSchemeBearer,
			},
			hasError: false,
		},
		{
			name:       "valid bearer auth header with mixed case",
			authHeader: "BeArEr " + testutil.APIKeyValidSandboxMode,
			expectedResult: &AuthHeaderResult{
				TokenString: testutil.APIKeyValidSandboxMode,
				Scheme:      AuthSchemeBearer,
			},
			hasError: false,
		},

		// Invalid headers - empty
		{
			name:       "empty auth header",
			authHeader: "",
			hasError:   true,
		},

		// Invalid headers - missing token
		{
			name:       "bearer without token",
			authHeader: "Bearer",
			hasError:   true,
		},
		{
			name:       "basic without token",
			authHeader: "Basic",
			hasError:   true,
		},
		{
			name:       "bearer with only spaces",
			authHeader: "Bearer ",
			hasError:   true,
		},
		{
			name:       "basic with only spaces",
			authHeader: "Basic ",
			hasError:   true,
		},

		// Invalid headers - wrong scheme
		{
			name:       "invalid auth scheme",
			authHeader: "Digest token123",
			hasError:   true,
		},
		{
			name:       "no auth scheme",
			authHeader: "token123",
			hasError:   true,
		},

		// Invalid Basic auth - malformed base64
		{
			name:       "basic auth with invalid base64",
			authHeader: "Basic invalid-base64!@#",
			hasError:   true,
		},
		{
			name:       "basic auth with empty base64",
			authHeader: "Basic ",
			hasError:   true,
		},

		// Invalid Basic auth - missing colon
		{
			name:       "basic auth without colon",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("token-without-colon")),
			hasError:   true,
		},
		{
			name:       "basic auth with multiple colons",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("token:with:multiple:colons")),
			expectedResult: &AuthHeaderResult{
				TokenString: "token",
				Scheme:      AuthSchemeBasic,
			},
			hasError: false,
		},

		// Invalid token values
		{
			name:       "bearer with undefined token",
			authHeader: "Bearer undefined",
			hasError:   true,
		},
		{
			name:       "bearer with null token",
			authHeader: "Bearer null",
			hasError:   true,
		},
		{
			name:       "bearer with None token",
			authHeader: "Bearer None",
			hasError:   true,
		},
		{
			name:       "basic with undefined token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("undefined:")),
			hasError:   true,
		},
		{
			name:       "basic with null token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("null:")),
			hasError:   true,
		},
		{
			name:       "basic with None token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("None:")),
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndExtractAuthHeader(tt.authHeader)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for auth header: %s", tt.authHeader)
				}
				if result != nil {
					t.Errorf("expected nil result when error occurs, got: %+v", result)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for auth header: %s, error: %v", tt.authHeader, err)
				}
				if result == nil {
					t.Errorf("expected result for auth header: %s", tt.authHeader)
				} else {
					if result.TokenString != tt.expectedResult.TokenString {
						t.Errorf("expected token string %s, got %s", tt.expectedResult.TokenString, result.TokenString)
					}
					if result.Scheme != tt.expectedResult.Scheme {
						t.Errorf("expected scheme %s, got %s", tt.expectedResult.Scheme, result.Scheme)
					}
				}
			}
		})
	}
}

func TestValidateAndExtractAuthHeader_ErrorMessages(t *testing.T) {
	tests := []struct {
		name                string
		authHeader          string
		expectedErrorCode   apierror.ErrorCode
		expectedErrorType   apierror.ErrorType
		expectedMessagePart string
	}{
		{
			name:                "empty auth header error",
			authHeader:          "",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "Authorization header is required",
		},
		{
			name:                "bearer without token error",
			authHeader:          "Bearer",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "You provided 'Bearer' but no token",
		},
		{
			name:                "basic without token error",
			authHeader:          "Basic",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "You provided 'Basic' but no token",
		},
		{
			name:                "invalid auth scheme error",
			authHeader:          "Digest token123",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "Expected 'Basic <token>:' or 'Bearer <token>'",
		},
		{
			name:                "bearer with undefined token error",
			authHeader:          "Bearer undefined",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "If you are sending the request with JavaScript",
		},
		{
			name:                "bearer with null token error",
			authHeader:          "Bearer null",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "If you are sending the request with Java",
		},
		{
			name:                "bearer with None token error",
			authHeader:          "Bearer None",
			expectedErrorCode:   apierror.ErrorCodeInvalidCredentials,
			expectedErrorType:   apierror.ErrorTypeInvalidRequest,
			expectedMessagePart: "If you are sending the request with Python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAndExtractAuthHeader(tt.authHeader)

			if err == nil {
				t.Errorf("expected error for auth header: %s", tt.authHeader)
				return
			}

			if err.Code != tt.expectedErrorCode {
				t.Errorf("expected error code %s, got %s", tt.expectedErrorCode, err.Code)
			}

			if err.Type != tt.expectedErrorType {
				t.Errorf("expected error type %s, got %s", tt.expectedErrorType, err.Type)
			}

			if !strings.Contains(err.PublicMessage, tt.expectedMessagePart) {
				t.Errorf("expected error message to contain '%s', got: %s", tt.expectedMessagePart, err.PublicMessage)
			}
		})
	}
}

func TestValidateBasicAuthHeader(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		hasError      bool
	}{
		{
			name:          "valid basic auth with simple token",
			authHeader:    "Basic " + base64.StdEncoding.EncodeToString([]byte("token:")),
			expectedToken: "token",
			hasError:      false,
		},
		{
			name:          "valid basic auth with complex token",
			authHeader:    "Basic " + base64.StdEncoding.EncodeToString([]byte("aug_sk_test_12345:")),
			expectedToken: "aug_sk_test_12345",
			hasError:      false,
		},
		{
			name:          "valid basic auth with token containing special chars",
			authHeader:    "Basic " + base64.StdEncoding.EncodeToString([]byte("token-with_underscore.123:")),
			expectedToken: "token-with_underscore.123",
			hasError:      false,
		},
		{
			name:       "invalid base64 encoding",
			authHeader: "Basic invalid-base64!@#",
			hasError:   true,
		},
		{
			name:       "missing colon",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("token-without-colon")),
			hasError:   true,
		},
		{
			name:       "empty token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte(":")),
			hasError:   true,
		},
		{
			name:       "undefined token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("undefined:")),
			hasError:   true,
		},
		{
			name:       "null token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("null:")),
			hasError:   true,
		},
		{
			name:       "None token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("None:")),
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := validateBasicAuthHeader(tt.authHeader)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for auth header: %s", tt.authHeader)
				}
				if token != "" {
					t.Errorf("expected empty token when error occurs, got: %s", token)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for auth header: %s, error: %v", tt.authHeader, err)
				}
				if token != tt.expectedToken {
					t.Errorf("expected token %s, got %s", tt.expectedToken, token)
				}
			}
		})
	}
}

func TestValidateBearerHeader(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		hasError      bool
	}{
		{
			name:          "valid bearer with simple token",
			authHeader:    "Bearer simple-token",
			expectedToken: "simple-token",
			hasError:      false,
		},
		{
			name:          "valid bearer with API key",
			authHeader:    "Bearer " + testutil.APIKeyValidSandboxMode,
			expectedToken: testutil.APIKeyValidSandboxMode,
			hasError:      false,
		},
		{
			name:          "valid bearer with JWT token",
			authHeader:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectedToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			hasError:      false,
		},
		{
			name:       "empty token",
			authHeader: "Bearer ",
			hasError:   true,
		},
		{
			name:       "undefined token",
			authHeader: "Bearer undefined",
			hasError:   true,
		},
		{
			name:       "null token",
			authHeader: "Bearer null",
			hasError:   true,
		},
		{
			name:       "None token",
			authHeader: "Bearer None",
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := validateBearerHeader(tt.authHeader)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for auth header: %s", tt.authHeader)
				}
				if token != "" {
					t.Errorf("expected empty token when error occurs, got: %s", token)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for auth header: %s, error: %v", tt.authHeader, err)
				}
				if token != tt.expectedToken {
					t.Errorf("expected token %s, got %s", tt.expectedToken, token)
				}
			}
		})
	}
}

func TestValidateTokenString(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		context  string
		hasError bool
	}{
		{
			name:     "valid token",
			token:    "valid-token-123",
			context:  "test context",
			hasError: false,
		},
		{
			name:     "valid API key",
			token:    testutil.APIKeyValidSandboxMode,
			context:  "test context",
			hasError: false,
		},
		{
			name:     "empty token",
			token:    "",
			context:  "test context",
			hasError: true,
		},
		{
			name:     "undefined token",
			token:    "undefined",
			context:  "test context",
			hasError: true,
		},
		{
			name:     "null token",
			token:    "null",
			context:  "test context",
			hasError: true,
		},
		{
			name:     "None token",
			token:    "None",
			context:  "test context",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTokenString(tt.token, tt.context)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for token: %s", tt.token)
				} else {
					if !strings.Contains(err.PublicMessage, tt.context) {
						t.Errorf("expected error message to contain context '%s', got: %s", tt.context, err.PublicMessage)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for token: %s, error: %v", tt.token, err)
				}
			}
		})
	}
}

func TestValidateAndExtractAuthHeaderNotEmpty(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		hasError   bool
	}{
		{
			name:       "valid bearer with token",
			authHeader: "Bearer valid-token",
			hasError:   false,
		},
		{
			name:       "valid basic with token",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("token:")),
			hasError:   false,
		},
		{
			name:       "bearer without token",
			authHeader: "Bearer",
			hasError:   true,
		},
		{
			name:       "basic without token",
			authHeader: "Basic",
			hasError:   true,
		},
		{
			name:       "bearer with spaces",
			authHeader: "Bearer ",
			hasError:   true,
		},
		{
			name:       "basic with spaces",
			authHeader: "Basic ",
			hasError:   true,
		},
		{
			name:       "bearer with only spaces",
			authHeader: "Bearer   ",
			hasError:   true,
		},
		{
			name:       "basic with only spaces",
			authHeader: "Basic   ",
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuthHeaderNotEmpty(tt.authHeader)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for auth header: %s", tt.authHeader)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for auth header: %s, error: %v", tt.authHeader, err)
				}
			}
		})
	}
}

func TestCleanTokenForResponse(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		expectedLength int
	}{
		{
			name:           "short token",
			token:          "abc",
			expectedLength: 3, // Should not be truncated
		},
		{
			name:           "medium token",
			token:          "medium-length-token",
			expectedLength: 12 + 4 + 3, // 12 visible + 4 asterisks + 3 visible
		},
		{
			name:           "long token",
			token:          "very-long-token-that-should-be-truncated-for-security",
			expectedLength: 12 + 4 + 4, // 12 visible + 4 asterisks + 4 visible
		},
		{
			name:           "API key",
			token:          testutil.APIKeyValidSandboxMode,
			expectedLength: 12 + 4 + 4, // Should be truncated
		},
		{
			name:           "empty token",
			token:          "",
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanTokenForResponse(tt.token)

			if len(result) != tt.expectedLength {
				t.Errorf("expected result length %d, got %d for token: %s", tt.expectedLength, len(result), tt.token)
			}

			// For non-empty tokens, ensure the result contains asterisks if truncated
			if tt.token != "" && len(tt.token) > 12+4 {
				if !strings.Contains(result, "*") {
					t.Errorf("expected result to contain asterisks for long token: %s", tt.token)
				}
			}
		})
	}
}

func TestAuthSchemeConstants(t *testing.T) {
	// Test that the constants are properly defined
	if AuthSchemeBasic != "basic" {
		t.Errorf("expected AuthSchemeBasic to be 'basic', got: %s", AuthSchemeBasic)
	}

	if AuthSchemeBearer != "bearer" {
		t.Errorf("expected AuthSchemeBearer to be 'bearer', got: %s", AuthSchemeBearer)
	}
}

func TestAuthHeaderResult(t *testing.T) {
	// Test the AuthHeaderResult struct
	result := &AuthHeaderResult{
		TokenString: "test-token",
		Scheme:      AuthSchemeBasic,
	}

	if result.TokenString != "test-token" {
		t.Errorf("expected TokenString to be 'test-token', got: %s", result.TokenString)
	}

	if result.Scheme != AuthSchemeBasic {
		t.Errorf("expected Scheme to be AuthSchemeBasic, got: %s", result.Scheme)
	}
}

func TestValidateAndExtractAuthHeader_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		hasError   bool
	}{
		{
			name:       "bearer with only spaces",
			authHeader: "Bearer   ",
			hasError:   true, // Should fail because token is empty after trimming
		},
		{
			name:       "basic with only spaces in base64",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("   :")),
			hasError:   false, // This should pass because the base64 contains a colon, so it's valid format
		},
		{
			name:       "bearer with newlines",
			authHeader: "Bearer \n\ntoken\n",
			hasError:   false, // Should pass as it has a token
		},
		{
			name:       "basic with newlines in base64",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("token\n:")),
			hasError:   false, // Should pass as it has a token
		},
		{
			name:       "bearer with tabs",
			authHeader: "Bearer \t\ttoken\t",
			hasError:   false, // Should pass as it has a token
		},
		{
			name:       "basic with tabs in base64",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("token\t:")),
			hasError:   false, // Should pass as it has a token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAndExtractAuthHeader(tt.authHeader)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for auth header: %s", tt.authHeader)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for auth header: %s, error: %v", tt.authHeader, err)
				}
			}
		})
	}
}

func TestValidateAndExtractAuthHeader_ErrorConstants(t *testing.T) {
	// Test that error constants are properly defined
	if ErrHeaderPrefix == "" {
		t.Error("expected ErrHeaderPrefix to be defined")
	}

	if ErrAPIKeyInvalid == "" {
		t.Error("expected ErrAPIKeyInvalid to be defined")
	}

	if ErrEnvMisconfigured == "" {
		t.Error("expected ErrEnvMisconfigured to be defined")
	}

	// Test that error messages contain expected content
	if !strings.Contains(ErrHeaderPrefix, "Invalid Authorization header format") {
		t.Errorf("expected ErrHeaderPrefix to contain 'Invalid Authorization header format', got: %s", ErrHeaderPrefix)
	}

	if !strings.Contains(ErrAPIKeyInvalid, "Invalid API Key provided") {
		t.Errorf("expected ErrAPIKeyInvalid to contain 'Invalid API Key provided', got: %s", ErrAPIKeyInvalid)
	}

	if !strings.Contains(ErrEnvMisconfigured, "forgotten to setup your API key") {
		t.Errorf("expected ErrEnvMisconfigured to contain 'forgotten to setup your API key', got: %s", ErrEnvMisconfigured)
	}
}
