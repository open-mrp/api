package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list agent definitions.
type ListAgentsRequest struct {
	apiresource.PaginationRequest
	// Restricts results to agents with one of the given account-level statuses.
	//
	// Omit to return agents of every status; repeat the parameter to match more than one status.
	Status []constants.AgentAccountStatus `query:"statuses"`
	// Restricts results to agents of one of the given definition types.
	DefinitionType []constants.AgentDefinitionType `query:"definition_types"`
	// Restricts results to agents with one of the given trigger types.
	TriggerType []constants.AgentTriggerType `query:"trigger_types"`
}

// Returns a paginated list of agent definitions for the current account.
type ListAgentsEndpoint struct{}

func (e *ListAgentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAgentsRequest, *apiresource.List[apiresource.AgentDefinition]] {
	return (&apiendpoint.APIEndpoint[*ListAgentsRequest, *apiresource.List[apiresource.AgentDefinition]]{
		Title:             "List Agents",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/agents",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentDefinition,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError) {
			return svc.(AgentSvc).ListAgents
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
