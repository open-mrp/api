package messaging

import (
	"context"
	"time"
)

// InboxStatus represents the lifecycle state of an inbox record as it moves from initial receipt through successful processing or failure.
type InboxStatus string

const (
	// InboxStatusReceived means the record exists but the handler has not completed. It is held under a lease (LockOwner, LockExpiresAt): a live lease means an attempt is in flight and a re-delivery must not run the handler again, and an expired one means the attempt was abandoned and the message may be retried.
	InboxStatusReceived InboxStatus = "received"
	// InboxStatusProcessed means the handler ran to completion and the message must never be processed again. Duplicate deliveries with this status are silently ACKed.
	InboxStatusProcessed InboxStatus = "processed"
	// InboxStatusDiscarded means the handler rejected the message as one that can never succeed — a malformed payload, or state the message can no longer be reconciled against. It is terminal and visible to the failure monitor, unlike an ACK that quietly recorded the message as processed.
	InboxStatusDiscarded InboxStatus = "discarded"
	// InboxStatusIgnored means the message was never this handler's work: an event type the service does not subscribe to, a routing key it does not serve. It is terminal like "discarded" and equally distinguishable from work that was applied, but it is not a failure and the failure monitor does not alert on it — these arrive constantly, and alerting on them buries the records that do need a human.
	InboxStatusIgnored InboxStatus = "ignored"
)

// InboxRecordInput contains the data needed to create an inbox record. It is populated from the AMQP delivery metadata and message body by InboxConsumer.Wrap.
type InboxRecordInput struct {
	// MessageID is the globally unique identifier from the AMQP MessageId header.
	MessageID string
	// ServiceName identifies which service is consuming the message.
	ServiceName string
	// Handler is the logical handler name (e.g. "notification.send_email") used to scope deduplication — the same message delivered to different handlers is processed independently.
	Handler string
	// MessageType is the AMQP routing key / message type for observability.
	MessageType string
	// RequestID ties this message back to the originating request for tracing.
	RequestID string
	// ParentMessageID links to the message that caused this one to be emitted.
	ParentMessageID string
	// LockOwner identifies the consumer taking the lease on the new record, so a later delivery can tell an attempt that is still running from one that was abandoned.
	LockOwner string
	// LockTTLSeconds is how long that lease is good for.
	LockTTLSeconds int
}

// InboxRecord represents a row in the inbox table. It tracks delivery state, attempt count, and any error from the most recent processing attempt. The InboxConsumer reads this record on duplicate detection to decide whether to skip, retry, or trigger crash recovery.
type InboxRecord struct {
	ID              int64
	MessageID       string
	ServiceName     string
	Handler         string
	MessageType     string
	RequestID       *string
	ParentMessageID *string
	Status          InboxStatus
	Attempts        int
	LastError       *string
	ReceivedAt      time.Time
	ProcessedAt     *time.Time
	FailedAt        *time.Time
	LockOwner       *string
	LockExpiresAt   *time.Time
}

// LeaseHeld reports whether another attempt is still working this record. A re-delivery that arrives while the lease is held must leave the work alone; once it lapses the record is abandoned and may be retried.
func (r *InboxRecord) LeaseHeld(now time.Time) bool {
	return r.LockExpiresAt != nil && r.LockExpiresAt.After(now)
}

// InboxCheckResult contains the result of checking for a duplicate message. It is used internally by the InboxConsumer to decide the outcome for a re-delivered message.
type InboxCheckResult struct {
	// IsDuplicate is true when an inbox record already exists for this (message_id, handler) pair.
	IsDuplicate bool
	// AlreadyProcessed is true when the existing record has status "processed", meaning the handler already ran to completion.
	AlreadyProcessed bool
	// HasPreviousError is true when the existing record has a non-nil LastError, indicating the previous attempt failed.
	HasPreviousError bool
	// PreviousError contains the error message from the last failed attempt.
	PreviousError *string
	// ExistingRecord is the full inbox record, available for detailed inspection.
	ExistingRecord *InboxRecord
}

// InboxRepo defines the persistence interface for inbox-based message deduplication. Implementations are provided by each service's repository layer, backed by the shared message_inbox table.
type InboxRepo interface {
	// TryInsert attempts to insert a new inbox record with status "received", holding the lease for the caller. On success it returns the auto-generated record ID. On duplicate (the unique index on message_id + handler), it returns 0 and the driver error so the caller can branch into duplicate-handling logic.
	TryInsert(ctx context.Context, input InboxRecordInput) (int64, error)

	// GetByMessageAndHandler retrieves the existing inbox record for a given message and handler combination. Used by handleDuplicate to inspect the prior record's status and decide whether to skip, retry, or trigger crash recovery.
	GetByMessageAndHandler(ctx context.Context, messageID, handler string) (*InboxRecord, error)

	// Claim takes the lease on an existing record whose own lease has lapsed, returning false when another attempt still holds it. Used on re-delivery to decide whether this consumer may retry an unfinished message.
	Claim(ctx context.Context, id int64, owner string, ttlSeconds int) (bool, error)

	// Complete transitions the record to "processed" and releases the lease, returning false when the caller no longer holds the lease on a "received" record — meaning a concurrent attempt finished first and this one must not commit its own work.
	//
	// Handlers whose entire effect is one local transaction should call this on the transaction-scoped repo as the last statement inside that transaction, so the marker and the work commit together. That closes the window where the work commits and the process dies before the marker, which leaves the record indistinguishable from one that never ran and invites a replay to apply it twice.
	Complete(ctx context.Context, id int64, owner string) (bool, error)

	// MarkFailed increments the attempt count, stamps failed_at, stores the error, and releases the lease. The record stays "received" so a re-delivery can retry it. It is a no-op unless the caller still holds the lease: an attempt whose lease already lapsed must not clear the lease of the consumer that claimed the record after it.
	MarkFailed(ctx context.Context, id int64, owner, errMsg string) error

	// MarkDiscarded moves the record to the terminal "discarded" state with the reason that made it unprocessable, releasing the lease. It is a no-op unless the caller still holds the lease on a "received" record.
	MarkDiscarded(ctx context.Context, id int64, owner, reason string) error

	// MarkIgnored moves the record to the terminal "ignored" state, for a message that was never this handler's work. Guarded like MarkDiscarded.
	MarkIgnored(ctx context.Context, id int64, owner, reason string) error
}

// InboxPurgerRepo defines the persistence interface used by the InboxPurger to delete processed inbox records that have exceeded the retention period.
type InboxPurgerRepo interface {
	// PurgeProcessed deletes processed inbox records older than retentionHours, up to limit rows per call. Returns the number of rows deleted.
	PurgeProcessed(ctx context.Context, retentionHours int, limit int32) (int64, error)
}
