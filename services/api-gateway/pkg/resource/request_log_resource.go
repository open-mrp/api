package apiresource

import (
	"encoding/json"
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
const SampleRequestLogClientIP = "198.51.100.7"
const SampleRequestLogUserAgent = "Mozilla/5.0"
const SampleRequestLogResponseBody = `{"object":"list","page_info":{"next_cursor":null,"prev_cursor":null,"has_next_page":false,"has_prev_page":false},"data":[]}`

// RequestLog is an API request log entry.
type RequestLog struct {
	// Request log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=request_log"`
	// HTTP method.
	Method string `json:"method" validate:"required"`
	// Request host. Usually `api.augno.com`.
	Host string `json:"host" validate:"required"`
	// Non-normalized request path.
	Path string `json:"path" validate:"required"`
	// _Normalized_ route template. For example `PATCH /v1/sales/customers/{id}` is the normalized route for a request route `PUT /v1/sales/customers/ac_...`.
	NormalizedRoute string `json:"normalized_route" validate:"required"`
	// Query parameters.
	QueryJSON json.RawMessage `json:"query_params" expandable:"true"`
	// HTTP status code.
	StatusCode int32 `json:"status_code" validate:"required"`
	// Request latency in microseconds.
	LatencyUs int64 `json:"latency_us" validate:"required"`
	// API version used.
	APIVersion *string `json:"api_version"`
	// Client IP address.
	ClientIP *string `json:"client_ip"`
	// User agent.
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
	// Account _targeted_ by the request.
	Account *Account `json:"account" expandable:"true"`
	// Actor who made the request.
	Actor *Actor `json:"actor" expandable:"true"`
	// User-provided idempotency key.
	IdempotencyKey *string `json:"idempotency_key"`
	// Request body.
	RequestBodyJSON json.RawMessage `json:"request_body" expandable:"true"`
	// Response body.
	ResponseBodyJSON json.RawMessage `json:"response_body" expandable:"true"`
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
	QueryJSON:        json.RawMessage(SampleRequestLogQueryJSON),
	StatusCode:       200,
	LatencyUs:        12345,
	APIVersion:       new(SampleRequestLogAPIVersion),
	ClientIP:         new(SampleRequestLogClientIP),
	UserAgent:        new(SampleRequestLogUserAgent),
	OccurredAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Account:          SampleAccount,
	Actor:            SampleRequestLogActor,
	ResponseBodyJSON: json.RawMessage(SampleRequestLogResponseBody),
}

func (*RequestLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRequestLog)
}
