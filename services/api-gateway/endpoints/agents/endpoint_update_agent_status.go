package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update the per-account status of an agent.
type UpdateAgentStatusRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
	// Account-level status: "active" or "inactive".
	Status string `json:"status" validate:"required,max=255"`
}

var sampleUpdateAgentStatusRequest = &UpdateAgentStatusRequest{
	Status: "active",
}

func (*UpdateAgentStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAgentStatusRequest)
}

// Upserts the per-account status for an agent definition.
type UpdateAgentStatusEndpoint struct{}

func (e *UpdateAgentStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAgentStatusRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*UpdateAgentStatusRequest, *apiresource.AgentDefinition]{
		Title:             "Update Agent Status",
		Method:            http.MethodPut,
		Route:             "/v1/ai/agents/{id}/status",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentDefinition,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).UpdateAgentStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
