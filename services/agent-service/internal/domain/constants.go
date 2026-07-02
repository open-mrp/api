package domain

import "github.com/augno/api/shared/constants"

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

// RunStatusIsCancellable reports whether a run in this status can still be stopped by a user. A run is cancellable while it is doing or waiting to do work — actively running/pending, or paused awaiting the user (a chat run between turns, or one blocked on tool approval). The terminal states
// (completed/failed/cancelled) have nothing left to stop.
func RunStatusIsCancellable(status string) bool {
	switch status {
	case RunStatusPending, RunStatusRunning, RunStatusAwaitingInput, RunStatusAwaitingApproval:
		return true
	default:
		return false
	}
}

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

// ModelInfo describes an LLM model agents can be configured to use.
type ModelInfo struct {
	// Code is the stable identifier used to select the model (matches the Stripe AI Gateway naming convention, no date suffixes).
	Code constants.Model
	// Name is the human-readable display name.
	Name string
	// Provider is the display name of the company that makes the model.
	Provider string
}

// AvailableModels is the ordered catalog of LLM models agents may use, derived from the shared constants.ModelCatalog (the single source of truth) so the model-list endpoint and the AllowedModels validation set never drift from the create-agent enum.
var AvailableModels = func() []ModelInfo {
	out := make([]ModelInfo, len(constants.ModelCatalog))
	for i, s := range constants.ModelCatalog {
		out[i] = ModelInfo{Code: s.ID, Name: s.Name, Provider: s.Provider}
	}
	return out
}()

// AllowedModels is the strict allowlist of model codes agents may use, derived from AvailableModels.
var AllowedModels = func() map[string]bool {
	allowed := make(map[string]bool, len(AvailableModels))
	for _, m := range AvailableModels {
		allowed[string(m.Code)] = true
	}
	return allowed
}()

// DefaultModel is the model used when none is specified.
const DefaultModel = string(constants.ModelClaudeSonnet4)
