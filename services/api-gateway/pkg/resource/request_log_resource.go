package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRequestLogID = "rl_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleRequestLogHost = "https://api.augno.com"
const SampleRequestLogPath = "/v1/core/sandboxes"
const SampleRequestLogQueryJSON = `{"limit":10}`
const SampleRequestLogAPIVersion = "2026-01-01"
const SampleRequestLogIdentityType = "user"
const SampleRequestLogClientIP = "198.51.100.7"
const SampleRequestLogUserAgent = "Mozilla/5.0"
const SampleRequestLogResponseBody = `{"object":"list","data":[...]}`

var SampleRequestLogActor = &RequestLogActor{
	ID:     SampleUserID,
	Object: constants.ObjectTypeUser,
	Name:   new(SampleUserName),
	Email:  new(SampleUserEmail),
	Role:   SampleLightRole,
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
	Account:          SampleLightAccount,
	Actor:            SampleRequestLogActor,
	ResponseBodyJSON: new(SampleRequestLogResponseBody),
}

var SampleRequestLogListItem = &RequestLogListItem{
	ID:              SampleRequestLogID,
	Object:          constants.ObjectTypeRequestLog,
	Method:          "GET",
	Host:            SampleRequestLogHost,
	Path:            SampleRequestLogPath,
	NormalizedRoute: SampleRequestLogPath,
	QueryJSON:       new(SampleRequestLogQueryJSON),
	StatusCode:      200,
	LatencyUs:       12345,
	APIVersion:      new(SampleRequestLogAPIVersion),
	IdentityType:    new(SampleRequestLogIdentityType),
	ClientIP:        new(SampleRequestLogClientIP),
	UserAgent:       new(SampleRequestLogUserAgent),
	OccurredAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Account:         SampleLightAccount,
	Actor:           SampleRequestLogActor,
}

// RequestLogActor contains the resolved actor details for a request log.
type RequestLogActor struct {
	// The actor's ID (user ID or API key type_id).
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user,api_key"`
	// The actor's display name.
	Name *string `json:"name"`
	// The actor's email (users only).
	Email *string `json:"email"`
	// The redacted API key value (API keys only).
	RedactedValue *string `json:"redacted_value"`
	// The role assigned to the actor. Expandable.
	Role *LightRole `json:"role" expandable:"true"`
}

// RequestLog represents a single API request log entry.
type RequestLog struct {
	// The unique identifier for the request log.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
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
	// The account targeted by the request. Expandable.
	Account *LightAccount `json:"account" expandable:"true"`
	// Actor details (user or API key). Expandable.
	Actor *RequestLogActor `json:"actor" expandable:"true"`
	// The user-provided idempotency key value.
	IdempotencyKey *string `json:"idempotency_key"`
	// The JSON request body.
	RequestBodyJSON *string `json:"request_body_json"`
	// The JSON response body.
	ResponseBodyJSON *string `json:"response_body_json"`
}

func (*RequestLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRequestLog)
}

// RequestLogListItem is the list representation of a request log entry.
// It omits the request and response body JSON fields which are only
// available when retrieving a single request log by ID.
type RequestLogListItem struct {
	// The unique identifier for the request log.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
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
	// The account targeted by the request. Expandable.
	Account *LightAccount `json:"account" expandable:"true"`
	// Actor details (user or API key). Expandable.
	Actor *RequestLogActor `json:"actor" expandable:"true"`
	// The user-provided idempotency key value.
	IdempotencyKey *string `json:"idempotency_key"`
}

func (*RequestLogListItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRequestLogListItem)
}
