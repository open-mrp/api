package messaging

import (
	"context"
	"time"
)

// InboxStatus represents the lifecycle state of an inbox record as it moves from initial receipt through successful processing or failure.
type InboxStatus string

const (
	// InboxStatusReceived means the message was inserted into the inbox table but the handler has not yet completed. If the process crashes at this point, the record stays in "received" and the InboxConsumer treats re-delivery as a crash-recovery retry.
	InboxStatusReceived InboxStatus = "received"
	// InboxStatusProcessed means the handler ran to completion and the message should not be processed again. Duplicate deliveries with this status are silently ACKed.
	InboxStatusProcessed InboxStatus = "processed"
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
	// TryInsert attempts to insert a new inbox record with status "received". On success it returns the auto-generated record ID. On duplicate (MySQL error 1062 from the unique index on message_id + handler), it returns 0 and the MySQL error so the caller can branch into duplicate-handling logic.
	TryInsert(ctx context.Context, input InboxRecordInput) (int64, error)

	// GetByMessageAndHandler retrieves the existing inbox record for a given message and handler combination. Used by handleDuplicate to inspect the prior record's status and decide whether to skip, retry, or trigger crash recovery.
	GetByMessageAndHandler(ctx context.Context, messageID, handler string) (*InboxRecord, error)

	// MarkProcessed transitions the record to "processed" status and sets processed_at. Called after the handler completes successfully.
	MarkProcessed(ctx context.Context, id int64) error

	// MarkFailed increments the attempt count and stores the error message from the most recent failed attempt. The record stays in "received" status so it can be retried on re-delivery.
	MarkFailed(ctx context.Context, id int64, errMsg string) error
}

// InboxPurgerRepo defines the persistence interface used by the InboxPurger to delete processed inbox records that have exceeded the retention period.
type InboxPurgerRepo interface {
	// PurgeProcessed deletes processed inbox records older than retentionHours, up to limit rows per call. Returns the number of rows deleted.
	PurgeProcessed(ctx context.Context, retentionHours int, limit int32) (int64, error)
}
