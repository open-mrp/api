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

// ToolInput represents a tool to attach to an agent definition.
type ToolInput struct {
	// The identifier of the available tool to attach.
	ToolID string `json:"tool_id" validate:"required"`
	// Optional JSON configuration for this tool instance.
	ConfigJSON string `json:"config_json,omitempty"`
	// Display order among the agent's tools (lower values appear first).
	SortOrder int32 `json:"sort_order,omitempty"`
	// Whether actions from this tool require human review before execution.
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

// TriggerConfigInput holds trigger-type-specific settings for agent creation/update requests.
type TriggerConfigInput struct {
	// Cron expression for scheduled triggers (e.g. "0 9 * * *").
	CronSchedule *string `json:"cron_schedule"`
	// IANA timezone for the cron schedule (e.g. "America/New_York").
	Timezone *string `json:"timezone"`
	// Event types that trigger this agent (e.g. ["email.received", "order.created"]).
	EventFilters []string `json:"event_filters"`
}

var sampleTriggerConfigInput = &TriggerConfigInput{
	EventFilters: []string{"email.received"},
}

func (*TriggerConfigInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleTriggerConfigInput)
}

// ConfigInput holds agent-level configuration for creation/update requests.
type ConfigInput struct {
	// The system prompt / instructions given to the agent.
	SystemPrompt *string `json:"system_prompt"`
	// The LLM model identifier (e.g. "claude-sonnet-4").
	Model *string `json:"model"`
	// The LLM provider name (e.g. "anthropic", "openai"). Inferred from model if omitted.
	Provider *string `json:"provider"`
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

// CreateAgentRequest is the request to create a new agent definition.
type CreateAgentRequest struct {
	// The display name of the agent.
	Name string `json:"name" validate:"required"`
	// A unique URL-friendly identifier for the agent.
	Slug string `json:"slug" validate:"required"`
	// A human-readable description of what the agent does.
	Description string `json:"description,omitempty"`
	// The category code that classifies this agent (e.g. "order_processing").
	CategoryCode string `json:"category_code" validate:"required"`
	// How this agent is triggered: "manual", "scheduled", or "event".
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config ConfigInput `json:"config"`
	// The tools to attach to this agent.
	Tools []ToolInput `json:"tools,omitempty"`
	// The ID of the role that defines this agent's permissions.
	RoleID string `json:"role_id,omitempty"`
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

type CreateAgentEndpoint struct{}

func (e *CreateAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAgentRequest, *apiresource.AgentDefinition] {
	return &apiendpoint.APIEndpoint[*CreateAgentRequest, *apiresource.AgentDefinition]{
		Title:             "Create Agent",
		Description:       "Creates a new custom agent definition with optional tool configuration.",
		Method:            http.MethodPost,
		Route:             "/v1/ai/agents",
		ContentType:       "application/json",
		Request:           &CreateAgentRequest{},
		Response:          &apiresource.AgentDefinition{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAgentRequest) (*apiresource.AgentDefinition, *apierror.APIError) {
			return svc.(AgentSvc).CreateAgent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentDefinition,
			Fields:     []string{"config", "tools", "role", "role.permissions"},
		}),
	}
}
