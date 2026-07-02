package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAgentRunID = "agrn_01502aa6da9bbdbaa595915fa4"
const SampleAgentActionID = "agax_018eddea543007633706d37109"
const SampleAgentRunStepID = "agrnev_01148232974cd53b3ef1b6d437"

// A single execution of an agent, from trigger through completion.
type AgentRun struct {
	// Agent run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_run"`
	// How this run was initiated.
	//
	// - `scheduled`: started by the agent's cron schedule.
	// - `event`: started in response to a platform event.
	// - `manual`: started by an explicit request; see `triggered_by`.
	// - `chat`: started by a message in a conversation, with the agent's reply posted back into that conversation.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Current run status.
	//
	// - `pending`: queued but not yet started.
	// - `running`: currently executing.
	// - `awaiting_input`: paused, waiting for user input before continuing.
	// - `awaiting_approval`: paused, waiting for a pending action to be approved.
	// - `completed`: finished successfully.
	// - `failed`: stopped after an error; see `error_message`.
	// - `cancelled`: stopped before completion by a user.
	Status constants.AgentRunStatus `json:"status" validate:"required"`
	// Full agent definition for this run.
	Definition *AgentDefinition `json:"definition" expandable:"true"`
	// Input provided to the agent at the start of the run, as JSON.
	Input json.RawMessage `json:"input"`
	// Final output produced by the agent, as JSON.
	//
	// Populated only once the run has completed successfully.
	Output json.RawMessage `json:"output"`
	// Error message if the run failed.
	ErrorMessage *string `json:"error_message"`
	// Actor that triggered this run.
	//
	// Null for scheduled or event-triggered runs.
	TriggeredBy *Actor `json:"triggered_by" expandable:"true"`
	// When the run started executing.
	StartedAt *time.Time `json:"started_at"`
	// When the run completed.
	CompletedAt *time.Time `json:"completed_at"`
	// Duration in milliseconds.
	DurationMs *int32 `json:"duration_ms"`
	// Actions performed during this run.
	Actions *List[AgentAction] `json:"actions" expandable:"true"`
	// Timeline steps for this run.
	Steps *List[AgentRunStep] `json:"steps" expandable:"true"`
	// When this run was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this run was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A single event in an agent run's execution timeline.
type AgentRunStep struct {
	// Agent run step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_run_step"`
	// The kind of timeline event (e.g. `trigger_received`, `user_message`, `assistant_message`, `tool_call`, `tool_result`, `awaiting_approval`, `completion`, `error`).
	StepType string `json:"step_type" validate:"required"`
	// Short title for the step.
	Title string `json:"title" validate:"required"`
	// Text payload for the step, such as a message body or a tool result.
	Content *string `json:"content"`
	// Zero-based position of this step within the run's timeline.
	Sequence int32 `json:"sequence"`
	// Actor who produced this step.
	Actor *Actor `json:"actor"`
	// Duration in milliseconds.
	DurationMs *int32 `json:"duration_ms"`
	// Additional structured data for the step, as JSON.
	Metadata json.RawMessage `json:"metadata"`
	// When this step was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleAgentRunStep = &AgentRunStep{
	ID:         SampleAgentRunStepID,
	Object:     constants.ObjectTypeAgentRunStep,
	StepType:   "trigger_received",
	Title:      "Run triggered",
	Content:    new("Process order #1234"),
	Sequence:   0,
	Actor:      SampleActor,
	DurationMs: new(int32(12)),
	Metadata:   json.RawMessage(`{"trigger_type":"manual"}`),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AgentRunStep) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentRunStep)
}

var SampleAgentRun = &AgentRun{
	ID:          SampleAgentRunID,
	Object:      constants.ObjectTypeAgentRun,
	TriggerType: constants.AgentTriggerTypeManual,
	Status:      constants.AgentRunStatusCompleted,
	Definition:  SampleAgentDefinition,
	TriggeredBy: SampleActor,
	Input:       json.RawMessage(`{"message":"Process order #1234"}`),
	Output:      json.RawMessage(`{"response":"Order processed successfully"}`),
	StartedAt:   new(timeutil.TimestampToTime(sampleCreatedAtTimestamp)),
	CompletedAt: new(timeutil.TimestampToTime(sampleUpdatedAtTimestamp)),
	DurationMs:  new(int32(1250)),
	Actions:     NewList([]AgentAction{*SampleAgentAction}, PageInfo{}),
	Steps:       NewList([]AgentRunStep{*SampleAgentRunStep}, PageInfo{}),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentRun) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentRun)
}
