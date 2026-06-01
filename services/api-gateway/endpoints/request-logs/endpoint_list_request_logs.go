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

// ListRequestLogsRequest is a request to list request logs.
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
	// Filter by API error code.
	ErrorCodes []apierror.ErrorCode `query:"error_codes"`
	// Filter by the account ID _targeted_ by the request. The actor may be operating on behalf of a separate account.
	AccountIDs []string `query:"account_ids"`
	// Filter by the actor identifier. `account_user.id` when `identity_type`=`user`, or an `api_key.id` when `identity_type`=`api_key`.
	ActorIDs []string `query:"actor_ids"`
	// Filter by the actor type.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Filter by the _normalized_ route template. For example `PATCH /v1/sales/customers/{id}` is the normalized route for a request route `PUT /v1/sales/customers/ac_...`.
	NormalizedRoutes []string `query:"normalized_routes"`
	// Filter by the request host. Typically, `api.augno.com`.
	Hosts []string `query:"hosts"`
	// Filter by the minimum latency in microseconds.
	MinLatencyUs *int64 `query:"min_latency_us"`
	// Filter by the user-provided idempotency key.
	IdempotencyKey *string `query:"idempotency_key"`
}

// Returns a paginated list of request logs.
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
