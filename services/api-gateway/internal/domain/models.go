package domain

import "time"

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
}
