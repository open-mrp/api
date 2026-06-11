package agentep

import (
	"context"
	"fmt"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Tool to attach to an agent definition.
type ToolInput struct {
	// ID of the tool to attach.
	//
	// Available tool IDs can be discovered with the List Tools endpoint (`GET /v1/ai/tools`).
	ToolID string `json:"tool_id" validate:"required"`
	// JSON-encoded configuration for this tool instance.
	//
	// The expected structure depends on the tool (see the tool's `config_schema`).
	ConfigJSON field.Optional[string] `json:"config_json,omitzero"`
	// Display order among the agent's tools (lower values appear first).
	SortOrder field.Optional[int32] `json:"sort_order,omitzero"`
	// Whether actions from this tool require human review before they execute.
	RequireReview field.Optional[bool] `json:"require_review,omitzero"`
}

var sampleToolInput = &ToolInput{
	ToolID:        apiresource.SampleAvailableToolID,
	SortOrder:     field.Some(int32(1)),
	RequireReview: field.Some(true),
}

func (*ToolInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleToolInput)
}

// Trigger-type-specific settings for agent creation/update requests.
//
// Required contents depend on the agent's `trigger_type`:
//
// - `scheduled`: `cron_schedule` is required.
// - `event`: at least one entry in `event_filters` is required.
// - `manual`: no trigger configuration is needed.
type TriggerConfigInput struct {
	// Cron expression for scheduled triggers (e.g. `0 9 * * *`).
	CronSchedule field.Optional[string] `json:"cron_schedule,omitzero" validate:"omitempty,max=255"`
	// IANA timezone for the cron schedule (e.g. `America/New_York`).
	Timezone field.Optional[string] `json:"timezone,omitzero" validate:"omitempty,max=255"`
	// Event types that trigger this agent (e.g. `["email.received", "order.created"]`).
	EventFilters []string `json:"event_filters"`
}

var sampleTriggerConfigInput = &TriggerConfigInput{
	EventFilters: []string{"email.received"},
}

func (*TriggerConfigInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleTriggerConfigInput)
}

// Agent-level configuration for creation/update requests.
type ConfigInput struct {
	// System prompt / instructions for the agent.
	SystemPrompt field.Optional[string] `json:"system_prompt,omitzero"`
	// LLM model identifier (e.g. `claude-sonnet-4`).
	Model field.Optional[string] `json:"model,omitzero" validate:"omitempty,max=255"`
	// LLM provider name (e.g. `anthropic`, `openai`).
	//
	// Inferred from `model` if omitted.
	Provider field.Optional[string] `json:"provider,omitzero" validate:"omitempty,max=255"`
	// LLM sampling temperature between 0 and 1.
	Temperature field.Optional[float64] `json:"temperature,omitzero" validate:"omitempty,min=0,max=1"`
	// Trigger-specific configuration.
	//
	// Required contents depend on the agent's `trigger_type`; see the trigger config schema.
	TriggerConfig field.Optional[TriggerConfigInput] `json:"trigger_config,omitzero"`
}

var sampleConfigInput = &ConfigInput{
	SystemPrompt:  field.Some("You are an order processing agent. Parse incoming emails and create draft orders."),
	Model:         field.Some("claude-sonnet-4"),
	Provider:      field.Some("anthropic"),
	Temperature:   field.Some(0.2),
	TriggerConfig: field.Some(*sampleTriggerConfigInput),
}

func (*ConfigInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConfigInput)
}

// Validate checks that the config is consistent with the given trigger type.
func (c *ConfigInput) Validate(triggerType constants.AgentTriggerType) error {
	tc, hasTC := c.TriggerConfig.Value()
	if triggerType == constants.AgentTriggerTypeScheduled {
		cron, hasCron := tc.CronSchedule.Value()
		if !hasTC || !hasCron || cron == "" {
			return fmt.Errorf("trigger_config with cron_schedule is required for scheduled triggers")
		}
	}
	if triggerType == constants.AgentTriggerTypeEvent && (!hasTC || len(tc.EventFilters) == 0) {
		return fmt.Errorf("trigger_config with at least one event_filter is required for event triggers")
	}
	return nil
}

// Request to create an agent definition.
type CreateAgentRequest struct {
	// Human-readable name of the agent.
	Name string `json:"name" validate:"required,max=255"`
	// URL-friendly identifier for the agent.
	Slug string `json:"slug" validate:"required,max=255"`
	// Description of what the agent does.
	Description field.Optional[string] `json:"description,omitzero"`
	// Category grouping for the agent (e.g. `order_processing`), used to organize agents in the UI.
	CategoryCode string `json:"category_code" validate:"required,max=255"`
	// How runs of this agent are initiated.
	//
	// - `scheduled`: runs on a cron schedule; `config.trigger_config.cron_schedule` is required.
	// - `event`: runs in response to platform events; at least one `config.trigger_config.event_filters` entry is required.
	// - `manual`: runs only when explicitly invoked.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config ConfigInput `json:"config"`
	// Tools to attach to the agent.
	Tools []ToolInput `json:"tools,omitzero"`
	// ID of the role that defines the permissions the agent operates with.
	RoleID field.Optional[string] `json:"role_id,omitzero" validate:"omitempty,max=191"`
}

var sampleCreateAgentRequest = &CreateAgentRequest{
	Name:         "Inventory Monitor",
	Slug:         "inventory_monitor",
	Description:  field.Some("Monitors inventory levels and creates restock alerts."),
	CategoryCode: "inventory",
	TriggerType:  constants.AgentTriggerTypeEvent,
	Config:       *sampleConfigInput,
	Tools:        []ToolInput{*sampleToolInput},
	RoleID:       field.Some(apiresource.SampleRoleID),
}

func (*CreateAgentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAgentRequest)
}

// Creates a custom agent definition with optional tool configuration.
//
// The new agent has `definition_type` `custom` and is immediately `active` for the account.
type CreateAgentEndpoint struct{}

func (e *CreateAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAgentRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*CreateAgentRequest, *apiresource.AgentDefinition]{
		Title:             "Create Agent",
		Method:            http.MethodPost,
		Route:             "/v1/ai/agents",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentDefinition,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).CreateAgent
		},
		LocationFunc: func(resp *apiresource.AgentDefinition) string {
			return "/v1/ai/agents/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	})
}
