package requestlogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list request logs.
type ListRequestLogsRequest struct {
	apiresource.PaginationRequest
	// Restricts results to request logs on or after this timestamp.
	StartDate *time.Time `query:"start_date"`
	// Restricts results to request logs on or before this timestamp.
	EndDate *time.Time `query:"end_date"`
	// Filter by the HTTP method.
	Methods []constants.HTTPMethod `query:"methods"`
	// Filter by the HTTP status code.
	StatusCodes []int32 `query:"status_codes"`
	// Filter by the HTTP status class, expressed as the leading digit: `1`–`5` for 1xx–5xx.
	//
	// Combined with `status_codes` using OR — e.g. `status_codes=401` and `status_code_classes=5` matches 401 responses and any 5xx response.
	StatusCodeClasses []int32 `query:"status_code_classes"`
	// Filter by API error code.
	ErrorCodes []apierror.ErrorCode `query:"error_codes"`
	// Filter by the _acting_ account: the account the actor belongs to, not the account targeted by the request.
	//
	// This is usually your own account, but differs when another account acts on yours — for example a customer using a customer-portal API key, whose acting account is the customer's account. The request's target account is always your own account (the only account you are authorized to view request logs for), so this filter narrows by _who acted_, not by which account was acted upon.
	AccountIDs []string `query:"account_ids"`
	// Filter by the actor identifier.
	//
	// Matches the log's `actor.id`: a user ID for `user` actors or an API key ID for `api_key` actors.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the actor type.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Filter by the _normalized_ route template.
	//
	// For example `/v1/sales/customers/{id}` matches every request to that route regardless of the specific customer ID. Parameter names inside `{}` are ignored when matching, so `{customer_id}` and `{id}` are equivalent.
	NormalizedRoutes []string `query:"normalized_routes"`
	// Filter by the request host.
	//
	// Typically `api.augno.com`.
	Hosts []string `query:"hosts"`
	// Restricts results to requests that took at least this many microseconds.
	MinLatencyUs *int64 `query:"min_latency_us"`
	// Filter by the user-provided idempotency key.
	IdempotencyKey *string `query:"idempotency_key"`
}

// Returns a paginated list of request logs for the current account.
type ListRequestLogsEndpoint struct{}

func (e *ListRequestLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]] {
	return (&apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]]{
		Title:             "List Request Logs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/request-logs",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
			return svc.(RequestLogSvc).ListRequestLogs
		},
		ObjectType: constants.ObjectTypeRequestLog,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRequestLog,
			Fields:     []string{"account", "actor", "actor.role"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
	})
}
