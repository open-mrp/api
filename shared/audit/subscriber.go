package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ObservedEvent is an audit event as seen by a subscriber reacting to it, e.g. to invalidate a cache.
type ObservedEvent struct {
	// AccountID is the account the mutation was made in.
	AccountID        string
	Action           constants.AuditAction
	ResourceType     constants.ObjectType
	ResourceID       string
	RootResourceType constants.ObjectType
	RootResourceID   string
	// ChangedFields names the fields an update touched; empty for creates and deletes.
	ChangedFields []string
}

// SubscribeConfig configures Subscribe.
type SubscribeConfig struct {
	// QueueBaseName (required) names this subscriber's per-replica queue, e.g. "auth_event_cache_invalidation".
	QueueBaseName string

	// OnEvent (required) is called once per audit event; it must be idempotent, since the stream is at-least-once.
	OnEvent func(context.Context, ObservedEvent)

	// OnResync (optional; default: no-op) runs whenever this replica's queue is (re)bound. Events published while it was unbound were lost to this replica, so state derived from the stream must be rebuilt.
	OnResync func()
}

// Subscribe delivers every audit event published by any service to this replica. Delivery is best-effort: the queue lives only as long as the broker connection, so a subscriber must tolerate gaps, which OnResync marks.
func Subscribe(ctx context.Context, broker messaging.MessageBroker, cfg SubscribeConfig) error {
	opts := []messaging.ConsumeOption{}
	if cfg.OnResync != nil {
		opts = append(opts, messaging.WithOnBound(cfg.OnResync))
	}
	return broker.ConsumeFanout(ctx, cfg.QueueBaseName, []string{string(contracts.PlatformEventAuditLogged)}, func(ctx context.Context, d amqp.Delivery) error {
		event, ok := decodeObservedEvent(d.Body)
		if !ok {
			slog.Warn("audit subscriber: dropping undecodable audit event", "queue", cfg.QueueBaseName, "message_id", d.MessageId)
			return nil
		}
		cfg.OnEvent(ctx, event)
		return nil
	}, opts...)
}

func decodeObservedEvent(body []byte) (ObservedEvent, bool) {
	var envelope contracts.AmqpMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ObservedEvent{}, false
	}
	var published PublishedEvent
	if err := json.Unmarshal(envelope.Data, &published); err != nil {
		return ObservedEvent{}, false
	}

	event := ObservedEvent{
		Action:           published.Action,
		ResourceType:     published.ResourceType,
		ResourceID:       published.ResourceID,
		RootResourceType: published.RootResourceType,
		RootResourceID:   published.RootResourceID,
	}
	if envelope.Identity.IsTargetAccountSet() {
		event.AccountID = envelope.Identity.Target.AccountID
	}
	for _, change := range published.Changes {
		event.ChangedFields = append(event.ChangedFields, change.Field)
	}
	return event, true
}
