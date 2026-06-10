package messaging

import (
	"context"

	"github.com/augno/api/shared/contracts"
	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler is the callback signature for processing a single AMQP delivery.
// Implementations should return nil on success (the message will be ACKed) or an
// error on failure (the message will be rejected to the dead-letter queue after
// retry exhaustion).
type MessageHandler func(context.Context, amqp.Delivery) error

// ConsumeOptions configures a single queue consumer.
type ConsumeOptions struct {
	// Concurrency is the number of worker goroutines processing deliveries for
	// this queue. The default of 1 preserves strict in-order, one-at-a-time
	// processing. Values above 1 are only safe for queues whose messages are
	// independently processable — no cross-message ordering requirements —
	// such as request-log and audit-event persistence, where each message is
	// an independent row and the inbox pattern deduplicates redeliveries.
	Concurrency int
}

// ConsumeOption mutates ConsumeOptions. Pass options to ConsumeMessages to
// override the per-queue defaults.
type ConsumeOption func(*ConsumeOptions)

// WithConcurrency sets the number of worker goroutines for the consumer. See
// ConsumeOptions.Concurrency for the safety requirements.
func WithConcurrency(n int) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.Concurrency = n
	}
}

// MessageBroker defines the interface for publishing and consuming AMQP messages.
type MessageBroker interface {
	// PublishMessage publishes the message to the given exchange with the specified routing key.
	PublishMessage(ctx context.Context, exchange, routingKey string, message contracts.AmqpMessage) error
	// ConsumeMessages consumes messages from the given queue and invokes the handler for each delivery.
	ConsumeMessages(ctx context.Context, queueName string, handler MessageHandler, opts ...ConsumeOption) error
	// IsReady reports whether the broker connection and channel are ready for use.
	IsReady() bool
	// Close shuts down the AMQP channel and connection. Safe to call multiple times.
	Close()
}
