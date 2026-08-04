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

// Request to list tool groups.
type ListToolGroupsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of the groups the agent tool catalog is organized into.
//
// The catalog is platform-defined and identical for every account. Pagination applies to the groups themselves, so a group requested with `include=tools` always carries its complete tool list regardless of the page limit. The `q` search term matches against group names.
type ListToolGroupsEndpoint struct{}

func (e *ListToolGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListToolGroupsRequest, *apiresource.List[apiresource.ToolGroup]] {
	return (&apiendpoint.APIEndpoint[*ListToolGroupsRequest, *apiresource.List[apiresource.ToolGroup]]{
		Title:             "List Tool Groups",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/tool-groups",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeToolGroup,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainAgents, Action: types.ActionRead},
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeToolGroup,
			Fields:     []string{"tools"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolGroupsRequest) (*apiresource.List[apiresource.ToolGroup], *apierror.APIError) {
			return svc.(AgentToolSvc).ListToolGroups
		},
	})
}
