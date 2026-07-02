package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAgentDefinitionID = "agdf_01b9ef28feb99e6954201aca63"

// An AI agent available to the account.
//
// The definition describes what the agent does, how its runs are triggered, the tools it can use, and whether it is currently enabled for the account.
type AgentDefinition struct {
	// Agent definition ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition"`
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
	// - `chat`: runs in response to a chat message; the run is linked to a conversation and posts its reply back into it.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Human-readable name of the agent.
	Name string `json:"name" validate:"required"`
	// URL-friendly identifier for the agent.
	Slug string `json:"slug" validate:"required"`
	// Description of what the agent does.
	Description *string `json:"description"`
	// Whether the current user can edit this agent definition.
	//
	// Always `read_only` for `system` definitions.
	Editability constants.Editability `json:"editability" validate:"required"`
	// Whether this agent is enabled for the current account.
	//
	// Activation is per-account: a `system` agent shared across accounts can be `active` for one account and `inactive` for another. An `inactive` agent does not run.
	AccountStatus constants.AgentAccountStatus `json:"status" validate:"required"`
	// Role defining the permissions the agent operates with.
	Role *Role `json:"role" expandable:"true"`
	// Agent-level configuration controlling LLM behavior and trigger settings.
	Config *AgentDefinitionConfig `json:"config" expandable:"true"`
	// Tools attached to this agent.
	Tools *List[AgentDefinitionTool] `json:"tools" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentDefinition = &AgentDefinition{
	ID:             SampleAgentDefinitionID,
	Object:         constants.ObjectTypeAgentDefinition,
	DefinitionType: constants.AgentDefinitionTypeSystem,
	CategoryCode:   "order_processing",
	TriggerType:    constants.AgentTriggerTypeEvent,
	Name:           "Email Order Agent",
	Slug:           "email_order",
	Description:    new("Processes incoming emails and creates draft orders."),
	Editability:    constants.EditabilityReadOnly,
	AccountStatus:  constants.AgentAccountStatusInactive,
	Config:         &SampleAgentDefinitionConfig,
	Tools: NewList([]AgentDefinitionTool{
		*SampleAgentDefinitionTool,
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentDefinition) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentDefinition)
}
