package domain

import (
	"encoding/json"
	"time"
)

// AgentDefinitionInfo is the domain representation of an agent definition
// with its linked tools.
type AgentDefinitionInfo struct {
	ID             string
	Name           string `audit:"name"`
	Slug           string `audit:"slug"`
	Description    string `audit:"description"`
	DefinitionType string `audit:"definition_type"`
	CategoryCode   string `audit:"category_code"`
	TriggerType    string `audit:"trigger_type"`
	IsEditable     bool
	Config         json.RawMessage `audit:"config"`
	RoleID         string          `audit:"role_id"`
	Tools          []AgentDefinitionToolInfo
	AccountStatus  *AgentAccountStatusInfo
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AgentAccountStatusInfo is the domain representation of a per-account
// status for an agent definition.
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

// AgentDefinitionToolInfo is the domain representation of a tool linked to
// an agent definition, including denormalized tool metadata.
type AgentDefinitionToolInfo struct {
	ID                  string
	ToolID              string
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

// AvailableToolInfo is the domain representation of a platform tool that
// can be attached to an agent definition.
type AvailableToolInfo struct {
	ID                  string
	DisplayName         string
	Description         string
	ConfigSchema        json.RawMessage
	Category            string
	GroupID             string
	GroupName           string
	RequiredPermissions []string
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
	Cursor *string
	Limit  int32
	Query  *string
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

// UpdateCustomAgentParams holds the parameters for updating a custom agent definition.
// Nil pointer fields indicate that the field should not be updated.
type UpdateCustomAgentParams struct {
	AgentDefinitionID string
	Name              *string
	Slug              *string
	Description       *string
	CategoryCode      *string
	TriggerType       *string
	ConfigJSON        *string
	RoleID            *string
	Tools             []ToolLinkParams
	ToolsProvided     bool
	Includes          []string
}

// DeleteCustomAgentParams holds the parameters for deleting a custom agent definition.
type DeleteCustomAgentParams struct {
	AgentDefinitionID string
}

// ToolLinkParams holds the parameters for linking a tool to an agent definition.
type ToolLinkParams struct {
	ToolID        string
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

// CancelRunParams holds the parameters for cancelling an agent run.
type CancelRunParams struct {
	AgentRunID string
}

// ContinueRunParams holds the parameters for continuing an agent run that is awaiting input.
type ContinueRunParams struct {
	AgentRunID        string
	Message           string
	ApprovedToolSlugs []string
	AllowedToolSlugs  []string
}

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

// AcknowledgeAgentAlertParams holds the parameters for acknowledging an agent alert.
type AcknowledgeAgentAlertParams struct {
	AlertID string
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

// AgentAlertInfo is the domain representation of an agent alert.
type AgentAlertInfo struct {
	ID                      string
	AccountID               string
	AgentRunID              string
	AgentActionID           string
	SeverityCode            string
	StatusCode              string
	Title                   string
	Message                 string
	Metadata                string
	AcknowledgedAt          string
	AcknowledgedBy          string
	AcknowledgedByActorType string
	AcknowledgedByActorName string
	CreatedAt               string
	UpdatedAt               string
}
