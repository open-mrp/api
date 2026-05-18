package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list agent token usage records.
type ListUsageRequest struct {
	// Number of days of usage history to return. Defaults to 30.
	Days int32 `query:"days" default:"30" validate:"min=1,max=365"`
	// Maximum number of records to return per page. Defaults to 100.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
	// Pagination cursor from a previous response.
	Cursor *string `query:"cursor"`
}

// Returns a paginated list of daily agent token usage records for the current account.
type ListUsageEndpoint struct{}

func (e *ListUsageEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUsageRequest, *apiresource.List[apiresource.AgentTokenUsage]] {
	return (&apiendpoint.APIEndpoint[*ListUsageRequest, *apiresource.List[apiresource.AgentTokenUsage]]{
		Title:             "List Agent Usage",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/usage",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUsageRequest) (*apiresource.List[apiresource.AgentTokenUsage], *apierror.APIError) {
			return svc.(AgentSvc).ListUsage
		},
	})
}
