package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/core-service/internal/hubspotsync"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HubspotSyncConsumer runs the long-running HubSpot backfill passes (preview and execute) out-of-band. The queue is bound to both command routing keys; handleMessage dispatches on the routing key.
type HubspotSyncConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	hubspotSync   hubspotsync.Service
	tracer        trace.Tracer
}

func NewHubspotSyncConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, hubspotSync hubspotsync.Service) *HubspotSyncConsumer {
	return &HubspotSyncConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		hubspotSync:   hubspotSync,
		tracer:        tracing.GetTracer("core-service.hubspot_sync_consumer"),
	}
}

func (c *HubspotSyncConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdHubspotSyncQueue,
		c.inboxConsumer.Wrap("core.hubspot_sync", c.handleMessage))
}

func (c *HubspotSyncConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.hubspot_sync",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[hubspot_sync] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}
	var data messaging.HubspotSyncCommandData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[hubspot_sync] Failed to unmarshal command payload: %v", err)
		span.RecordError(err)
		return err
	}
	if data.JobID == "" || data.AccountID == "" {
		log.Printf("[hubspot_sync] Missing job or account id in command; dropping")
		return nil
	}
	span.SetAttributes(
		attribute.String("hubspot_sync.job_id", data.JobID),
		attribute.String("hubspot_sync.account_id", data.AccountID),
	)

	switch msg.RoutingKey {
	case string(contracts.CoreCmdHubspotSyncPreview):
		if apiErr := c.hubspotSync.RunPreview(ctx, data.AccountID, data.JobID); apiErr != nil {
			// The engine has already recorded the failure on the job. Swallow so the message is acked (no poison-loop); the user re-triggers a fresh backfill.
			log.Printf("[hubspot_sync] preview failed for job %s: %v", data.JobID, apiErr)
			span.RecordError(apiErr)
		}
	case string(contracts.CoreCmdHubspotSyncExecute):
		if apiErr := c.hubspotSync.RunExecute(ctx, data.AccountID, data.JobID); apiErr != nil {
			// The engine marks the job failed and checkpoints cursors. Swallow; the user re-triggers execute, which resumes from the checkpoint.
			log.Printf("[hubspot_sync] execute failed for job %s: %v", data.JobID, apiErr)
			span.RecordError(apiErr)
		}
	default:
		log.Printf("[hubspot_sync] unknown routing key %q; dropping", msg.RoutingKey)
	}
	return nil
}
