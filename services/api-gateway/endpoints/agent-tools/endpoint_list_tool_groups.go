package agenttoolep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type ListToolGroupsRequest struct {
	apiresource.PaginationRequest
}

type ListToolGroupsEndpoint struct{}

func (e *ListToolGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListToolGroupsRequest, *apiresource.List[apiresource.ToolGroup]] {
	return &apiendpoint.APIEndpoint[*ListToolGroupsRequest, *apiresource.List[apiresource.ToolGroup]]{
		Title:             "List Tool Groups",
		Description:       "Returns all tool groups used to organize available platform tools.",
		Method:            http.MethodGet,
		Route:             "/v1/ai/tool-groups",
		Request:           &ListToolGroupsRequest{},
		Response:          apiresource.NewList[apiresource.ToolGroup](nil, apiresource.PageInfo{}),
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeToolGroup,
			Fields:     []string{"tools"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolGroupsRequest) (*apiresource.List[apiresource.ToolGroup], *apierror.APIError) {
			return svc.(AgentToolSvc).ListToolGroups
		},
	}
}
