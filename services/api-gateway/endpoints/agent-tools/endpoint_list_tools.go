package agenttoolep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type ListToolsRequest struct {
	apiresource.PaginationRequest
}

type ListToolsEndpoint struct{}

func (e *ListToolsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListToolsRequest, *apiresource.List[apiresource.AvailableTool]] {
	return &apiendpoint.APIEndpoint[*ListToolsRequest, *apiresource.List[apiresource.AvailableTool]]{
		Title:             "List Tools",
		Description:       "Returns all available platform tools that can be assigned to agents.",
		Method:            http.MethodGet,
		Route:             "/v1/ai/tools",
		Request:           &ListToolsRequest{},
		Response:          &apiresource.List[apiresource.AvailableTool]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolsRequest) (*apiresource.List[apiresource.AvailableTool], *apierror.APIError) {
			return svc.(AgentToolSvc).ListTools
		},
	}
}
