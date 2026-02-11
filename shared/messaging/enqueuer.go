package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/augno/api/shared/appctx"
)

// EnqueuerConfig holds the configuration for the outbox enqueuer.
type EnqueuerConfig struct {
	// ServiceName (required) identifies which service owns this enqueuer instance.
	// Used in log messages and to scope outbox queries to the owning service's messages.
	ServiceName string

	// LockOwner (optional; default: "{hostname}-{pid}") is a unique identifier for this
	// process instance, used to claim outbox messages via optimistic locking.
	LockOwner string

	// PollInterval (optional; default: 30s) controls how frequently the enqueuer polls
	// the outbox table for pending messages.
	PollInterval time.Duration

	// BatchSize (optional; default: 100) is the maximum number of outbox messages to lock
	// and publish in a single poll cycle.
	BatchSize int

	// LockDurationSeconds (optional; default: 60) is how long (in seconds) a message
	// remains locked to this enqueuer before the lock expires and the message becomes
	// eligible for another instance to pick up.
	LockDurationSeconds int

	// CleanupInterval (optional; default: 30s) controls how frequently the enqueuer runs
	// its expired-lock cleanup pass, releasing locks held by crashed processes.
	CleanupInterval time.Duration
}

// WithDefaults fills zero-value fields with production defaults and returns the config.
// It computes a unique lock owner from the hostname and process ID when not set.
func (c *EnqueuerConfig) WithDefaults() *EnqueuerConfig {
	if c == nil {
		c = &EnqueuerConfig{}
	}

	if c.LockOwner == "" {
		hostname, _ := os.Hostname()
		c.LockOwner = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}
	if c.PollInterval == 0 {
		c.PollInterval = 30 * time.Second
	}
	if c.BatchSize == 0 {
		c.BatchSize = 100
	}
	if c.LockDurationSeconds == 0 {
		c.LockDurationSeconds = 60
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = 30 * time.Second
	}
	return c
}

// validate checks that the config has all required fields.
func (c *EnqueuerConfig) validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("enqueuer: service name is required")
	}
	return nil
}

// Enqueuer implements the publishing side of the transactional outbox pattern.
// It runs two background goroutines:
//   - pollLoop: on each tick, acquires a batch of pending outbox messages (with
//     optimistic locking), publishes each to RabbitMQ, and marks them as published
//     or failed in the database.
//   - cleanupLoop: on each tick, releases expired locks left by crashed instances
//     so those messages become eligible for re-processing.
//
// Start and Stop control the goroutine lifecycle. Tracing is disabled for outbox
// operations (via appctx.WithNoTrace) to avoid cluttering the trace backend with
// high-volume background traffic.
type Enqueuer struct {
	config EnqueuerConfig
	repo   OutboxEnqueuerRepo
	broker MessageBroker

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEnqueuer creates a new outbox enqueuer. Pass a config with at least
// ServiceName set; zero-value fields are filled with production defaults.
func NewEnqueuer(config *EnqueuerConfig, repo OutboxEnqueuerRepo, broker MessageBroker) (*Enqueuer, error) {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &Enqueuer{
		config: *config,
		repo:   repo,
		broker: broker,
	}, nil
}

// Start launches the poll and cleanup goroutines. The provided context is used as
// the parent for all background operations; cancelling it (or calling Stop) shuts
// down both loops. Tracing is disabled on the derived context so outbox polling
// does not generate trace spans.
func (e *Enqueuer) Start(ctx context.Context) error {
	// Disable tracing for background outbox operations to avoid cluttering traces
	ctx = appctx.WithNoTrace(ctx)
	e.ctx, e.cancel = context.WithCancel(ctx)

	e.wg.Add(2)
	go e.pollLoop()
	go e.cleanupLoop()

	slog.Info("Outbox enqueuer started",
		"service", e.config.ServiceName,
		"lock_owner", e.config.LockOwner,
		"poll_interval", e.config.PollInterval,
		"batch_size", e.config.BatchSize,
	)

	return nil
}

// Stop cancels the background context and blocks until both the poll and cleanup
// goroutines have exited. It is safe to call from a deferred shutdown path.
func (e *Enqueuer) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
	slog.Info("Outbox enqueuer stopped", "service", e.config.ServiceName)
}

// pollLoop ticks at PollInterval and calls processBatch on each tick. It exits
// when the enqueuer's context is cancelled.
func (e *Enqueuer) pollLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.processBatch()
		}
	}
}

// cleanupLoop ticks at CleanupInterval and releases expired outbox locks so
// messages owned by crashed instances become eligible for re-processing. It exits
// when the enqueuer's context is cancelled.
func (e *Enqueuer) cleanupLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.cleanupExpiredLocks()
		}
	}
}

// processBatch acquires up to BatchSize pending outbox messages (locked to this
// instance's LockOwner), publishes each to RabbitMQ via publishMessage, and marks
// the result in the database. Successfully published messages are deleted from the
// outbox (MarkPublished). Failed messages have their attempt count incremented and
// are scheduled for retry with exponential backoff (MarkFailed).
func (e *Enqueuer) processBatch() {
	messages, err := e.repo.AcquireAndLock(e.ctx, e.config.LockOwner, e.config.BatchSize, e.config.LockDurationSeconds)
	if err != nil {
		slog.Error("Failed to acquire outbox messages", "error", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	slog.Debug("Processing outbox messages", "count", len(messages))

	for _, msg := range messages {
		if err := e.publishMessage(msg); err != nil {
			slog.Error("Failed to publish outbox message",
				"message_id", msg.MessageID,
				"message_type", msg.MessageType,
				"error", err,
			)
			if markErr := e.repo.MarkFailed(e.ctx, msg.ID, err.Error()); markErr != nil {
				slog.Error("Failed to mark message as failed", "id", msg.ID, "error", markErr)
			}
			continue
		}

		if err := e.repo.MarkPublished(e.ctx, msg.ID); err != nil {
			slog.Error("Failed to mark message as published", "id", msg.ID, "error", err)
		} else {
			slog.Debug("Published outbox message",
				"message_id", msg.MessageID,
				"message_type", msg.MessageType,
			)
		}
	}
}

// publishMessage publishes an OutboxMessage via the message broker. It copies
// metadata fields (MessageID, RequestID, ParentMessageID) from the outbox record
// onto the AMQP payload to ensure tracing correlation survives the outbox hop.
func (e *Enqueuer) publishMessage(msg *OutboxMessage) error {
	payload := msg.Payload
	payload.MessageID = msg.MessageID
	if msg.RequestID != nil && *msg.RequestID != "" {
		payload.RequestID = *msg.RequestID
	}
	if msg.ParentMessageID != nil && *msg.ParentMessageID != "" {
		payload.ParentMessageID = *msg.ParentMessageID
	}

	return e.broker.PublishMessage(e.ctx, msg.Destination, msg.RoutingKey, payload)
}

// cleanupExpiredLocks delegates to the repository to release outbox locks that have
// exceeded LockDurationSeconds. This handles the case where a process crashes after
// locking messages but before publishing them — the locks expire and the messages
// become available for another enqueuer instance to pick up.
func (e *Enqueuer) cleanupExpiredLocks() {
	count, err := e.repo.CleanupExpiredLocks(e.ctx)
	if err != nil {
		slog.Error("Failed to cleanup expired locks", "error", err)
		return
	}

	if count > 0 {
		slog.Info("Cleaned up expired outbox locks", "count", count)
	}
}
