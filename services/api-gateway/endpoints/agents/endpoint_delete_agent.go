package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a custom agent definition.
type DeleteAgentRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
}

// Deletes a custom agent definition.
//
// The agent is soft-deleted and can no longer be run or modified. System agents cannot be deleted.
type DeleteAgentEndpoint struct{}

func (e *DeleteAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAgentRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAgentRequest, *apiresource.EmptyResource]{
		Title:               "Delete Agent",
		Method:              http.MethodDelete,
		Route:               "/v1/ai/agents/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAgentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AgentSvc).DeleteAgent
		},
	})
}
