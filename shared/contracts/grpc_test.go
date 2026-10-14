package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/apperr"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/querytag"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

// TestConvertAPIErrorToGRPC_AllErrorCodes tests that every API error code
// can be converted to a gRPC error without panicking.
func TestConvertAPIErrorToGRPC_AllErrorCodes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		apiErr   *apierror.APIError
		wantCode grpccodes.Code
	}{
		// Authentication errors
		{
			name:     "InvalidCredentials",
			apiErr:   apierror.NewAuthenticationError("Invalid credentials"),
			wantCode: grpccodes.Unauthenticated,
		},
		{
			name:     "ExpiredToken",
			apiErr:   apierror.New(apierror.CodeExpiredToken, "Token expired"),
			wantCode: grpccodes.Unauthenticated,
		},
		{
			name:     "ExpiredAPIKey",
			apiErr:   apperr.NewExpiredAPIKeyError("API key expired"),
			wantCode: grpccodes.Unauthenticated,
		},
		// Authorization errors
		{
			name:     "InsufficientPerms",
			apiErr:   apierror.NewAuthorizationError("Insufficient permissions"),
			wantCode: grpccodes.PermissionDenied,
		},
		// Validation errors
		{
			name:     "ValidationFailed",
			apiErr:   apierror.NewValidationError("Validation failed"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "MissingField",
			apiErr:   apierror.New(apierror.CodeMissingField, "Missing field"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "InvalidFormat",
			apiErr:   apierror.New(apierror.CodeInvalidFormat, "Invalid format"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterMissing",
			apiErr:   apierror.New(apierror.CodeParameterMissing, "Parameter missing"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterInvalid",
			apiErr:   apierror.New(apierror.CodeParameterInvalid, "Parameter invalid"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterUnknown",
			apiErr:   apierror.New(apierror.CodeParameterUnknown, "Parameter unknown"),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParametersExclusive",
			apiErr:   apierror.New(apierror.CodeParametersExclusive, "Parameters exclusive"),
			wantCode: grpccodes.InvalidArgument,
		},
		// Resource errors
		{
			name:     "ResourceNotFound",
			apiErr:   apierror.NewNotFoundError("Resource not found"),
			wantCode: grpccodes.NotFound,
		},
		{
			name:     "ResourceExists",
			apiErr:   apierror.New(apierror.CodeResourceExists, "Resource exists"),
			wantCode: grpccodes.AlreadyExists,
		},
		{
			name:     "ResourceConflict",
			apiErr:   apierror.NewConflictError("Resource conflict"),
			wantCode: grpccodes.Aborted,
		},
		// Business logic errors
		{
			name:     "RateLimitExceeded",
			apiErr:   apierror.NewRateLimitedError("Rate limit exceeded"),
			wantCode: grpccodes.ResourceExhausted,
		},
		// Server errors
		{
			name:     "SvcUnavailable",
			apiErr:   apierror.New(apierror.CodeServiceUnavailable, "Service unavailable"),
			wantCode: grpccodes.Unavailable,
		},
		{
			name:     "RequestTimeout",
			apiErr:   apierror.NewRequestTimeoutError("Request timed out"),
			wantCode: grpccodes.DeadlineExceeded,
		},
		{
			name:     "Timeout",
			apiErr:   apierror.New(apierror.CodeTimeout, "Timeout"),
			wantCode: grpccodes.DeadlineExceeded,
		},
		{
			name:     "MethodNotAllowed",
			apiErr:   apierror.NewMethodNotAllowedError("Method not allowed"),
			wantCode: grpccodes.Unimplemented,
		},
		{
			name:     "InternalError",
			apiErr:   apierror.NewInternalError(errors.New("internal error"), "Something went wrong"),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ExternalSvcError",
			apiErr:   apierror.New(apierror.CodeExternalServiceError, "External service error"),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ConnectionError",
			apiErr:   apierror.New(apierror.CodeConnectionError, "Connection error"),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ClientClosedRequest",
			apiErr:   apierror.NewClientClosedRequestError(),
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

			// Parse and verify all fields are preserved
			message := st.Message()
			jsonData, ok := strings.CutPrefix(message, "__API_ERROR__:")
			if !ok {
				t.Errorf("expected message to start with __API_ERROR__:, got %q", message)
			}

			// Parse and verify all fields are preserved
			var decoded struct {
				Code            apierror.Code `json:"code"`
				Type            apierror.Type `json:"type"`
				PublicMessage   string        `json:"message"`
				Param           string        `json:"param,omitempty"`
				InternalMessage string        `json:"internal_message,omitempty"`
				InternalError   string        `json:"internal_error,omitempty"`
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
			if decoded.InternalMessage != tc.apiErr.InternalMessage {
				t.Errorf("expected internal message %q, got %q", tc.apiErr.InternalMessage, decoded.InternalMessage)
			}
		})
	}
}

// TestConvertAPIErrorToGRPC_RoundTrip tests that converting an API error to gRPC
// and back preserves the error code mapping (even if not exact due to many-to-one mapping).
func TestConvertAPIErrorToGRPC_RoundTrip(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		apiErr            *apierror.APIError
		expectedRoundTrip apierror.Code // The error code we expect after round-trip
		expectExactMatch  bool          // Whether we expect exact code match
	}{
		// These should map exactly
		{
			name:              "InvalidCredentials",
			apiErr:            apierror.NewAuthenticationError("Invalid credentials"),
			expectedRoundTrip: apierror.CodeInvalidCredentials,
			expectExactMatch:  true,
		},
		{
			name:              "InsufficientPerms",
			apiErr:            apierror.NewAuthorizationError("Insufficient permissions"),
			expectedRoundTrip: apierror.CodeInsufficientPermissions,
			expectExactMatch:  true,
		},
		{
			name:              "ValidationFailed",
			apiErr:            apierror.NewValidationError("Validation failed"),
			expectedRoundTrip: apierror.CodeValidationFailed,
			expectExactMatch:  true,
		},
		{
			name:              "ResourceNotFound",
			apiErr:            apierror.NewNotFoundError("Resource not found"),
			expectedRoundTrip: apierror.CodeResourceNotFound,
			expectExactMatch:  true,
		},
		{
			name:              "ResourceConflict",
			apiErr:            apierror.NewConflictError("Resource conflict"),
			expectedRoundTrip: apierror.CodeResourceConflict,
			expectExactMatch:  true,
		},
		{
			name:              "RateLimitExceeded",
			apiErr:            apierror.NewRateLimitedError("Rate limit exceeded"),
			expectedRoundTrip: apierror.CodeRateLimited,
			expectExactMatch:  true,
		},
		{
			name:              "SvcUnavailable",
			apiErr:            apierror.New(apierror.CodeServiceUnavailable, "Service unavailable", apierror.WithInternalMessage("internal msg")),
			expectedRoundTrip: apierror.CodeServiceUnavailable,
			expectExactMatch:  true, // Now with zero information loss
		},
		{
			name:              "RequestTimeout",
			apiErr:            apierror.NewRequestTimeoutError("Request timed out"),
			expectedRoundTrip: apierror.CodeRequestTimeout,
			expectExactMatch:  true,
		},
		// With zero information loss, all should match exactly
		{
			name:              "ExpiredToken",
			apiErr:            apierror.New(apierror.CodeExpiredToken, "Token expired"),
			expectedRoundTrip: apierror.CodeExpiredToken,
			expectExactMatch:  true,
		},
		{
			name:              "ExpiredAPIKey",
			apiErr:            apperr.NewExpiredAPIKeyError("API key expired"),
			expectedRoundTrip: apperr.CodeAPIKeyExpired,
			expectExactMatch:  true,
		},
		{
			name:              "MissingField",
			apiErr:            apierror.New(apierror.CodeMissingField, "Missing field"),
			expectedRoundTrip: apierror.CodeMissingField,
			expectExactMatch:  true,
		},
		{
			name:              "MethodNotAllowed",
			apiErr:            apierror.NewMethodNotAllowedError("Method not allowed"),
			expectedRoundTrip: apierror.CodeMethodNotAllowed,
			expectExactMatch:  true,
		},
		{
			name:              "InternalError",
			apiErr:            apierror.NewInternalError(errors.New("test"), "Internal error"),
			expectedRoundTrip: apierror.CodeInternalError,
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
	t.Parallel()
	grpcErr := ConvertAPIErrorToGRPC(nil)
	if grpcErr != nil {
		t.Errorf("expected nil, got %v", grpcErr)
	}
}

// TestConvertAPIErrorToGRPC_MessagePreservation tests that all fields are preserved correctly.
func TestConvertAPIErrorToGRPC_MessagePreservation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		apiErr      *apierror.APIError
		description string
	}{
		{
			name:        "PublicMessage only",
			apiErr:      apierror.NewValidationError("Public message"),
			description: "PublicMessage should be preserved",
		},
		{
			name:        "InternalMessage fallback",
			apiErr:      apierror.New(apierror.CodeValidationFailed, "", apierror.WithInternalMessage("Internal message")),
			description: "InternalMessage should be preserved",
		},
		{
			name:        "PublicMessage preferred",
			apiErr:      apierror.New(apierror.CodeValidationFailed, "Public", apierror.WithInternalMessage("Internal")),
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
				Code            apierror.Code `json:"code"`
				Type            apierror.Type `json:"type"`
				PublicMessage   string        `json:"message"`
				InternalMessage string        `json:"internal_message,omitempty"`
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
	t.Parallel()
	testCases := []struct {
		name         string
		grpcCode     grpccodes.Code
		message      string
		expectedCode apierror.Code
		expectedType apierror.Type
		description  string
	}{
		{
			name:         "InvalidArgument",
			grpcCode:     grpccodes.InvalidArgument,
			message:      "Invalid argument",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "NotFound",
			grpcCode:     grpccodes.NotFound,
			message:      "Not found",
			expectedCode: apierror.CodeResourceNotFound,
			expectedType: apierror.TypeInvalidRequest,
		},
		{
			name:         "AlreadyExists",
			grpcCode:     grpccodes.AlreadyExists,
			message:      "Already exists",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "PermissionDenied",
			grpcCode:     grpccodes.PermissionDenied,
			message:      "Permission denied",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "ResourceExhausted",
			grpcCode:     grpccodes.ResourceExhausted,
			message:      "Resource exhausted",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "FailedPrecondition",
			grpcCode:     grpccodes.FailedPrecondition,
			message:      "Failed precondition",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "Aborted",
			grpcCode:     grpccodes.Aborted,
			message:      "Aborted",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "OutOfRange",
			grpcCode:     grpccodes.OutOfRange,
			message:      "Out of range",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "Unauthenticated",
			grpcCode:     grpccodes.Unauthenticated,
			message:      "Unauthenticated",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "Unavailable",
			grpcCode:     grpccodes.Unavailable,
			message:      "Unavailable",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "DeadlineExceeded",
			grpcCode:     grpccodes.DeadlineExceeded,
			message:      "Deadline exceeded",
			expectedCode: apierror.CodeRequestTimeout,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "Internal",
			grpcCode:     grpccodes.Internal,
			message:      "Internal error",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "Unimplemented",
			grpcCode:     grpccodes.Unimplemented,
			message:      "Unimplemented",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
		},
		{
			name:         "DataLoss",
			grpcCode:     grpccodes.DataLoss,
			message:      "Data loss",
			expectedCode: apierror.CodeInternalError,
			expectedType: apierror.TypeAPI,
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
	t.Parallel()
	ctx := context.Background()
	serviceName := "test-service"

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}

		if apiErr.Code != apierror.CodeInternalError {
			t.Errorf("expected error code %q, got %q", apierror.CodeInternalError, apiErr.Code)
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

		if apiErr.Code != apierror.CodeRequestTimeout {
			t.Errorf("expected error code %q, got %q", apierror.CodeRequestTimeout, apiErr.Code)
		}
	})
}

// TestConvertGRPCError_ClientCancellation tests that client disconnections are
// correctly identified and mapped to client_closed_request (HTTP 499).
func TestConvertGRPCError_ClientCancellation(t *testing.T) {
	t.Parallel()
	serviceName := "test-service"

	t.Run("canceled context with context.Canceled error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // simulate client disconnect

		err := ctx.Err() // context.Canceled
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.CodeClientClosedRequest {
			t.Errorf("expected error code %q, got %q", apierror.CodeClientClosedRequest, apiErr.Code)
		}
	})

	t.Run("canceled context with gRPC Canceled status", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // simulate client disconnect

		err := grpcstatus.Error(grpccodes.Canceled, "context canceled")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.CodeClientClosedRequest {
			t.Errorf("expected error code %q, got %q", apierror.CodeClientClosedRequest, apiErr.Code)
		}
	})

	t.Run("canceled context with gRPC DeadlineExceeded status", func(t *testing.T) {
		// If the parent (HTTP) context is canceled but the gRPC error is
		// DeadlineExceeded, the root cause is the client disconnecting.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := grpcstatus.Error(grpccodes.DeadlineExceeded, "deadline exceeded")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.CodeClientClosedRequest {
			t.Errorf("expected error code %q, got %q", apierror.CodeClientClosedRequest, apiErr.Code)
		}
	})

	t.Run("non-canceled context with gRPC DeadlineExceeded is timeout", func(t *testing.T) {
		ctx := context.Background() // not canceled

		err := grpcstatus.Error(grpccodes.DeadlineExceeded, "deadline exceeded")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.CodeRequestTimeout {
			t.Errorf("expected error code %q, got %q", apierror.CodeRequestTimeout, apiErr.Code)
		}
	})

	t.Run("non-canceled context with gRPC Canceled is internal error", func(t *testing.T) {
		ctx := context.Background() // not canceled

		err := grpcstatus.Error(grpccodes.Canceled, "canceled by server")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.CodeInternalError {
			t.Errorf("expected error code %q, got %q", apierror.CodeInternalError, apiErr.Code)
		}
	})
}

// TestConvertGRPCError_Nil tests nil handling.
func TestConvertGRPCError_Nil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	apiErr := ConvertGRPCError(ctx, nil, "test-service")
	if apiErr != nil {
		t.Errorf("expected nil, got %v", apiErr)
	}
}

// TestRoundTrip_AllAPIErrorCodes tests that every single API error code
// can survive a round-trip conversion without losing critical information.
func TestRoundTrip_AllAPIErrorCodes(t *testing.T) {
	t.Parallel()
	allErrorCodes := []struct {
		name   string
		code   apierror.Code
		create func() *apierror.APIError
	}{
		{name: "ExpiredToken", code: apierror.CodeExpiredToken, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeExpiredToken, "Token expired")
		}},
		{name: "ExpiredAPIKey", code: apperr.CodeAPIKeyExpired, create: func() *apierror.APIError {
			return apperr.NewExpiredAPIKeyError("API key expired")
		}},
		{name: "InvalidCredentials", code: apierror.CodeInvalidCredentials, create: func() *apierror.APIError {
			return apierror.NewAuthenticationError("Invalid credentials")
		}},
		{name: "InsufficientPerms", code: apierror.CodeInsufficientPermissions, create: func() *apierror.APIError {
			return apierror.NewAuthorizationError("Insufficient permissions")
		}},
		{name: "ValidationFailed", code: apierror.CodeValidationFailed, create: func() *apierror.APIError {
			return apierror.NewValidationError("Validation failed")
		}},
		{name: "MissingField", code: apierror.CodeMissingField, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeMissingField, "Missing field")
		}},
		{name: "InvalidFormat", code: apierror.CodeInvalidFormat, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeInvalidFormat, "Invalid format")
		}},
		{name: "MethodNotAllowed", code: apierror.CodeMethodNotAllowed, create: func() *apierror.APIError {
			return apierror.NewMethodNotAllowedError("Method not allowed")
		}},
		{name: "RequestTooLarge", code: apierror.CodeRequestTooLarge, create: func() *apierror.APIError {
			return apierror.NewRequestTooLargeError("Request too large")
		}},
		{name: "ResourceNotFound", code: apierror.CodeResourceNotFound, create: func() *apierror.APIError {
			return apierror.NewNotFoundError("Resource not found")
		}},
		{name: "ResourceExists", code: apierror.CodeResourceExists, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeResourceExists, "Resource exists")
		}},
		{name: "ResourceConflict", code: apierror.CodeResourceConflict, create: func() *apierror.APIError {
			return apierror.NewConflictError("Resource conflict")
		}},
		{name: "RateLimitExceeded", code: apierror.CodeRateLimited, create: func() *apierror.APIError {
			return apierror.NewRateLimitedError("Rate limit exceeded")
		}},
		{name: "ParameterMissing", code: apierror.CodeParameterMissing, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeParameterMissing, "Parameter missing")
		}},
		{name: "ParameterInvalid", code: apierror.CodeParameterInvalid, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeParameterInvalid, "Parameter invalid")
		}},
		{name: "ParameterUnknown", code: apierror.CodeParameterUnknown, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeParameterUnknown, "Parameter unknown")
		}},
		{name: "ParametersExclusive", code: apierror.CodeParametersExclusive, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeParametersExclusive, "Parameters exclusive")
		}},
		{name: "InternalError", code: apierror.CodeInternalError, create: func() *apierror.APIError {
			return apierror.NewInternalError(errors.New("test"), "Internal error")
		}},
		{name: "SvcUnavailable", code: apierror.CodeServiceUnavailable, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeServiceUnavailable, "Service unavailable", apierror.WithInternalMessage("internal"))
		}},
		{name: "ExternalSvcError", code: apierror.CodeExternalServiceError, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeExternalServiceError, "External service error")
		}},
		{name: "Timeout", code: apierror.CodeTimeout, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeTimeout, "Timeout")
		}},
		{name: "ConnectionError", code: apierror.CodeConnectionError, create: func() *apierror.APIError {
			return apierror.New(apierror.CodeConnectionError, "Connection error")
		}},
		{name: "RequestTimeout", code: apierror.CodeRequestTimeout, create: func() *apierror.APIError {
			return apierror.NewRequestTimeoutError("Request timed out")
		}},
		{name: "ClientClosedRequest", code: apierror.CodeClientClosedRequest, create: func() *apierror.APIError {
			return apierror.NewClientClosedRequestError()
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

func TestSetAPIVersionInMetadata(t *testing.T) {
	t.Parallel()
	md := metadata.New(nil)
	SetAPIVersionInMetadata(md, "2026-02-01")

	values := md.Get(APIVersionHeader)
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if values[0] != "2026-02-01" {
		t.Errorf("expected version '2026-02-01', got '%s'", values[0])
	}
}

func TestGetAPIVersionFromMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		md       metadata.MD
		expected string
	}{
		{
			name:     "version present",
			md:       metadata.Pairs(APIVersionHeader, "2026-02-01"),
			expected: "2026-02-01",
		},
		{
			name:     "version absent",
			md:       metadata.New(nil),
			expected: "",
		},
		{
			name:     "empty metadata",
			md:       nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAPIVersionFromMetadata(tt.md)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetClientIPFromMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		md       metadata.MD
		expected string
	}{
		{
			name:     "ip present",
			md:       metadata.Pairs(ClientIPHeader, "198.51.100.7"),
			expected: "198.51.100.7",
		},
		{
			name:     "absent",
			md:       metadata.New(nil),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetClientIPFromMetadata(tt.md); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestClientIPUnaryServerInterceptor_SetsIPInContext(t *testing.T) {
	t.Parallel()
	interceptor := ClientIPUnaryServerInterceptor()

	md := metadata.Pairs(ClientIPHeader, "198.51.100.7")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var handlerCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCtx = ctx
		return nil, nil
	}

	_, _ = interceptor(ctx, nil, nil, handler)

	ip, ok := appctx.GetPropagatedClientIP(handlerCtx)
	if !ok || ip != "198.51.100.7" {
		t.Errorf("expected propagated IP 198.51.100.7, ok=%v ip=%q", ok, ip)
	}
}

func TestAPIVersionUnaryServerInterceptor_SetsVersionInContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		version     string
		expectInCtx bool
	}{
		{
			name:        "valid version",
			version:     "1.0.forge-preview.1",
			expectInCtx: true,
		},
		{
			name:        "empty version",
			version:     "",
			expectInCtx: false,
		},
		{
			name:        "invalid version",
			version:     "not-a-version",
			expectInCtx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := APIVersionUnaryServerInterceptor()

			md := metadata.New(nil)
			if tt.version != "" {
				md.Set(APIVersionHeader, tt.version)
			}
			ctx := metadata.NewIncomingContext(context.Background(), md)

			var handlerCtx context.Context
			handler := func(ctx context.Context, req any) (any, error) {
				handlerCtx = ctx
				return nil, nil
			}

			_, _ = interceptor(ctx, nil, nil, handler)

			v, ok := appctx.GetAPIVersionFromContext(handlerCtx)
			if tt.expectInCtx {
				if !ok {
					t.Error("expected version in context")
				}
				if v.String() != tt.version {
					t.Errorf("expected version '%s', got '%s'", tt.version, v.String())
				}
			} else {
				if ok {
					t.Error("expected no version in context")
				}
			}
		})
	}
}

func TestQueryTagUnaryServerInterceptorTagsTheMethod(t *testing.T) {
	var got map[string]string
	_, err := QueryTagUnaryServerInterceptor()(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/core.CoreService/AnalyzeSalesSummary"},
		func(ctx context.Context, _ any) (any, error) {
			got = querytag.From(ctx)
			return nil, nil
		})
	if err != nil || got[querytag.GRPCMethod] != "/core.CoreService/AnalyzeSalesSummary" {
		t.Fatalf("tags = %v, err = %v; want the method", got, err)
	}
}
