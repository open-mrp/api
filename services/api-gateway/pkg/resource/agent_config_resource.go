package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// Agent-level configuration controlling LLM behavior and trigger settings.
//
// Distinct from per-tool configuration (`tools[].config`), which configures individual tools attached to the agent.
type AgentDefinitionConfig struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition_config"`
	// Standing instructions that define the agent's role and how it should behave on every run.
	SystemPrompt *string `json:"system_prompt"`
	// Intelligence and cost tier for the agent's reasoning.
	//
	// Selects how capable and expensive a model the agent uses without pinning a specific model; higher tiers reason better but cost more. Each tier resolves to an ordered chain of equivalent models, so a run automatically fails over to another provider's model if the preferred one is unavailable.
	//
	// - `frontier`: the most capable tier, for multi-step planning, ambiguous agent work, and hard coding or architecture tasks.
	// - `high`: for normal planning, code edits, synthesis, and customer-facing reasoning.
	// - `balanced`: for research, summarization, classification, structured extraction, and light tool use.
	// - `cheap`: for simple transforms, validation, formatting, and routing.
	// - `legacy`: older-generation models kept for compatibility and regression comparison; avoid unless you specifically need them.
	//
	// Leaving the tier unset picks one from how the agent is triggered: chat and manual runs use `high`, while scheduled and event-driven runs use `balanced` so background work stays cheap.
	Tier *constants.ModelTier `json:"tier"`
	// LLM sampling temperature between 0 and 1.
	//
	// Lower values make the agent's output more repeatable and literal; higher values make it more varied.
	Temperature *float64 `json:"temperature"`
	// Trigger-specific configuration.
	//
	// Shape depends on the agent's `trigger_type`.
	TriggerConfig *TriggerConfig `json:"trigger_config"`
	// API-endpoint tools the agent may discover and use, by slug (e.g. `create_account_group`).
	//
	// These correspond to tools listed by the List Tools endpoint with category `api_endpoint`. A single entry `*` grants the entire endpoint-tool catalog.
	EndpointToolSlugs []string `json:"endpoint_tool_slugs"`
	// Per-endpoint-tool human-review overrides, keyed by tool slug.
	//
	// When an entry is `true`, the run pauses in `awaiting_approval` each time the agent calls that endpoint-tool until it is approved via the Continue Agent Run endpoint. Slugs absent from the map do not require review.
	EndpointToolReview map[string]bool `json:"endpoint_tool_review"`
}

// Trigger-type-specific configuration.
//
// Which fields are populated depends on the agent's `trigger_type`:
//
// - `scheduled`: `cron_schedule` (and optionally `timezone`) is set.
// - `event`: `event_filters` is set.
// - `manual`: all fields are empty.
type TriggerConfig struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=trigger_config"`
	// Cron expression for scheduled triggers (e.g. `0 9 * * *`).
	CronSchedule *string `json:"cron_schedule"`
	// IANA timezone for the cron schedule (e.g. `America/New_York`).
	Timezone *string `json:"timezone"`
	// Event types that trigger this agent (e.g. `["email.received", "order.created"]`).
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
	Object:             constants.ObjectTypeAgentDefinitionConfig,
	SystemPrompt:       new("You are an order processing agent. Parse incoming emails and create draft orders."),
	Tier:               new(constants.ModelTierHigh),
	Temperature:        new(0.2),
	TriggerConfig:      SampleTriggerConfig,
	EndpointToolSlugs:  []string{"create_account_group"},
	EndpointToolReview: map[string]bool{"create_account_group": true},
}

func (*AgentDefinitionConfig) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentDefinitionConfig)
}
