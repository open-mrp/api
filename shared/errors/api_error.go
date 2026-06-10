// Package apierror defines the structured error types used across all services.
//
// Every user-facing error is represented as an APIError, which carries both a
// public message (safe to return in HTTP responses) and an internal message
// (for logs and traces only). Errors are categorized by an ErrorCode (machine-
// readable, e.g. "validation_failed") and an ErrorType (broad category, e.g.
// "invalid_request_error").
//
// The package provides constructor functions for common error scenarios (validation,
// auth, not-found, etc.) and handles mapping error codes to HTTP status codes via
// GetHTTPStatusCode. For gRPC transport, errors can be serialized to/from JSON
// using ToJSON/APIErrorFromJSON.
package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/augno/api/shared/version"
)

// QuotaInfo provides machine-readable details about a plan-imposed resource limit.
// Included in limit_exceeded errors so clients can display upgrade prompts, usage bars,
// or implement programmatic retry/backoff logic.
type QuotaInfo struct {
	// Limit is the maximum number of resources allowed by the current plan.
	Limit int32 `json:"limit"`
	// Used is the number of resources currently consumed.
	Used int32 `json:"used"`
	// ResetAt is the time when the quota resets, if applicable. Nil for static (non-metered) limits.
	ResetAt *time.Time `json:"reset_at"`
}

// ErrorCode is a machine-readable identifier for a specific error condition.
// These codes are returned in API responses and used by clients to programmatically
// handle errors without parsing human-readable messages.
type ErrorCode string

const (
	// --- Authentication & authorization errors (401/403) ---

	// ErrorCodeExpiredToken indicates a JWT access or refresh token has expired.
	ErrorCodeExpiredToken ErrorCode = "expired_token"
	// ErrorCodeExpiredAPIKey indicates an API key has passed its expiration date.
	ErrorCodeExpiredAPIKey ErrorCode = "api_key_expired" // #nosec G101 - This is an error code constant, not a hardcoded credential
	// ErrorCodeRevokedAPIKey indicates an API key was explicitly revoked by the owner or Augno.
	ErrorCodeRevokedAPIKey ErrorCode = "api_key_revoked" // #nosec G101 - This is an error code constant, not a hardcoded credential
	// ErrorCodeInvalidCredentials indicates the provided credentials are wrong.
	ErrorCodeInvalidCredentials ErrorCode = "invalid_credentials" // #nosec G101 - This is an error code constant, not a hardcoded credential
	// ErrorCodeInsufficientPerms indicates the caller is authenticated but lacks the required role or permission.
	ErrorCodeInsufficientPerms ErrorCode = "insufficient_permissions"
	// ErrorCodePaymentRequired indicates the account's subscription is in a non-active state
	// (past_due, canceled, unpaid) and must be resolved before the account can continue using the platform.
	ErrorCodePaymentRequired ErrorCode = "payment_required"

	// --- Validation errors (400) ---

	// ErrorCodeValidationFailed is a general validation failure indicating that the request is invalid.
	ErrorCodeValidationFailed ErrorCode = "validation_failed"
	// ErrorCodeMissingField indicates a required field was not provided in the request body.
	ErrorCodeMissingField ErrorCode = "missing_field"
	// ErrorCodeInvalidFormat indicates a field value does not match the expected format (e.g. invalid email).
	ErrorCodeInvalidFormat ErrorCode = "invalid_format"
	// ErrorCodeMethodNotAllowed indicates the HTTP method is not supported for this endpoint.
	ErrorCodeMethodNotAllowed ErrorCode = "method_not_allowed"

	// --- Resource errors (404/409/410) ---

	// ErrorCodeResourceNotFound indicates the requested resource does not exist.
	ErrorCodeResourceNotFound ErrorCode = "resource_not_found"
	// ErrorCodeResourceExists indicates a resource with the same unique constraint already exists.
	ErrorCodeResourceExists ErrorCode = "resource_exists"
	// ErrorCodeResourceConflict indicates a state conflict that prevents the operation
	// (e.g. duplicate username, optimistic locking failure).
	ErrorCodeResourceConflict ErrorCode = "resource_conflict"
	// ErrorCodeResourceGone indicates the resource existed but has been permanently deleted.
	ErrorCodeResourceGone ErrorCode = "resource_gone"

	// --- Idempotency errors (409) ---

	// ErrorCodeIdempotencyInProgress indicates a request with the same idempotency key is
	// currently being processed. The client should retry after a short delay.
	ErrorCodeIdempotencyInProgress ErrorCode = "idempotency_in_progress"

	// --- Limit errors (403) ---

	// ErrorCodeLimitExceeded indicates the account has reached a plan-imposed resource limit
	// (e.g. maximum sandbox accounts). The caller must upgrade their plan to increase the limit.
	ErrorCodeLimitExceeded ErrorCode = "limit_exceeded"
	// ErrorCodeRegistrationClosed indicates that public registration for the
	// requested plan code has reached its capacity. Distinct from limit_exceeded,
	// which applies to per-account resource quotas (e.g. sandbox count).
	ErrorCodeRegistrationClosed ErrorCode = "registration_closed"

	// --- Rate limiting (429) ---

	// ErrorCodeRateLimitExceeded indicates the caller has exceeded the allowed request rate.
	// The client should back off and retry.
	ErrorCodeRateLimitExceeded ErrorCode = "rate_limit_exceeded"

	// --- Parameter errors (400) ---

	// ErrorCodeParameterMissing indicates a required query or path parameter was not provided.
	ErrorCodeParameterMissing ErrorCode = "parameter_missing"
	// ErrorCodeParameterInvalid indicates a parameter value is not valid (e.g. negative limit).
	ErrorCodeParameterInvalid ErrorCode = "parameter_invalid"
	// ErrorCodeParameterUnknown indicates an unrecognized parameter was sent in the request.
	ErrorCodeParameterUnknown ErrorCode = "parameter_unknown"
	// ErrorCodeParametersExclusive indicates two mutually exclusive parameters were both provided.
	ErrorCodeParametersExclusive ErrorCode = "parameters_exclusive"

	// --- Server errors (5xx) ---

	// ErrorCodeInternalError is an unexpected server-side failure. Transient; safe to retry.
	ErrorCodeInternalError ErrorCode = "internal_error"
	// ErrorCodeSvcUnavailable indicates a downstream service is temporarily unreachable. Transient.
	ErrorCodeSvcUnavailable ErrorCode = "service_unavailable"
	// ErrorCodeExternalSvcError indicates a third-party dependency returned an error. Transient.
	ErrorCodeExternalSvcError ErrorCode = "external_service_error"
	// ErrorCodeTimeout indicates the operation exceeded its deadline. Transient.
	ErrorCodeTimeout ErrorCode = "timeout"
	// ErrorCodeConnectionError indicates a network-level failure to a downstream service. Transient.
	ErrorCodeConnectionError ErrorCode = "connection_error"
	// ErrorCodeRequestTimeout indicates the overall request exceeded its deadline. Transient.
	ErrorCodeRequestTimeout ErrorCode = "request_timeout"
	// ErrorCodeClientClosedRequest indicates the client disconnected before the server finished.
	// Not transient — the server cannot meaningfully retry on behalf of a disconnected client.
	ErrorCodeClientClosedRequest ErrorCode = "client_closed_request"

	// --- API version errors (400) ---

	// ErrorCodeAPIVersionRequired indicates the Augno-Version header was missing from the request.
	ErrorCodeAPIVersionRequired ErrorCode = "api_version_required"
	// ErrorCodeAPIVersionInvalid indicates the requested API version string is not recognized.
	ErrorCodeAPIVersionInvalid ErrorCode = "api_version_invalid"
	// ErrorCodeAPIVersionTooOld indicates the requested API version is below the minimum
	// supported by the endpoint.
	ErrorCodeAPIVersionTooOld ErrorCode = "api_version_too_old"
)

// IsValid reports whether the ErrorCode is a recognized value.
func (c ErrorCode) IsValid() bool {
	switch c {
	case ErrorCodeExpiredToken, ErrorCodeExpiredAPIKey, ErrorCodeRevokedAPIKey,
		ErrorCodeInvalidCredentials, ErrorCodeInsufficientPerms, ErrorCodePaymentRequired,
		ErrorCodeValidationFailed, ErrorCodeMissingField, ErrorCodeInvalidFormat,
		ErrorCodeMethodNotAllowed, ErrorCodeResourceNotFound, ErrorCodeResourceExists,
		ErrorCodeResourceConflict, ErrorCodeResourceGone, ErrorCodeIdempotencyInProgress,
		ErrorCodeLimitExceeded, ErrorCodeRegistrationClosed, ErrorCodeRateLimitExceeded,
		ErrorCodeParameterMissing, ErrorCodeParameterInvalid, ErrorCodeParameterUnknown,
		ErrorCodeParametersExclusive, ErrorCodeInternalError, ErrorCodeSvcUnavailable,
		ErrorCodeExternalSvcError, ErrorCodeTimeout, ErrorCodeConnectionError,
		ErrorCodeRequestTimeout, ErrorCodeClientClosedRequest,
		ErrorCodeAPIVersionRequired, ErrorCodeAPIVersionInvalid, ErrorCodeAPIVersionTooOld:
		return true
	default:
		return false
	}
}

// EnumValues returns all valid ErrorCode values for use in schema generation.
func (c ErrorCode) EnumValues() []string {
	return []string{
		string(ErrorCodeExpiredToken),
		string(ErrorCodeExpiredAPIKey),
		string(ErrorCodeRevokedAPIKey),
		string(ErrorCodeInvalidCredentials),
		string(ErrorCodeInsufficientPerms),
		string(ErrorCodePaymentRequired),
		string(ErrorCodeValidationFailed),
		string(ErrorCodeMissingField),
		string(ErrorCodeInvalidFormat),
		string(ErrorCodeMethodNotAllowed),
		string(ErrorCodeResourceNotFound),
		string(ErrorCodeResourceExists),
		string(ErrorCodeResourceConflict),
		string(ErrorCodeResourceGone),
		string(ErrorCodeIdempotencyInProgress),
		string(ErrorCodeLimitExceeded),
		string(ErrorCodeRegistrationClosed),
		string(ErrorCodeRateLimitExceeded),
		string(ErrorCodeParameterMissing),
		string(ErrorCodeParameterInvalid),
		string(ErrorCodeParameterUnknown),
		string(ErrorCodeParametersExclusive),
		string(ErrorCodeInternalError),
		string(ErrorCodeSvcUnavailable),
		string(ErrorCodeExternalSvcError),
		string(ErrorCodeTimeout),
		string(ErrorCodeConnectionError),
		string(ErrorCodeRequestTimeout),
		string(ErrorCodeClientClosedRequest),
		string(ErrorCodeAPIVersionRequired),
		string(ErrorCodeAPIVersionInvalid),
		string(ErrorCodeAPIVersionTooOld),
	}
}

// ErrorType categorizes errors into broad classes for client-side handling.
//   - ErrorTypeAPI: server-side failures (5xx-class), safe to retry if transient.
//   - ErrorTypeIdempotency: idempotency key conflicts (concurrent or mismatched requests).
//   - ErrorTypeInvalidRequest: client-side mistakes (bad input, missing auth, not found).
type ErrorType string

const (
	// ErrorTypeAPI covers server-side failures (5xx). Generally transient and safe to retry.
	ErrorTypeAPI ErrorType = "api_error"
	// ErrorTypeIdempotency covers idempotency key conflicts. May be transient (concurrent
	// duplicate request) or permanent (reused key with different parameters).
	ErrorTypeIdempotency ErrorType = "idempotency_error"
	// ErrorTypeInvalidRequest covers client-side mistakes (bad input, missing auth, not found).
	// Generally not retryable — the client must fix the request before resending.
	ErrorTypeInvalidRequest ErrorType = "invalid_request_error"
)

// IsValid reports whether the ErrorType is a recognized value.
func (t ErrorType) IsValid() bool {
	switch t {
	case ErrorTypeAPI, ErrorTypeIdempotency, ErrorTypeInvalidRequest:
		return true
	default:
		return false
	}
}

// EnumValues returns all valid ErrorType values for use in schema generation.
func (t ErrorType) EnumValues() []string {
	return []string{
		string(ErrorTypeAPI),
		string(ErrorTypeIdempotency),
		string(ErrorTypeInvalidRequest),
	}
}

// IsTransientError returns true if the error code and type combination represents a
// temporary condition that may resolve on retry. The error type provides context that
// the code alone cannot — for example, a resource_conflict with ErrorTypeInvalidRequest
// (e.g. "username already taken") is not retryable, while server-side errors generally are.
//
// Transient combinations:
//   - ErrorTypeAPI: all server errors except client_closed_request
//   - ErrorTypeIdempotency: only idempotency_in_progress (concurrent duplicate request)
//   - ErrorTypeInvalidRequest: only rate_limit_exceeded (back off and retry)
func IsTransientError(code ErrorCode, errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeAPI:
		switch code {
		case ErrorCodeInternalError,
			ErrorCodeSvcUnavailable,
			ErrorCodeExternalSvcError,
			ErrorCodeTimeout,
			ErrorCodeConnectionError,
			ErrorCodeRequestTimeout:
			return true
		}
	case ErrorTypeIdempotency:
		return code == ErrorCodeIdempotencyInProgress
	case ErrorTypeInvalidRequest:
		return code == ErrorCodeRateLimitExceeded
	}
	return false
}

// APIError is the canonical error type used throughout the platform. It implements
// the error interface and separates public-facing information (Code, Type, PublicMessage)
// from internal diagnostics (InternalMessage, Internal) that are never exposed to clients.
//
// Create instances using the constructor functions (NewValidationError, NewInternalError, etc.)
// rather than building APIError literals directly.
type APIError struct {
	// Code is the machine-readable error code included in API responses.
	Code ErrorCode `json:"code"`
	// Type is the broad error category included in API responses.
	Type ErrorType `json:"type"`
	// PublicMessage is the human-readable message returned to the client.
	PublicMessage string `json:"message"`
	// Param is the request parameter that caused the error (e.g. "email"), if applicable.
	Param string `json:"param,omitempty"`
	// DocURL links to documentation about this specific error, if available.
	DocURL string `json:"doc_url,omitempty"`
	// IsTransient indicates whether the client should retry the request.
	// Automatically set by NewAPIError based on the error code and type.
	IsTransient bool `json:"is_transient"`
	// Quota provides machine-readable details about a plan-imposed resource limit.
	// Only populated for limit_exceeded errors. Included in API responses when non-nil.
	Quota *QuotaInfo `json:"quota,omitempty"`
	// InternalMessage is a developer-facing message for logs and traces. Never sent to clients.
	InternalMessage string `json:"-"`
	// Internal is the underlying error, if any. Never sent to clients. Accessible via Unwrap().
	Internal error `json:"-"`
	// Stack is the goroutine stack captured at the point a 5xx error is created, so the
	// recorded trace points at the failing code rather than the response-writing layer.
	// Empty for non-5xx errors. Never sent to clients. Propagated across gRPC via ToJSON.
	Stack string `json:"-"`
}

// Error returns the internal (non-public) error string for logging. If an underlying
// error is wrapped, it is appended to the internal message.
func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Internal != nil {
		if inner := e.Internal.Error(); inner != "" {
			return e.InternalMessage + ": " + inner
		}
	}
	return e.InternalMessage
}

// Unwrap returns the underlying error for use with errors.Is/errors.As.
func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Internal
}

// WithParam sets the parameter name on the error and returns the same pointer for chaining.
func (e *APIError) WithParam(param string) *APIError {
	e.Param = param
	return e
}

// WithDocURL sets the documentation URL on the error and returns the same pointer for chaining.
func (e *APIError) WithDocURL(url string) *APIError {
	e.DocURL = url
	return e
}

// WithInternal wraps an underlying error and returns the same pointer for chaining.
func (e *APIError) WithInternal(err error) *APIError {
	e.Internal = err
	return e
}

// WithQuota attaches quota details to the error and returns the same pointer for chaining.
// Intended for limit_exceeded errors so clients receive the plan limit, current usage,
// and optional reset time in the response envelope.
func (e *APIError) WithQuota(limit, used int32, resetAt *time.Time) *APIError {
	e.Quota = &QuotaInfo{Limit: limit, Used: used, ResetAt: resetAt}
	return e
}

// APIErrorOption is a functional option applied during NewAPIError construction.
// Use the package-level WithParam, WithDocURL, and WithInternal functions to create options.
type APIErrorOption func(*APIError)

// WithParam returns an option that sets the offending parameter name on the error.
func WithParam(param string) APIErrorOption {
	return func(e *APIError) {
		e.Param = param
	}
}

// WithDocURL returns an option that sets the documentation URL on the error.
func WithDocURL(url string) APIErrorOption {
	return func(e *APIError) {
		e.DocURL = url
	}
}

// WithInternal returns an option that wraps an underlying error for logging/tracing.
func WithInternal(err error) APIErrorOption {
	return func(e *APIError) {
		e.Internal = err
	}
}

// nestInternalMessage concatenates the internal messages of nested APIErrors to form
// a chain like "outer context: inner context". This preserves diagnostic context when
// one service wraps an error received from another.
func nestInternalMessage(internal error, internalMessage string) string {
	if internal == nil {
		return internalMessage
	}

	// if nested Internal Error combine messages
	if nestedErr, ok := internal.(*APIError); ok {
		if nestedErr.InternalMessage != "" {
			if internalMessage != "" {
				return internalMessage + ": " + nestedErr.InternalMessage
			}
			return nestedErr.InternalMessage
		}
	}

	return internalMessage
}

// NewAPIError is the base constructor for all API errors. It sets IsTransient automatically
// based on the error code. Prefer the domain-specific constructors (NewValidationError,
// NewInternalError, etc.) for common cases; use this directly only when none of those fit.
func NewAPIError(code ErrorCode, errorType ErrorType, publicMessage string, internalMessage string, opts ...APIErrorOption) *APIError {
	apiError := &APIError{
		Code:            code,
		Type:            errorType,
		PublicMessage:   publicMessage,
		InternalMessage: internalMessage,
		IsTransient:     IsTransientError(code, errorType),
	}

	for _, opt := range opts {
		opt(apiError)
	}

	// For server-side (5xx) errors, capture a stack at the origin so the recorded trace
	// points at the failing code. If the wrapped error is itself an APIError that already
	// captured a stack (e.g. arriving from a downstream service), inherit it rather than
	// overwrite it with this less-useful outer frame. 4xx errors skip this entirely.
	if Is5XXErrorCode(apiError.Code) {
		if inner, ok := apiError.Internal.(*APIError); ok && inner.Stack != "" {
			apiError.Stack = inner.Stack
		} else {
			apiError.Stack = captureStack()
		}
	}

	return apiError
}

// captureStack returns a formatted stack trace for the current goroutine, used to
// pinpoint where a 5xx error originated.
func captureStack() string {
	buf := make([]byte, 32768) // 32KB
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// NewValidationError creates a 400 Bad Request error for general input validation failures.
func NewValidationError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeValidationFailed, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLValidationFailed))
}

// NewValidationErrorWithParam creates a 400 Bad Request error tied to a specific request parameter.
func NewValidationErrorWithParam(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeValidationFailed, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLValidationFailed))
}

// NewMissingFieldError creates a 400 error when a required field is not provided in the request body.
func NewMissingFieldError(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeMissingField, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLMissingField))
}

// NewInvalidFormatError creates a 400 error when a field value does not match the expected format.
func NewInvalidFormatError(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeInvalidFormat, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLInvalidFormat))
}

// NewParameterMissingError creates a 400 error when a required query or path parameter is not provided.
func NewParameterMissingError(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeParameterMissing, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLParameterMissing))
}

// NewParameterInvalidError creates a 400 error when a query or path parameter value is not valid.
func NewParameterInvalidError(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeParameterInvalid, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLParameterInvalid))
}

// NewParameterUnknownError creates a 400 error when an unrecognized parameter is sent in the request.
func NewParameterUnknownError(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeParameterUnknown, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLParameterUnknown))
}

// NewAuthenticationError creates a 401 Unauthorized error for invalid credentials.
func NewAuthenticationError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeInvalidCredentials, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLInvalidCredentials))
}

// NewExpiredAPIKeyError creates a 401 error when an API key has passed its expiration date.
func NewExpiredAPIKeyError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeExpiredAPIKey, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLExpiredAPIKey))
}

// NewRevokedAPIKeyError creates a 401 error when an API key has been explicitly revoked.
func NewRevokedAPIKeyError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeRevokedAPIKey, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLRevokedAPIKey))
}

// NewAuthorizationError creates a 403 Forbidden error when the caller lacks required permissions.
func NewAuthorizationError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeInsufficientPerms, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLInsufficientPerms))
}

// NewPaymentRequiredError creates a 402 Payment Required error when the account's subscription
// is in a non-active state and must be resolved before continuing.
func NewPaymentRequiredError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodePaymentRequired, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLPaymentRequired))
}

// NewLimitExceededError creates a 403 Forbidden error when an account has reached
// a plan-imposed resource limit (e.g. maximum sandbox accounts).
func NewLimitExceededError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeLimitExceeded, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLLimitExceeded))
}

// NewRegistrationClosedError creates a 403 Forbidden error when public
// registration for a plan code has reached its capacity.
func NewRegistrationClosedError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeRegistrationClosed, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLRegistrationClosed))
}

// NewResourceNotFoundError creates a 404 Not Found error for missing resources.
func NewResourceNotFoundError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeResourceNotFound, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLResourceNotFound))
}

// NewResourceConflictError creates a 409 Conflict error for state conflicts (e.g. concurrent updates).
func NewResourceConflictError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeResourceConflict, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLResourceConflict))
}

// NewResourceExistsError creates a 409 Conflict error for duplicate resource creation
// (e.g. unique constraint violation). Uses ErrorCodeResourceExists rather than
// ErrorCodeResourceConflict because the semantics are "this resource already exists"
// rather than a generic state conflict.
func NewResourceExistsError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeResourceExists, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLResourceConflict))
}

// NewAlreadyDeletedError creates a 410 Gone error for resources that were already deleted.
func NewAlreadyDeletedError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeResourceGone, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLResourceGone))
}

// NewIdempotencyInProgressError creates a 409 Conflict error when a request with the
// same idempotency key is already being processed concurrently.
func NewIdempotencyInProgressError(idempotencyKey string) *APIError {
	return NewAPIError(ErrorCodeIdempotencyInProgress, ErrorTypeIdempotency, fmt.Sprintf("Request for the idempotency key '%s' is already being processed.", idempotencyKey), "", WithDocURL(docURLIdempotencyInProgress))
}

// NewIdempotencyHashMismatchError creates a validation error when an idempotency key
// is reused with different request parameters than the original request.
func NewIdempotencyHashMismatchError(idempotencyKey string) *APIError {
	return NewAPIError(ErrorCodeValidationFailed, ErrorTypeIdempotency, fmt.Sprintf("Idempotency key '%s' was used with different request parameters; use a new key.", idempotencyKey), "", WithDocURL(docURLValidationFailed))
}

// NewConflictErrorWithParam creates a 409 Conflict error tied to a specific parameter
// (e.g. a duplicate email address).
func NewConflictErrorWithParam(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeResourceConflict, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param), WithDocURL(docURLResourceConflict))
}

// NewRateLimitExceededError creates a 429 Too Many Requests error. Marked as transient.
func NewRateLimitExceededError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeRateLimitExceeded, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLRateLimit))
}

// NewInternalError creates a 500 Internal Server Error with a generic public message
// ("Something went wrong.") and wraps the underlying error for internal logging.
// If the underlying error is itself an APIError, their internal messages are chained.
func NewInternalError(internal error, internalMessage string) *APIError {
	combinedMessage := nestInternalMessage(internal, internalMessage)
	return NewAPIError(ErrorCodeInternalError, ErrorTypeAPI, "Something went wrong.", combinedMessage, WithInternal(internal), WithDocURL(docURLInternalError))
}

// NewInvariantViolationError creates a 500 error for conditions that should never occur
// in correct code (e.g. missing required data after a successful query). The public
// message is generic; the internal message captures what invariant was violated.
func NewInvariantViolationError(internalMessage string) *APIError {
	internal := errors.New("")
	return NewAPIError(ErrorCodeInternalError, ErrorTypeAPI, "Something went wrong.", internalMessage, WithInternal(internal), WithDocURL(docURLInternalError))
}

// NewMethodNotAllowedError creates a 405 Method Not Allowed error.
func NewMethodNotAllowedError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeMethodNotAllowed, ErrorTypeInvalidRequest, publicMessage, "", WithDocURL(docURLMethodNotAllowed))
}

// NewRequestTimeoutError creates a 408 Request Timeout error. Marked as transient.
func NewRequestTimeoutError(internalMessage string) *APIError {
	return NewAPIError(ErrorCodeRequestTimeout, ErrorTypeAPI, "Request timed out.", internalMessage, WithDocURL(docURLRequestTimeout))
}

// NewClientClosedRequestError creates a 499 error (nginx convention) when the client
// disconnects before the server finishes processing.
func NewClientClosedRequestError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeClientClosedRequest, ErrorTypeAPI, publicMessage, "Client closed request.", WithDocURL(docURLClientClosedRequest))
}

// NewAPIVersionRequiredError creates a 400 error when the Augno-Version header is missing.
func NewAPIVersionRequiredError() *APIError {
	return NewAPIError(ErrorCodeAPIVersionRequired, ErrorTypeInvalidRequest, fmt.Sprintf("The Augno-Version header is required. Please include a valid API version. The latest version is %s.", version.Latest.String()), "", WithDocURL(docURLAPIVersionRequired))
}

// NewAPIVersionInvalidError creates a 400 error when the requested API version is not recognized.
func NewAPIVersionInvalidError(version string, supported []string) *APIError {
	return NewAPIError(ErrorCodeAPIVersionInvalid, ErrorTypeInvalidRequest, fmt.Sprintf("Invalid API version '%s'. Supported versions: %v", version, supported), "", WithDocURL(docURLAPIVersionInvalid))
}

// NewAPIVersionTooOldError creates a 400 error when the requested API version is below
// the minimum version required by the endpoint.
func NewAPIVersionTooOldError(requested, minimum string) *APIError {
	return NewAPIError(ErrorCodeAPIVersionTooOld, ErrorTypeInvalidRequest, fmt.Sprintf("This endpoint requires API version %s or newer. You requested %s.", minimum, requested), "", WithDocURL(docURLAPIVersionTooOld))
}

// ResponseError is the JSON-serializable error body returned to API clients. It contains
// only public information. This struct is used by the OpenAPI schema generator to produce
// documentation.
type ResponseError struct {
	// A machine-readable code for the error.
	Code ErrorCode `json:"code"`
	// The type of error.
	Type ErrorType `json:"type"`
	// A human-readable message providing more details about the error.
	Message string `json:"message"`
	// The parameter that caused the error, if applicable.
	Param *string `json:"param"`
	// A URL to documentation about the error.
	DocURL *string `json:"doc_url"`
	// Whether this error is transient and the request can be retried.
	IsTransient bool `json:"is_transient"`
	// Quota provides plan limit details when the error is limit_exceeded. Nil otherwise.
	Quota *QuotaInfo `json:"quota"`
	// RequestLogURL is a link to the dashboard page for this request's log entry.
	// Nil when no request log is available.
	RequestLogURL *string `json:"request_log_url"`
}

// SchemaExample returns a representative instance for OpenAPI documentation generation.
func (r ResponseError) SchemaExample() any {
	return ResponseError{
		Code:          ErrorCodeValidationFailed,
		Type:          ErrorTypeInvalidRequest,
		Message:       "The request was invalid.",
		Param:         new("email"),
		DocURL:        new(docURLValidationFailed),
		IsTransient:   false,
		RequestLogURL: new("https://augno.com/dashboard/request-logs/rq_fbv1ygmybo3eauykr74"),
	}
}

// APIErrorResponse is the top-level envelope wrapping a ResponseError.
// All API error responses use this shape: { "error": { ... } }.
type APIErrorResponse struct {
	// The error object containing details about what went wrong.
	Error ResponseError `json:"error"`
}

// SchemaExample returns a representative instance for OpenAPI documentation generation.
func (r APIErrorResponse) SchemaExample() any {
	return APIErrorResponse{
		Error: ResponseError{
			Code:        ErrorCodeValidationFailed,
			Type:        ErrorTypeInvalidRequest,
			Message:     "The request was invalid.",
			Param:       new("email"),
			DocURL:      new(docURLValidationFailed),
			IsTransient: false,
		},
	}
}

// ToResponseMap converts the APIError into an APIErrorResponse suitable for JSON
// serialization in HTTP responses. Only public fields are included.
func (e *APIError) ToResponseMap() any {
	if e == nil {
		return APIErrorResponse{}
	}
	resp := ResponseError{
		Code:        e.Code,
		Type:        e.Type,
		Message:     e.PublicMessage,
		IsTransient: e.IsTransient,
	}
	if e.Param != "" {
		resp.Param = new(e.Param)
	}
	if e.DocURL != "" {
		resp.DocURL = new(e.DocURL)
	}
	if e.Quota != nil {
		resp.Quota = e.Quota
	}
	return APIErrorResponse{Error: resp}
}

// GetHTTPStatusCode maps an ErrorCode to the appropriate HTTP status code.
// Used by the API gateway to set the response status when writing error responses.
func GetHTTPStatusCode(code ErrorCode) int {
	switch code {
	case ErrorCodeExpiredToken,
		ErrorCodeExpiredAPIKey,
		ErrorCodeRevokedAPIKey,
		ErrorCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrorCodeInsufficientPerms, ErrorCodeLimitExceeded, ErrorCodeRegistrationClosed:
		return http.StatusForbidden
	case ErrorCodePaymentRequired:
		return http.StatusPaymentRequired
	case ErrorCodeValidationFailed, ErrorCodeMissingField, ErrorCodeInvalidFormat,
		ErrorCodeParameterMissing, ErrorCodeParameterInvalid, ErrorCodeParameterUnknown,
		ErrorCodeParametersExclusive, ErrorCodeAPIVersionRequired, ErrorCodeAPIVersionInvalid,
		ErrorCodeAPIVersionTooOld:
		return http.StatusBadRequest
	case ErrorCodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case ErrorCodeResourceNotFound:
		return http.StatusNotFound
	case ErrorCodeResourceGone:
		return http.StatusGone
	case ErrorCodeResourceExists, ErrorCodeResourceConflict, ErrorCodeIdempotencyInProgress:
		return http.StatusConflict
	case ErrorCodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case ErrorCodeSvcUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeExternalSvcError, ErrorCodeConnectionError:
		return http.StatusBadGateway
	case ErrorCodeTimeout:
		return http.StatusGatewayTimeout
	case ErrorCodeRequestTimeout:
		return http.StatusRequestTimeout
	case ErrorCodeClientClosedRequest:
		return 499
	case ErrorCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Is5XXErrorCode returns true if the error code maps to a 5xx HTTP status.
func Is5XXErrorCode(code ErrorCode) bool {
	return GetHTTPStatusCode(code) >= 500
}

// IsNotFound is a convenience check for whether an APIError is a resource_not_found error.
func IsNotFound(err *APIError) bool {
	return err != nil && err.Code == ErrorCodeResourceNotFound
}

// apiErrorSerializable is a wire format for encoding APIErrors as JSON when passing
// them across gRPC service boundaries. Unlike the public ResponseError, this includes
// internal diagnostic fields so they survive the hop between services.
type apiErrorSerializable struct {
	Code            ErrorCode  `json:"code"`
	Type            ErrorType  `json:"type"`
	PublicMessage   string     `json:"message"`
	Param           string     `json:"param,omitempty"`
	DocURL          string     `json:"doc_url,omitempty"`
	IsTransient     bool       `json:"is_transient"`
	Quota           *QuotaInfo `json:"quota,omitempty"`
	InternalMessage string     `json:"internal_message,omitempty"`
	InternalError   string     `json:"internal_error,omitempty"`
	Stack           string     `json:"stack,omitempty"`
}

// ToJSON serializes the full APIError (including internal fields) to JSON for
// transport over gRPC. Use APIErrorFromJSON on the receiving side to reconstruct it.
func (e *APIError) ToJSON() ([]byte, error) {
	if e == nil {
		return nil, nil
	}

	serializable := apiErrorSerializable{
		Code:            e.Code,
		Type:            e.Type,
		PublicMessage:   e.PublicMessage,
		Param:           e.Param,
		DocURL:          e.DocURL,
		IsTransient:     e.IsTransient,
		Quota:           e.Quota,
		InternalMessage: e.InternalMessage,
		Stack:           e.Stack,
	}

	if e.Internal != nil {
		serializable.InternalError = e.Internal.Error()
	}

	return json.Marshal(serializable)
}

// APIErrorFromJSON reconstructs an APIError from JSON bytes produced by ToJSON.
// Used on the receiving side of gRPC calls to restore the full error with internal details.
func APIErrorFromJSON(jsonData []byte) (*APIError, error) {
	var serializable apiErrorSerializable
	if err := json.Unmarshal(jsonData, &serializable); err != nil {
		return nil, err
	}

	apiErr := &APIError{
		Code:            serializable.Code,
		Type:            serializable.Type,
		PublicMessage:   serializable.PublicMessage,
		Param:           serializable.Param,
		DocURL:          serializable.DocURL,
		IsTransient:     serializable.IsTransient,
		Quota:           serializable.Quota,
		InternalMessage: serializable.InternalMessage,
		Stack:           serializable.Stack,
	}

	if serializable.InternalError != "" {
		apiErr.Internal = errors.New(serializable.InternalError)
	}

	return apiErr, nil
}
