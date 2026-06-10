package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an agent definition.
type UpdateAgentRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// URL-friendly identifier.
	Slug field.Optional[string] `json:"slug,omitzero" validate:"omitempty,max=255"`
	// Description of what the agent does.
	Description field.Optional[string] `json:"description,omitzero"`
	// Category code (e.g. "order_processing").
	CategoryCode field.Optional[string] `json:"category_code,omitzero" validate:"omitempty,max=255"`
	// Trigger type.
	TriggerType field.Optional[constants.AgentTriggerType] `json:"trigger_type,omitzero"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config field.Optional[ConfigInput] `json:"config,omitzero"`
	// Tools to attach. Replaces the existing tool set when provided.
	Tools field.Optional[[]ToolInput] `json:"tools,omitzero"`
	// Role ID defining agent permissions.
	RoleID field.Optional[string] `json:"role_id,omitzero" validate:"omitempty"`
}

var sampleUpdateAgentRequest = &UpdateAgentRequest{
	Name: field.Some("Inventory Monitor"),
}

func (*UpdateAgentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAgentRequest)
}

// Partially updates a custom agent definition. System agents cannot be modified.
type UpdateAgentEndpoint struct{}

func (e *UpdateAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAgentRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*UpdateAgentRequest, *apiresource.AgentDefinition]{
		Title:             "Update Agent",
		Method:            http.MethodPatch,
		Route:             "/v1/ai/agents/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentDefinition,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).UpdateAgent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
