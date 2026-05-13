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

// Request to partially update an agent definition.
type UpdateAgentRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// URL-friendly identifier.
	Slug *string `json:"slug,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Description of what the agent does.
	Description *string `json:"description,omitempty" nullable:"true"`
	// Category code (e.g. "order_processing").
	CategoryCode *string `json:"category_code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Trigger type: "manual", "scheduled", or "event".
	TriggerType *constants.AgentTriggerType `json:"trigger_type,omitempty" nullable:"false"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config *ConfigInput `json:"config,omitempty" nullable:"false"`
	// Tools to attach. Replaces the existing tool set when provided.
	Tools *[]ToolInput `json:"tools,omitempty" nullable:"false"`
	// Role ID defining agent permissions.
	RoleID *string `json:"role_id,omitempty" nullable:"true" validate:"omitempty"`
}

var sampleUpdateAgentRequest = &UpdateAgentRequest{
	Name: new("Inventory Monitor"),
}

func (*UpdateAgentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAgentRequest)
}

type UpdateAgentEndpoint struct{}

func (e *UpdateAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAgentRequest, *apiresource.AgentDefinition] {
	return &apiendpoint.APIEndpoint[*UpdateAgentRequest, *apiresource.AgentDefinition]{
		Title:             "Update Agent",
		Description:       "Partially updates a custom agent definition. System agents cannot be modified.",
		Method:            http.MethodPatch,
		Route:             "/v1/ai/agents/{id}",
		ContentType:       "application/json",
		Request:           &UpdateAgentRequest{},
		Response:          &apiresource.AgentDefinition{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).UpdateAgent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	}
}
