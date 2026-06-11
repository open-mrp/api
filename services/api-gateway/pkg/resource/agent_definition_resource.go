package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// An AI agent available to the account.
//
// The definition describes what the agent does, how its runs are triggered, the tools it can use, and whether it is currently enabled for the account.
type AgentDefinition struct {
	// Agent definition ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition"`
	// Human-readable name of the agent.
	Name string `json:"name" validate:"required"`
	// URL-friendly identifier for the agent.
	Slug string `json:"slug" validate:"required"`
	// Description of what the agent does.
	Description *string `json:"description"`
	// Whether the agent is provided by Augno or created in this account.
	//
	// - `system`: provided by Augno; cannot be edited or deleted.
	// - `custom`: created by a user in this account.
	DefinitionType constants.AgentDefinitionType `json:"definition_type" validate:"required"`
	// Category grouping for the agent (e.g. `order_processing`), used to organize agents in the UI.
	CategoryCode string `json:"category_code" validate:"required"`
	// How runs of this agent are initiated.
	//
	// - `scheduled`: runs on a cron schedule (see `config.trigger_config.cron_schedule`).
	// - `event`: runs in response to platform events (see `config.trigger_config.event_filters`).
	// - `manual`: runs only when explicitly invoked.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Whether the current user can edit this agent definition.
	//
	// Always `false` for `system` definitions.
	IsEditable bool `json:"is_editable"`
	// Role defining the permissions the agent operates with.
	Role *Role `json:"role" expandable:"true"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config *AgentDefinitionConfig `json:"config" expandable:"true"`
	// Tools attached to this agent.
	Tools *List[AgentDefinitionTool] `json:"tools" expandable:"true"`
	// Whether this agent is enabled for the current account.
	//
	// Activation is per-account: a `system` agent shared across accounts can be `active` for one account and `inactive` for another. An `inactive` agent does not run.
	AccountStatus constants.AgentAccountStatus `json:"status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentDefinition = &AgentDefinition{
	ID:             SampleAgentDefinitionID,
	Object:         constants.ObjectTypeAgentDefinition,
	Name:           "Email Order Agent",
	Slug:           "email_order",
	Description:    new("Processes incoming emails and creates draft orders."),
	DefinitionType: constants.AgentDefinitionTypeSystem,
	CategoryCode:   "order_processing",
	TriggerType:    constants.AgentTriggerTypeEvent,
	IsEditable:     false,
	Config:         &SampleAgentDefinitionConfig,
	Tools: NewList([]AgentDefinitionTool{
		*SampleAgentDefinitionTool,
	}, PageInfo{}),
	AccountStatus: constants.AgentAccountStatusInactive,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentDefinition) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentDefinition)
}
