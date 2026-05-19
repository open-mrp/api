package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAgentRunID = "agrn_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleAgentActionID = "agax_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleAgentRunStepID = "agrnev_01jm4r6700f8nwq3v5hx2d9ktp"

// Agent run resource.
type AgentRun struct {
	// Agent run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_run"`
	// Current run status.
	Status constants.AgentRunStatus `json:"status" validate:"required"`
	// Trigger type.
	TriggerType constants.AgentTriggerType `json:"trigger_type" validate:"required"`
	// Input provided to the agent.
	Input json.RawMessage `json:"input"`
	// Output produced by the agent.
	Output json.RawMessage `json:"output"`
	// Error message if the run failed.
	ErrorMessage *string `json:"error_message"`
	// When the run started executing.
	StartedAt *time.Time `json:"started_at"`
	// When the run completed.
	CompletedAt *time.Time `json:"completed_at"`
	// Duration in milliseconds.
	DurationMs *int32 `json:"duration_ms"`
	// Total input tokens consumed.
	TotalInputTokens *int64 `json:"total_input_tokens"`
	// Total output tokens consumed.
	TotalOutputTokens *int64 `json:"total_output_tokens"`
	// Actor that triggered this run. Null for scheduled or event-triggered runs.
	TriggeredBy *Actor `json:"triggered_by"`
	// When this run was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this run was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
	// Actions performed during this run.
	Actions *List[AgentAction] `json:"actions" expandable:"true"`
	// Full agent definition for this run.
	Definition *AgentDefinition `json:"definition" expandable:"true"`
	// Timeline steps for this run.
	Steps *List[AgentRunStep] `json:"steps" expandable:"true"`
}

// Agent run step resource.
type AgentRunStep struct {
	// Agent run step ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_run_step"`
	// Step type.
	StepType string `json:"step_type" validate:"required"`
	// Short title for the step.
	Title string `json:"title" validate:"required"`
	// Step content.
	Content *string `json:"content"`
	// Sequence number.
	Sequence int32 `json:"sequence"`
	// Duration in milliseconds.
	DurationMs *int32 `json:"duration_ms"`
	// Actor who produced this event.
	Actor *Actor `json:"actor"`
	// Metadata for the step.
	Metadata json.RawMessage `json:"metadata"`
	// When this step was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleAgentRunStep = &AgentRunStep{
	ID:        SampleAgentRunStepID,
	Object:    constants.ObjectTypeAgentRunStep,
	StepType:  "trigger_received",
	Title:     "Run triggered",
	Content:   new("Process order #1234"),
	Sequence:  0,
	Actor:     SampleActor,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AgentRunStep) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentRunStep)
}

var SampleAgentRun = &AgentRun{
	ID:                SampleAgentRunID,
	Object:            constants.ObjectTypeAgentRun,
	Status:            constants.AgentRunStatusCompleted,
	TriggerType:       constants.AgentTriggerTypeManual,
	TriggeredBy:       SampleActor,
	Input:             json.RawMessage(`{"message":"Process order #1234"}`),
	Output:            json.RawMessage(`{"response":"Order processed successfully"}`),
	DurationMs:        new(int32(1250)),
	TotalInputTokens:  new(int64(500)),
	TotalOutputTokens: new(int64(300)),
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	Actions:           NewList([]AgentAction{*SampleAgentAction}, PageInfo{}),
	Steps:             NewList([]AgentRunStep{*SampleAgentRunStep}, PageInfo{}),
}

func (*AgentRun) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentRun)
}
