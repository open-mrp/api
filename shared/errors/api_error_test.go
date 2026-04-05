package apierror

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAPIErrorResponse_SchemaExample(t *testing.T) {
	t.Parallel()
	resp := APIErrorResponse{}
	example := resp.SchemaExample()

	if example == nil {
		t.Fatal("SchemaExample returned nil")
	}

	errResp, ok := example.(APIErrorResponse)
	if !ok {
		t.Fatalf("expected APIErrorResponse, got %T", example)
	}

	if errResp.Error.Code == "" {
		t.Error("expected non-empty error code in example")
	}
}

func TestResponseError_SchemaExample(t *testing.T) {
	t.Parallel()
	resp := ResponseError{}
	example := resp.SchemaExample()

	if example == nil {
		t.Fatal("SchemaExample returned nil")
	}

	errResp, ok := example.(ResponseError)
	if !ok {
		t.Fatalf("expected ResponseError, got %T", example)
	}

	if errResp.Code == "" {
		t.Error("expected non-empty error code in example")
	}
}

func TestAPIError_nilReceiver_implementsError(t *testing.T) {
	t.Parallel()
	var apiErr *APIError
	var err error = apiErr
	if err.Error() != "" {
		t.Errorf("nil *APIError Error() = %q, want empty string", err.Error())
	}
	if unw := errors.Unwrap(err); unw != nil {
		t.Errorf("errors.Unwrap(nil *APIError) = %v, want nil", unw)
	}
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *APIError
		expected bool
	}{
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "not found error returns true",
			err:      NewResourceNotFoundError("Resource not found"),
			expected: true,
		},
		{
			name:     "internal error returns false",
			err:      NewInternalError(nil, "internal error"),
			expected: false,
		},
		{
			name:     "validation error returns false",
			err:      NewValidationError("validation failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAPIError_ToResponseMap_Order(t *testing.T) {
	t.Parallel()
	apiErr := &APIError{
		Code:          ErrorCodeValidationFailed,
		Type:          ErrorTypeInvalidRequest,
		PublicMessage: "Invalid input",
		Param:         "email",
		DocURL:        "https://docs.example.com/api/errors",
	}

	resp := apiErr.ToResponseMap()
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	jsonStr := string(data)

	// Current implementation uses map[string]any, which sorts keys alphabetically during marshal.
	// Expected alphabetical order: code, doc_url, message, param, type
	// Desired order (as defined in ToResponseMap): code, type, message, param, doc_url

	expectedOrder := []string{
		`"code":"validation_failed"`,
		`"type":"invalid_request_error"`,
		`"message":"Invalid input"`,
		`"param":"email"`,
		`"doc_url":"https://docs.example.com/api/errors"`,
	}

	lastIdx := -1
	for _, expected := range expectedOrder {
		idx := strings.Index(jsonStr, expected)
		if idx == -1 {
			t.Errorf("expected %s to be in JSON, but not found in %s", expected, jsonStr)
			continue
		}
		if idx < lastIdx {
			t.Errorf("expected %s to appear after previous field, but it appeared before in %s", expected, jsonStr)
		}
		lastIdx = idx
	}
}
