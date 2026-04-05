package domain

import (
	"encoding/json"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
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

// Trigger types — aliases for shared constants.
const (
	TriggerScheduled = string(constants.AgentTriggerTypeScheduled)
	TriggerManual    = string(constants.AgentTriggerTypeManual)
	TriggerEvent     = string(constants.AgentTriggerTypeEvent)
)

// Definition types — aliases for shared constants.
const (
	DefinitionTypeSystem = string(constants.AgentDefinitionTypeSystem)
	DefinitionTypeCustom = string(constants.AgentDefinitionTypeCustom)
)

// Alert severity — aliases for shared constants.
const (
	SeverityInfo     = string(constants.AgentAlertSeverityInfo)
	SeverityWarning  = string(constants.AgentAlertSeverityWarning)
	SeverityUrgent   = string(constants.AgentAlertSeverityUrgent)
	SeverityCritical = string(constants.AgentAlertSeverityCritical)
)

// Alert statuses — aliases for shared constants.
const (
	AlertStatusOpen         = string(constants.AgentAlertStatusOpen)
	AlertStatusAcknowledged = string(constants.AgentAlertStatusAcknowledged)
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

// RunContext holds the loaded context for an agent run.
type RunContext struct {
	AccountID  string
	RunID      string
	Definition *sqlc.AgentDefinition
	Config     *sqlc.AgentConfig
	Memories   []sqlc.AgentMemory
}

// RunResult holds the outputs of an agent run.
type RunResult struct {
	Output           json.RawMessage
	Actions          []PendingAction
	Artifacts        []PendingArtifact
	Memories         []PendingMemory
	Alerts           []PendingAlert
	InputTokens      int
	OutputTokens     int
	LLMProvider      string
	LLMModel         string
	AwaitingApproval bool
}

// PendingAction represents an action to be persisted after a run completes.
type PendingAction struct {
	ToolSlug       string
	Label          string
	Description    string
	Input          json.RawMessage
	Output         json.RawMessage
	RequiresReview bool
	EntityType     string
	EntityID       string
}

// PendingArtifact represents an artifact to be persisted after a run completes.
type PendingArtifact struct {
	ActionIndex  int
	ArtifactType string
	Name         string
	Content      string
	Metadata     json.RawMessage
	MimeType     string
}

// PendingMemory represents a memory to be persisted after a run completes.
type PendingMemory struct {
	Category   string
	Content    string
	Metadata   json.RawMessage
	EntityType string
	EntityID   string
	Importance float64
}

// PendingAlert represents an alert to be persisted after a run completes.
type PendingAlert struct {
	SeverityCode string
	Title        string
	Message      string
	Metadata     json.RawMessage
}

// ProductResult holds product data returned by core-service.
type ProductResult struct {
	ProductID   string
	ItemID      string
	SKU         string
	Description string
	UnitPrice   string
}

// CustomerResult holds customer data returned by core-service.
type CustomerResult struct {
	RelationID            string
	OwnerAccountID        string
	CounterpartyAccountID string
	RoleCode              string
	Alias                 string
	Email                 string
	UserName              string
}
