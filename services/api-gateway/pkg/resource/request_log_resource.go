package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRequestLogID = "rq_01304bffe90e8cce9690cbefd4"
const SampleRequestLogHost = "https://api.augno.com"
const SampleRequestLogPath = "/v1/core/sandboxes"
const SampleRequestLogQueryJSON = `{"limit":10}`
const SampleRequestLogAPIVersion = "2026-01-01"
const SampleRequestLogClientIP = "198.51.100.7"
const SampleRequestLogUserAgent = "Mozilla/5.0"
const SampleRequestLogResponseBody = `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null,"has_next_page":false,"has_prev_page":false},"data":[]}`

// A log of a single API request, capturing its route, outcome, latency, and actor.
type RequestLog struct {
	// Request log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=request_log"`
	// HTTP method.
	Method string `json:"method" validate:"required"`
	// Request host.
	//
	// Usually `api.augno.com`.
	Host string `json:"host" validate:"required"`
	// The exact path the request was made to, including path parameter values.
	Path string `json:"path" validate:"required"`
	// The route template the request matched, with path parameters left as placeholders.
	//
	// For example `/v1/sales/customers/{id}` is the normalized route for the request path `/v1/sales/customers/ac_...`. Falls back to the raw path when the request did not match a registered route.
	NormalizedRoute string `json:"normalized_route" validate:"required"`
	// Query parameters.
	QueryJSON json.RawMessage `json:"query_params" expandable:"true"`
	// HTTP response status code (e.g. `200`, `404`).
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
	// Machine-readable API error code.
	//
	// Populated only for failed requests.
	ErrorCode *string `json:"error_code"`
	// Human-readable error message.
	//
	// Populated only for failed requests.
	ErrorMessage *string `json:"error_message"`
	// When the request occurred.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// When the log entry was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Account _targeted_ by the request: the account the request acted upon.
	//
	// Results are scoped to logs where your account is either the acting account or the target account. Use the `target_account_ids` query parameter to filter by which account was acted upon, and `actor_account_ids` to filter by who acted.
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
	ID:        SampleUserID,
	Object:    constants.ObjectTypeActor,
	Type:      constants.ActorTypeUser,
	Name:      new(SampleUserName),
	Handle:    new(SampleUserEmail),
	AvatarURL: new(SampleUserImageUrl),
	Role:      SampleRole,
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
	Referrer:         new("https://www.augno.com"),
	OccurredAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Account:          SampleAccount,
	Actor:            SampleRequestLogActor,
	ResponseBodyJSON: json.RawMessage(SampleRequestLogResponseBody),
}

func (*RequestLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRequestLog)
}

// RedactedRequestLogHost is the host shown for internal/agent request logs in
// customer-facing responses, so the gateway's internal listener hostname (a k8s
// service name:port) never leaks. We surface the public API host the agent's call
// is logically equivalent to, rather than a meaningless "internal" placeholder. The
// true internal host stays in platform-service storage for operator debugging.
const RedactedRequestLogHost = "https://api.augno.com"

// internalListenerIdentityType is the request_log.identity_type stamped on requests
// authenticated by the gateway's trusted internal listener (agents). It is the
// signal that a log may carry internal infrastructure details (internal host, pod IP).
const internalListenerIdentityType = "agent"

// ScrubInternalInfra blanks fields that expose internal infrastructure — the
// internal listener host and the in-cluster client (pod) IP — when the log
// represents an internal/agent request (identityType == "agent"). Every
// customer-facing presenter that renders a request_log MUST call this; the real
// values remain in platform-service storage for operators. No-op for external
// (user/api_key) requests, whose host/IP are legitimately customer-visible.
func (r *RequestLog) ScrubInternalInfra(identityType *string) {
	if r == nil || identityType == nil || *identityType != internalListenerIdentityType {
		return
	}
	r.Host = RedactedRequestLogHost
	r.ClientIP = nil
}
