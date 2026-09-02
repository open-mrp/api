package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
)

// DefaultInboxLeaseSeconds bounds how long a crashed consumer can hold a message before another attempt may retry it. It is generous relative to the handlers here (the slowest inventory scans run in seconds) because the cost of expiring too early is a duplicate application, while the cost of expiring too late is only delay.
const DefaultInboxLeaseSeconds = 300

// ErrInboxLeaseHeld is returned when a re-delivery arrives while another attempt still holds the record's lease.
//
// The consumer's backoff ladder is far shorter than a lease, so in practice this dead-letters the delivery rather than waiting the lease out. That is the intended trade: a message parked on the DLQ is visible, alerted on, and re-drivable by the replay commands, where running it alongside the attempt that holds the lease is how the same work gets applied twice. It should be rare — an attempt that ends normally releases its lease, so this only fires for a redelivery that lands while a genuinely live attempt is running, or inside the lease window after a process was killed outright.
var ErrInboxLeaseHeld = errors.New("inbox record is leased by another attempt")

// ErrInboxDiscarded is returned by Discard. It signals that the message ended deliberately in a terminal state, so Wrap ACKs it instead of recording a failure that would invite a retry.
var ErrInboxDiscarded = errors.New("inbox record was discarded")

// ErrInboxAlreadyCompleted is returned from a transactional recovery point when the record was no longer "received", meaning a concurrent attempt completed the message first. Handlers must let it abort their transaction so the duplicate work rolls back.
var ErrInboxAlreadyCompleted = errors.New("inbox record was already completed by another attempt")

type inboxRecordIDKey struct{}

// WithInboxRecordID puts the record id on the context so a handler can commit its own recovery point.
func WithInboxRecordID(ctx context.Context, recordID int64) context.Context {
	return context.WithValue(ctx, inboxRecordIDKey{}, recordID)
}

// InboxRecordIDFromContext returns the inbox record id for the delivery being handled.
//
// A handler whose entire effect is one local transaction uses this to call InboxRepo.Complete on the transaction-scoped repo inside that transaction, so the marker commits with the work. Handlers that mutate foreign state cannot do this — there is no transaction spanning the foreign call — and rely on the lease instead.
func InboxRecordIDFromContext(ctx context.Context) (int64, bool) {
	recordID, ok := ctx.Value(inboxRecordIDKey{}).(int64)
	return recordID, ok
}

// InboxConsumer wraps message handlers with inbox-based deduplication. For each delivery it:
//  1. Extracts the message ID and metadata from the delivery and body.
//  2. Attempts to insert an inbox record (status "received", lease held by this consumer).
//  3. If the insert succeeds, the handler is invoked under that lease.
//  4. If the insert fails with a duplicate-key error, handleDuplicate inspects the existing record to decide whether to skip it (terminal), leave it alone (leased by a live attempt), or claim and retry it (lease lapsed).
//
// The lease is what makes step 4 safe. Without it "received" cannot distinguish an attempt still running, an attempt that died before doing the work, and an attempt that did the work and died before recording it — and re-invoking the handler on that last case applies the message twice.
//
// The lease bounds duplicate work but does not by itself prevent it, because a handler can still commit and then lose the process before the marker is written. Handlers that need exactly-once commit their own recovery point inside their transaction; see InboxRecordIDFromContext.
type InboxConsumer struct {
	repo         InboxRepo
	serviceName  string
	owner        string
	leaseSeconds int
	tracer       trace.Tracer
}

// NewInboxConsumer creates an InboxConsumer that uses the given repository for persistence and derives a tracer scoped to "{serviceName}.inbox_consumer". Each consumer mints its own lease owner id so a record's lease identifies the attempt holding it.
func NewInboxConsumer(repo InboxRepo, serviceName string) *InboxConsumer {
	owner, err := id.GenID(id.MessageIDPrefix, nil)
	if err != nil {
		owner = serviceName
	}
	return &InboxConsumer{
		repo:         repo,
		serviceName:  serviceName,
		owner:        owner,
		leaseSeconds: DefaultInboxLeaseSeconds,
		tracer:       tracing.GetTracer(serviceName + ".inbox_consumer"),
	}
}

// WithLeaseSeconds overrides how long this consumer's leases last. Raise it for handlers that legitimately run longer than DefaultInboxLeaseSeconds, so a slow attempt is not mistaken for an abandoned one.
func (c *InboxConsumer) WithLeaseSeconds(seconds int) *InboxConsumer {
	if seconds > 0 {
		c.leaseSeconds = seconds
	}
	return c
}

// Wrap returns a new MessageHandler that guards fn with inbox deduplication. The handler parameter is a human-readable name that scopes the deduplication — the same message ID processed by different handlers (e.g. "notification.send_email" vs "notification.log_email") is treated as distinct and both execute.
//
// Metadata (message ID, request ID, parent message ID) is extracted from the AMQP delivery headers and body. If no message ID can be found, the handler runs without deduplication (with a warning log) to avoid silently dropping messages.
func (c *InboxConsumer) Wrap(handler string, fn MessageHandler) MessageHandler {
	return func(ctx context.Context, msg amqp.Delivery) error {
		ctx, span := c.tracer.Start(ctx, "inbox.wrap."+handler)
		defer span.End()

		// Extract message metadata from the AMQP message
		messageID := msg.MessageId
		messageType := msg.RoutingKey
		var requestID, parentMessageID string

		// Try to extract additional metadata from the message body
		var amqpMsg contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &amqpMsg); err == nil {
			if amqpMsg.RequestID != "" {
				requestID = amqpMsg.RequestID
			}
			if amqpMsg.ParentMessageID != "" {
				parentMessageID = amqpMsg.ParentMessageID
			}
			if amqpMsg.MessageID != "" && messageID == "" {
				messageID = amqpMsg.MessageID
			}
		}

		// If no message ID, we can't deduplicate - just process
		if messageID == "" {
			slog.Warn("No message ID found, processing without deduplication", "handler", handler)
			return fn(ctx, msg)
		}

		// Try to insert the inbox record
		input := InboxRecordInput{
			MessageID:       messageID,
			ServiceName:     c.serviceName,
			Handler:         handler,
			MessageType:     messageType,
			RequestID:       requestID,
			ParentMessageID: parentMessageID,
			LockOwner:       c.owner,
			LockTTLSeconds:  c.leaseSeconds,
		}

		recordID, err := c.repo.TryInsert(ctx, input)
		if err != nil {
			// A duplicate (message_id, handler) means this delivery is a redelivery — route to the
			// dedup path. db.IsDuplicateEntry covers both MySQL (1062) and PostgreSQL (23505), so this
			// works for inbox tables on either engine; a MySQL-only check silently fell through here for
			// Postgres-backed services (e.g. agent-service), defeating deduplication entirely.
			if db.IsDuplicateEntry(err) {
				return c.handleDuplicate(ctx, messageID, handler, fn, msg)
			}
			// Some other error - let the handler proceed but log the issue
			slog.Warn("Failed to insert inbox record", "handler", handler, "message_id", messageID, "error", err)
			return fn(ctx, msg)
		}

		// New message - process it
		return c.executeAndRecord(ctx, recordID, messageID, handler, fn, msg)
	}
}

// bookkeepingTimeout bounds the detached writes that close out a delivery.
const bookkeepingTimeout = 5 * time.Second

// bookkeepingContext detaches the record-keeping writes from the delivery's context.
//
// Shutdown cancels that context, and a MarkFailed that fails with it leaves the lease held for its full duration: the message is then neither retryable nor visibly failed until the lease lapses. The outcome of an attempt has to be recorded even when the attempt was cut short, which is the one case where it matters most.
func bookkeepingContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), bookkeepingTimeout)
}

// executeAndRecord invokes the handler under this consumer's lease and records the outcome. It is called both for new messages and for duplicates this consumer has claimed.
//
// Complete is best-effort here: a handler that committed its own recovery point has already marked the record inside its transaction, and this call is then a no-op repeat. For handlers that did not, this is the only marker, and the window between the handler returning and this write is exactly why those handlers get at-most-once rather than exactly-once.
func (c *InboxConsumer) executeAndRecord(ctx context.Context, recordID int64, messageID, handler string, fn MessageHandler, msg amqp.Delivery) error {
	if err := fn(WithInboxRecordID(ctx, recordID), msg); err != nil {
		// The handler ended the message deliberately; the terminal state is already recorded and must not be overwritten with a failure that invites a retry.
		if errors.Is(err, ErrInboxDiscarded) {
			return nil
		}
		// A concurrent attempt completed this message and the handler rolled its own work back. Nothing failed and nothing is owed.
		if errors.Is(err, ErrInboxAlreadyCompleted) {
			slog.Info("Message completed concurrently by another attempt", "handler", handler, "message_id", messageID)
			return nil
		}
		markCtx, cancel := bookkeepingContext(ctx)
		defer cancel()
		if markErr := c.repo.MarkFailed(markCtx, recordID, apierror.Describe(err)); markErr != nil {
			slog.Warn("Failed to mark inbox record as failed", "handler", handler, "message_id", messageID, "error", markErr)
		}
		return err
	}

	completeCtx, cancel := bookkeepingContext(ctx)
	defer cancel()
	if _, err := c.repo.Complete(completeCtx, recordID); err != nil {
		slog.Warn("Failed to mark inbox record as processed", "handler", handler, "message_id", messageID, "error", err)
	}

	return nil
}

// handleDuplicate is called when the inbox insert fails with a duplicate-key error, meaning this (message_id, handler) pair was seen before. It fetches the existing record and decides the outcome from its status and lease:
//   - "processed" or "discarded": terminal — skip silently (return nil so the delivery is ACKed).
//   - "received" with a live lease: another attempt is working it — return ErrInboxLeaseHeld so this delivery backs off rather than running the handler alongside it.
//   - "received" with a lapsed lease: the previous attempt was abandoned — claim the lease and retry.
func (c *InboxConsumer) handleDuplicate(ctx context.Context, messageID, handler string, fn MessageHandler, msg amqp.Delivery) error {
	record, err := c.repo.GetByMessageAndHandler(ctx, messageID, handler)
	if err != nil {
		// Can't determine state - log and skip (ACK to prevent infinite redelivery)
		slog.Warn("Duplicate detected but couldn't fetch record", "handler", handler, "message_id", messageID, "error", err)
		return nil
	}

	switch record.Status {
	case InboxStatusProcessed:
		slog.Info("Skipping already-processed message", "handler", handler, "message_id", messageID)
		return nil
	case InboxStatusDiscarded:
		slog.Info("Skipping discarded message", "handler", handler, "message_id", messageID)
		return nil
	}

	if record.LeaseHeld(time.Now()) {
		slog.Info("Message is leased by another attempt, backing off",
			"handler", handler, "message_id", messageID, "lock_expires_at", record.LockExpiresAt)
		return ErrInboxLeaseHeld
	}

	// Claim is conditional on the lease still being free, so two consumers racing the same abandoned record cannot both proceed.
	claimed, err := c.repo.Claim(ctx, record.ID, c.owner, c.leaseSeconds)
	if err != nil {
		slog.Warn("Failed to claim inbox record", "handler", handler, "message_id", messageID, "error", err)
		return err
	}
	if !claimed {
		return ErrInboxLeaseHeld
	}

	if record.LastError != nil {
		slog.Warn("Retrying previously-failed message",
			"handler", handler, "message_id", messageID, "attempt", record.Attempts+1, "last_error", *record.LastError)
	} else {
		slog.Warn("Retrying message abandoned by a previous attempt",
			"handler", handler, "message_id", messageID, "attempts", record.Attempts)
	}

	return c.executeAndRecord(ctx, record.ID, messageID, handler, fn, msg)
}

// Discard ends the in-flight message in the terminal "discarded" state and returns ErrInboxDiscarded.
//
// Use it wherever a handler would otherwise return nil to drop an unprocessable message. Returning nil ACKs the delivery and lets Wrap record it as processed, which claims work was applied that never was; the message then looks identical to a successful one in the inbox, in the failure monitor, and to the replay tools.
func (c *InboxConsumer) Discard(ctx context.Context, reason string) error {
	recordID, ok := InboxRecordIDFromContext(ctx)
	if !ok {
		slog.WarnContext(ctx, "Discarding message with no inbox record", "reason", reason)
		return nil
	}
	discardCtx, cancel := bookkeepingContext(ctx)
	defer cancel()
	if err := c.repo.MarkDiscarded(discardCtx, recordID, reason); err != nil {
		slog.WarnContext(ctx, "Failed to mark inbox record as discarded", "reason", reason, "error", err)
	}
	return ErrInboxDiscarded
}
