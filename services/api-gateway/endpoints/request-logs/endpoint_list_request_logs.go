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
	// Filter: start of date range for occurred_at.
	StartDate *time.Time `query:"start_date"`
	// Filter: end of date range for occurred_at.
	EndDate *time.Time `query:"end_date"`
	// Filter: HTTP methods (repeatable, exact match).
	Methods []string `query:"method"`
	// Filter: HTTP status codes (repeatable, exact match).
	StatusCodes []int32 `query:"status_code"`
	// Filter: API error codes (repeatable, exact match).
	ErrorCodes []string `query:"error_code"`
	// Filter: actor home account IDs (repeatable, exact match).
	AccountIDs []string `query:"account_id"`
	// Filter: actor IDs (repeatable, exact match). Each value is a user.id when
	// identity_type=user, or an api_key.type_id when identity_type=api_key —
	// not an account_user.id.
	ActorIDs []string `query:"actor_id"`
	// Filter: actor types (repeatable, exact match — "user" or "api_key").
	ActorTypes []string `query:"actor_type"`
	// Filter: normalized route templates (repeatable, exact match).
	NormalizedRoutes []string `query:"normalized_route"`
	// Filter: request hosts (repeatable, exact match).
	Hosts []string `query:"host"`
	// Filter: minimum latency in microseconds.
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
