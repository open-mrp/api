package contracts

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ErrorCode string

const (
	// Authentication errors
	ErrorCodeExpiredToken ErrorCode = "expired_token"
	// #nosec G101 - This is an error code constants, not a hardcoded credential
	ErrorCodeExpiredAPIKey ErrorCode = "api_key_expired"
	// #nosec G101 - This is an error code constants, not a hardcoded credential
	ErrorCodeRevokedAPIKey ErrorCode = "api_key_revoked"
	// #nosec G101 - This is an error code constants, not a hardcoded credential
	ErrorCodeInvalidCredentials ErrorCode = "invalid_credentials"
	ErrorCodeInsufficientPerms  ErrorCode = "insufficient_permissions"
	ErrorCodeHTTPDisabled       ErrorCode = "http_disabled"

	// Validation errors
	ErrorCodeValidationFailed ErrorCode = "validation_failed"
	ErrorCodeMissingField     ErrorCode = "missing_field"
	ErrorCodeInvalidFormat    ErrorCode = "invalid_format"
	ErrorCodeMethodNotAllowed ErrorCode = "method_not_allowed"

	// Resource errors
	ErrorCodeResourceNotFound ErrorCode = "resource_not_found"
	ErrorCodeResourceExists   ErrorCode = "resource_exists"
	ErrorCodeResourceConflict ErrorCode = "resource_conflict"

	// Business logic errors
	ErrorCodeRateLimitExceeded ErrorCode = "rate_limit_exceeded"

	// Parameter errors
	ErrorCodeParameterMissing    ErrorCode = "parameter_missing"
	ErrorCodeParameterInvalid    ErrorCode = "parameter_invalid"
	ErrorCodeParameterUnknown    ErrorCode = "parameter_unknown"
	ErrorCodeParametersExclusive ErrorCode = "parameters_exclusive"

	// Server errors
	ErrorCodeInternalError       ErrorCode = "internal_error"
	ErrorCodeSvcUnavailable      ErrorCode = "service_unavailable"
	ErrorCodeExternalSvcError    ErrorCode = "external_service_error"
	ErrorCodeTimeout             ErrorCode = "timeout"
	ErrorCodeConnectionError     ErrorCode = "connection_error"
	ErrorCodeRequestTimeout      ErrorCode = "request_timeout"
	ErrorCodeClientClosedRequest ErrorCode = "client_closed_request"
)

type ErrorType string

const (
	ErrorTypeAPI            ErrorType = "api_error"
	ErrorTypeIdempotency    ErrorType = "idempotency_error"
	ErrorTypeInvalidRequest ErrorType = "invalid_request_error"
)

type APIError struct {
	Code            ErrorCode `json:"code"`
	Type            ErrorType `json:"type"`
	PublicMessage   string    `json:"message"`
	Param           string    `json:"param,omitempty"`
	DocURL          string    `json:"doc_url,omitempty"`
	InternalMessage string    `json:"-"`
	Internal        error     `json:"-"`
}

func (e *APIError) Error() string {
	if e.Internal != nil {
		return e.InternalMessage + ": " + e.Internal.Error()
	}
	return e.InternalMessage
}

func (e *APIError) Unwrap() error {
	return e.Internal
}

func (e *APIError) WithParam(param string) *APIError {
	e.Param = param
	return e
}

func (e *APIError) WithDocURL(url string) *APIError {
	e.DocURL = url
	return e
}

func (e *APIError) WithInternal(err error) *APIError {
	e.Internal = err
	return e
}

type APIErrorOption func(*APIError)

func WithParam(param string) APIErrorOption {
	return func(e *APIError) {
		e.Param = param
	}
}

func WithDocURL(url string) APIErrorOption {
	return func(e *APIError) {
		e.DocURL = url
	}
}

func WithInternal(err error) APIErrorOption {
	return func(e *APIError) {
		e.Internal = err
	}
}

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

func NewAPIError(code ErrorCode, errorType ErrorType, publicMessage string, internalMessage string, opts ...APIErrorOption) *APIError {
	apiError := &APIError{
		Code:            code,
		Type:            errorType,
		PublicMessage:   publicMessage,
		InternalMessage: internalMessage,
	}

	for _, opt := range opts {
		opt(apiError)
	}

	return apiError
}

func NewValidationError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeValidationFailed, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewValidationErrorWithParam(publicMessage string, param string) *APIError {
	return NewAPIError(ErrorCodeValidationFailed, ErrorTypeInvalidRequest, publicMessage, "", WithParam(param))
}

func NewAuthenticationError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeInvalidCredentials, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewExpiredAPIKeyError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeExpiredAPIKey, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewRevokedAPIKeyError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeRevokedAPIKey, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewAuthorizationError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeInsufficientPerms, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewResourceNotFoundError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeResourceNotFound, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewResourceConflictError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeResourceConflict, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewRateLimitExceededError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeRateLimitExceeded, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewInternalError(internal error, internalMessage string) *APIError {
	combinedMessage := nestInternalMessage(internal, internalMessage)
	return NewAPIError(ErrorCodeInternalError, ErrorTypeAPI, "Something went wrong.", combinedMessage, WithInternal(internal))
}

func NewMethodNotAllowedError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeMethodNotAllowed, ErrorTypeInvalidRequest, publicMessage, "")
}

func NewRequestTimeoutError(internalMessage string) *APIError {
	return NewAPIError(ErrorCodeRequestTimeout, ErrorTypeAPI, "Request timed out.", internalMessage)
}

func NewClientClosedRequestError(publicMessage string) *APIError {
	return NewAPIError(ErrorCodeClientClosedRequest, ErrorTypeAPI, publicMessage, "Client closed request.")
}

func (e *APIError) ToResponseMap() map[string]any {
	errorMap := map[string]any{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.PublicMessage,
	}

	if e.Param != "" {
		errorMap["param"] = e.Param
	}

	if e.DocURL != "" {
		errorMap["doc_url"] = e.DocURL
	}

	return map[string]any{
		"error": errorMap,
	}
}

func GetHTTPStatusCode(code ErrorCode) int {
	switch code {
	case ErrorCodeExpiredToken,
		ErrorCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrorCodeInsufficientPerms:
		return http.StatusForbidden
	case ErrorCodeValidationFailed, ErrorCodeMissingField, ErrorCodeInvalidFormat,
		ErrorCodeParameterMissing, ErrorCodeParameterInvalid, ErrorCodeParameterUnknown,
		ErrorCodeParametersExclusive:
		return http.StatusBadRequest
	case ErrorCodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case ErrorCodeResourceNotFound:
		return http.StatusNotFound
	case ErrorCodeResourceExists, ErrorCodeResourceConflict:
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

func Is5XXErrorCode(code ErrorCode) bool {
	return GetHTTPStatusCode(code) >= 500
}

// apiErrorSerializable is a serializable version of APIError for JSON encoding
type apiErrorSerializable struct {
	Code            ErrorCode `json:"code"`
	Type            ErrorType `json:"type"`
	PublicMessage   string    `json:"message"`
	Param           string    `json:"param,omitempty"`
	DocURL          string    `json:"doc_url,omitempty"`
	InternalMessage string    `json:"internal_message,omitempty"`
	InternalError   string    `json:"internal_error,omitempty"`
}

// ToJSON converts an APIError to JSON bytes.
func (e *APIError) toJSON() ([]byte, error) {
	if e == nil {
		return nil, nil
	}

	serializable := apiErrorSerializable{
		Code:            e.Code,
		Type:            e.Type,
		PublicMessage:   e.PublicMessage,
		Param:           e.Param,
		DocURL:          e.DocURL,
		InternalMessage: e.InternalMessage,
	}

	if e.Internal != nil {
		serializable.InternalError = e.Internal.Error()
	}

	return json.Marshal(serializable)
}

// apiErrorFromJSON converts JSON bytes back to an APIError.
func apiErrorFromJSON(jsonData []byte) (*APIError, error) {
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
		InternalMessage: serializable.InternalMessage,
	}

	if serializable.InternalError != "" {
		apiErr.Internal = errors.New(serializable.InternalError)
	}

	return apiErr, nil
}
