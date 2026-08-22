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
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update an agent definition.
type UpdateAgentRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
	// Human-readable name of the agent.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// URL-friendly identifier for the agent.
	Slug field.Optional[string] `json:"slug,omitzero" validate:"omitempty,max=255"`
	// Description of what the agent does.
	//
	// Send `null` to clear the description; omit to leave it unchanged.
	Description field.Clearable[string] `json:"description,omitzero"`
	// Category grouping for the agent (e.g. `order_processing`), used to organize agents in the UI.
	CategoryCode field.Optional[string] `json:"category_code,omitzero" validate:"omitempty,max=255"`
	// How runs of this agent are initiated.
	//
	// When changing the trigger type, also provide a `config` with a `trigger_config` appropriate for the new type (a cron schedule for `scheduled`, at least one event filter for `event`).
	TriggerType field.Optional[constants.AgentTriggerType] `json:"trigger_type,omitzero"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	//
	// Replaces the stored configuration in full rather than merging into it, so send every setting you want to keep — anything you leave out is cleared.
	Config field.Optional[ConfigInput] `json:"config,omitzero"`
	// Built-in tools to attach to the agent.
	//
	// Replaces the existing tool set when provided.
	Tools field.Optional[[]ToolInput] `json:"tools,omitzero"`
	// ID of the role that defines the permissions the agent operates with.
	//
	// Send `null` to detach the role; omit to leave it unchanged. An agent with no role cannot execute, so detaching the role makes its runs fail immediately.
	RoleID field.Clearable[string] `json:"role_id,omitzero" validate:"omitempty"`
}

var sampleUpdateAgentRequest = &UpdateAgentRequest{
	Name: field.Some("Inventory Monitor"),
}

func (*UpdateAgentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAgentRequest)
}

// Updates a custom agent.
//
// Only the fields provided in the request are changed. OpenMRP's `system` agents cannot be edited — the only thing you can change about them is whether they are enabled for your account, with the Update Agent Status endpoint.
type UpdateAgentEndpoint struct{}

func (e *UpdateAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAgentRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*UpdateAgentRequest, *apiresource.AgentDefinition]{
		Title:               "Update Agent",
		Method:              http.MethodPatch,
		Route:               "/v1/ai/agents/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentDefinition,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).UpdateAgent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
