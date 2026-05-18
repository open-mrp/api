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
)

// Tool to attach to an agent definition.
type ToolInput struct {
	// Available tool ID.
	ToolID string `json:"tool_id" validate:"required"`
	// JSON configuration for this tool instance.
	ConfigJSON string `json:"config_json,omitempty"`
	// Display order among the agent's tools (lower values appear first).
	SortOrder int32 `json:"sort_order,omitempty"`
	// Requires human review before execution.
	RequireReview bool `json:"require_review,omitempty"`
}

var sampleToolInput = &ToolInput{
	ToolID:        apiresource.SampleAvailableToolID,
	SortOrder:     1,
	RequireReview: true,
}

func (*ToolInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleToolInput)
}

// Trigger-type-specific settings for agent creation/update requests.
type TriggerConfigInput struct {
	// Cron expression for scheduled triggers (e.g. "0 9 * * *").
	CronSchedule *string `json:"cron_schedule" validate:"omitempty,max=255"`
	// IANA timezone for the cron schedule (e.g. "America/New_York").
	Timezone *string `json:"timezone" validate:"omitempty,max=255"`
	// Event types that trigger this agent (e.g. ["email.received", "order.created"]).
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
	SystemPrompt *string `json:"system_prompt"`
	// LLM model identifier (e.g. "claude-sonnet-4").
	Model *string `json:"model" validate:"omitempty,max=255"`
	// LLM provider name (e.g. "anthropic", "openai"). Inferred from model if omitted.
	Provider *string `json:"provider" validate:"omitempty,max=255"`
	// LLM sampling temperature between 0 and 1.
	Temperature *float64 `json:"temperature" validate:"omitempty,min=0,max=1"`
	// Trigger-specific configuration. Shape depends on the agent's trigger_type.
	TriggerConfig *TriggerConfigInput `json:"trigger_config"`
}

var sampleConfigInput = &ConfigInput{
	SystemPrompt:  new("You are an order processing agent. Parse incoming emails and create draft orders."),
	Model:         new("claude-sonnet-4"),
	Provider:      new("anthropic"),
	Temperature:   new(0.2),
	TriggerConfig: sampleTriggerConfigInput,
}

func (*ConfigInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConfigInput)
}

// Validate checks that the config is consistent with the given trigger type.
func (c *ConfigInput) Validate(triggerType constants.AgentTriggerType) error {
	if triggerType == constants.AgentTriggerTypeScheduled && (c.TriggerConfig == nil || c.TriggerConfig.CronSchedule == nil || *c.TriggerConfig.CronSchedule == "") {
		return fmt.Errorf("trigger_config with cron_schedule is required for scheduled triggers")
	}
	if triggerType == constants.AgentTriggerTypeEvent && (c.TriggerConfig == nil || len(c.TriggerConfig.EventFilters) == 0) {
		return fmt.Errorf("trigger_config with at least one event_filter is required for event triggers")
	}
	return nil
}

// Request to create an agent definition.
type CreateAgentRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// URL-friendly identifier.
	Slug string `json:"slug" validate:"required,max=255"`
	// Description of what the agent does.
	Description string `json:"description,omitempty"`
	// Category code (e.g. "order_processing").
	CategoryCode string `json:"category_code" validate:"required,max=255"`
	// Trigger type: "manual", "scheduled", or "event".
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config ConfigInput `json:"config"`
	// Tools to attach.
	Tools []ToolInput `json:"tools,omitempty"`
	// Role ID defining agent permissions.
	RoleID string `json:"role_id,omitempty" validate:"max=191"`
}

var sampleCreateAgentRequest = &CreateAgentRequest{
	Name:         "Inventory Monitor",
	Slug:         "inventory_monitor",
	Description:  "Monitors inventory levels and creates restock alerts.",
	CategoryCode: "inventory",
	TriggerType:  constants.AgentTriggerTypeEvent,
	Config:       *sampleConfigInput,
	Tools:        []ToolInput{*sampleToolInput},
	RoleID:       apiresource.SampleRoleID,
}

func (*CreateAgentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAgentRequest)
}

// Creates a custom agent definition with optional tool configuration.
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
