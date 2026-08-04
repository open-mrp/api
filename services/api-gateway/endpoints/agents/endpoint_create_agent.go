package agentep

import (
	"context"
	"fmt"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Tool to attach to an agent definition.
type ToolInput struct {
	// The built-in tool to attach.
	//
	// Only Augno's built-in tools are attached here. Access to API-endpoint tools (creating a customer, listing orders, and so on) is granted separately through `config.endpoint_tool_slugs`. The List Tools endpoint (`GET /v1/ai/tools`) returns both kinds, with API-endpoint tools in the `api_endpoint` category.
	Tool constants.Tool `json:"tool" validate:"required"`
	// JSON-encoded configuration for this tool instance.
	//
	// The expected structure depends on the tool (see the tool's `config_schema`).
	ConfigJSON field.Optional[string] `json:"config_json,omitzero"`
	// Display order among the agent's tools (lower values appear first).
	SortOrder field.Optional[int32] `json:"sort_order,omitzero"`
	// Whether actions from this tool require human review before they execute.
	//
	// When review is required, a call to this tool pauses the run in `awaiting_approval` and records an action in `pending_review` until someone approves or rejects it through the Continue Agent Run endpoint. Approvals are one-time, so a later call to the same tool pauses again.
	RequireReview field.Optional[bool] `json:"require_review,omitzero"`
}

var sampleToolInput = &ToolInput{
	Tool:          apiresource.SampleAvailableToolSlug,
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
// - `manual` and `chat`: no trigger configuration is needed.
type TriggerConfigInput struct {
	// Cron expression for scheduled triggers (e.g. `0 9 * * *`).
	CronSchedule field.Optional[string] `json:"cron_schedule,omitzero" validate:"omitempty,max=255"`
	// IANA timezone for the cron schedule (e.g. `America/New_York`).
	Timezone field.Optional[string] `json:"timezone,omitzero" validate:"omitempty,max=255"`
	// Event types that trigger this agent (e.g. `["email.received", "order.created"]`).
	EventFilters []string `json:"event_filters,omitzero"`
}

var sampleTriggerConfigInput = &TriggerConfigInput{
	EventFilters: []string{"email.received"},
}

func (*TriggerConfigInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleTriggerConfigInput)
}

// Agent-level configuration for creation/update requests.
type ConfigInput struct {
	// Instructions that define the agent's role and how it should behave.
	//
	// Sent to the model on every turn of a run, alongside the platform guidance Augno adds automatically.
	SystemPrompt field.Optional[string] `json:"system_prompt,omitzero"`
	// Intelligence and cost tier for the agent's reasoning.
	//
	// Selects how capable (and how expensive) a model the agent uses without pinning a specific model, so the agent keeps working as the underlying model catalog changes.
	//
	// - `frontier`: the most capable and most expensive; multi-step planning, ambiguous work, tool-heavy workflows.
	// - `high`: normal planning, synthesis, and customer-facing reasoning.
	// - `balanced`: research, summarization, classification, structured extraction, and light tool use.
	// - `cheap`: simple transforms, validation, formatting, keyword lookup, and routing.
	// - `legacy`: older models kept for compatibility and regression comparison; avoid unless you specifically need them.
	Tier field.Optional[constants.ModelTier] `json:"tier,omitzero" default:"high"`
	// How much randomness the model uses when generating text.
	//
	// Lower values make the agent's output more repeatable; higher values make it more varied.
	Temperature field.Optional[float64] `json:"temperature,omitzero" validate:"omitempty,min=0,max=1"`
	// Trigger-specific configuration.
	//
	// Required contents depend on the agent's `trigger_type`; see the trigger config schema.
	TriggerConfig field.Optional[TriggerConfigInput] `json:"trigger_config,omitzero"`
	// API-endpoint tools the agent may discover and use, by slug (e.g. `create_account_group`).
	//
	// These are the tools listed by the List Tools endpoint with category `api_endpoint`. The single entry `*` grants the entire endpoint-tool catalog. Omit or leave empty to grant none.
	EndpointToolSlugs []string `json:"endpoint_tool_slugs,omitzero"`
	// Per-endpoint-tool human-review overrides, keyed by tool slug.
	//
	// Set a slug to `true` to require human approval before the agent may execute that endpoint-tool; the run pauses in `awaiting_approval` until approved via the Continue Agent Run endpoint. Slugs omitted from the map do not require review.
	EndpointToolReview field.Optional[map[string]bool] `json:"endpoint_tool_review,omitzero"`
}

var sampleConfigInput = &ConfigInput{
	SystemPrompt:  field.Some("You are an order processing agent. Parse incoming emails and create draft orders."),
	Tier:          field.Some(constants.ModelTierHigh),
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
	//
	// Must be unique within your account.
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
	// - `chat`: runs when a user messages the agent in a conversation, and the agent's reply is posted back into that conversation.
	//
	// Whatever the trigger type, a run can always be started by hand with the Trigger Agent Run endpoint.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config ConfigInput `json:"config"`
	// Built-in tools to attach to the agent.
	Tools []ToolInput `json:"tools,omitzero"`
	// ID of the role that defines the permissions the agent operates with.
	//
	// Every API call the agent makes is authorized against this role, so it bounds what the agent can see and change. An agent created without a role cannot execute — its runs fail immediately — so attach one before triggering it.
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

// Creates a custom agent for your account.
//
// The new agent is a `custom` definition and is immediately `active`, so it can start running as soon as it has a role.
type CreateAgentEndpoint struct{}

func (e *CreateAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAgentRequest, *apiresource.AgentDefinition] {
	return (&apiendpoint.APIEndpoint[*CreateAgentRequest, *apiresource.AgentDefinition]{
		Title:               "Create Agent",
		Method:              http.MethodPost,
		Route:               "/v1/ai/agents",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentDefinition,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgents, Action: types.ActionCreate}},
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
