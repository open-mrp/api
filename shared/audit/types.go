package audit

import (
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
)

// FieldChange represents a single field-level before/after transition.
//
// OldValue/NewValue are JSON fragments (possibly "null").
type FieldChange struct {
	Field    string          `json:"field"`
	OldValue json.RawMessage `json:"old_value"`
	NewValue json.RawMessage `json:"new_value"`
}

// EventData is the producer-side input for publishing an audit event via the platform's transactional outbox pipeline.
type EventData struct {
	// ServiceName identifies the service that produced the audit event.
	ServiceName  string
	Action       constants.AuditAction
	ResourceType constants.ObjectType
	ResourceID   string
	Changes      []FieldChange

	// RootResourceType / RootResourceID identify the root record this audited entity belongs to (e.g. the sales_order that a sales_order_line, pick, shipment, or invoice hangs off of). Optional; when set they let a single query return an entire record tree's history, including deleted children. Leave empty for top-level records with no parent.
	RootResourceType constants.ObjectType
	RootResourceID   string

	// Metadata is any additional JSON-serializable context associated with the mutation (e.g. "reason", "source", "tags").
	Metadata map[string]any
}

// PublishedEvent is the JSON payload persisted in the outbox and later processed by the platform-service audit consumer.
type PublishedEvent struct {
	TypeID           string                `json:"type_id"`
	Action           constants.AuditAction `json:"action"`
	ResourceType     constants.ObjectType  `json:"resource_type"`
	ResourceID       string                `json:"resource_id"`
	RootResourceType constants.ObjectType  `json:"root_resource_type,omitempty"`
	RootResourceID   string                `json:"root_resource_id,omitempty"`
	Changes          []FieldChange         `json:"changes"`
	Metadata         map[string]any        `json:"metadata"`

	ServiceName      string  `json:"service_name"`
	IdempotencyKeyID *string `json:"idempotency_key_id,omitempty"`
	SourceIP         *string `json:"source_ip,omitempty"`

	// OccurredAt is the time the mutation took effect (UTC).
	OccurredAt time.Time `json:"occurred_at"`
}
