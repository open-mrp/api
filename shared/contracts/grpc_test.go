package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// TestConvertAPIErrorToGRPC_AllErrorCodes tests that every API error code
// can be converted to a gRPC error without panicking.
func TestConvertAPIErrorToGRPC_AllErrorCodes(t *testing.T) {
	testCases := []struct {
		name     string
		apiErr   *APIError
		wantCode grpccodes.Code
	}{
		// Authentication errors
		{
			name:     "InvalidCredentials",
			apiErr:   NewAuthenticationError("Invalid credentials"),
			wantCode: grpccodes.Unauthenticated,
		},
		{
			name:     "ExpiredToken",
			apiErr:   NewAPIError(ErrorCodeExpiredToken, ErrorTypeInvalidRequest, "Token expired", ""),
			wantCode: grpccodes.Unauthenticated,
		},
		{
			name:     "ExpiredAPIKey",
			apiErr:   NewExpiredAPIKeyError("API key expired"),
			wantCode: grpccodes.Unauthenticated,
		},
		// Authorization errors
		{
			name:     "InsufficientPerms",
			apiErr:   NewAuthorizationError("Insufficient permissions"),
			wantCode: grpccodes.PermissionDenied,
		},
		// Validation errors
		{
			name:     "ValidationFailed",
			apiErr:   NewValidationError("Validation failed"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "MissingField",
			apiErr:   NewAPIError(ErrorCodeMissingField, ErrorTypeInvalidRequest, "Missing field", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "InvalidFormat",
			apiErr:   NewAPIError(ErrorCodeInvalidFormat, ErrorTypeInvalidRequest, "Invalid format", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterMissing",
			apiErr:   NewAPIError(ErrorCodeParameterMissing, ErrorTypeInvalidRequest, "Parameter missing", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterInvalid",
			apiErr:   NewAPIError(ErrorCodeParameterInvalid, ErrorTypeInvalidRequest, "Parameter invalid", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterUnknown",
			apiErr:   NewAPIError(ErrorCodeParameterUnknown, ErrorTypeInvalidRequest, "Parameter unknown", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParametersExclusive",
			apiErr:   NewAPIError(ErrorCodeParametersExclusive, ErrorTypeInvalidRequest, "Parameters exclusive", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		// Resource errors
		{
			name:     "ResourceNotFound",
			apiErr:   NewResourceNotFoundError("Resource not found"),
			wantCode: grpccodes.NotFound,
		},
		{
			name:     "ResourceExists",
			apiErr:   NewAPIError(ErrorCodeResourceExists, ErrorTypeInvalidRequest, "Resource exists", ""),
			wantCode: grpccodes.AlreadyExists,
		},
		{
			name:     "ResourceConflict",
			apiErr:   NewResourceConflictError("Resource conflict"),
			wantCode: grpccodes.Aborted,
		},
		// Business logic errors
		{
			name:     "RateLimitExceeded",
			apiErr:   NewRateLimitExceededError("Rate limit exceeded"),
			wantCode: grpccodes.ResourceExhausted,
		},
		// Server errors
		{
			name:     "SvcUnavailable",
			apiErr:   NewAPIError(ErrorCodeSvcUnavailable, ErrorTypeAPI, "Service unavailable", ""),
			wantCode: grpccodes.Unavailable,
		},
		{
			name:     "RequestTimeout",
			apiErr:   NewRequestTimeoutError("Request timed out"),
			wantCode: grpccodes.DeadlineExceeded,
		},
		{
			name:     "Timeout",
			apiErr:   NewAPIError(ErrorCodeTimeout, ErrorTypeAPI, "Timeout", ""),
			wantCode: grpccodes.DeadlineExceeded,
		},
		{
			name:     "MethodNotAllowed",
			apiErr:   NewMethodNotAllowedError("Method not allowed"),
			wantCode: grpccodes.Unimplemented,
		},
		{
			name:     "InternalError",
			apiErr:   NewInternalError(errors.New("internal error"), "Something went wrong"),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ExternalSvcError",
			apiErr:   NewAPIError(ErrorCodeExternalSvcError, ErrorTypeAPI, "External service error", ""),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ConnectionError",
			apiErr:   NewAPIError(ErrorCodeConnectionError, ErrorTypeAPI, "Connection error", ""),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ClientClosedRequest",
			apiErr:   NewClientClosedRequestError("Client closed request"),
			wantCode: grpccodes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			grpcErr := ConvertAPIErrorToGRPC(tc.apiErr)
			if grpcErr == nil {
				t.Fatal("expected gRPC error, got nil")
			}

			st, ok := grpcstatus.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}

			if st.Code() != tc.wantCode {
				t.Errorf("expected gRPC code %v, got %v", tc.wantCode, st.Code())
			}

			// Verify message contains encoded API error information
			message := st.Message()
			jsonData, ok := strings.CutPrefix(message, "__API_ERROR__:")
			if !ok {
				t.Errorf("expected message to start with __API_ERROR__:, got %q", message)
			}

			// Parse and verify all fields are preserved
			var decoded struct {
				Code            ErrorCode `json:"code"`
				Type            ErrorType `json:"type"`
				PublicMessage   string    `json:"message"`
				Param           string    `json:"param,omitempty"`
				DocURL          string    `json:"doc_url,omitempty"`
				InternalMessage string    `json:"internal_message,omitempty"`
				InternalError   string    `json:"internal_error,omitempty"`
			}
			if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
				t.Errorf("failed to decode JSON: %v", err)
				return
			}

			// Verify all fields match
			if decoded.Code != tc.apiErr.Code {
				t.Errorf("expected code %q, got %q", tc.apiErr.Code, decoded.Code)
			}
			if decoded.Type != tc.apiErr.Type {
				t.Errorf("expected type %q, got %q", tc.apiErr.Type, decoded.Type)
			}
			if decoded.PublicMessage != tc.apiErr.PublicMessage {
				t.Errorf("expected public message %q, got %q", tc.apiErr.PublicMessage, decoded.PublicMessage)
			}
			if decoded.Param != tc.apiErr.Param {
				t.Errorf("expected param %q, got %q", tc.apiErr.Param, decoded.Param)
			}
			if decoded.DocURL != tc.apiErr.DocURL {
				t.Errorf("expected doc_url %q, got %q", tc.apiErr.DocURL, decoded.DocURL)
			}
			if decoded.InternalMessage != tc.apiErr.InternalMessage {
				t.Errorf("expected internal message %q, got %q", tc.apiErr.InternalMessage, decoded.InternalMessage)
			}
		})
	}
}

// TestConvertAPIErrorToGRPC_RoundTrip tests that converting an API error to gRPC
// and back preserves the error code mapping (even if not exact due to many-to-one mapping).
func TestConvertAPIErrorToGRPC_RoundTrip(t *testing.T) {
	testCases := []struct {
		name              string
		apiErr            *APIError
		expectedRoundTrip ErrorCode // The error code we expect after round-trip
		expectExactMatch  bool      // Whether we expect exact code match
	}{
		// These should map exactly
		{
			name:              "InvalidCredentials",
			apiErr:            NewAuthenticationError("Invalid credentials"),
			expectedRoundTrip: ErrorCodeInvalidCredentials,
			expectExactMatch:  true,
		},
		{
			name:              "InsufficientPerms",
			apiErr:            NewAuthorizationError("Insufficient permissions"),
			expectedRoundTrip: ErrorCodeInsufficientPerms,
			expectExactMatch:  true,
		},
		{
			name:              "ValidationFailed",
			apiErr:            NewValidationError("Validation failed"),
			expectedRoundTrip: ErrorCodeValidationFailed,
			expectExactMatch:  true,
		},
		{
			name:              "ResourceNotFound",
			apiErr:            NewResourceNotFoundError("Resource not found"),
			expectedRoundTrip: ErrorCodeResourceNotFound,
			expectExactMatch:  true,
		},
		{
			name:              "ResourceConflict",
			apiErr:            NewResourceConflictError("Resource conflict"),
			expectedRoundTrip: ErrorCodeResourceConflict,
			expectExactMatch:  true,
		},
		{
			name:              "RateLimitExceeded",
			apiErr:            NewRateLimitExceededError("Rate limit exceeded"),
			expectedRoundTrip: ErrorCodeRateLimitExceeded,
			expectExactMatch:  true,
		},
		{
			name:              "SvcUnavailable",
			apiErr:            NewAPIError(ErrorCodeSvcUnavailable, ErrorTypeAPI, "Service unavailable", "internal msg"),
			expectedRoundTrip: ErrorCodeSvcUnavailable,
			expectExactMatch:  true, // Now with zero information loss
		},
		{
			name:              "RequestTimeout",
			apiErr:            NewRequestTimeoutError("Request timed out"),
			expectedRoundTrip: ErrorCodeRequestTimeout,
			expectExactMatch:  true,
		},
		// With zero information loss, all should match exactly
		{
			name:              "ExpiredToken",
			apiErr:            NewAPIError(ErrorCodeExpiredToken, ErrorTypeInvalidRequest, "Token expired", ""),
			expectedRoundTrip: ErrorCodeExpiredToken,
			expectExactMatch:  true,
		},
		{
			name:              "ExpiredAPIKey",
			apiErr:            NewExpiredAPIKeyError("API key expired"),
			expectedRoundTrip: ErrorCodeExpiredAPIKey,
			expectExactMatch:  true,
		},
		{
			name:              "MissingField",
			apiErr:            NewAPIError(ErrorCodeMissingField, ErrorTypeInvalidRequest, "Missing field", ""),
			expectedRoundTrip: ErrorCodeMissingField,
			expectExactMatch:  true,
		},
		{
			name:              "MethodNotAllowed",
			apiErr:            NewMethodNotAllowedError("Method not allowed"),
			expectedRoundTrip: ErrorCodeMethodNotAllowed,
			expectExactMatch:  true,
		},
		{
			name:              "InternalError",
			apiErr:            NewInternalError(errors.New("test"), "Internal error"),
			expectedRoundTrip: ErrorCodeInternalError,
			expectExactMatch:  true,
		},
	}

	ctx := context.Background()
	serviceName := "test-service"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert API error to gRPC
			grpcErr := ConvertAPIErrorToGRPC(tc.apiErr)
			if grpcErr == nil {
				t.Fatal("expected gRPC error, got nil")
			}

			// Convert gRPC error back to API error
			roundTripErr := ConvertGRPCError(ctx, grpcErr, serviceName)
			if roundTripErr == nil {
				t.Fatal("expected API error, got nil")
			}

			// With zero information loss, all fields should match exactly
			if roundTripErr.Code != tc.apiErr.Code {
				t.Errorf("expected error code %q, got %q", tc.apiErr.Code, roundTripErr.Code)
			}
			if roundTripErr.Type != tc.apiErr.Type {
				t.Errorf("expected error type %q, got %q", tc.apiErr.Type, roundTripErr.Type)
			}
			if roundTripErr.PublicMessage != tc.apiErr.PublicMessage {
				t.Errorf("expected public message %q, got %q", tc.apiErr.PublicMessage, roundTripErr.PublicMessage)
			}
			if roundTripErr.Param != tc.apiErr.Param {
				t.Errorf("expected param %q, got %q", tc.apiErr.Param, roundTripErr.Param)
			}
			if roundTripErr.DocURL != tc.apiErr.DocURL {
				t.Errorf("expected doc_url %q, got %q", tc.apiErr.DocURL, roundTripErr.DocURL)
			}
			if roundTripErr.InternalMessage != tc.apiErr.InternalMessage {
				t.Errorf("expected internal message %q, got %q", tc.apiErr.InternalMessage, roundTripErr.InternalMessage)
			}
			// Internal error is converted to string, so compare string representations
			if tc.apiErr.Internal != nil {
				if roundTripErr.Internal == nil {
					t.Errorf("expected internal error to be preserved, got nil")
				} else if tc.apiErr.Internal.Error() != roundTripErr.Internal.Error() {
					t.Errorf("expected internal error %q, got %q", tc.apiErr.Internal.Error(), roundTripErr.Internal.Error())
				}
			} else if roundTripErr.Internal != nil {
				t.Errorf("expected no internal error, got %q", roundTripErr.Internal.Error())
			}
		})
	}
}

// TestConvertAPIErrorToGRPC_Nil tests nil handling.
func TestConvertAPIErrorToGRPC_Nil(t *testing.T) {
	grpcErr := ConvertAPIErrorToGRPC(nil)
	if grpcErr != nil {
		t.Errorf("expected nil, got %v", grpcErr)
	}
}

// TestConvertAPIErrorToGRPC_MessagePreservation tests that all fields are preserved correctly.
func TestConvertAPIErrorToGRPC_MessagePreservation(t *testing.T) {
	tests := []struct {
		name        string
		apiErr      *APIError
		description string
	}{
		{
			name:        "PublicMessage only",
			apiErr:      NewValidationError("Public message"),
			description: "PublicMessage should be preserved",
		},
		{
			name: "InternalMessage fallback",
			apiErr: NewAPIError(
				ErrorCodeValidationFailed,
				ErrorTypeInvalidRequest,
				"", // Empty public message
				"Internal message",
			),
			description: "InternalMessage should be preserved",
		},
		{
			name:        "PublicMessage preferred",
			apiErr:      NewAPIError(ErrorCodeValidationFailed, ErrorTypeInvalidRequest, "Public", "Internal"),
			description: "Both PublicMessage and InternalMessage should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := ConvertAPIErrorToGRPC(tt.apiErr)
			if grpcErr == nil {
				t.Fatal("expected gRPC error, got nil")
			}

			st, ok := grpcstatus.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}

			// Message should contain encoded JSON
			message := st.Message()
			jsonData, ok := strings.CutPrefix(message, "__API_ERROR__:")
			if !ok {
				t.Errorf("expected message to start with __API_ERROR__:, got %q", message)
			}

			// Parse and verify all fields are preserved
			var decoded struct {
				Code            ErrorCode `json:"code"`
				Type            ErrorType `json:"type"`
				PublicMessage   string    `json:"message"`
				InternalMessage string    `json:"internal_message,omitempty"`
			}
			if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
				t.Fatalf("failed to decode JSON: %v", err)
			}

			// Verify all fields match
			if decoded.Code != tt.apiErr.Code {
				t.Errorf("%s: expected code %q, got %q", tt.description, tt.apiErr.Code, decoded.Code)
			}
			if decoded.Type != tt.apiErr.Type {
				t.Errorf("%s: expected type %q, got %q", tt.description, tt.apiErr.Type, decoded.Type)
			}
			if decoded.PublicMessage != tt.apiErr.PublicMessage {
				t.Errorf("%s: expected public message %q, got %q", tt.description, tt.apiErr.PublicMessage, decoded.PublicMessage)
			}
			if decoded.InternalMessage != tt.apiErr.InternalMessage {
				t.Errorf("%s: expected internal message %q, got %q", tt.description, tt.apiErr.InternalMessage, decoded.InternalMessage)
			}
		})
	}
}

// TestConvertGRPCError_AllCodes tests that all gRPC codes can be converted to API errors.
func TestConvertGRPCError_AllCodes(t *testing.T) {
	testCases := []struct {
		name         string
		grpcCode     grpccodes.Code
		message      string
		expectedCode ErrorCode
		expectedType ErrorType
		description  string
	}{
		{
			name:         "InvalidArgument",
			grpcCode:     grpccodes.InvalidArgument,
			message:      "Invalid argument",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "NotFound",
			grpcCode:     grpccodes.NotFound,
			message:      "Not found",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "AlreadyExists",
			grpcCode:     grpccodes.AlreadyExists,
			message:      "Already exists",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "PermissionDenied",
			grpcCode:     grpccodes.PermissionDenied,
			message:      "Permission denied",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "ResourceExhausted",
			grpcCode:     grpccodes.ResourceExhausted,
			message:      "Resource exhausted",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "FailedPrecondition",
			grpcCode:     grpccodes.FailedPrecondition,
			message:      "Failed precondition",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "Aborted",
			grpcCode:     grpccodes.Aborted,
			message:      "Aborted",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "OutOfRange",
			grpcCode:     grpccodes.OutOfRange,
			message:      "Out of range",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "Unauthenticated",
			grpcCode:     grpccodes.Unauthenticated,
			message:      "Unauthenticated",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "Unavailable",
			grpcCode:     grpccodes.Unavailable,
			message:      "Unavailable",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "DeadlineExceeded",
			grpcCode:     grpccodes.DeadlineExceeded,
			message:      "Deadline exceeded",
			expectedCode: ErrorCodeRequestTimeout,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "Internal",
			grpcCode:     grpccodes.Internal,
			message:      "Internal error",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "Unimplemented",
			grpcCode:     grpccodes.Unimplemented,
			message:      "Unimplemented",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
		{
			name:         "DataLoss",
			grpcCode:     grpccodes.DataLoss,
			message:      "Data loss",
			expectedCode: ErrorCodeInternalError,
			expectedType: ErrorTypeAPI,
		},
	}

	ctx := context.Background()
	serviceName := "test-service"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			grpcErr := grpcstatus.Error(tc.grpcCode, tc.message)
			apiErr := ConvertGRPCError(ctx, grpcErr, serviceName)

			if apiErr == nil {
				t.Fatal("expected API error, got nil")
			}

			if apiErr.Code != tc.expectedCode {
				t.Errorf("expected error code %q, got %q", tc.expectedCode, apiErr.Code)
			}

			if apiErr.Type != tc.expectedType {
				t.Errorf("expected error type %q, got %q", tc.expectedType, apiErr.Type)
			}

			// Verify message exists (non-API errors use generic messages to avoid leaking internal details)
			if apiErr.PublicMessage == "" && apiErr.InternalMessage == "" {
				t.Errorf("expected message to exist, got empty")
			}
		})
	}
}

// TestConvertGRPCError_NonStatusError tests handling of non-status errors.
func TestConvertGRPCError_NonStatusError(t *testing.T) {
	ctx := context.Background()
	serviceName := "test-service"

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}

		if apiErr.Code != ErrorCodeInternalError {
			t.Errorf("expected error code %q, got %q", ErrorCodeInternalError, apiErr.Code)
		}
	})

	t.Run("deadline exceeded context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		<-ctx.Done() // Ensure context is done

		err := ctx.Err()
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}

		if apiErr.Code != ErrorCodeRequestTimeout {
			t.Errorf("expected error code %q, got %q", ErrorCodeRequestTimeout, apiErr.Code)
		}
	})
}

// TestConvertGRPCError_Nil tests nil handling.
func TestConvertGRPCError_Nil(t *testing.T) {
	ctx := context.Background()
	apiErr := ConvertGRPCError(ctx, nil, "test-service")
	if apiErr != nil {
		t.Errorf("expected nil, got %v", apiErr)
	}
}

// TestRoundTrip_AllAPIErrorCodes tests that every single API error code
// can survive a round-trip conversion without losing critical information.
func TestRoundTrip_AllAPIErrorCodes(t *testing.T) {
	allErrorCodes := []struct {
		name   string
		code   ErrorCode
		create func() *APIError
	}{
		{name: "ExpiredToken", code: ErrorCodeExpiredToken, create: func() *APIError {
			return NewAPIError(ErrorCodeExpiredToken, ErrorTypeInvalidRequest, "Token expired", "")
		}},
		{name: "ExpiredAPIKey", code: ErrorCodeExpiredAPIKey, create: func() *APIError {
			return NewExpiredAPIKeyError("API key expired")
		}},
		{name: "InvalidCredentials", code: ErrorCodeInvalidCredentials, create: func() *APIError {
			return NewAuthenticationError("Invalid credentials")
		}},
		{name: "InsufficientPerms", code: ErrorCodeInsufficientPerms, create: func() *APIError {
			return NewAuthorizationError("Insufficient permissions")
		}},
		{name: "ValidationFailed", code: ErrorCodeValidationFailed, create: func() *APIError {
			return NewValidationError("Validation failed")
		}},
		{name: "MissingField", code: ErrorCodeMissingField, create: func() *APIError {
			return NewAPIError(ErrorCodeMissingField, ErrorTypeInvalidRequest, "Missing field", "")
		}},
		{name: "InvalidFormat", code: ErrorCodeInvalidFormat, create: func() *APIError {
			return NewAPIError(ErrorCodeInvalidFormat, ErrorTypeInvalidRequest, "Invalid format", "")
		}},
		{name: "MethodNotAllowed", code: ErrorCodeMethodNotAllowed, create: func() *APIError {
			return NewMethodNotAllowedError("Method not allowed")
		}},
		{name: "ResourceNotFound", code: ErrorCodeResourceNotFound, create: func() *APIError {
			return NewResourceNotFoundError("Resource not found")
		}},
		{name: "ResourceExists", code: ErrorCodeResourceExists, create: func() *APIError {
			return NewAPIError(ErrorCodeResourceExists, ErrorTypeInvalidRequest, "Resource exists", "")
		}},
		{name: "ResourceConflict", code: ErrorCodeResourceConflict, create: func() *APIError {
			return NewResourceConflictError("Resource conflict")
		}},
		{name: "RateLimitExceeded", code: ErrorCodeRateLimitExceeded, create: func() *APIError {
			return NewRateLimitExceededError("Rate limit exceeded")
		}},
		{name: "ParameterMissing", code: ErrorCodeParameterMissing, create: func() *APIError {
			return NewAPIError(ErrorCodeParameterMissing, ErrorTypeInvalidRequest, "Parameter missing", "")
		}},
		{name: "ParameterInvalid", code: ErrorCodeParameterInvalid, create: func() *APIError {
			return NewAPIError(ErrorCodeParameterInvalid, ErrorTypeInvalidRequest, "Parameter invalid", "")
		}},
		{name: "ParameterUnknown", code: ErrorCodeParameterUnknown, create: func() *APIError {
			return NewAPIError(ErrorCodeParameterUnknown, ErrorTypeInvalidRequest, "Parameter unknown", "")
		}},
		{name: "ParametersExclusive", code: ErrorCodeParametersExclusive, create: func() *APIError {
			return NewAPIError(ErrorCodeParametersExclusive, ErrorTypeInvalidRequest, "Parameters exclusive", "")
		}},
		{name: "InternalError", code: ErrorCodeInternalError, create: func() *APIError {
			return NewInternalError(errors.New("test"), "Internal error")
		}},
		{name: "SvcUnavailable", code: ErrorCodeSvcUnavailable, create: func() *APIError {
			return NewAPIError(ErrorCodeSvcUnavailable, ErrorTypeAPI, "Service unavailable", "internal")
		}},
		{name: "ExternalSvcError", code: ErrorCodeExternalSvcError, create: func() *APIError {
			return NewAPIError(ErrorCodeExternalSvcError, ErrorTypeAPI, "External service error", "")
		}},
		{name: "Timeout", code: ErrorCodeTimeout, create: func() *APIError {
			return NewAPIError(ErrorCodeTimeout, ErrorTypeAPI, "Timeout", "")
		}},
		{name: "ConnectionError", code: ErrorCodeConnectionError, create: func() *APIError {
			return NewAPIError(ErrorCodeConnectionError, ErrorTypeAPI, "Connection error", "")
		}},
		{name: "RequestTimeout", code: ErrorCodeRequestTimeout, create: func() *APIError {
			return NewRequestTimeoutError("Request timed out")
		}},
		{name: "ClientClosedRequest", code: ErrorCodeClientClosedRequest, create: func() *APIError {
			return NewClientClosedRequestError("Client closed request")
		}},
	}

	ctx := context.Background()
	serviceName := "test-service"

	for _, tc := range allErrorCodes {
		t.Run(tc.name, func(t *testing.T) {
			original := tc.create()

			// Convert to gRPC
			grpcErr := ConvertAPIErrorToGRPC(original)
			if grpcErr == nil {
				t.Fatal("expected gRPC error, got nil")
			}

			// Convert back to API error
			roundTrip := ConvertGRPCError(ctx, grpcErr, serviceName)
			if roundTrip == nil {
				t.Fatal("expected API error, got nil")
			}

			// With zero information loss, all fields should match exactly
			if roundTrip.Code != original.Code {
				t.Errorf("error code not preserved: expected %q, got %q", original.Code, roundTrip.Code)
			}
			if roundTrip.Type != original.Type {
				t.Errorf("error type not preserved: expected %q, got %q", original.Type, roundTrip.Type)
			}
			if roundTrip.PublicMessage != original.PublicMessage {
				t.Errorf("public message not preserved: expected %q, got %q",
					original.PublicMessage, roundTrip.PublicMessage)
			}
			if roundTrip.Param != original.Param {
				t.Errorf("param not preserved: expected %q, got %q", original.Param, roundTrip.Param)
			}
			if roundTrip.DocURL != original.DocURL {
				t.Errorf("doc_url not preserved: expected %q, got %q", original.DocURL, roundTrip.DocURL)
			}
			if roundTrip.InternalMessage != original.InternalMessage {
				t.Errorf("internal message not preserved: expected %q, got %q",
					original.InternalMessage, roundTrip.InternalMessage)
			}
			// Internal error is converted to string, so compare string representations
			if original.Internal != nil {
				if roundTrip.Internal == nil {
					t.Errorf("internal error not preserved: expected %q, got nil", original.Internal.Error())
				} else if original.Internal.Error() != roundTrip.Internal.Error() {
					t.Errorf("internal error not preserved: expected %q, got %q",
						original.Internal.Error(), roundTrip.Internal.Error())
				}
			} else if roundTrip.Internal != nil {
				t.Errorf("unexpected internal error: got %q", roundTrip.Internal.Error())
			}

			// Verify gRPC codes remain consistent
			grpcSt, _ := grpcstatus.FromError(grpcErr)
			roundTripGrpcErr := ConvertAPIErrorToGRPC(roundTrip)
			roundTripSt, _ := grpcstatus.FromError(roundTripGrpcErr)

			if grpcSt.Code() != roundTripSt.Code() {
				t.Errorf("gRPC code changed during round-trip: original %v, round-trip %v",
					grpcSt.Code(), roundTripSt.Code())
			}
		})
	}
}
