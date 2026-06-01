package agenttoolep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type ListToolsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of tools that can be assigned to agents.
type ListToolsEndpoint struct{}

func (e *ListToolsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListToolsRequest, *apiresource.List[apiresource.AvailableTool]] {
	return (&apiendpoint.APIEndpoint[*ListToolsRequest, *apiresource.List[apiresource.AvailableTool]]{
		Title:             "List Tools",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/tools",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAvailableTool,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolsRequest) (*apiresource.List[apiresource.AvailableTool], *apierror.APIError) {
			return svc.(AgentToolSvc).ListTools
		},
	})
}
