package messaging

import (
	"context"
	"time"

	"github.com/augno/api/shared/contracts"
)

// OutboxStatus represents the lifecycle state of an outbox message as it moves from
// creation through publishing or failure.
type OutboxStatus string

const (
	// OutboxStatusPending means the message has been written to the outbox table (inside
	// the service's business transaction) and is waiting for the Enqueuer to pick it up.
	OutboxStatusPending OutboxStatus = "pending"
	// OutboxStatusPublished means the Enqueuer successfully published the message to
	// RabbitMQ and the outbox record has been deleted (or marked) as complete.
	OutboxStatusPublished OutboxStatus = "published"
	// OutboxStatusFailed means the Enqueuer attempted to publish but encountered an
	// error. The attempt count is incremented and the message is scheduled for retry
	// with exponential backoff until MaxAttempts is reached.
	OutboxStatusFailed OutboxStatus = "failed"
)

// OutboxMessageInput contains the data needed to insert a new outbox message. It is
// passed to OutboxRepo.Create inside the same database transaction as the business
// operation, ensuring atomicity between the domain write and the intent to publish.
type OutboxMessageInput struct {
	// MessageID is the globally unique identifier for this message (e.g. mg_abc123).
	MessageID string
	// ServiceName identifies the service that created the message (e.g. "auth-service").
	ServiceName string
	// MessageType categorizes the message for routing and consumer dispatch
	// (e.g. "notification.cmd.send_email").
	MessageType string
	// Destination is the AMQP exchange the message should be published to.
	Destination string
	// RoutingKey is the AMQP routing key used for topic-based queue binding.
	RoutingKey string
	// Payload is the structured message body that will be JSON-serialized and published.
	Payload contracts.AmqpMessage
	// MaxAttempts caps how many times the enqueuer will retry publishing before giving up.
	MaxAttempts int
}

// OutboxMessage represents a row in the outbox table as read by the Enqueuer. It
// includes all columns needed for locking, publishing, retry scheduling, and
// failure tracking.
type OutboxMessage struct {
	ID              int64
	MessageID       string
	ServiceName     string
	MessageType     string
	Destination     string
	RoutingKey      string
	Headers         map[string]any
	Payload         contracts.AmqpMessage
	Status          OutboxStatus
	Attempts        int
	MaxAttempts     int
	NextRunAt       time.Time
	LockedAt        *time.Time
	LockOwner       *string
	LockExpiresAt   *time.Time
	LastError       *string
	PublishedAt     *time.Time
	RequestID       *string
	ParentMessageID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OutboxRepo defines the write-side interface for outbox message persistence. It is
// used by service-layer code to enqueue a message inside the same database transaction
// as the business operation (transactional outbox pattern).
type OutboxRepo interface {
	// Create inserts a new outbox message within the current transaction. The message
	// starts in "pending" status and will be picked up by the Enqueuer's poll loop.
	Create(ctx context.Context, input OutboxMessageInput) (int64, error)
}

// OutboxEnqueuerRepo defines the read/update interface used exclusively by the
// Enqueuer to process outbox messages. It is kept separate from OutboxRepo because
// the enqueuer operates outside of business transactions and needs different
// operations (locking, bulk fetch, status updates).
type OutboxEnqueuerRepo interface {
	// AcquireAndLock atomically selects up to `limit` pending messages whose
	// next_run_at has passed and locks them to the given lockOwner for
	// lockDurationSeconds. Returns the locked messages for publishing.
	AcquireAndLock(ctx context.Context, lockOwner string, limit int, lockDurationSeconds int) ([]*OutboxMessage, error)

	// MarkPublished updates the message status to 'published' with a timestamp,
	// preserving the record for audit and debugging purposes.
	MarkPublished(ctx context.Context, id int64) error

	// MarkFailed increments the message's attempt count, records the error message,
	// and schedules the next retry after retryDelaySecs seconds. If MaxAttempts is
	// exceeded the message remains in "failed" status for manual investigation.
	MarkFailed(ctx context.Context, id int64, errorMsg string, retryDelaySecs int) error

	// CleanupExpiredLocks releases locks held by enqueuer instances that have
	// crashed or stalled (lock_expires_at < now), making those messages eligible
	// for re-acquisition by any healthy enqueuer.
	CleanupExpiredLocks(ctx context.Context, limit int32) (int64, error)

	// PurgePublished deletes published outbox messages older than retentionHours,
	// up to limit rows per call. Returns the number of rows deleted.
	PurgePublished(ctx context.Context, retentionHours int, limit int32) (int64, error)
}
