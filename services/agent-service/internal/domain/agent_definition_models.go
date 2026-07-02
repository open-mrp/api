package domain

import (
	"encoding/json"
	"time"
)

// AgentDefinitionInfo is the domain representation of an agent definition with its linked tools.
type AgentDefinitionInfo struct {
	ID             string
	Name           string  `audit:"name"`
	Slug           string  `audit:"slug"`
	Description    *string `audit:"description"`
	DefinitionType string  `audit:"definition_type"`
	CategoryCode   string  `audit:"category_code"`
	TriggerType    string  `audit:"trigger_type"`
	IsEditable     bool
	Config         json.RawMessage `audit:"config"`
	RoleID         string          `audit:"role_id"`
	Tools          []AgentDefinitionToolInfo
	AccountStatus  *AgentAccountStatusInfo
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AgentAccountStatusInfo is the domain representation of a per-account status for an agent definition.
type AgentAccountStatusInfo struct {
	ID                string
	AccountID         string
	AgentDefinitionID string
	StatusCode        string `audit:"status_code"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ToolGroupInfo is the domain representation of a tool group.
type ToolGroupInfo struct {
	ID          string
	Name        string
	Description string
	Slug        string
	Icon        string
	SortOrder   int32
}

// AgentDefinitionToolInfo is the domain representation of a tool linked to an agent definition. Display metadata (DisplayName, Description, group, permissions) is resolved from the code catalog (agents.BuiltinTools) by ToolSlug, not stored.
type AgentDefinitionToolInfo struct {
	ID                  string
	ToolSlug            string
	DisplayName         string
	Description         string
	ConfigSchema        json.RawMessage
	Category            string
	Config              json.RawMessage
	SortOrder           int32
	RequireReview       bool
	GroupID             string
	GroupName           string
	RequiredPermissions []string
}

// AvailableToolInfo is the domain representation of a platform tool that can be attached to an agent definition. Slug is the tool's stable identifier (e.g. "lookup_customer").
type AvailableToolInfo struct {
	Slug                string
	DisplayName         string
	Description         string
	ConfigSchema        json.RawMessage
	Category            string
	GroupID             string
	GroupName           string
	RequiredPermissions []string
	RequiredRoleType    string
	// Mutating reports whether the tool takes an externally-visible or irreversible action: any non-GET endpoint-tool, or a built-in tool flagged mutating in the catalog (e.g. send_email). Surfaced so the UI can default such tools to requiring human review.
	Mutating bool
}

// ListAgentDefinitionsParams holds the parameters for listing agent definitions.
type ListAgentDefinitionsParams struct {
	Includes        []string
	Statuses        []string
	DefinitionTypes []string
	TriggerTypes    []string
	Cursor          *string
	Limit           int32
	Query           *string
}

// ListAgentDefinitionsResult holds the result of listing agent definitions.
type ListAgentDefinitionsResult struct {
	Items    []AgentDefinitionInfo
	PageInfo PageInfo
}

// PageInfo holds cursor-based pagination metadata.
type PageInfo struct {
	NextCursor  *string
	PrevCursor  *string
	HasNextPage bool
	HasPrevPage bool
}

// ListAvailableToolsParams holds the parameters for listing available tools.
type ListAvailableToolsParams struct {
	Cursor           *string
	Limit            int32
	Query            *string
	PaginateResource string
}

// CreateCustomAgentParams holds the parameters for creating a custom agent definition.
type CreateCustomAgentParams struct {
	Name         string
	Slug         string
	Description  string
	CategoryCode string
	TriggerType  string
	ConfigJSON   string
	RoleID       string
	Tools        []ToolLinkParams
	Includes     []string
}

// UpdateCustomAgentParams holds the parameters for updating a custom agent definition. Nil pointer fields indicate that the field should not be updated.
type UpdateCustomAgentParams struct {
	AgentDefinitionID string
	Name              *string
	Slug              *string
	Description       *string
	CategoryCode      *string
	TriggerType       *string
	ConfigJSON        *string
	RoleID            *string
	// ClearDescription / ClearRoleID set the respective column to NULL (the value fields are ignored when set).
	ClearDescription bool
	ClearRoleID      bool
	Tools            []ToolLinkParams
	ToolsProvided    bool
	Includes         []string
}

// DeleteCustomAgentParams holds the parameters for deleting a custom agent definition.
type DeleteCustomAgentParams struct {
	AgentDefinitionID string
}

// ToolLinkParams holds the parameters for linking a built-in tool (by slug) to an agent definition.
type ToolLinkParams struct {
	ToolSlug      string
	ConfigJSON    string
	SortOrder     int32
	RequireReview bool
}

// UpdateAgentAccountStatusParams holds the parameters for upserting a per-account agent status.
type UpdateAgentAccountStatusParams struct {
	AgentDefinitionID string
	StatusCode        string
}

// TriggerRunParams holds the parameters for triggering an agent run.
type TriggerRunParams struct {
	AgentDefinitionCode string
	Input               string
}

// ChatRunInput starts a chat-triggered agent run. AgentDefinitionID is the participant's agent identifier; ConversationID/TriggerMessageID link the run to the conversation it replies into.
type ChatRunInput struct {
	AccountID         string
	AgentDefinitionID string
	ConversationID    string
	TriggerMessageID  string
	Message           string
	// History is the recent thread context preceding the trigger (oldest-first), seeded as prior turns so the agent can follow the conversation rather than seeing only the trigger message.
	History []ChatHistoryMessage
	// ContinueRunID, when set, is an existing run to continue (the user replied to that run's message) rather than starting a new one. Falls back to a new run if it isn't continuable.
	ContinueRunID string
}

// ChatHistoryMessage is one prior conversation turn for a chat-triggered run. Role is "assistant" for this agent's own earlier replies, "user" for everyone else; Name is the sender's display name when known (people), empty for agents. AgentConfigID is set when a different agent authored the turn — its Name is resolved from the agent definition when the run is created.
type ChatHistoryMessage struct {
	Role          string `json:"role"`
	Name          string `json:"name,omitempty"`
	AgentConfigID string `json:"agent_config_id,omitempty"`
	Body          string `json:"body"`
}

// CancelRunParams holds the parameters for cancelling an agent run.
type CancelRunParams struct {
	AgentRunID string
}

// ContinueRunParams holds the parameters for continuing an agent run that is awaiting input.
type ContinueRunParams struct {
	AgentRunID        string
	Message           string
	ApprovedToolSlugs []string
	RejectedToolSlugs []string
	// ApprovedToolCallIDs / RejectedToolCallIDs are per-call decisions: the tool_use_ids of individual
	// blocked calls, so two calls of the same slug can be decided independently. See ContinueRunRequest.
	ApprovedToolCallIDs []string
	RejectedToolCallIDs []string
}

// RetryRunParams holds the parameters for retrying a failed run.
type RetryRunParams struct {
	AgentRunID string
}

// MaxManualRetries bounds how many times a failed run may be re-attempted (manual + automatic combined, tracked by agent_run.retry_count) before retry is refused.
const MaxManualRetries = 5

// MaxAutoRetries bounds how many times the runner will transparently auto-retry a run that failed on a transient, whole-chain-unavailable error before it leaves side effects (runCtx.Actions empty). It shares the agent_run.retry_count budget with manual retries and is intentionally smaller than MaxManualRetries so an automatic retry storm cannot exhaust a user's ability to retry by hand.
const MaxAutoRetries = 3

// CreateAgentMemoryParams holds the parameters for creating an agent memory.
type CreateAgentMemoryParams struct {
	Category     string
	Content      string
	MetadataJSON string
	EntityType   string
	EntityID     string
	Importance   float64
	ExpiresAt    string
}

// UpdateAgentMemoryParams holds the parameters for updating an agent memory.
// UpdateAgentMemoryParams is a partial update: nil fields leave the column unchanged. ClearEntity nulls
// entity_type + entity_id (unscopes); ClearExpiresAt nulls expires_at (makes the memory permanent).
type UpdateAgentMemoryParams struct {
	MemoryID       string
	Category       *string
	Content        *string
	MetadataJSON   *string
	EntityType     *string
	EntityID       *string
	Importance     *float64
	ExpiresAt      *string
	ClearEntity    bool
	ClearExpiresAt bool
}

// DeleteAgentMemoryParams holds the parameters for deleting an agent memory.
type DeleteAgentMemoryParams struct {
	MemoryID string
}

// AgentMemoryInfo is the domain representation of an agent memory.
type AgentMemoryInfo struct {
	ID         string
	AccountID  string
	Category   string  `audit:"category"`
	Content    string  `audit:"content"`
	Metadata   string  `audit:"metadata"`
	EntityType string  `audit:"entity_type"`
	EntityID   string  `audit:"entity_id"`
	Importance float64 `audit:"importance"`
	ExpiresAt  string  `audit:"expires_at"`
	CreatedAt  string
	UpdatedAt  string
}
