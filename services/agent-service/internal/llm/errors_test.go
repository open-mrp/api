package llm

import (
	"net/http"
	"testing"
	"time"
)

func TestNewGatewayError_429Retryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(429, `{"error":{"message":"rate limited"}}`, nil)
	if !ge.Retryable {
		t.Error("expected 429 to be retryable")
	}
	if ge.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", ge.StatusCode)
	}
}

func TestNewGatewayError_529Retryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(529, "overloaded", nil)
	if !ge.Retryable {
		t.Error("expected 529 to be retryable")
	}
}

func TestNewGatewayError_500Retryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(500, "internal server error", nil)
	if !ge.Retryable {
		t.Error("expected 500 to be retryable")
	}
}

func TestNewGatewayError_502Retryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(502, "bad gateway", nil)
	if !ge.Retryable {
		t.Error("expected 502 to be retryable")
	}
}

func TestNewGatewayError_400NotRetryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(400, "bad request", nil)
	if ge.Retryable {
		t.Error("expected 400 to not be retryable")
	}
}

func TestNewGatewayError_401NotRetryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(401, "unauthorized", nil)
	if ge.Retryable {
		t.Error("expected 401 to not be retryable")
	}
}

func TestNewGatewayError_403NotRetryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(403, "forbidden", nil)
	if ge.Retryable {
		t.Error("expected 403 to not be retryable")
	}
}

func TestNewGatewayError_ContextLengthNotRetryable(t *testing.T) {
	t.Parallel(
	// Even though 429 is normally retryable, context_length errors are never retryable.
	)

	ge := NewGatewayError(400, `{"error":{"message":"maximum context length exceeded"}}`, nil)
	if ge.Retryable {
		t.Error("expected context length error to not be retryable")
	}
	if !ge.IsContextLengthError() {
		t.Error("expected IsContextLengthError to return true")
	}
}

func TestNewGatewayError_ContextWindowNotRetryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(400, "this request exceeds the context window", nil)
	if ge.Retryable {
		t.Error("expected context window error to not be retryable")
	}
}

func TestNewGatewayError_PaymentRequiredIsBillingLimit(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(402, "payment required", nil)
	if !ge.IsBillingLimitError() {
		t.Error("expected 402 to be a billing-limit error")
	}
	if ge.Retryable {
		t.Error("expected billing-limit error to not be retryable")
	}
}

func TestNewGatewayError_BillingBody429NotRetryable(t *testing.T) {
	t.Parallel()
	// A 429 is normally retryable, but a billing/quota body reclassifies it as a non-retryable
	// account-wide limit so the run stops cleanly instead of failing over across every model.
	ge := NewGatewayError(429, `{"error":{"message":"You have exceeded your spending limit","type":"insufficient_quota"}}`, nil)
	if !ge.IsBillingLimitError() {
		t.Error("expected quota body to be a billing-limit error")
	}
	if ge.Retryable {
		t.Error("expected billing-limit 429 to not be retryable")
	}
}

func TestNewGatewayError_RateLimit429StillRetryable(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(429, `{"error":{"message":"rate limited","type":"rate_limit_error"}}`, nil)
	if ge.IsBillingLimitError() {
		t.Error("expected a plain rate-limit 429 to not be a billing-limit error")
	}
	if !ge.Retryable {
		t.Error("expected a plain rate-limit 429 to remain retryable")
	}
}

func TestGatewayError_RetryAfterSeconds(t *testing.T) {
	t.Parallel()
	headers := http.Header{}
	headers.Set("Retry-After", "5")
	ge := NewGatewayError(429, "rate limited", headers)

	ra := ge.RetryAfter()
	if ra != 5*time.Second {
		t.Errorf("expected 5s retry-after, got %v", ra)
	}
}

func TestGatewayError_RetryAfterMissing(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(429, "rate limited", nil)
	if ge.RetryAfter() != 0 {
		t.Error("expected 0 retry-after when header missing")
	}
}

func TestGatewayError_ErrorMessage(t *testing.T) {
	t.Parallel()
	ge := NewGatewayError(500, "oops", nil)
	expected := "gateway returned status 500: oops"
	if ge.Error() != expected {
		t.Errorf("expected %q, got %q", expected, ge.Error())
	}
}

func TestExtractErrorMessage_JSON(t *testing.T) {
	t.Parallel()
	body := `{"error":{"message":"rate limited","type":"rate_limit_error"}}`
	msg := ExtractErrorMessage(body)
	if msg != "rate limited" {
		t.Errorf("expected 'rate limited', got %q", msg)
	}
}

func TestExtractErrorMessage_PlainText(t *testing.T) {
	t.Parallel()
	body := "plain text error"
	msg := ExtractErrorMessage(body)
	if msg != body {
		t.Errorf("expected %q, got %q", body, msg)
	}
}
