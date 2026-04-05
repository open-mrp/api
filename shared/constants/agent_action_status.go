package constants

// AgentActionStatus represents the status of an agent action.
type AgentActionStatus string

const (
	// AgentActionStatusPendingReview indicates the action is awaiting human review.
	AgentActionStatusPendingReview AgentActionStatus = "pending_review"
	// AgentActionStatusAutoApproved indicates the action was automatically approved by policy.
	AgentActionStatusAutoApproved AgentActionStatus = "auto_approved"
	// AgentActionStatusApproved indicates the action was manually approved by a user.
	AgentActionStatusApproved AgentActionStatus = "approved"
	// AgentActionStatusRejected indicates the action was rejected by a user.
	AgentActionStatusRejected AgentActionStatus = "rejected"
	// AgentActionStatusExecuted indicates the action was successfully executed.
	AgentActionStatusExecuted AgentActionStatus = "executed"
	// AgentActionStatusFailed indicates the action failed during execution.
	AgentActionStatusFailed AgentActionStatus = "failed"
)

func (s AgentActionStatus) IsValid() bool {
	switch s {
	case AgentActionStatusPendingReview, AgentActionStatusAutoApproved, AgentActionStatusApproved, AgentActionStatusRejected, AgentActionStatusExecuted, AgentActionStatusFailed:
		return true
	default:
		return false
	}
}

func (s AgentActionStatus) EnumValues() []string {
	return []string{string(AgentActionStatusPendingReview), string(AgentActionStatusAutoApproved), string(AgentActionStatusApproved), string(AgentActionStatusRejected), string(AgentActionStatusExecuted), string(AgentActionStatusFailed)}
}
