package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
)

func TestRespondWithAPIError_NilErrorPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when apiErr is nil")
		}
		expectedMsg := "RespondWithAPIError: apiErr received is nil."
		if !strings.Contains(r.(string), expectedMsg) {
			t.Fatalf("expected panic message to contain %q, got %q", expectedMsg, r)
		}
	}()

	ctx := context.Background()
	w := httptest.NewRecorder()
	RespondWithAPIError(ctx, w, nil)
}

func TestRespondWithAPIError_WithoutRequestLog(t *testing.T) {
	apiErr := apierror.NewValidationError("Invalid input")
	ctx := context.Background()
	w := httptest.NewRecorder()

	RespondWithAPIError(ctx, w, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errorMap, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatal("expected response to have 'error' key with map value")
	}

	if errorMap["code"] != string(apierror.ErrorCodeValidationFailed) {
		t.Fatalf("expected error code %q, got %q", apierror.ErrorCodeValidationFailed, errorMap["code"])
	}

	if errorMap["message"] != "Invalid input" {
		t.Fatalf("expected error message %q, got %q", "Invalid input", errorMap["message"])
	}
}

func TestRespondWithAPIError_WithRequestLog_NonInternalError(t *testing.T) {
	apiErr := apierror.NewValidationError("Invalid input")
	rl := &appctx.RequestLog{
		ID: "test-request-id",
	}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	w := httptest.NewRecorder()

	RespondWithAPIError(ctx, w, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	if rl.ErrorCode == nil || *rl.ErrorCode != string(apierror.ErrorCodeValidationFailed) {
		t.Fatalf("expected ErrorCode %q, got %v", apierror.ErrorCodeValidationFailed, rl.ErrorCode)
	}

	if rl.ErrorMessage == nil || *rl.ErrorMessage != "Invalid input" {
		t.Fatalf("expected ErrorMessage %q, got %v", "Invalid input", rl.ErrorMessage)
	}

	if rl.InternalErrorMessage != nil && *rl.InternalErrorMessage != "" {
		t.Fatalf("expected InternalErrorMessage to be empty for non-internal error, got %v", rl.InternalErrorMessage)
	}

	if rl.StackTrace != nil && *rl.StackTrace != "" {
		t.Fatalf("expected StackTrace to be empty for non-internal error, got %v", rl.StackTrace)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errorMap, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatal("expected response to have 'error' key with map value")
	}

	if errorMap["code"] != string(apierror.ErrorCodeValidationFailed) {
		t.Fatalf("expected error code %q, got %q", apierror.ErrorCodeValidationFailed, errorMap["code"])
	}
}

func TestRespondWithAPIError_WithRequestLog_InternalError(t *testing.T) {
	internalErr := apierror.NewInvariantViolationError("Database connection failed.")
	rl := &appctx.RequestLog{
		ID: "test-request-id",
	}
	ctx := appctx.WithRequestLog(context.Background(), rl)
	w := httptest.NewRecorder()

	RespondWithAPIError(ctx, w, internalErr)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if rl.ErrorCode == nil || *rl.ErrorCode != string(apierror.ErrorCodeInternalError) {
		t.Fatalf("expected ErrorCode %q, got %v", apierror.ErrorCodeInternalError, rl.ErrorCode)
	}

	if rl.ErrorMessage == nil || *rl.ErrorMessage != "Something went wrong." {
		t.Fatalf("expected ErrorMessage %q, got %v", "Something went wrong.", rl.ErrorMessage)
	}

	if rl.InternalErrorMessage == nil || *rl.InternalErrorMessage != "Database connection failed." {
		t.Fatalf("expected InternalErrorMessage %q, got %v", "Database connection failed.", rl.InternalErrorMessage)
	}

	if rl.StackTrace == nil || *rl.StackTrace == "" {
		t.Fatal("expected StackTrace to be set for internal error")
	}

	if !strings.Contains(*rl.StackTrace, "RespondWithAPIError") {
		t.Fatal("expected StackTrace to contain function name")
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errorMap, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatal("expected response to have 'error' key with map value")
	}

	if errorMap["code"] != string(apierror.ErrorCodeInternalError) {
		t.Fatalf("expected error code %q, got %q", apierror.ErrorCodeInternalError, errorMap["code"])
	}
}

func TestRespondWithAPIError_DifferentErrorCodes(t *testing.T) {
	testCases := []struct {
		name           string
		apiErr         *apierror.APIError
		expectedStatus int
	}{
		{
			name:           "Unauthorized",
			apiErr:         apierror.NewAuthenticationError("Invalid credentials"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Forbidden",
			apiErr:         apierror.NewAuthorizationError("Insufficient permissions"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "NotFound",
			apiErr:         apierror.NewResourceNotFoundError("Resource not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Conflict",
			apiErr:         apierror.NewResourceConflictError("Resource conflict"),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "TooManyRequests",
			apiErr:         apierror.NewRateLimitExceededError("Rate limit exceeded"),
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "MethodNotAllowed",
			apiErr:         apierror.NewMethodNotAllowedError("Method not allowed"),
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			w := httptest.NewRecorder()

			RespondWithAPIError(ctx, w, tc.apiErr)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			var response map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			errorMap, ok := response["error"].(map[string]any)
			if !ok {
				t.Fatal("expected response to have 'error' key with map value")
			}

			if errorMap["code"] == nil {
				t.Fatal("expected error code to be present")
			}
		})
	}
}
