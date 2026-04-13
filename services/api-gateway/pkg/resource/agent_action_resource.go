package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Agent action resource.
type AgentAction struct {
	// Agent action ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_action"`
	// Agent run this action belongs to.
	Run *AgentRun `json:"run" validate:"required" expandable:"true"`
	// Tool slug.
	ToolSlug constants.ToolSlug `json:"tool_slug" validate:"required"`
	// Current action status.
	Status constants.AgentActionStatus `json:"status" validate:"required"`
	// Short label.
	Label *string `json:"label"`
	// Action description.
	Description *string `json:"description"`
	// Action input.
	Input json.RawMessage `json:"input"`
	// Action output.
	Output json.RawMessage `json:"output"`
	// Error message if the action failed.
	ErrorMessage *string `json:"error_message"`
	// Entity this action relates to.
	Entity *Entity `json:"entity"`
	// Whether human review is required.
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
