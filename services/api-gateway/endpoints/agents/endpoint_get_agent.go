package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAgentRequest is the request to retrieve a single agent definition.
type GetAgentRequest struct {
	// The ID of the agent definition to retrieve.
	AgentDefinitionID string `path:"id" validate:"required"`
}

type GetAgentEndpoint struct{}

func (e *GetAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAgentRequest, *apiresource.AgentDefinition] {
	return &apiendpoint.APIEndpoint[*GetAgentRequest, *apiresource.AgentDefinition]{
		Title:             "Get Agent",
		Description:       "Returns a single agent definition with its tool configuration.",
		Method:            http.MethodGet,
		Route:             "/v1/ai/agents/{id}",
		Request:           &GetAgentRequest{},
		Response:          &apiresource.AgentDefinition{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).GetAgent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	}
}
