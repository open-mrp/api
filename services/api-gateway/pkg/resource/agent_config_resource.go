package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Agent-level configuration controlling LLM behavior.
// Separate from AgentDefinitionTool.Config, which configures individual tools.
type AgentDefinitionConfig struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition_config"`
	// System prompt / instructions for the agent.
	SystemPrompt *string `json:"system_prompt"`
	// LLM model identifier (e.g. "claude-sonnet-4").
	Model *string `json:"model"`
	// LLM provider name (e.g. "anthropic", "openai"). Inferred from model if omitted.
	Provider *string `json:"provider"`
	// LLM sampling temperature between 0 and 1.
	Temperature *float64 `json:"temperature"`
	// Trigger-specific configuration. Shape depends on the agent's trigger_type.
	TriggerConfig *TriggerConfig `json:"trigger_config"`
}

// Trigger-type-specific configuration.
// For "scheduled": CronSchedule is populated.
// For "event": EventFilters is populated.
// For "manual": all fields are empty.
type TriggerConfig struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=trigger_config"`
	// Cron expression for scheduled triggers (e.g. "0 9 * * *").
	CronSchedule *string `json:"cron_schedule"`
	// IANA timezone for the cron schedule (e.g. "America/New_York").
	Timezone *string `json:"timezone"`
	// Event types that trigger this agent (e.g. ["email.received", "order.created"]).
	EventFilters []string `json:"event_filters"`
}

var SampleTriggerConfig = &TriggerConfig{
	Object:       constants.ObjectTypeTriggerConfig,
	EventFilters: []string{"email.received"},
}

func (*TriggerConfig) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTriggerConfig)
}

// SampleAgentDefinitionConfig is a realistic example config for docs.
var SampleAgentDefinitionConfig = AgentDefinitionConfig{
	Object:        constants.ObjectTypeAgentDefinitionConfig,
	SystemPrompt:  new("You are an order processing agent. Parse incoming emails and create draft orders."),
	Model:         new("claude-sonnet-4"),
	Provider:      new("anthropic"),
	Temperature:   new(0.2),
	TriggerConfig: SampleTriggerConfig,
}

func (*AgentDefinitionConfig) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentDefinitionConfig)
}
