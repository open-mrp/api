package agenttoolep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the tools available to agents.
type ListToolsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of every capability that can be granted to an agent.
//
// The catalog is platform-defined and identical for every account, and covers both built-in runtime capabilities and the API operations agents are allowed to perform. The `q` search term matches against tool names and the name of the group a tool belongs to.
type ListToolsEndpoint struct{}

func (e *ListToolsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListToolsRequest, *apiresource.List[apiresource.AvailableTool]] {
	return (&apiendpoint.APIEndpoint[*ListToolsRequest, *apiresource.List[apiresource.AvailableTool]]{
		Title:             "List Tools",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/tools",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAvailableTool,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainAgents, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolsRequest) (*apiresource.List[apiresource.AvailableTool], *apierror.APIError) {
			return svc.(AgentToolSvc).ListTools
		},
	})
}
