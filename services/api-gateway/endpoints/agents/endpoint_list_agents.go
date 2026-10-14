package agentep

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

// Request to list agent definitions.
type ListAgentsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Restricts results to agents with one of the given account-level statuses.
	//
	// `inactive` also matches agents that have never been enabled for your account.
	Status []constants.AgentAccountStatus `query:"statuses"`
	// Restricts results to agents of one of the given definition types.
	DefinitionType []constants.AgentDefinitionType `query:"definition_types"`
	// Restricts results to agents with one of the given trigger types.
	TriggerType []constants.AgentTriggerType `query:"trigger_types"`
}

// Lists the agents available to your account, newest first.
//
// Covers both the `system` agents OpenMRP provides to every account and the `custom` agents created in yours. Deleted agents are never returned. The `q` parameter matches an agent's name, slug, description, or ID.
type ListAgentsEndpoint struct{}

func (e *ListAgentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAgentsRequest, *apiresource.List[apiresource.AgentDefinition]] {
	return (&apiendpoint.APIEndpoint[*ListAgentsRequest, *apiresource.List[apiresource.AgentDefinition]]{
		Title:             "List Agents",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/agents",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentDefinition,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError) {
			return svc.(AgentSvc).ListAgents
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
