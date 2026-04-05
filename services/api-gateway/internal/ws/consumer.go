package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

// zeroTime is used to clear write deadlines on WebSocket connections.
var zeroTime time.Time

// StartEventConsumer starts a RabbitMQ consumer that reads agent run step
// events and fans them out to WebSocket clients via the hub.
func StartEventConsumer(ctx context.Context, broker messaging.MessageBroker, hub *Hub) error {
	return broker.ConsumeMessages(ctx, messaging.AgentEventRunStepQueue, func(ctx context.Context, d amqp.Delivery) error {
		// Unwrap the AmqpMessage envelope.
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(d.Body, &envelope); err != nil {
			slog.Error("WS consumer: failed to unmarshal envelope", "error", err)
			return nil // don't retry malformed messages
		}

		// Parse the step data.
		var stepData messaging.AgentRunStepData
		if err := json.Unmarshal(envelope.Data, &stepData); err != nil {
			slog.Error("WS consumer: failed to unmarshal step data", "error", err)
			return nil
		}

		// Build WebSocket message.
		wsMsg := contracts.WSMessage{
			Type: contracts.WSTypeRunEvent,
			Data: stepData,
		}
		msgBytes, err := json.Marshal(wsMsg)
		if err != nil {
			slog.Error("WS consumer: failed to marshal WS message", "error", err)
			return nil
		}

		hub.Publish(stepData.AgentRunID, stepData.AccountID, msgBytes)
		return nil
	})
}

// StartRunCompletedConsumer starts a RabbitMQ consumer that reads agent run
// completed events and fans them out to WebSocket clients via the hub.
func StartRunCompletedConsumer(ctx context.Context, broker messaging.MessageBroker, hub *Hub) error {
	return broker.ConsumeMessages(ctx, messaging.AgentEventRunCompletedQueue, func(ctx context.Context, d amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(d.Body, &envelope); err != nil {
			slog.Error("WS run-completed consumer: failed to unmarshal envelope", "error", err)
			return nil
		}

		var data messaging.AgentRunCompletedData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			slog.Error("WS run-completed consumer: failed to unmarshal completed data", "error", err)
			return nil
		}

		wsMsg := contracts.WSMessage{
			Type: contracts.WSTypeRunComplete,
			Data: data,
		}
		msgBytes, err := json.Marshal(wsMsg)
		if err != nil {
			slog.Error("WS run-completed consumer: failed to marshal WS message", "error", err)
			return nil
		}

		hub.Publish(data.AgentRunID, data.AccountID, msgBytes)
		return nil
	})
}
