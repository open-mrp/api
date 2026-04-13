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
	// Account-level status code: "active" or "inactive".
	StatusCode string `json:"status_code" validate:"required,max=255"`
}

var sampleUpdateAgentStatusRequest = &UpdateAgentStatusRequest{
	StatusCode: "active",
}

func (*UpdateAgentStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAgentStatusRequest)
}

type UpdateAgentStatusEndpoint struct{}

func (e *UpdateAgentStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAgentStatusRequest, *apiresource.AgentDefinition] {
	return &apiendpoint.APIEndpoint[*UpdateAgentStatusRequest, *apiresource.AgentDefinition]{
		Title:             "Update Agent Status",
		Description:       "Upserts the per-account status for an agent definition.",
		Method:            http.MethodPut,
		Route:             "/v1/ai/agents/{id}/status",
		ContentType:       "application/json",
		Request:           &UpdateAgentStatusRequest{},
		Response:          &apiresource.AgentDefinition{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAgentStatusRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).UpdateAgentStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	}
}
