package appctx

import (
	"context"
	"time"
)

const requestLogKey contextKey = "request_log"

// RequestLog captures metadata about an HTTP request for logging and auditing.
type RequestLog struct {
	ID                   string    `json:"id"`
	Method               string    `json:"method"`
	Host                 string    `json:"host"`
	Path                 string    `json:"path"`
	NormalizedRoute      string    `json:"normalized_route"`
	QueryJSON            *string   `json:"query_json,omitempty"`
	StatusCode           int       `json:"status_code"`
	LatencyUs            int64     `json:"latency_us"`
	AccountID            *string   `json:"account_id,omitempty"`
	TargetAccountID      *string   `json:"target_account_id,omitempty"`
	ActorID              *string   `json:"actor_id,omitempty"`
	ActorType            *string   `json:"actor_type"`
	IdentityType         *string   `json:"identity_type"`
	ClientIP             []byte    `json:"client_ip"`
	ClientIPString       *string   `json:"client_ip_string"`
	UserAgent            *string   `json:"user_agent"`
	Referrer             *string   `json:"referrer,omitempty"`
	ErrorCode            *string   `json:"error_code,omitempty"`
	ErrorMessage         *string   `json:"error_message,omitempty"`
	OccurredAt           time.Time `json:"occurred_at"`
	IdempotencyKeyID     *string   `json:"idempotency_key_id"`
	InternalErrorMessage *string   `json:"internal_error_message,omitempty"`
	StackTrace           *string   `json:"stack_trace,omitempty"`
	APIVersion           *string   `json:"api_version,omitempty"`
	TraceID              *string   `json:"trace_id,omitempty"`
}

// WithRequestLog returns a child context carrying the request log.
func WithRequestLog(ctx context.Context, rl *RequestLog) context.Context {
	return context.WithValue(ctx, requestLogKey, rl)
}

// GetRequestLog retrieves the request log from the context.
func GetRequestLog(ctx context.Context) (*RequestLog, bool) {
	rl, ok := ctx.Value(requestLogKey).(*RequestLog)
	return rl, ok && rl != nil
}
