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

// ListRequestLogsRequest is the request to list request logs with filters.
type ListRequestLogsRequest struct {
	apiresource.PaginationRequest
	// Filter: start of date range for occurred_at.
	StartDate *time.Time `query:"start_date"`
	// Filter: end of date range for occurred_at.
	EndDate *time.Time `query:"end_date"`
	// Filter: HTTP method.
	Method *string `query:"method"`
	// Filter: HTTP status code.
	StatusCode *int32 `query:"status_code"`
	// Filter: API error code.
	ErrorCode *string `query:"error_code"`
	// Filter: actor's home account ID.
	AccountID *string `query:"account_id"`
	// Filter: actor ID.
	ActorID *string `query:"actor_id"`
	// Filter: actor type ("user" or "api_key").
	ActorType *string `query:"actor_type"`
	// Filter: actor name (partial or exact match).
	ActorName *string `query:"actor_name"`
	// When true, string filters use exact match instead of partial (LIKE).
	ExactMatch *bool `query:"exact_match"`
}

type ListRequestLogsEndpoint struct{}

func (e *ListRequestLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]] {
	return &apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]]{
		Title:             "List Request Logs",
		Description:       "Returns a paginated, filterable list of request logs for the target account.",
		Method:            http.MethodGet,
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
