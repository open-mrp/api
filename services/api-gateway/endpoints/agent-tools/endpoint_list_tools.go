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

// Request to list the tools available to agents.
type ListToolsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAvailableTool,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainAgents, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListToolsRequest) (*apiresource.List[apiresource.AvailableTool], *apierror.APIError) {
			return svc.(AgentToolSvc).ListTools
		},
	})
}
