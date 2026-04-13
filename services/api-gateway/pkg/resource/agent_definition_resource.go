package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Agent definition resource.
type AgentDefinition struct {
	// Agent definition ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// URL-friendly slug.
	Slug string `json:"slug" validate:"required"`
	// Description of what the agent does.
	Description *string `json:"description"`
	// Agent definition type.
	DefinitionType constants.AgentDefinitionType `json:"definition_type" validate:"required"`
	// Category code.
	CategoryCode string `json:"category_code" validate:"required"`
	// How this agent is triggered.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Whether the current user can edit this agent definition.
	IsEditable bool `json:"is_editable"`
	// Role defining agent permissions.
	Role *Role `json:"role" expandable:"true"`
	// Agent configuration.
	Config *AgentDefinitionConfig `json:"config" expandable:"true"`
	// Tools attached to this agent.
	Tools *List[AgentDefinitionTool] `json:"tools" expandable:"true"`
	// Per-account activation status.
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
