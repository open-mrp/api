package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestNewInternalError_CapturesOriginStack(t *testing.T) {
	err := NewInternalError(errors.New("boom"), "something failed")

	if err.Stack == "" {
		t.Fatal("expected a stack to be captured for a 5xx error")
	}
	// The stack must include this test function — i.e. the origin, not just the
	// constructor — so the recorded trace points at the failing code.
	if !strings.Contains(err.Stack, "TestNewInternalError_CapturesOriginStack") {
		t.Errorf("expected captured stack to include the calling frame, got:\n%s", err.Stack)
	}
}

func TestNewValidationError_DoesNotCaptureStack(t *testing.T) {
	err := NewValidationError("bad input")

	if err.Stack != "" {
		t.Errorf("expected no stack for a 4xx error, got:\n%s", err.Stack)
	}
}

func TestNewInternalError_InheritsWrappedOriginStack(t *testing.T) {
	origin := NewInternalError(errors.New("db exploded"), "query failed")
	wrapped := NewInternalError(origin, "handler failed")

	if wrapped.Stack != origin.Stack {
		t.Error("expected outer error to inherit the wrapped error's origin stack rather than re-capture")
	}
}

func TestAPIError_Stack_SurvivesJSONRoundTrip(t *testing.T) {
	original := NewInternalError(errors.New("db exploded"), "query failed")

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	restored, err := APIErrorFromJSON(data)
	if err != nil {
		t.Fatalf("APIErrorFromJSON: %v", err)
	}

	if restored.Stack != original.Stack {
		t.Errorf("stack not preserved across gRPC serialization\nwant:\n%s\ngot:\n%s", original.Stack, restored.Stack)
	}
}

func TestGetHTTPStatusCode(t *testing.T) {
	t.Parallel()
	statuses := map[ErrorCode]int{
		ErrorCodeExpiredToken:            http.StatusUnauthorized,
		ErrorCodeExpiredAPIKey:           http.StatusUnauthorized,
		ErrorCodeRevokedAPIKey:           http.StatusUnauthorized,
		ErrorCodeInvalidCredentials:      http.StatusUnauthorized,
		ErrorCodeInsufficientPerms:       http.StatusForbidden,
		ErrorCodeLimitExceeded:           http.StatusForbidden,
		ErrorCodeRegistrationClosed:      http.StatusForbidden,
		ErrorCodePaymentRequired:         http.StatusPaymentRequired,
		ErrorCodeAgentSpendingCapReached: http.StatusPaymentRequired,
		ErrorCodeValidationFailed:        http.StatusBadRequest,
		ErrorCodeMissingField:            http.StatusBadRequest,
		ErrorCodeInvalidFormat:           http.StatusBadRequest,
		ErrorCodeParameterMissing:        http.StatusBadRequest,
		ErrorCodeParameterInvalid:        http.StatusBadRequest,
		ErrorCodeParameterUnknown:        http.StatusBadRequest,
		ErrorCodeParametersExclusive:     http.StatusBadRequest,
		ErrorCodeAPIVersionRequired:      http.StatusBadRequest,
		ErrorCodeAPIVersionInvalid:       http.StatusBadRequest,
		ErrorCodeAPIVersionTooOld:        http.StatusBadRequest,
		ErrorCodeMethodNotAllowed:        http.StatusMethodNotAllowed,
		ErrorCodeResourceNotFound:        http.StatusNotFound,
		ErrorCodeResourceGone:            http.StatusGone,
		ErrorCodeResourceExists:          http.StatusConflict,
		ErrorCodeResourceConflict:        http.StatusConflict,
		ErrorCodeIdempotencyInProgress:   http.StatusConflict,
		ErrorCodeRateLimitExceeded:       http.StatusTooManyRequests,
		ErrorCodeSvcUnavailable:          http.StatusServiceUnavailable,
		ErrorCodeExternalSvcError:        http.StatusBadGateway,
		ErrorCodeConnectionError:         http.StatusBadGateway,
		ErrorCodeTimeout:                 http.StatusGatewayTimeout,
		ErrorCodeRequestTimeout:          http.StatusGatewayTimeout,
		ErrorCodeClientClosedRequest:     499,
		ErrorCodeInternalError:           http.StatusInternalServerError,
	}

	for _, value := range ErrorCode("").EnumValues() {
		code := ErrorCode(value)
		want, ok := statuses[code]
		if !ok {
			t.Errorf("error code %q has no expected status in this test — add one", value)
			continue
		}
		if got := GetHTTPStatusCode(code); got != want {
			t.Errorf("GetHTTPStatusCode(%q) = %d, want %d", value, got, want)
		}
	}

	if len(statuses) != len(ErrorCode("").EnumValues()) {
		t.Errorf("expectation table has %d codes, EnumValues has %d", len(statuses), len(ErrorCode("").EnumValues()))
	}
}

// An APIError reconstructed from a newer service's JSON can carry a code this binary does
// not know; it must still produce a usable status rather than a zero one.
func TestGetHTTPStatusCode_UnrecognizedCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		code ErrorCode
	}{
		{name: "empty code", code: ""},
		{name: "code from a newer service", code: "quantum_flux_exceeded"},
		{name: "near miss of a known code", code: "resource_not_found "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GetHTTPStatusCode(tt.code); got != http.StatusInternalServerError {
				t.Errorf("GetHTTPStatusCode(%q) = %d, want %d", tt.code, got, http.StatusInternalServerError)
			}
		})
	}
}

func TestIs5XXErrorCode(t *testing.T) {
	t.Parallel()
	for _, value := range ErrorCode("").EnumValues() {
		code := ErrorCode(value)
		want := GetHTTPStatusCode(code) >= 500
		if got := Is5XXErrorCode(code); got != want {
			t.Errorf("Is5XXErrorCode(%q) = %v, want %v", value, got, want)
		}
	}

	// 499 is below the 5xx band, so a disconnected client must not pay for stack capture.
	if Is5XXErrorCode(ErrorCodeClientClosedRequest) {
		t.Error("Is5XXErrorCode(client_closed_request) = true, want false")
	}
	if !Is5XXErrorCode("unknown_code_from_a_newer_service") {
		t.Error("Is5XXErrorCode(unknown code) = false, want true")
	}
}

func TestIsTransientError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		code      ErrorCode
		errorType ErrorType
		expected  bool
	}{
		{name: "api internal_error is transient", code: ErrorCodeInternalError, errorType: ErrorTypeAPI, expected: true},
		{name: "api service_unavailable is transient", code: ErrorCodeSvcUnavailable, errorType: ErrorTypeAPI, expected: true},
		{name: "api external_service_error is transient", code: ErrorCodeExternalSvcError, errorType: ErrorTypeAPI, expected: true},
		{name: "api timeout is transient", code: ErrorCodeTimeout, errorType: ErrorTypeAPI, expected: true},
		{name: "api connection_error is transient", code: ErrorCodeConnectionError, errorType: ErrorTypeAPI, expected: true},
		{name: "api request_timeout is transient", code: ErrorCodeRequestTimeout, errorType: ErrorTypeAPI, expected: true},
		{name: "api client_closed_request is not transient", code: ErrorCodeClientClosedRequest, errorType: ErrorTypeAPI, expected: false},
		{name: "api rate_limit_exceeded is not transient", code: ErrorCodeRateLimitExceeded, errorType: ErrorTypeAPI, expected: false},
		{name: "api unknown code is not transient", code: "unknown_code", errorType: ErrorTypeAPI, expected: false},

		{name: "idempotency in_progress is transient", code: ErrorCodeIdempotencyInProgress, errorType: ErrorTypeIdempotency, expected: true},
		{name: "idempotency validation_failed is not transient", code: ErrorCodeValidationFailed, errorType: ErrorTypeIdempotency, expected: false},
		{name: "idempotency internal_error is not transient", code: ErrorCodeInternalError, errorType: ErrorTypeIdempotency, expected: false},

		{name: "invalid_request rate_limit_exceeded is transient", code: ErrorCodeRateLimitExceeded, errorType: ErrorTypeInvalidRequest, expected: true},
		{name: "invalid_request resource_conflict is not transient", code: ErrorCodeResourceConflict, errorType: ErrorTypeInvalidRequest, expected: false},
		{name: "invalid_request internal_error is not transient", code: ErrorCodeInternalError, errorType: ErrorTypeInvalidRequest, expected: false},
		{name: "invalid_request idempotency_in_progress is not transient", code: ErrorCodeIdempotencyInProgress, errorType: ErrorTypeInvalidRequest, expected: false},

		{name: "unrecognized type is not transient", code: ErrorCodeInternalError, errorType: "some_new_error_type", expected: false},
		{name: "empty type is not transient", code: ErrorCodeInternalError, errorType: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsTransientError(tt.code, tt.errorType); got != tt.expected {
				t.Errorf("IsTransientError(%q, %q) = %v, want %v", tt.code, tt.errorType, got, tt.expected)
			}
		})
	}
}

func TestIsTransientError_SetOnConstructedError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *APIError
		expected bool
	}{
		{name: "internal error retries", err: NewInternalError(errors.New("boom"), "failed"), expected: true},
		{name: "request timeout retries", err: NewRequestTimeoutError("deadline"), expected: true},
		{name: "rate limit retries", err: NewRateLimitExceededError("slow down"), expected: true},
		{name: "idempotency in progress retries", err: NewIdempotencyInProgressError("key_123"), expected: true},
		{name: "client closed request does not retry", err: NewClientClosedRequestError("client gone"), expected: false},
		{name: "idempotency hash mismatch does not retry", err: NewIdempotencyHashMismatchError("key_123"), expected: false},
		{name: "validation failure does not retry", err: NewValidationError("bad input"), expected: false},
		{name: "conflict does not retry", err: NewResourceConflictError("taken"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err.IsTransient != tt.expected {
				t.Errorf("IsTransient = %v, want %v", tt.err.IsTransient, tt.expected)
			}
		})
	}
}

func TestAPIError_Error_ComposedMessage(t *testing.T) {
	t.Parallel()
	inner := NewInternalError(errors.New("db exploded"), "query failed")

	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "internal message with no wrapped error",
			err:      &APIError{InternalMessage: "boom"},
			expected: "boom",
		},
		{
			name:     "wrapped error is appended after the internal message",
			err:      inner,
			expected: "query failed: db exploded",
		},
		{
			name:     "wrapped error with an empty message is not appended",
			err:      NewInvariantViolationError("expected a row after insert"),
			expected: "expected a row after insert",
		},
		{
			name:     "APIError behind a %w wrap is rendered by its own Error",
			err:      NewInternalError(fmt.Errorf("calling core: %w", inner), "handler failed"),
			expected: "handler failed: calling core: query failed: db exploded",
		},
		{
			name:     "joined errors are rendered whole",
			err:      NewInternalError(errors.Join(errors.New("first"), inner), "handler failed"),
			expected: "handler failed: first\nquery failed: db exploded",
		},
		{
			name:     "nested APIError with no internal message contributes nothing",
			err:      NewInternalError(NewValidationError("bad input"), "handler failed"),
			expected: "handler failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAPIErrorFromJSON_MalformedInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
	}{
		{name: "nil input", data: nil},
		{name: "empty input", data: []byte("")},
		{name: "truncated object", data: []byte(`{"code":"resource_not_found"`)},
		{name: "not an object", data: []byte(`"resource_not_found"`)},
		{name: "wrong field type", data: []byte(`{"code":404}`)},
		{name: "garbage", data: []byte("\x00\x01not json")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			apiErr, err := APIErrorFromJSON(tt.data)
			if err == nil {
				t.Fatalf("APIErrorFromJSON(%q) = %+v, want an error", tt.data, apiErr)
			}
			if apiErr != nil {
				t.Errorf("APIErrorFromJSON(%q) returned %+v alongside the error, want nil", tt.data, apiErr)
			}
		})
	}
}

// A well-formed payload that carries nothing usable decodes without error, so callers must
// key off the empty code rather than assume a decoded error is meaningful.
func TestAPIErrorFromJSON_EmptyPayloads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
	}{
		{name: "json null", data: []byte("null")},
		{name: "empty object", data: []byte("{}")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			apiErr, err := APIErrorFromJSON(tt.data)
			if err != nil {
				t.Fatalf("APIErrorFromJSON(%q) error = %v, want nil", tt.data, err)
			}
			if apiErr == nil {
				t.Fatalf("APIErrorFromJSON(%q) = nil, want a non-nil error", tt.data)
			}
			if apiErr.Code != "" {
				t.Errorf("Code = %q, want empty", apiErr.Code)
			}
			if apiErr.Internal != nil {
				t.Errorf("Internal = %v, want nil", apiErr.Internal)
			}
			if got := GetHTTPStatusCode(apiErr.Code); got != http.StatusInternalServerError {
				t.Errorf("GetHTTPStatusCode(%q) = %d, want %d", apiErr.Code, got, http.StatusInternalServerError)
			}
		})
	}
}

// A code minted by a newer service must survive decoding verbatim rather than be dropped
// or coerced, even though this binary cannot validate it.
func TestAPIErrorFromJSON_UnknownCode(t *testing.T) {
	t.Parallel()
	apiErr, err := APIErrorFromJSON([]byte(`{"code":"quantum_flux_exceeded","type":"reactor_error","message":"Flux capacitor overloaded.","is_transient":true}`))
	if err != nil {
		t.Fatalf("APIErrorFromJSON: %v", err)
	}
	if apiErr.Code != "quantum_flux_exceeded" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "quantum_flux_exceeded")
	}
	if apiErr.Code.IsValid() {
		t.Error("IsValid() = true for an unknown code, want false")
	}
	if apiErr.Type != "reactor_error" {
		t.Errorf("Type = %q, want %q", apiErr.Type, "reactor_error")
	}
	if apiErr.Type.IsValid() {
		t.Error("Type.IsValid() = true for an unknown type, want false")
	}
	if !apiErr.IsTransient {
		t.Error("IsTransient = false, want true — the sender's judgement is carried, not recomputed")
	}
	if apiErr.PublicMessage != "Flux capacitor overloaded." {
		t.Errorf("PublicMessage = %q, want %q", apiErr.PublicMessage, "Flux capacitor overloaded.")
	}
}

func TestAPIError_JSONRoundTrip_PreservesQuotaAndFields(t *testing.T) {
	t.Parallel()
	resetAt := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	original := NewLimitExceededError("Sandbox limit reached.").
		WithParam("plan").
		WithInternal(errors.New("account at cap")).
		WithQuota(5, 5, &resetAt)
	original.InternalMessage = "sandbox quota exhausted"

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	restored, err := APIErrorFromJSON(data)
	if err != nil {
		t.Fatalf("APIErrorFromJSON: %v", err)
	}

	if restored.Quota == nil {
		t.Fatal("Quota = nil after round trip, want the plan limit details")
	}
	if restored.Quota.Limit != 5 || restored.Quota.Used != 5 {
		t.Errorf("Quota = %+v, want limit 5 used 5", *restored.Quota)
	}
	if restored.Quota.ResetAt == nil || !restored.Quota.ResetAt.Equal(resetAt) {
		t.Errorf("Quota.ResetAt = %v, want %v", restored.Quota.ResetAt, resetAt)
	}
	if restored.Code != original.Code || restored.Type != original.Type {
		t.Errorf("code/type = %q/%q, want %q/%q", restored.Code, restored.Type, original.Code, original.Type)
	}
	if restored.Param != "plan" || restored.DocURL != original.DocURL {
		t.Errorf("param/doc_url = %q/%q, want %q/%q", restored.Param, restored.DocURL, "plan", original.DocURL)
	}
	if restored.InternalMessage != "sandbox quota exhausted" {
		t.Errorf("InternalMessage = %q, want %q", restored.InternalMessage, "sandbox quota exhausted")
	}
	// The wrapped error crosses the wire as flattened text, not as the original type.
	if restored.Internal == nil || restored.Internal.Error() != "account at cap" {
		t.Errorf("Internal = %v, want an error reading %q", restored.Internal, "account at cap")
	}
}

// Quota is absent from the wire when unset, so the receiver must not invent an empty one.
func TestAPIError_JSONRoundTrip_NilQuota(t *testing.T) {
	t.Parallel()
	data, err := NewValidationError("bad input").ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	restored, err := APIErrorFromJSON(data)
	if err != nil {
		t.Fatalf("APIErrorFromJSON: %v", err)
	}
	if restored.Quota != nil {
		t.Errorf("Quota = %+v, want nil", *restored.Quota)
	}
	if restored.Internal != nil {
		t.Errorf("Internal = %v, want nil", restored.Internal)
	}
}

// EnumValues feeds the public OpenAPI enum for ResponseError.code, so a code that exists as
// a constant but is missing here can never be represented by a generated SDK.
func TestErrorCode_EnumValuesCoversEveryConstant(t *testing.T) {
	t.Parallel()
	declared := parseErrorCodeConstants(t)
	if len(declared) == 0 {
		t.Fatal("no ErrorCode constants parsed — the parser is broken")
	}

	listed := map[string]bool{}
	for _, value := range ErrorCode("").EnumValues() {
		if listed[value] {
			t.Errorf("EnumValues() lists %q more than once", value)
		}
		listed[value] = true
	}

	for _, value := range declared {
		if !listed[value] {
			t.Errorf("ErrorCode %q is declared as a constant but missing from EnumValues()", value)
		}
	}

	declaredSet := map[string]bool{}
	for _, value := range declared {
		declaredSet[value] = true
	}
	for value := range listed {
		if !declaredSet[value] {
			t.Errorf("EnumValues() lists %q, which is not a declared ErrorCode constant", value)
		}
	}
}

func TestErrorCode_IsValid(t *testing.T) {
	t.Parallel()
	for _, value := range ErrorCode("").EnumValues() {
		if !ErrorCode(value).IsValid() {
			t.Errorf("IsValid() = false for EnumValues() entry %q", value)
		}
	}

	for _, value := range parseErrorCodeConstants(t) {
		if !ErrorCode(value).IsValid() {
			t.Errorf("IsValid() = false for declared constant %q", value)
		}
	}

	invalid := []ErrorCode{"", "quantum_flux_exceeded", "Validation_Failed", "validation_failed "}
	for _, code := range invalid {
		if code.IsValid() {
			t.Errorf("IsValid() = true for %q, want false", code)
		}
	}
}

// parseErrorCodeConstants reads the string values of every `ErrorCode` constant declared in
// the package source, so the enum plumbing is checked against the declarations themselves
// rather than against a hand-copied list.
func parseErrorCodeConstants(t *testing.T) []string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	fset := token.NewFileSet()
	var values []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || valueSpec.Type == nil {
					continue
				}
				ident, ok := valueSpec.Type.(*ast.Ident)
				if !ok || ident.Name != "ErrorCode" {
					continue
				}
				for _, value := range valueSpec.Values {
					lit, ok := value.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						t.Fatalf("ErrorCode constant in %s is not a string literal", entry.Name())
					}
					unquoted, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatalf("failed to unquote %s: %v", lit.Value, err)
					}
					values = append(values, unquoted)
				}
			}
		}
	}
	return values
}

func TestNewAPIError_StackCaptureByStatusClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		code      ErrorCode
		errorType ErrorType
		wantStack bool
	}{
		{name: "internal_error captures", code: ErrorCodeInternalError, errorType: ErrorTypeAPI, wantStack: true},
		{name: "service_unavailable captures", code: ErrorCodeSvcUnavailable, errorType: ErrorTypeAPI, wantStack: true},
		{name: "external_service_error captures", code: ErrorCodeExternalSvcError, errorType: ErrorTypeAPI, wantStack: true},
		{name: "connection_error captures", code: ErrorCodeConnectionError, errorType: ErrorTypeAPI, wantStack: true},
		{name: "timeout captures", code: ErrorCodeTimeout, errorType: ErrorTypeAPI, wantStack: true},
		{name: "request_timeout captures", code: ErrorCodeRequestTimeout, errorType: ErrorTypeAPI, wantStack: true},
		{name: "unknown code captures because it maps to 500", code: "quantum_flux_exceeded", errorType: ErrorTypeAPI, wantStack: true},
		{name: "client_closed_request skips", code: ErrorCodeClientClosedRequest, errorType: ErrorTypeAPI, wantStack: false},
		{name: "rate_limit_exceeded skips", code: ErrorCodeRateLimitExceeded, errorType: ErrorTypeInvalidRequest, wantStack: false},
		{name: "resource_not_found skips", code: ErrorCodeResourceNotFound, errorType: ErrorTypeInvalidRequest, wantStack: false},
		{name: "insufficient_permissions skips", code: ErrorCodeInsufficientPerms, errorType: ErrorTypeInvalidRequest, wantStack: false},
		{name: "idempotency_in_progress skips", code: ErrorCodeIdempotencyInProgress, errorType: ErrorTypeIdempotency, wantStack: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewAPIError(tt.code, tt.errorType, "public", "internal")
			if tt.wantStack && err.Stack == "" {
				t.Errorf("no stack captured for %q, want one", tt.code)
			}
			if !tt.wantStack && err.Stack != "" {
				t.Errorf("stack captured for %q, want none:\n%s", tt.code, err.Stack)
			}
		})
	}
}

// A 5xx wrapping a 4xx has no origin stack to inherit, so it must capture its own.
func TestNewAPIError_CapturesOwnStackWhenWrappedErrorHasNone(t *testing.T) {
	t.Parallel()
	err := NewInternalError(NewResourceNotFoundError("missing"), "lookup failed")
	if err.Stack == "" {
		t.Error("expected a stack when the wrapped error carries none")
	}
}

func TestAPIError_ToResponseError(t *testing.T) {
	t.Parallel()
	resetAt := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)

	t.Run("nil receiver yields the zero response", func(t *testing.T) {
		t.Parallel()
		var apiErr *APIError
		if got := apiErr.ToResponseError(); got != (ResponseError{}) {
			t.Errorf("ToResponseError() = %+v, want the zero value", got)
		}
	})

	t.Run("empty param and doc_url stay null", func(t *testing.T) {
		t.Parallel()
		apiErr := &APIError{
			Code:          ErrorCodeResourceNotFound,
			Type:          ErrorTypeInvalidRequest,
			PublicMessage: "Not found.",
		}
		resp := apiErr.ToResponseError()
		if resp.Param != nil {
			t.Errorf("Param = %q, want nil", *resp.Param)
		}
		if resp.DocURL != nil {
			t.Errorf("DocURL = %q, want nil", *resp.DocURL)
		}
		if resp.Quota != nil {
			t.Errorf("Quota = %+v, want nil", *resp.Quota)
		}
		if resp.RequestLogURL != nil {
			t.Errorf("RequestLogURL = %q, want nil", *resp.RequestLogURL)
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		for _, field := range []string{`"param":null`, `"doc_url":null`, `"quota":null`} {
			if !strings.Contains(string(data), field) {
				t.Errorf("expected %s in %s", field, data)
			}
		}
	})

	t.Run("populated fields are carried through", func(t *testing.T) {
		t.Parallel()
		apiErr := NewLimitExceededError("Sandbox limit reached.").
			WithParam("plan").
			WithQuota(5, 5, &resetAt)
		apiErr.IsTransient = true

		resp := apiErr.ToResponseError()
		if resp.Code != ErrorCodeLimitExceeded || resp.Type != ErrorTypeInvalidRequest {
			t.Errorf("code/type = %q/%q, want %q/%q", resp.Code, resp.Type, ErrorCodeLimitExceeded, ErrorTypeInvalidRequest)
		}
		if resp.Message != "Sandbox limit reached." {
			t.Errorf("Message = %q, want %q", resp.Message, "Sandbox limit reached.")
		}
		if resp.Param == nil || *resp.Param != "plan" {
			t.Errorf("Param = %v, want %q", resp.Param, "plan")
		}
		if resp.DocURL == nil || *resp.DocURL != docURLLimitExceeded {
			t.Errorf("DocURL = %v, want %q", resp.DocURL, docURLLimitExceeded)
		}
		if resp.Quota == nil || resp.Quota.Limit != 5 || resp.Quota.Used != 5 {
			t.Errorf("Quota = %v, want limit 5 used 5", resp.Quota)
		}
		if resp.Quota != nil && (resp.Quota.ResetAt == nil || !resp.Quota.ResetAt.Equal(resetAt)) {
			t.Errorf("Quota.ResetAt = %v, want %v", resp.Quota.ResetAt, resetAt)
		}
		if !resp.IsTransient {
			t.Error("IsTransient = false, want true")
		}
	})

	t.Run("internal fields never reach the response", func(t *testing.T) {
		t.Parallel()
		apiErr := NewInternalError(errors.New("db exploded"), "query failed")
		data, err := json.Marshal(apiErr.ToResponseError())
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		for _, leaked := range []string{"db exploded", "query failed", "goroutine"} {
			if strings.Contains(string(data), leaked) {
				t.Errorf("response leaked %q: %s", leaked, data)
			}
		}
	})
}

func TestAPIError_ToResponseMap_NilReceiver(t *testing.T) {
	t.Parallel()
	var apiErr *APIError
	resp, ok := apiErr.ToResponseMap().(APIErrorResponse)
	if !ok {
		t.Fatalf("ToResponseMap() = %T, want APIErrorResponse", apiErr.ToResponseMap())
	}
	if resp.Error != (ResponseError{}) {
		t.Errorf("Error = %+v, want the zero value", resp.Error)
	}
}

// The gateway recognises APIErrors coming back through wrapped chains; if Unwrap regresses
// those errors are recoerced into a generic 400 carrying raw internal text.
func TestAPIError_Unwrap_ErrorsIsAndAs(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sql: no rows in result set")
	apiErr := NewInternalError(sentinel, "lookup failed")

	if !errors.Is(apiErr, sentinel) {
		t.Error("errors.Is did not find the wrapped sentinel through the APIError")
	}
	if !errors.Is(fmt.Errorf("calling core: %w", apiErr), sentinel) {
		t.Error("errors.Is did not find the sentinel through a %w wrap of the APIError")
	}
	if errors.Is(NewValidationError("bad input"), sentinel) {
		t.Error("errors.Is matched a sentinel that was never wrapped")
	}

	wrapped := fmt.Errorf("service failed: %w", fmt.Errorf("calling core: %w", apiErr))
	var found *APIError
	if !errors.As(wrapped, &found) {
		t.Fatal("errors.As did not find the APIError behind two %w wraps")
	}
	if found != apiErr {
		t.Errorf("errors.As found %v, want the original APIError", found)
	}
	if got, ok := errors.AsType[*APIError](wrapped); !ok || got != apiErr {
		t.Errorf("errors.AsType found %v (ok=%v), want the original APIError", got, ok)
	}

	joined := errors.Join(errors.New("unrelated"), apiErr)
	if !errors.As(joined, &found) {
		t.Error("errors.As did not find the APIError inside a joined error")
	}

	// The innermost APIError wins so nested service errors keep their original code.
	outer := NewInternalError(apiErr, "handler failed")
	if inner, ok := errors.AsType[*APIError](outer.Unwrap()); !ok || inner != apiErr {
		t.Errorf("Unwrap() = %v (ok=%v), want the nested APIError", inner, ok)
	}
	if _, ok := errors.AsType[*APIError](NewValidationError("bad input").Unwrap()); ok {
		t.Error("Unwrap() on an error with no internal cause returned a value, want none")
	}
}

// nestInternalMessage is the only place an outer message is joined to a nested APIError's
// own, so its contract is pinned here independently of how Error() later renders it.
func TestNestInternalMessage(t *testing.T) {
	t.Parallel()
	nested := NewInternalError(errors.New("db exploded"), "query failed")

	tests := []struct {
		name            string
		internal        error
		internalMessage string
		expected        string
	}{
		{
			name:            "no wrapped error keeps the message as given",
			internal:        nil,
			internalMessage: "handler failed",
			expected:        "handler failed",
		},
		{
			name:            "a plain wrapped error contributes nothing",
			internal:        errors.New("db exploded"),
			internalMessage: "handler failed",
			expected:        "handler failed",
		},
		{
			name:            "a nested APIError's message is chained after the outer one",
			internal:        nested,
			internalMessage: "handler failed",
			expected:        "handler failed: query failed",
		},
		{
			name:            "an empty outer message yields the nested message alone",
			internal:        nested,
			internalMessage: "",
			expected:        "query failed",
		},
		{
			name:            "a nested APIError with no message contributes nothing",
			internal:        NewValidationError("bad input"),
			internalMessage: "handler failed",
			expected:        "handler failed",
		},
		{
			name:            "both empty yields an empty message",
			internal:        NewValidationError("bad input"),
			internalMessage: "",
			expected:        "",
		},
		// Chaining is a direct type assertion, not errors.As, so an APIError reached only
		// through a %w wrap is left to Error() to render.
		{
			name:            "an APIError behind a %w wrap is not chained",
			internal:        fmt.Errorf("calling core: %w", nested),
			internalMessage: "handler failed",
			expected:        "handler failed",
		},
		{
			name:            "a joined APIError is not chained",
			internal:        errors.Join(errors.New("first"), nested),
			internalMessage: "handler failed",
			expected:        "handler failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := nestInternalMessage(tt.internal, tt.internalMessage); got != tt.expected {
				t.Errorf("nestInternalMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Error() is what lands in request_logs.internal_error_message, so it must read the same on
// the receiving service as it did where the error was raised.
func TestAPIError_Error_SurvivesJSONRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "message and wrapped cause",
			err:      NewInternalError(errors.New("db exploded"), "query failed"),
			expected: "query failed: db exploded",
		},
		{
			name:     "message with no wrapped cause",
			err:      NewRequestTimeoutError("core deadline exceeded"),
			expected: "core deadline exceeded",
		},
		{
			name:     "wrapped cause with an empty message is not appended",
			err:      NewInvariantViolationError("expected a row after insert"),
			expected: "expected a row after insert",
		},
		{
			name:     "APIError behind a %w wrap is flattened to its rendered text",
			err:      NewInternalError(fmt.Errorf("calling core: %w", NewInternalError(errors.New("db exploded"), "query failed")), "handler failed"),
			expected: "handler failed: calling core: query failed: db exploded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.expected {
				t.Fatalf("Error() = %q, want %q", got, tt.expected)
			}

			data, err := tt.err.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON: %v", err)
			}
			restored, err := APIErrorFromJSON(data)
			if err != nil {
				t.Fatalf("APIErrorFromJSON: %v", err)
			}
			if got := restored.Error(); got != tt.expected {
				t.Errorf("Error() after round trip = %q, want %q", got, tt.expected)
			}
		})
	}
}

// A nil error must serialize to nothing rather than to a null literal the receiver would
// decode into an empty APIError carrying a 500.
func TestAPIError_ToJSON_NilReceiver(t *testing.T) {
	t.Parallel()
	var apiErr *APIError
	data, err := apiErr.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if data != nil {
		t.Errorf("ToJSON() = %q, want nil", data)
	}
}

// Describe exists because Error() is a composition primitive, not a reporting one. Most constructors
// set only a public message, so their Error() is empty by design — and an empty string reaching
// message_inbox.last_error records that a handler failed without recording what failed.
func TestDescribe_NeverEmptyForANonNilError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil is the only empty case",
			err:      nil,
			expected: "",
		},
		{
			name:     "an internal message is used as-is",
			err:      NewInternalError(errors.New("db exploded"), "query failed"),
			expected: "query failed: db exploded",
		},
		{
			name:     "a public-message-only error falls back to code and message",
			err:      NewValidationError("bad input"),
			expected: "validation_failed: bad input",
		},
		{
			name:     "a plain error is unchanged",
			err:      errors.New("boom"),
			expected: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Describe(tt.err); got != tt.expected {
				t.Errorf("Describe() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// The composition semantics Error() has must not change: a nested APIError with no internal message
// still contributes nothing to its parent's text. Describe is the reporting path, not a replacement.
func TestDescribe_DoesNotChangeErrorComposition(t *testing.T) {
	t.Parallel()

	nested := NewInternalError(NewValidationError("bad input"), "handler failed")
	if got := nested.Error(); got != "handler failed" {
		t.Errorf("Error() = %q, want %q — Describe must not have leaked into composition", got, "handler failed")
	}
}
