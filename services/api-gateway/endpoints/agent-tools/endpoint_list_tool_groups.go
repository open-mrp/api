package agenttoolep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list tool groups.
type ListToolGroupsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeToolGroup,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainAgents, Action: types.ActionRead},
		}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeToolGroup,
			Fields:     []string{"tools"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolGroupsRequest) (*apiresource.List[apiresource.ToolGroup], *apierror.APIError) {
			return svc.(AgentToolSvc).ListToolGroups
		},
	})
}
