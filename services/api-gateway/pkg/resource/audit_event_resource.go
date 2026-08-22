package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAuditEventID = "ae_emripvn8t1xl"

const SampleAuditEventAction = constants.AuditActionUpdate
const SampleAuditEventResourceType = constants.ObjectTypeUser
const SampleAuditEventResourceID = SampleUserID

const SampleAuditEventSourceIP = "198.51.100.8"

const SampleAuditEventIdempotencyKey = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"

const SampleAuditEventMetadataReason = `{"reason":"operator override"}`

// Field-level before/after transition recorded during a mutation.
type AuditFieldChange struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=audit_field_change"`
	// Name of the changed field.
	//
	// Field names come from the audited record's stored representation and can differ slightly from the corresponding field on the API resource — for example `commission_policy_code` rather than `commission_policy`.
	Field string `json:"field" validate:"required"`
	// Previous value as a JSON fragment.
	//
	// `null` on `create` events, where the field had no prior value.
	OldValue json.RawMessage `json:"old_value"`
	// New value as a JSON fragment.
	//
	// `null` on `delete` events, where the field has no remaining value.
	NewValue json.RawMessage `json:"new_value"`
}

// An immutable record of a single change to a resource, capturing who made the change, what changed, and when.
//
// Audit events are recorded automatically as mutations happen; they cannot be created, edited, or deleted through the API. Recording is asynchronous, so an event may take a moment to become readable after the request that caused it has returned. An update that leaves every tracked field at its existing value records no event unless the mutation attaches metadata of its own — a password rotation, for example, records metadata and no field changes.
type AuditEvent struct {
	// Audit event ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=audit_event"`
	// The type of action this event records.
	//
	// - `create`: the resource was created.
	// - `update`: one or more fields were changed.
	// - `delete`: the resource was deleted.
	// - `restore`: a previously deleted resource was restored.
	// - `archive`: the resource was archived.
	// - `approve`: a human approved a gated action, such as allowing a review-gated agent tool to run.
	// - `deny`: a human denied a gated action, such as rejecting a review-gated agent tool.
	Action constants.AuditAction `json:"action" validate:"required"`
	// Resource type of the audited entity.
	ResourceType constants.ObjectType `json:"resource_type" validate:"required"`
	// Audited resource ID.
	ResourceID string `json:"resource_id" validate:"required"`
	// Actor who performed the mutation.
	Actor *Actor `json:"actor" expandable:"true"`
	// Account the audited mutation was performed against.
	//
	// For a mutation on one of your own resources this is your account; when you act on a customer's or supplier's account, it is that account.
	Account *Account `json:"account" expandable:"true"`
	// Field-level changes recorded for this event.
	//
	// Only fields OpenMRP tracks for that resource type are compared, and only those whose value actually differs are listed. Actions that do not alter stored fields, such as `approve` and `deny`, generally record no changes.
	Changes *List[AuditFieldChange] `json:"changes" expandable:"true"`
	// Arbitrary JSON metadata for the mutation (e.g. reason, source, tags).
	Metadata json.RawMessage `json:"metadata"`
	// Log of the API request that caused the mutation.
	//
	// Changes that did not originate from an API request have no originating request log.
	Request *RequestLog `json:"request" expandable:"true"`
	// Idempotency key of the originating request.
	IdempotencyKey *string `json:"idempotency_key"`
	// Originating client IP address.
	SourceIP *string `json:"source_ip"`
	// When the audited mutation occurred.
	//
	// Audit events are ordered and date-filtered by this timestamp rather than by `created_at`.
	OccurredAt time.Time `json:"occurred_at" validate:"required"`
	// When the audit event record was written.
	//
	// Slightly later than `occurred_at`, since events are recorded out of band from the request that caused them.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleAuditEvent = &AuditEvent{
	ID:           SampleAuditEventID,
	Object:       constants.ObjectTypeAuditEvent,
	Action:       SampleAuditEventAction,
	ResourceType: SampleAuditEventResourceType,
	ResourceID:   SampleAuditEventResourceID,
	Actor:        SampleActor,
	Account:      SampleAccount,
	Changes: NewList([]AuditFieldChange{
		{
			Object:   constants.ObjectTypeAuditFieldChange,
			Field:    "email",
			OldValue: json.RawMessage(`"old@example.com"`),
			NewValue: json.RawMessage(`"new@example.com"`),
		},
	}, PageInfo{}),
	Metadata:       json.RawMessage(SampleAuditEventMetadataReason),
	Request:        SampleRequestLog,
	IdempotencyKey: new(SampleAuditEventIdempotencyKey),
	SourceIP:       new(SampleAuditEventSourceIP),
	OccurredAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*AuditEvent) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAuditEvent)
}
