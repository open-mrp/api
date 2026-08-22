package domain

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
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
	InputTokens      int
	OutputTokens     int
	LLMProvider      string
	LLMModel         string
	AwaitingApproval bool
	// Cancelled is true when the run was stopped mid-flight (via CancelRun) rather than finishing on its own. The caller finalizes the run as cancelled and skips the normal completed/awaiting_input transition so a stop request isn't silently overwritten.
	Cancelled bool
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

// HandlerRunContext provides tool handlers with access to repos and accumulated run state.
type HandlerRunContext struct {
	AccountID string
	RunID     string
	// ToolUseID is the LLM-assigned ID of the tool call currently being handled. The runner sets it per-call before dispatch so handlers (notably endpoint-tools) can derive a deterministic idempotency key from RunID+ToolUseID.
	ToolUseID          string
	Definition         *sqlc.AgentDefinition
	Config             *sqlc.AgentConfig
	Repos              RepoFactory
	CoreClient         CoreClient
	GatewayClient      GatewayClient
	NotificationClient NotificationClient
	// ConversationID is the chat conversation this run is linked to (empty for non-chat runs). The email tools use it to address the bound inbox.
	ConversationID       string
	Identity             *types.Identity
	Actions              []PendingAction
	Artifacts            []PendingArtifact
	RequireReviewBySlug  map[string]bool
	AlwaysAllowedSlugs   map[string]bool // reserved approval-bypass; always empty (no "always allow")
	OneTimeApprovedSlugs map[string]bool // from approved_tool_slugs / "approve all": approves EVERY pending call of the slug. Consumed after execution.
	RejectedSlugs        map[string]bool // from rejected_tool_slugs: slugs the human denied this resume — answered with a "denied by user" result so the run continues without them (never paused)

	// OneTimeApprovedKeys / RejectedKeys carry PER-CALL decisions keyed by ToolCallApprovalKey(slug, input),
	// so two blocked calls of the same slug with different inputs can be approved or denied independently
	// (the slug maps above cannot distinguish them). Populated from approved_tool_call_ids / rejected_tool_call_ids
	// on resume; the gates check these alongside the slug maps. Consumed after execution like the slug approvals.
	OneTimeApprovedKeys map[string]bool
	RejectedKeys        map[string]bool

	// AllowedEndpointToolSlugs is the set of endpoint-tools this agent may use (resolved per-agent from config). search_api_tools only surfaces tools in this set, and execution is denied for anything outside it.
	AllowedEndpointToolSlugs map[string]bool
	// RevealedToolSlugs accumulates endpoint-tool slugs surfaced via search_api_tools during this run. The runner reads it to add those tools to the live tool list.
	RevealedToolSlugs map[string]bool
}

// ToolHandlerFunc is the signature for a single tool's execution handler.
type ToolHandlerFunc func(ctx context.Context, input json.RawMessage, runCtx *HandlerRunContext) (string, error)
