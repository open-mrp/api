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

// StartEventConsumer starts a RabbitMQ consumer that reads agent run step events and fans them out to WebSocket clients via the hub.
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

		hub.Publish(topicPrefixRun+stepData.AgentRunID, stepData.AccountID, msgBytes)

		// A terminal step (e.g. the "Run failed" marker) has no corresponding run-completed event — that event only fires on the success/awaiting paths. Without a terminal frame here, the frontend's live run view stays stuck loading on a failed run until a hard refresh. Emit a run_complete frame so the client leaves its loading state and re-fetches the run's authoritative (failed) status.
		if stepData.Terminal {
			completeMsg := contracts.WSMessage{
				Type: contracts.WSTypeRunComplete,
				Data: contracts.RunCompleteData{
					AgentRunID: stepData.AgentRunID,
					AccountID:  stepData.AccountID,
				},
			}
			if completeBytes, marshalErr := json.Marshal(completeMsg); marshalErr == nil {
				hub.Publish(topicPrefixRun+stepData.AgentRunID, stepData.AccountID, completeBytes)
			} else {
				slog.Error("WS consumer: failed to marshal terminal run_complete message", "error", marshalErr)
			}
		}
		return nil
	})
}

// StartNotificationConsumer starts a RabbitMQ consumer that reads in-app notification / messaging realtime-delivery events and fans them out to WebSocket clients via the hub, routing to the per-user topic (the bell) and/or a conversation topic (live chat).
func StartNotificationConsumer(ctx context.Context, broker messaging.MessageBroker, hub *Hub) error {
	return broker.ConsumeMessages(ctx, messaging.NotificationEventDeliveredQueue, func(ctx context.Context, d amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(d.Body, &envelope); err != nil {
			slog.Error("WS notification consumer: failed to unmarshal envelope", "error", err)
			return nil
		}

		var data messaging.RealtimeDeliveryData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			slog.Error("WS notification consumer: failed to unmarshal delivery data", "error", err)
			return nil
		}

		// Map the producer's event name to the client-facing WS message type.
		wsType := contracts.WSTypeNotification
		switch data.Event {
		case "message.created", "message.updated", "message.deleted":
			wsType = contracts.WSTypeMessage
		case "conversation.updated":
			wsType = contracts.WSTypeConversationUpdated
		case "unread.changed":
			wsType = contracts.WSTypeUnread
		case "typing":
			wsType = contracts.WSTypeTyping
		case "agent_run_started":
			wsType = contracts.WSTypeAgentRunStarted
		}

		wsMsg := contracts.WSMessage{Type: wsType, Data: data}
		msgBytes, err := json.Marshal(wsMsg)
		if err != nil {
			slog.Error("WS notification consumer: failed to marshal WS message", "error", err)
			return nil
		}

		// The per-user (bell) topic is keyed by user id — the gateway subscribes connections to user:<user_id> from the validated identity's actor id, which is the user id, not the account_user id the notification row is keyed by.
		if data.RecipientUserID != "" {
			hub.Publish(topicPrefixUser+data.RecipientUserID, data.AccountID, msgBytes)
		}
		// Account-wide broadcast announcements reach every connected user in the account.
		if data.AnnouncementID != "" {
			hub.Publish(topicPrefixAccount+data.AccountID, data.AccountID, msgBytes)
		}
		if data.ConversationID != "" {
			hub.Publish(topicPrefixConversation+data.ConversationID, data.AccountID, msgBytes)
		}

		// Cross-account hint: a per-user event also nudges the user's connections viewing OTHER accounts so the bell can show an unread dot for this account. The hint crosses tenant isolation (it is keyed by user id) and carries nothing beyond the account id.
		if data.RecipientUserID != "" {
			hint := contracts.WSMessage{
				Type: contracts.WSTypeAccountUnreadHint,
				Data: messaging.RealtimeDeliveryData{
					AccountID:       data.AccountID,
					RecipientUserID: data.RecipientUserID,
					Event:           "account.unread_hint",
				},
			}
			if hintBytes, marshalErr := json.Marshal(hint); marshalErr == nil {
				hub.PublishGlobal(topicPrefixUserGlobal+data.RecipientUserID, hintBytes)
			}
		}
		return nil
	})
}

// StartRunCompletedConsumer starts a RabbitMQ consumer that reads agent run completed events and fans them out to WebSocket clients via the hub.
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

		// Forward only a frontend-safe payload. The internal event carries token usage and model metadata for billing aggregation, which must never reach the browser.
		wsMsg := contracts.WSMessage{
			Type: contracts.WSTypeRunComplete,
			Data: contracts.RunCompleteData{
				AgentRunID: data.AgentRunID,
				AccountID:  data.AccountID,
			},
		}
		msgBytes, err := json.Marshal(wsMsg)
		if err != nil {
			slog.Error("WS run-completed consumer: failed to marshal WS message", "error", err)
			return nil
		}

		hub.Publish(topicPrefixRun+data.AgentRunID, data.AccountID, msgBytes)
		return nil
	})
}
