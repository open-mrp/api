package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

// TestConvertAPIErrorToGRPC_AllErrorCodes tests that every API error code
// can be converted to a gRPC error without panicking.
func TestConvertAPIErrorToGRPC_AllErrorCodes(t *testing.T) {
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
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeExpiredToken, apierror.ErrorTypeInvalidRequest, "Token expired", ""),
			wantCode: grpccodes.Unauthenticated,
		},
		{
			name:     "ExpiredAPIKey",
			apiErr:   apierror.NewExpiredAPIKeyError("API key expired"),
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
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeMissingField, apierror.ErrorTypeInvalidRequest, "Missing field", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "InvalidFormat",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeInvalidFormat, apierror.ErrorTypeInvalidRequest, "Invalid format", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterMissing",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeParameterMissing, apierror.ErrorTypeInvalidRequest, "Parameter missing", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterInvalid",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeParameterInvalid, apierror.ErrorTypeInvalidRequest, "Parameter invalid", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParameterUnknown",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeParameterUnknown, apierror.ErrorTypeInvalidRequest, "Parameter unknown", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		{
			name:     "ParametersExclusive",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeParametersExclusive, apierror.ErrorTypeInvalidRequest, "Parameters exclusive", ""),
			wantCode: grpccodes.InvalidArgument,
		},
		// Resource errors
		{
			name:     "ResourceNotFound",
			apiErr:   apierror.NewResourceNotFoundError("Resource not found"),
			wantCode: grpccodes.NotFound,
		},
		{
			name:     "ResourceExists",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeResourceExists, apierror.ErrorTypeInvalidRequest, "Resource exists", ""),
			wantCode: grpccodes.AlreadyExists,
		},
		{
			name:     "ResourceConflict",
			apiErr:   apierror.NewResourceConflictError("Resource conflict"),
			wantCode: grpccodes.Aborted,
		},
		// Business logic errors
		{
			name:     "RateLimitExceeded",
			apiErr:   apierror.NewRateLimitExceededError("Rate limit exceeded"),
			wantCode: grpccodes.ResourceExhausted,
		},
		// Server errors
		{
			name:     "SvcUnavailable",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeSvcUnavailable, apierror.ErrorTypeAPI, "Service unavailable", ""),
			wantCode: grpccodes.Unavailable,
		},
		{
			name:     "RequestTimeout",
			apiErr:   apierror.NewRequestTimeoutError("Request timed out"),
			wantCode: grpccodes.DeadlineExceeded,
		},
		{
			name:     "Timeout",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeTimeout, apierror.ErrorTypeAPI, "Timeout", ""),
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
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeExternalSvcError, apierror.ErrorTypeAPI, "External service error", ""),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ConnectionError",
			apiErr:   apierror.NewAPIError(apierror.ErrorCodeConnectionError, apierror.ErrorTypeAPI, "Connection error", ""),
			wantCode: grpccodes.Internal,
		},
		{
			name:     "ClientClosedRequest",
			apiErr:   apierror.NewClientClosedRequestError("Client closed request"),
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
				Code            apierror.ErrorCode `json:"code"`
				Type            apierror.ErrorType `json:"type"`
				PublicMessage   string             `json:"message"`
				Param           string             `json:"param,omitempty"`
				DocURL          string             `json:"doc_url,omitempty"`
				InternalMessage string             `json:"internal_message,omitempty"`
				InternalError   string             `json:"internal_error,omitempty"`
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
		apiErr            *apierror.APIError
		expectedRoundTrip apierror.ErrorCode // The error code we expect after round-trip
		expectExactMatch  bool               // Whether we expect exact code match
	}{
		// These should map exactly
		{
			name:              "InvalidCredentials",
			apiErr:            apierror.NewAuthenticationError("Invalid credentials"),
			expectedRoundTrip: apierror.ErrorCodeInvalidCredentials,
			expectExactMatch:  true,
		},
		{
			name:              "InsufficientPerms",
			apiErr:            apierror.NewAuthorizationError("Insufficient permissions"),
			expectedRoundTrip: apierror.ErrorCodeInsufficientPerms,
			expectExactMatch:  true,
		},
		{
			name:              "ValidationFailed",
			apiErr:            apierror.NewValidationError("Validation failed"),
			expectedRoundTrip: apierror.ErrorCodeValidationFailed,
			expectExactMatch:  true,
		},
		{
			name:              "ResourceNotFound",
			apiErr:            apierror.NewResourceNotFoundError("Resource not found"),
			expectedRoundTrip: apierror.ErrorCodeResourceNotFound,
			expectExactMatch:  true,
		},
		{
			name:              "ResourceConflict",
			apiErr:            apierror.NewResourceConflictError("Resource conflict"),
			expectedRoundTrip: apierror.ErrorCodeResourceConflict,
			expectExactMatch:  true,
		},
		{
			name:              "RateLimitExceeded",
			apiErr:            apierror.NewRateLimitExceededError("Rate limit exceeded"),
			expectedRoundTrip: apierror.ErrorCodeRateLimitExceeded,
			expectExactMatch:  true,
		},
		{
			name:              "SvcUnavailable",
			apiErr:            apierror.NewAPIError(apierror.ErrorCodeSvcUnavailable, apierror.ErrorTypeAPI, "Service unavailable", "internal msg"),
			expectedRoundTrip: apierror.ErrorCodeSvcUnavailable,
			expectExactMatch:  true, // Now with zero information loss
		},
		{
			name:              "RequestTimeout",
			apiErr:            apierror.NewRequestTimeoutError("Request timed out"),
			expectedRoundTrip: apierror.ErrorCodeRequestTimeout,
			expectExactMatch:  true,
		},
		// With zero information loss, all should match exactly
		{
			name:              "ExpiredToken",
			apiErr:            apierror.NewAPIError(apierror.ErrorCodeExpiredToken, apierror.ErrorTypeInvalidRequest, "Token expired", ""),
			expectedRoundTrip: apierror.ErrorCodeExpiredToken,
			expectExactMatch:  true,
		},
		{
			name:              "ExpiredAPIKey",
			apiErr:            apierror.NewExpiredAPIKeyError("API key expired"),
			expectedRoundTrip: apierror.ErrorCodeExpiredAPIKey,
			expectExactMatch:  true,
		},
		{
			name:              "MissingField",
			apiErr:            apierror.NewAPIError(apierror.ErrorCodeMissingField, apierror.ErrorTypeInvalidRequest, "Missing field", ""),
			expectedRoundTrip: apierror.ErrorCodeMissingField,
			expectExactMatch:  true,
		},
		{
			name:              "MethodNotAllowed",
			apiErr:            apierror.NewMethodNotAllowedError("Method not allowed"),
			expectedRoundTrip: apierror.ErrorCodeMethodNotAllowed,
			expectExactMatch:  true,
		},
		{
			name:              "InternalError",
			apiErr:            apierror.NewInternalError(errors.New("test"), "Internal error"),
			expectedRoundTrip: apierror.ErrorCodeInternalError,
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
		apiErr      *apierror.APIError
		description string
	}{
		{
			name:        "PublicMessage only",
			apiErr:      apierror.NewValidationError("Public message"),
			description: "PublicMessage should be preserved",
		},
		{
			name: "InternalMessage fallback",
			apiErr: apierror.NewAPIError(
				apierror.ErrorCodeValidationFailed,
				apierror.ErrorTypeInvalidRequest,
				"", // Empty public message
				"Internal message",
			),
			description: "InternalMessage should be preserved",
		},
		{
			name:        "PublicMessage preferred",
			apiErr:      apierror.NewAPIError(apierror.ErrorCodeValidationFailed, apierror.ErrorTypeInvalidRequest, "Public", "Internal"),
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
				Code            apierror.ErrorCode `json:"code"`
				Type            apierror.ErrorType `json:"type"`
				PublicMessage   string             `json:"message"`
				InternalMessage string             `json:"internal_message,omitempty"`
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
		expectedCode apierror.ErrorCode
		expectedType apierror.ErrorType
		description  string
	}{
		{
			name:         "InvalidArgument",
			grpcCode:     grpccodes.InvalidArgument,
			message:      "Invalid argument",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "NotFound",
			grpcCode:     grpccodes.NotFound,
			message:      "Not found",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "AlreadyExists",
			grpcCode:     grpccodes.AlreadyExists,
			message:      "Already exists",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "PermissionDenied",
			grpcCode:     grpccodes.PermissionDenied,
			message:      "Permission denied",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "ResourceExhausted",
			grpcCode:     grpccodes.ResourceExhausted,
			message:      "Resource exhausted",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "FailedPrecondition",
			grpcCode:     grpccodes.FailedPrecondition,
			message:      "Failed precondition",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "Aborted",
			grpcCode:     grpccodes.Aborted,
			message:      "Aborted",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "OutOfRange",
			grpcCode:     grpccodes.OutOfRange,
			message:      "Out of range",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "Unauthenticated",
			grpcCode:     grpccodes.Unauthenticated,
			message:      "Unauthenticated",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "Unavailable",
			grpcCode:     grpccodes.Unavailable,
			message:      "Unavailable",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "DeadlineExceeded",
			grpcCode:     grpccodes.DeadlineExceeded,
			message:      "Deadline exceeded",
			expectedCode: apierror.ErrorCodeRequestTimeout,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "Internal",
			grpcCode:     grpccodes.Internal,
			message:      "Internal error",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "Unimplemented",
			grpcCode:     grpccodes.Unimplemented,
			message:      "Unimplemented",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
		},
		{
			name:         "DataLoss",
			grpcCode:     grpccodes.DataLoss,
			message:      "Data loss",
			expectedCode: apierror.ErrorCodeInternalError,
			expectedType: apierror.ErrorTypeAPI,
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

		if apiErr.Code != apierror.ErrorCodeInternalError {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeInternalError, apiErr.Code)
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

		if apiErr.Code != apierror.ErrorCodeRequestTimeout {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeRequestTimeout, apiErr.Code)
		}
	})
}

// TestConvertGRPCError_ClientCancellation tests that client disconnections are
// correctly identified and mapped to client_closed_request (HTTP 499).
func TestConvertGRPCError_ClientCancellation(t *testing.T) {
	serviceName := "test-service"

	t.Run("canceled context with context.Canceled error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // simulate client disconnect

		err := ctx.Err() // context.Canceled
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.ErrorCodeClientClosedRequest {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeClientClosedRequest, apiErr.Code)
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
		if apiErr.Code != apierror.ErrorCodeClientClosedRequest {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeClientClosedRequest, apiErr.Code)
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
		if apiErr.Code != apierror.ErrorCodeClientClosedRequest {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeClientClosedRequest, apiErr.Code)
		}
	})

	t.Run("non-canceled context with gRPC DeadlineExceeded is timeout", func(t *testing.T) {
		ctx := context.Background() // not canceled

		err := grpcstatus.Error(grpccodes.DeadlineExceeded, "deadline exceeded")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.ErrorCodeRequestTimeout {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeRequestTimeout, apiErr.Code)
		}
	})

	t.Run("non-canceled context with gRPC Canceled is internal error", func(t *testing.T) {
		ctx := context.Background() // not canceled

		err := grpcstatus.Error(grpccodes.Canceled, "canceled by server")
		apiErr := ConvertGRPCError(ctx, err, serviceName)

		if apiErr == nil {
			t.Fatal("expected API error, got nil")
		}
		if apiErr.Code != apierror.ErrorCodeInternalError {
			t.Errorf("expected error code %q, got %q", apierror.ErrorCodeInternalError, apiErr.Code)
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
		code   apierror.ErrorCode
		create func() *apierror.APIError
	}{
		{name: "ExpiredToken", code: apierror.ErrorCodeExpiredToken, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeExpiredToken, apierror.ErrorTypeInvalidRequest, "Token expired", "")
		}},
		{name: "ExpiredAPIKey", code: apierror.ErrorCodeExpiredAPIKey, create: func() *apierror.APIError {
			return apierror.NewExpiredAPIKeyError("API key expired")
		}},
		{name: "InvalidCredentials", code: apierror.ErrorCodeInvalidCredentials, create: func() *apierror.APIError {
			return apierror.NewAuthenticationError("Invalid credentials")
		}},
		{name: "InsufficientPerms", code: apierror.ErrorCodeInsufficientPerms, create: func() *apierror.APIError {
			return apierror.NewAuthorizationError("Insufficient permissions")
		}},
		{name: "ValidationFailed", code: apierror.ErrorCodeValidationFailed, create: func() *apierror.APIError {
			return apierror.NewValidationError("Validation failed")
		}},
		{name: "MissingField", code: apierror.ErrorCodeMissingField, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeMissingField, apierror.ErrorTypeInvalidRequest, "Missing field", "")
		}},
		{name: "InvalidFormat", code: apierror.ErrorCodeInvalidFormat, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeInvalidFormat, apierror.ErrorTypeInvalidRequest, "Invalid format", "")
		}},
		{name: "MethodNotAllowed", code: apierror.ErrorCodeMethodNotAllowed, create: func() *apierror.APIError {
			return apierror.NewMethodNotAllowedError("Method not allowed")
		}},
		{name: "ResourceNotFound", code: apierror.ErrorCodeResourceNotFound, create: func() *apierror.APIError {
			return apierror.NewResourceNotFoundError("Resource not found")
		}},
		{name: "ResourceExists", code: apierror.ErrorCodeResourceExists, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeResourceExists, apierror.ErrorTypeInvalidRequest, "Resource exists", "")
		}},
		{name: "ResourceConflict", code: apierror.ErrorCodeResourceConflict, create: func() *apierror.APIError {
			return apierror.NewResourceConflictError("Resource conflict")
		}},
		{name: "RateLimitExceeded", code: apierror.ErrorCodeRateLimitExceeded, create: func() *apierror.APIError {
			return apierror.NewRateLimitExceededError("Rate limit exceeded")
		}},
		{name: "ParameterMissing", code: apierror.ErrorCodeParameterMissing, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeParameterMissing, apierror.ErrorTypeInvalidRequest, "Parameter missing", "")
		}},
		{name: "ParameterInvalid", code: apierror.ErrorCodeParameterInvalid, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeParameterInvalid, apierror.ErrorTypeInvalidRequest, "Parameter invalid", "")
		}},
		{name: "ParameterUnknown", code: apierror.ErrorCodeParameterUnknown, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeParameterUnknown, apierror.ErrorTypeInvalidRequest, "Parameter unknown", "")
		}},
		{name: "ParametersExclusive", code: apierror.ErrorCodeParametersExclusive, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeParametersExclusive, apierror.ErrorTypeInvalidRequest, "Parameters exclusive", "")
		}},
		{name: "InternalError", code: apierror.ErrorCodeInternalError, create: func() *apierror.APIError {
			return apierror.NewInternalError(errors.New("test"), "Internal error")
		}},
		{name: "SvcUnavailable", code: apierror.ErrorCodeSvcUnavailable, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeSvcUnavailable, apierror.ErrorTypeAPI, "Service unavailable", "internal")
		}},
		{name: "ExternalSvcError", code: apierror.ErrorCodeExternalSvcError, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeExternalSvcError, apierror.ErrorTypeAPI, "External service error", "")
		}},
		{name: "Timeout", code: apierror.ErrorCodeTimeout, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeTimeout, apierror.ErrorTypeAPI, "Timeout", "")
		}},
		{name: "ConnectionError", code: apierror.ErrorCodeConnectionError, create: func() *apierror.APIError {
			return apierror.NewAPIError(apierror.ErrorCodeConnectionError, apierror.ErrorTypeAPI, "Connection error", "")
		}},
		{name: "RequestTimeout", code: apierror.ErrorCodeRequestTimeout, create: func() *apierror.APIError {
			return apierror.NewRequestTimeoutError("Request timed out")
		}},
		{name: "ClientClosedRequest", code: apierror.ErrorCodeClientClosedRequest, create: func() *apierror.APIError {
			return apierror.NewClientClosedRequestError("Client closed request")
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

func TestSetAPIVersionInMetadata(t *testing.T) {
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

func TestAPIVersionUnaryServerInterceptor_SetsVersionInContext(t *testing.T) {
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
