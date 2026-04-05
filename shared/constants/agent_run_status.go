package constants

// AgentRunStatus represents the execution status of an agent run.
type AgentRunStatus string

const (
	// AgentRunStatusPending indicates the run is queued but not yet started.
	AgentRunStatusPending AgentRunStatus = "pending"
	// AgentRunStatusRunning indicates the run is currently executing.
	AgentRunStatusRunning AgentRunStatus = "running"
	// AgentRunStatusCompleted indicates the run finished successfully.
	AgentRunStatusCompleted AgentRunStatus = "completed"
	// AgentRunStatusFailed indicates the run encountered an error.
	AgentRunStatusFailed AgentRunStatus = "failed"
	// AgentRunStatusCancelled indicates the run was cancelled.
	AgentRunStatusCancelled AgentRunStatus = "cancelled"
	// AgentRunStatusAwaitingInput indicates the run is waiting for user input.
	AgentRunStatusAwaitingInput AgentRunStatus = "awaiting_input"
	// AgentRunStatusAwaitingApproval indicates the run is waiting for approval.
	AgentRunStatusAwaitingApproval AgentRunStatus = "awaiting_approval"
)

func (m AgentRunStatus) IsValid() bool {
	switch m {
	case AgentRunStatusPending, AgentRunStatusRunning, AgentRunStatusCompleted,
		AgentRunStatusFailed, AgentRunStatusCancelled, AgentRunStatusAwaitingInput,
		AgentRunStatusAwaitingApproval:
		return true
	default:
		return false
	}
}

func (m AgentRunStatus) EnumValues() []string {
	return []string{
		string(AgentRunStatusPending),
		string(AgentRunStatusRunning),
		string(AgentRunStatusCompleted),
		string(AgentRunStatusFailed),
		string(AgentRunStatusCancelled),
		string(AgentRunStatusAwaitingInput),
		string(AgentRunStatusAwaitingApproval),
	}
}
