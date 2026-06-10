package domain

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	types "github.com/augno/api/services/auth-service/pkg/types"
)

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

// HandlerRunContext provides tool handlers with access to repos and accumulated run state.
type HandlerRunContext struct {
	AccountID            string
	RunID                string
	Definition           *sqlc.AgentDefinition
	Config               *sqlc.AgentConfig
	Repos                RepoFactory
	CoreClient           CoreClient
	Identity             *types.Identity
	Actions              []PendingAction
	Artifacts            []PendingArtifact
	Memories             []PendingMemory
	Alerts               []PendingAlert
	RequireReviewBySlug  map[string]bool
	AlwaysAllowedSlugs   map[string]bool // from allowed_tool_slugs, never consumed
	OneTimeApprovedSlugs map[string]bool // from approved_tool_slugs, consumed after execution
}

// ToolHandlerFunc is the signature for a single tool's execution handler.
type ToolHandlerFunc func(ctx context.Context, input json.RawMessage, runCtx *HandlerRunContext) (string, error)
