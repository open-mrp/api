package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A single tool invocation performed by an agent during a run.
//
// Each action records the tool that was called, its input and output, and any human review decision.
type AgentAction struct {
	// Agent action ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_action"`
	// The tool the agent invoked for this action.
	//
	// - `create_artifact`: create an artifact such as a report, document, or data export.
	// - `read_doc`: read Augno documentation pages.
	// - `fetch_url`: fetch content from a public URL.
	// - `draft_reply`: propose a reply to the case's external party as a draft held for human approval (not sent).
	// - `send_email`: send an email reply through the conversation's bound inbox.
	Tool constants.Tool `json:"tool" validate:"required"`
	// Current action status.
	//
	// - `pending_review`: awaiting human review before it can execute.
	// - `auto_approved`: automatically approved by policy.
	// - `approved`: manually approved by a user.
	// - `rejected`: rejected by a user; will not execute.
	// - `executed`: successfully executed.
	// - `failed`: errored during execution; see `error_message`.
	Status constants.AgentActionStatus `json:"status" validate:"required"`
	// Short human-readable label summarizing the action.
	Label *string `json:"label"`
	// Longer description of what the action does.
	Description *string `json:"description"`
	// Agent run this action belongs to.
	Run *AgentRun `json:"run" validate:"required" expandable:"true"`
	// Arguments passed to the tool, as JSON.
	//
	// Shape depends on `tool`.
	Input json.RawMessage `json:"input"`
	// Result returned by the tool, as JSON.
	//
	// Recorded when the tool runs, so it is present even while the action is still `pending_review` or `auto_approved`; the shape depends on `tool`, and it is `{}` when the tool returned no output.
	Output json.RawMessage `json:"output"`
	// Error message if the action failed.
	ErrorMessage *string `json:"error_message"`
	// The resource this action operated on, when the tool targets a specific entity such as a customer or product.
	Entity *Entity `json:"entity"`
	// Whether this action must be reviewed by a human before it can execute.
	ReviewRequirement constants.ReviewRequirement `json:"review_requirement" validate:"required"`
	// When a human review decision was recorded for the action.
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
	ID:                SampleAgentActionID,
	Object:            constants.ObjectTypeAgentAction,
	Tool:              constants.ToolReadDoc,
	Status:            constants.AgentActionStatusExecuted,
	Label:             new("Read Doc"),
	Description:       new("Read the Augno documentation page on creating sales orders."),
	Run:               &AgentRun{ID: SampleAgentRunID, Object: constants.ObjectTypeAgentRun},
	Input:             json.RawMessage(`{"path":"/api/sales-orders/create"}`),
	Output:            json.RawMessage(`{"title":"Create a sales order","url":"https://docs.augno.com/api/sales-orders/create"}`),
	ReviewRequirement: constants.ReviewRequirementNotRequired,
	ExecutedAt:        new(timeutil.TimestampToTime(sampleUpdatedAtTimestamp)),
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentAction) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentAction)
}
