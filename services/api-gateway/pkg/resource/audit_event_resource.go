package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAuditEventID = "ae_01b1c07dc3085bbd84111edcbd"

const SampleAuditEventAction = constants.AuditActionUpdate
const SampleAuditEventResourceType = constants.ObjectTypeUser
const SampleAuditEventResourceID = SampleUserID

const SampleAuditEventSourceIP = "198.51.100.8"

const SampleAuditEventMetadataReason = `{"reason":"operator override"}`

// Field-level before/after transition recorded during a mutation.
type AuditFieldChange struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=audit_field_change"`
	// Name of the changed field.
	Field string `json:"field" validate:"required"`
	// Previous value as a JSON fragment. Null for creation events.
	OldValue json.RawMessage `json:"old_value"`
	// New value as a JSON fragment. Null for deletion events.
	NewValue json.RawMessage `json:"new_value"`
}

// Immutable audit event record. Captures the actor, changed resource, and timestamp.
type AuditEvent struct {
	// Audit event ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=audit_event"`
	// Mutation type.
	Action constants.AuditAction `json:"action" validate:"required"`
	// Resource type of the audited entity.
	ResourceType constants.ObjectType `json:"resource_type" validate:"required"`
	// Audited resource ID.
	ResourceID string `json:"resource_id" validate:"required"`
	// Actor who performed the mutation.
	Actor *Actor `json:"actor" expandable:"true"`
	// Field-level changes recorded for this event.
	Changes *List[AuditFieldChange] `json:"changes" expandable:"true"`
	// Arbitrary JSON metadata for the mutation (e.g. reason, source, tags).
	Metadata json.RawMessage `json:"metadata"`
	// Originating HTTP request. Expandable.
	Request *RequestLog `json:"request" expandable:"true"`
	// Idempotency key of the originating request.
	IdempotencyKey *string `json:"idempotency_key"`
	// Originating client IP address.
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
			Object:   constants.ObjectTypeAuditFieldChange,
			Field:    "email",
			OldValue: json.RawMessage(`"old@example.com"`),
			NewValue: json.RawMessage(`"new@example.com"`),
		},
	}, PageInfo{}),
	Metadata: json.RawMessage(SampleAuditEventMetadataReason),
	Request: &RequestLog{
		ID:     SampleRequestLogID,
		Object: constants.ObjectTypeRequestLog,
	},
	SourceIP:   new(SampleAuditEventSourceIP),
	OccurredAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AuditEvent) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAuditEvent)
}
