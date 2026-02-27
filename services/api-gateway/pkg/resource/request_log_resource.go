package apiresource

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// RequestLogActor contains the resolved actor details for a request log.
type RequestLogActor struct {
	// The actor's ID (user ID or API key type_id).
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	ObjectType constants.ObjectType `json:"object_type" validate:"required,enum=user,api_key"`
	// The actor's display name.
	Name *string `json:"name"`
	// The actor's email (users only).
	Email *string `json:"email"`
	// The redacted API key value (API keys only).
	RedactedValue *string `json:"redacted_value"`
	// The role assigned to the actor.
	Role *LightRole `json:"role"`
}

// RequestLog represents a single API request log entry.
type RequestLog struct {
	// The unique identifier for the request log.
	ID string `json:"id" validate:"required"`
	// The object type.
	Object constants.ObjectType `json:"object" validate:"required,enum=request_log"`
	// The HTTP method.
	Method string `json:"method" validate:"required"`
	// The request host.
	Host string `json:"host" validate:"required"`
	// The request path.
	Path string `json:"path" validate:"required"`
	// The normalized route pattern.
	NormalizedRoute string `json:"normalized_route" validate:"required"`
	// The query parameters as JSON.
	QueryJSON *string `json:"query_json"`
	// The HTTP status code.
	StatusCode int32 `json:"status_code" validate:"required"`
	// The request latency in microseconds.
	LatencyUs int64 `json:"latency_us" validate:"required"`
	// The API version used.
	APIVersion *string `json:"api_version"`
	// The identity type of the caller.
	IdentityType *string `json:"identity_type"`
	// The client IP address.
	ClientIP *string `json:"client_ip"`
	// The user agent string.
	UserAgent *string `json:"user_agent"`
	// The referrer header.
	Referrer *string `json:"referrer"`
	// The API error code, if any.
	ErrorCode *string `json:"error_code"`
	// The error message, if any.
	ErrorMessage *string `json:"error_message"`
	// When the request occurred.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// When the log entry was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The account targeted by the request.
	Account *LightAccount `json:"account"`
	// Actor details (user or API key).
	Actor *RequestLogActor `json:"actor"`
	// The user-provided idempotency key value.
	IdempotencyKey *string `json:"idempotency_key"`
	// The JSON request body.
	RequestBodyJSON *string `json:"request_body_json"`
	// The JSON response body.
	ResponseBodyJSON *string `json:"response_body_json"`
}

func (*RequestLog) SchemaExample() any {
	return &RequestLog{
		ID:              "rl_01jm4r6700f8nwq3v5hx2d9ktp",
		Object:          constants.ObjectTypeRequestLog,
		Method:          "GET",
		Path:            "/v1/core/sandboxes",
		NormalizedRoute: "/v1/core/sandboxes",
		StatusCode:      200,
		LatencyUs:       12345,
		OccurredAt:      time.Now(),
		CreatedAt:       time.Now(),
	}
}
