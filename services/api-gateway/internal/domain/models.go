package domain

import "time"

type RequestLog struct {
	ID             string `json:"id"`
	UserAgent      string `json:"user_agent"`
	IdempotencyKey string `json:"idempotency_key"`
	ClientIP       []byte `json:"client_ip"`
	ClientIPString string `json:"client_ip_string"`
	// HTTP
	Method          string `json:"method"`
	Host            string `json:"host"`
	Path            string `json:"path"`
	NormalizedRoute string `json:"normalized_route"`
	QueryJSON       string `json:"query_json,omitempty"`
	StatusCode      int    `json:"status_code"`
	Latency         int64  `json:"latency_ms"`
	// Principles
	IdentityType string `json:"identity_type"`
	AccountID    string `json:"account_id,omitempty"`
	ActorID      string `json:"actor_id,omitempty"`
	ActorType    string `json:"actor_type"`
	// Networking
	Referrer string `json:"referrer,omitempty"`
	// Error
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	// Timestamps
	OccurredAt time.Time `json:"occurred_at"`
	// Internal
	InternalErrorMessage string `json:"internal_error_message,omitempty"`
	StackTrace           string `json:"stack_trace,omitempty"`
}
