package messaging

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
)

// InboxConsumer wraps message handlers with inbox-based deduplication to achieve exactly-once processing semantics. For each incoming AMQP delivery it:
//  1. Extracts the message ID and metadata from the delivery and body.
//  2. Attempts to insert an inbox record (status = "received").
//  3. If the insert succeeds (new message), the handler is invoked.
//  4. If the insert fails with a duplicate-key error, handleDuplicate inspects the existing record's status to decide whether to skip (already processed), retry (previously failed), or retry (crash recovery — received but never completed).
//
// This pattern guarantees that a handler is invoked at most once for a given (message_id, handler) pair under normal operation, and provides safe retry semantics for crash-recovery scenarios.
type InboxConsumer struct {
	repo        InboxRepo
	serviceName string
	tracer      trace.Tracer
}

// NewInboxConsumer creates a new InboxConsumer that uses the given repository for persistence and derives a tracer scoped to "{serviceName}.inbox_consumer".
func NewInboxConsumer(repo InboxRepo, serviceName string) *InboxConsumer {
	return &InboxConsumer{
		repo:        repo,
		serviceName: serviceName,
		tracer:      tracing.GetTracer(serviceName + ".inbox_consumer"),
	}
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

// executeAndRecord invokes the handler and updates the inbox record based on the outcome. It is called both for new messages and for duplicates that need retry.
func (c *InboxConsumer) executeAndRecord(ctx context.Context, recordID int64, messageID, handler string, fn MessageHandler, msg amqp.Delivery) error {
	if err := fn(ctx, msg); err != nil {
		if markErr := c.repo.MarkFailed(ctx, recordID, err.Error()); markErr != nil {
			slog.Warn("Failed to mark inbox record as failed", "handler", handler, "message_id", messageID, "error", markErr)
		}
		return err
	}

	if err := c.repo.MarkProcessed(ctx, recordID); err != nil {
		slog.Warn("Failed to mark inbox record as processed", "handler", handler, "message_id", messageID, "error", err)
	}

	return nil
}

// handleDuplicate is called when the inbox insert fails with a duplicate-key error, meaning this (message_id, handler) pair was seen before. It fetches the existing record and decides the outcome based on its status:
//   - "processed": the handler already ran to completion — skip silently (return nil so the delivery is ACKed).
//   - has LastError or "received" with no error: a previous attempt failed or crashed — re-invoke the handler to retry processing.
func (c *InboxConsumer) handleDuplicate(ctx context.Context, messageID, handler string, fn MessageHandler, msg amqp.Delivery) error {
	record, err := c.repo.GetByMessageAndHandler(ctx, messageID, handler)
	if err != nil {
		// Can't determine state - log and skip (ACK to prevent infinite redelivery)
		slog.Warn("Duplicate detected but couldn't fetch record", "handler", handler, "message_id", messageID, "error", err)
		return nil
	}

	if record.Status == InboxStatusProcessed {
		slog.Info("Skipping already-processed message", "handler", handler, "message_id", messageID)
		return nil
	}

	// Message needs processing — either a previous failure or crash recovery.
	if record.LastError != nil {
		slog.Warn("Retrying previously-failed message",
			"handler", handler, "message_id", messageID, "attempt", record.Attempts+1, "last_error", *record.LastError)
	} else {
		slog.Warn("Crash recovery: re-processing message",
			"handler", handler, "message_id", messageID, "attempts", record.Attempts)
	}

	return c.executeAndRecord(ctx, record.ID, messageID, handler, fn, msg)
}
