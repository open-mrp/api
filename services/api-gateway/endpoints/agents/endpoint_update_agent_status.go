package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to update the per-account status of an agent.
type UpdateAgentStatusRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
	// Account-level status to set for the agent.
	//
	// Either `active` or `inactive`.
	Status string `json:"status" validate:"required,max=255"`
}

var sampleUpdateAgentStatusRequest = &UpdateAgentStatusRequest{
	Status: "active",
}

func (*UpdateAgentStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAgentStatusRequest)
}

// Enables or disables an agent for your account.
//
// Activation is per-account, so this works for the `system` agents OpenMRP shares across accounts as well as your own `custom` agents: disabling one here leaves the underlying agent untouched for everyone else. Triggering an inactive agent returns a validation error.
type UpdateAgentStatusEndpoint struct{}

func (e *UpdateAgentStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAgentStatusRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*UpdateAgentStatusRequest, *apiresource.AgentDefinition]{
		Title:               "Update Agent Status",
		Method:              http.MethodPut,
		Route:               "/v1/ai/agents/{id}/status",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentDefinition,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).UpdateAgentStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
