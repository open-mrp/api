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
	// Filter by account-level status.
	Status []constants.AgentAccountStatus `query:"statuses" default:"active"`
	// Filter by definition type.
	DefinitionType []constants.AgentDefinitionType `query:"definition_types"`
	// Filter by trigger type.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAgentsRequest) (*apiresource.List[apiresource.AgentDefinition], *apierror.APIError) {
			return svc.(AgentSvc).ListAgents
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
