package messaging

import (
	"context"

	"github.com/open-mrp/api/shared/contracts"
	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler is the callback signature for processing a single AMQP delivery. Implementations should return nil on success (the message will be ACKed) or an error on failure (the message will be rejected to the dead-letter queue after retry exhaustion).
type MessageHandler func(context.Context, amqp.Delivery) error

// ConsumeOptions configures a single queue consumer.
type ConsumeOptions struct {
	// Concurrency is the number of worker goroutines processing deliveries for this queue. The default of 1 preserves strict in-order, one-at-a-time processing. Values above 1 are only safe for queues whose messages are independently processable — no cross-message ordering requirements — such as request-log and audit-event persistence, where each message is an independent row and the inbox pattern deduplicates redeliveries.
	Concurrency int
}

// ConsumeOption mutates ConsumeOptions. Pass options to ConsumeMessages to override the per-queue defaults.
type ConsumeOption func(*ConsumeOptions)

// WithConcurrency sets the number of worker goroutines for the consumer. See ConsumeOptions.Concurrency for the safety requirements.
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
	// ConsumeFanout declares a per-instance ephemeral (non-durable, exclusive, auto-delete) queue named "<baseName>.<instance-suffix>", binds it to the given routing keys on the application exchange, and consumes from it. Because every process that calls this gets its OWN queue, each receives a copy of every matching message — true fan-out. This is the correct primitive for realtime WebSocket delivery, where every api-gateway replica holds a distinct set of client sockets and must therefore see every event. Contrast with ConsumeMessages, whose callers share one durable queue and thus compete for deliveries (work-queue semantics). The per-instance queue dies with the process, so undelivered realtime events are simply dropped — acceptable because the persisted rows remain the source of truth.
	ConsumeFanout(ctx context.Context, baseName string, routingKeys []string, handler MessageHandler, opts ...ConsumeOption) error
	// IsReady reports whether the broker connection and channel are ready for use.
	IsReady() bool
	// Close shuts down the AMQP channel and connection. Safe to call multiple times.
	Close()
}
