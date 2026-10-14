package agentep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a custom agent definition.
type DeleteAgentRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
}

// Deletes a custom agent.
//
// The agent is withdrawn from the API: it stops appearing in listings, no longer resolves by ID, and can no longer be run or modified. Runs it already produced are kept. OpenMRP's `system` agents cannot be deleted — disable one for your account with the Update Agent Status endpoint instead.
type DeleteAgentEndpoint struct{}

func (e *DeleteAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAgentRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAgentRequest, *object.Deleted]{
		Title:             "Delete Agent",
		Method:            http.MethodDelete,
		Route:             "/v1/ai/agents/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAgentRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AgentSvc).DeleteAgent, constants.ObjectTypeAgentDefinition, "id")
		},
	})
}
