package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GatewayError is a structured error returned when the LLM gateway responds
// with a non-200 status code. It classifies errors as retryable or not and
// extracts the Retry-After header when present.
type GatewayError struct {
	StatusCode int
	Body       string
	Headers    http.Header
	Retryable  bool
}

func (e *GatewayError) Error() string {
	return fmt.Sprintf("gateway returned status %d: %s", e.StatusCode, e.Body)
}

// RetryAfter returns the duration to wait before retrying, based on the
// Retry-After header. Returns 0 if the header is absent or unparseable.
func (e *GatewayError) RetryAfter() time.Duration {
	if e.Headers == nil {
		return 0
	}
	ra := e.Headers.Get("Retry-After")
	if ra == "" {
		return 0
	}
	// Try parsing as seconds (most common for API rate limits).
	if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	// Try parsing as HTTP-date.
	if t, err := http.ParseTime(ra); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// IsContextLengthError returns true if the error body indicates the input
// exceeded the model's context window. These errors should not be retried.
func (e *GatewayError) IsContextLengthError() bool {
	lower := strings.ToLower(e.Body)
	return strings.Contains(lower, "context_length") ||
		strings.Contains(lower, "context window") ||
		strings.Contains(lower, "maximum context length") ||
		strings.Contains(lower, "token limit")
}

// NewGatewayError creates a GatewayError from an HTTP response, classifying
// whether the error is retryable based on the status code and response body.
func NewGatewayError(statusCode int, body string, headers http.Header) *GatewayError {
	ge := &GatewayError{
		StatusCode: statusCode,
		Body:       body,
		Headers:    headers,
	}

	switch {
	case statusCode == 429, statusCode == 529:
		// Rate limited or overloaded — retryable.
		ge.Retryable = true
	case statusCode >= 500:
		// Server errors are generally retryable.
		ge.Retryable = true
	case statusCode == 400:
		// 400 may be context_length_exceeded — not retryable.
		ge.Retryable = false
	case statusCode == 401, statusCode == 403:
		// Auth errors are never retryable.
		ge.Retryable = false
	default:
		ge.Retryable = false
	}

	// Context length errors are never retryable regardless of status code.
	if ge.IsContextLengthError() {
		ge.Retryable = false
	}

	return ge
}

// gatewayErrorBody is used to extract error details from gateway JSON responses.
type gatewayErrorBody struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// ExtractErrorMessage attempts to extract a human-readable error message from
// the gateway response body. Falls back to the raw body if parsing fails.
func ExtractErrorMessage(body string) string {
	var parsed gatewayErrorBody
	if err := json.Unmarshal([]byte(body), &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return body
}
