package domain

const (
	ServiceName = "agent-service"
)

// Run statuses
const (
	RunStatusPending          = "pending"
	RunStatusRunning          = "running"
	RunStatusCompleted        = "completed"
	RunStatusFailed           = "failed"
	RunStatusCancelled        = "cancelled"
	RunStatusAwaitingInput    = "awaiting_input"
	RunStatusAwaitingApproval = "awaiting_approval"
)

// Action statuses
const (
	ActionStatusPendingReview = "pending_review"
	ActionStatusAutoApproved  = "auto_approved"
	ActionStatusApproved      = "approved"
	ActionStatusRejected      = "rejected"
	ActionStatusExecuted      = "executed"
	ActionStatusFailed        = "failed"
)

// Agent account statuses
const (
	AgentAccountStatusActive   = "active"
	AgentAccountStatusInactive = "inactive"
)

// AllowedModels is the strict allowlist of LLM models agents may use.
// These identifiers match the Stripe AI Gateway naming convention (no date suffixes).
var AllowedModels = map[string]bool{
	// Anthropic
	"claude-sonnet-4":  true,
	"claude-haiku-4.5": true,
	// OpenAI
	"gpt-4o":      true,
	"gpt-4o-mini": true,
}

// DefaultModel is the model used when none is specified.
const DefaultModel = "claude-sonnet-4"
