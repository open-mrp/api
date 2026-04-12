package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// AgentDefinition represents an agent definition.
type AgentDefinition struct {
	// The unique identifier for the agent definition.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition"`
	// The display name of the agent.
	Name string `json:"name" validate:"required"`
	// The unique slug identifier.
	Slug string `json:"slug" validate:"required"`
	// A description of what the agent does.
	Description *string `json:"description"`
	// Agent definition type.
	DefinitionType constants.AgentDefinitionType `json:"definition_type" validate:"required"`
	// The category code for this agent.
	CategoryCode string `json:"category_code" validate:"required"`
	// How this agent is triggered.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Whether the current user can edit this agent definition.
	IsEditable bool `json:"is_editable"`
	// The role that defines this agent's permissions.
	Role *Role `json:"role" expandable:"true"`
	// The agent configuration.
	Config *AgentDefinitionConfig `json:"config" expandable:"true"`
	// The tools attached to this agent.
	Tools *List[AgentDefinitionTool] `json:"tools" expandable:"true"`
	// The per-account activation status for this agent definition.
	AccountStatus constants.AgentAccountStatus `json:"status" validate:"required"`
	// When this agent definition was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this agent definition was last updated.
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
