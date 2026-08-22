package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an agent definition.
type RetrieveAgentRequest struct {
	// ID of the agent definition to retrieve.
	AgentDefinitionID string `path:"id" validate:"required"`
}

// Retrieves a single agent by ID.
//
// Resolves both the `system` agents OpenMRP provides and the `custom` agents in your account; the `status` reflects whether the agent is enabled for your account specifically.
type RetrieveAgentEndpoint struct{}

func (e *RetrieveAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAgentRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*RetrieveAgentRequest, *apiresource.AgentDefinition]{
		Title:               "Retrieve Agent",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/agents/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
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
