package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an agent definition.
type RetrieveAgentRequest struct {
	// ID of the agent definition to retrieve.
	AgentDefinitionID string `path:"id" validate:"required"`
}

// Returns an agent definition by ID.
type RetrieveAgentEndpoint struct{}

func (e *RetrieveAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAgentRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*RetrieveAgentRequest, *apiresource.AgentDefinition]{
		Title:               "Retrieve Agent",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/agents/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentDefinition,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).GetAgent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
