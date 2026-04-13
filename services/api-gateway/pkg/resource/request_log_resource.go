package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRequestLogID = "rq_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleRequestLogHost = "https://api.augno.com"
const SampleRequestLogPath = "/v1/core/sandboxes"
const SampleRequestLogQueryJSON = `{"limit":10}`
const SampleRequestLogAPIVersion = "2026-01-01"
const SampleRequestLogIdentityType = "user"
const SampleRequestLogClientIP = "198.51.100.7"
const SampleRequestLogUserAgent = "Mozilla/5.0"
const SampleRequestLogResponseBody = `{"object":"list","data":[...]}`

// RequestLog is an API request log entry.
type RequestLog struct {
	// Request log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=request_log"`
	// HTTP method.
	Method string `json:"method" validate:"required"`
	// Request host.
	Host string `json:"host" validate:"required"`
	// Request path.
	Path string `json:"path" validate:"required"`
	// Normalized route pattern.
	NormalizedRoute string `json:"normalized_route" validate:"required"`
	// Query parameters as JSON.
	QueryJSON *string `json:"query_json" expandable:"true"`
	// HTTP status code.
	StatusCode int32 `json:"status_code" validate:"required"`
	// Request latency in microseconds.
	LatencyUs int64 `json:"latency_us" validate:"required"`
	// API version used.
	APIVersion *string `json:"api_version"`
	// Caller identity type.
	IdentityType *string `json:"identity_type"`
	// Client IP address.
	ClientIP *string `json:"client_ip"`
	// User agent string.
	UserAgent *string `json:"user_agent"`
	// Referrer header.
	Referrer *string `json:"referrer"`
	// API error code.
	ErrorCode *string `json:"error_code"`
	// Error message.
	ErrorMessage *string `json:"error_message"`
	// When the request occurred.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// When the log entry was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Account targeted by the request.
	Account *Account `json:"account" expandable:"true"`
	// Actor details (user or API key).
	Actor *Actor `json:"actor" expandable:"true"`
	// User-provided idempotency key.
	IdempotencyKey *string `json:"idempotency_key"`
	// Request body as JSON.
	RequestBodyJSON *string `json:"request_body_json" expandable:"true"`
	// Response body as JSON.
	ResponseBodyJSON *string `json:"response_body_json" expandable:"true"`
}

var SampleRequestLogActor = &Actor{
	ID:     SampleUserID,
	Object: constants.ObjectTypeActor,
	Type:   constants.ActorTypeUser,
	Name:   new(SampleUserName),
	Handle: new(SampleUserEmail),
	Role:   SampleRole,
}

var SampleRequestLog = &RequestLog{
	ID:               SampleRequestLogID,
	Object:           constants.ObjectTypeRequestLog,
	Method:           "GET",
	Host:             SampleRequestLogHost,
	Path:             SampleRequestLogPath,
	NormalizedRoute:  SampleRequestLogPath,
	QueryJSON:        new(SampleRequestLogQueryJSON),
	StatusCode:       200,
	LatencyUs:        12345,
	APIVersion:       new(SampleRequestLogAPIVersion),
	IdentityType:     new(SampleRequestLogIdentityType),
	ClientIP:         new(SampleRequestLogClientIP),
	UserAgent:        new(SampleRequestLogUserAgent),
	OccurredAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Account:          SampleAccount,
	Actor:            SampleRequestLogActor,
	ResponseBodyJSON: new(SampleRequestLogResponseBody),
}

func (*RequestLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRequestLog)
}
