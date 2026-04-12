package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAuditEventID = "ae_01gq7s3f2m0y9h2t7z1w7q3v9k"

const SampleAuditEventAction = constants.AuditActionUpdate
const SampleAuditEventResourceType = constants.ObjectTypeUser
const SampleAuditEventResourceID = SampleUserID

const SampleAuditEventRequestID = "req_01gq7s3f2m0y9h2t7z1w7q3v9k"
const SampleAuditEventSourceIP = "198.51.100.8"

const SampleAuditEventMetadataReason = `{"reason":"operator override"}`

// AuditFieldChange represents a single field-level before/after transition
// recorded during an update mutation.
type AuditFieldChange struct {
	// The name of the field that changed.
	Field string `json:"field" validate:"required"`
	// The previous value of the field as a JSON fragment, or null for creation events.
	OldValue json.RawMessage `json:"old_value"`
	// The new value of the field as a JSON fragment, or null for deletion events.
	NewValue json.RawMessage `json:"new_value"`
}

// AuditEvent represents a single immutable audit event record. Audit events
// capture who mutated a resource, what changed, and when, providing a
// comprehensive edit history for compliance and traceability.
type AuditEvent struct {
	// The unique identifier for the audit event.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=audit_event"`
	// The type of mutation that occurred.
	Action constants.AuditAction `json:"action" validate:"required"`
	// The resource type of the audited entity.
	ResourceType constants.ObjectType `json:"resource_type" validate:"required"`
	// The unique identifier of the audited resource.
	ResourceID string `json:"resource_id" validate:"required"`

	// The actor who performed the mutation.
	Actor *Actor `json:"actor" expandable:"true"`
	// The field-level changes recorded for this event.
	Changes *List[AuditFieldChange] `json:"changes" expandable:"true"`
	// Arbitrary JSON metadata associated with the mutation (e.g. reason, source, tags).
	Metadata json.RawMessage `json:"metadata"`

	// The originating HTTP request ID, when available.
	RequestID *string `json:"request_id"`
	// The idempotency key ID associated with the originating request, when applicable.
	IdempotencyKeyID *string `json:"idempotency_key_id"`
	// The originating client IP address, when available.
	SourceIP *string `json:"source_ip"`

	// When the audited mutation occurred.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// When the audit event record was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleAuditEvent = &AuditEvent{
	ID:           SampleAuditEventID,
	Object:       constants.ObjectTypeAuditEvent,
	Action:       SampleAuditEventAction,
	ResourceType: SampleAuditEventResourceType,
	ResourceID:   SampleAuditEventResourceID,
	Actor:        SampleActor,
	Changes: NewList([]AuditFieldChange{
		{
			Field:    "email",
			OldValue: json.RawMessage(`"old@example.com"`),
			NewValue: json.RawMessage(`"new@example.com"`),
		},
	}, PageInfo{}),
	Metadata:   json.RawMessage(SampleAuditEventMetadataReason),
	RequestID:  new(SampleAuditEventRequestID),
	SourceIP:   new(SampleAuditEventSourceIP),
	OccurredAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AuditEvent) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAuditEvent)
}
