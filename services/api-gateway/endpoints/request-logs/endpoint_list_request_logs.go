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
	// Start of date range for occurred_at.
	StartDate *time.Time `query:"start_date"`
	// End of date range for occurred_at.
	EndDate *time.Time `query:"end_date"`
	// HTTP methods.
	Methods []constants.HTTPMethod `query:"methods"`
	// HTTP status codes.
	StatusCodes []int32 `query:"status_codes"`
	// API error codes.
	ErrorCodes []apierror.ErrorCode `query:"error_codes"`
	// Actor home account IDs.
	AccountIDs []string `query:"account_ids"`
	// Actor identifier. `user.id` when `identity_type`=`user`, or an `api_key.id` when `identity_type`=`api_key`.
	ActorIDs []string `query:"actor_ids"`
	// Actor types.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Normalized route templates.
	NormalizedRoutes []string `query:"normalized_routes"`
	// Request hosts.
	Hosts []string `query:"hosts"`
	// Minimum latency in microseconds.
	MinLatencyUs *int64 `query:"min_latency_us"`
}

type ListRequestLogsEndpoint struct{}

func (e *ListRequestLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]] {
	return &apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]]{
		Title:             "List Request Logs",
		Description:       "Returns a paginated list of request logs.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/request-logs",
		Request:           &ListRequestLogsRequest{},
		Response:          &apiresource.List[apiresource.RequestLog]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRequestLog,
			Fields:     []string{"account", "actor", "actor.role"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
			return svc.(RequestLogSvc).ListRequestLogs
		},
	}
}
