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

// MessageBroker defines the interface for publishing and consuming AMQP messages.
type MessageBroker interface {
	// PublishMessage publishes the message to the given exchange with the specified routing key.
	PublishMessage(ctx context.Context, exchange, routingKey string, message contracts.AmqpMessage) error
	// ConsumeMessages consumes messages from the given queue and invokes the handler for each delivery.
	ConsumeMessages(ctx context.Context, queueName string, handler MessageHandler) error
	// IsReady reports whether the broker connection and channel are ready for use.
	IsReady() bool
	// Close shuts down the AMQP channel and connection. Safe to call multiple times.
	Close()
}
