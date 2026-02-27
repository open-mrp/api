package requestlogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListRequestLogsRequest is the request to list request logs with filters.
type ListRequestLogsRequest struct {
	// Cursor for fetching the next page, from a previous response's next_cursor field.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return per page (default: 100, max: 1000).
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
	// Search query: matches against ID (exact), path (partial), or error message (partial).
	Query *string `query:"q"`
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

const listRequestLogsEndpointDescription string = `This endpoint returns a paginated, filterable list of request logs for the target account.
Supports cursor-based pagination and various filters.`

type ListRequestLogsEndpoint struct{}

func (e *ListRequestLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]] {
	return &apiendpoint.APIEndpoint[*ListRequestLogsRequest, *apiresource.List[apiresource.RequestLog]]{
		Title:             "List Request Logs",
		Description:       listRequestLogsEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/core/request-logs",
		Request:           &ListRequestLogsRequest{},
		Response:          &apiresource.List[apiresource.RequestLog]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging:     true,
			AllowUnknownJSONFields: false,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLog], *apierror.APIError) {
			return svc.(RequestLogSvc).ListRequestLogs
		},
	}
}
