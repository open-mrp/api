package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// AgentAction represents a single action performed during an agent run.
type AgentAction struct {
	// The unique identifier for the action.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_action"`
	// The agent run this action belongs to.
	Run *AgentRun `json:"run" validate:"required" expandable:"true"`
	// The tool slug identifier.
	ToolSlug constants.ToolSlug `json:"tool_slug" validate:"required"`
	// The current status of this action.
	Status constants.AgentActionStatus `json:"status" validate:"required"`
	// A short label for the action.
	Label *string `json:"label"`
	// A description of what the action does.
	Description *string `json:"description"`
	// The input to the action.
	Input json.RawMessage `json:"input"`
	// The output from the action.
	Output json.RawMessage `json:"output"`
	// Error message if the action failed.
	ErrorMessage *string `json:"error_message"`
	// The entity this action relates to.
	Entity *Entity `json:"entity"`
	// Whether this action requires human review.
	RequiresReview bool `json:"requires_review"`
	// When the action was reviewed.
	ReviewedAt *time.Time `json:"reviewed_at"`
	// Who reviewed the action.
	ReviewedBy *Actor `json:"reviewed_by"`
	// When the action was executed.
	ExecutedAt *time.Time `json:"executed_at"`
	// When this action was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this action was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentAction = &AgentAction{
	ID:     SampleAgentActionID,
	Object: constants.ObjectTypeAgentAction,
	Run: &AgentRun{
		ID:     SampleAgentRunID,
		Object: constants.ObjectTypeAgentRun,
	},
	ToolSlug:  constants.ToolSlugSearchProducts,
	Status:    constants.AgentActionStatusExecuted,
	Label:     new("Search Products"),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentAction) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentAction)
}
