package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/hubspotsync"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

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
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service").WithLeaseSeconds(jobInboxLeaseSeconds),
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
		return c.inboxConsumer.Discard(ctx, "missing job or account id")
	}
	span.SetAttributes(
		attribute.String("hubspot_sync.job_id", data.JobID),
		attribute.String("hubspot_sync.account_id", data.AccountID),
	)

	switch msg.RoutingKey {
	case string(contracts.CoreCmdHubspotSyncPreview):
		if apiErr := c.hubspotSync.RunPreview(ctx, data.AccountID, data.JobID); apiErr != nil {
			return c.handleRunError(span, "preview", data.JobID, apiErr)
		}
	case string(contracts.CoreCmdHubspotSyncExecute):
		if apiErr := c.hubspotSync.RunExecute(ctx, data.AccountID, data.JobID); apiErr != nil {
			return c.handleRunError(span, "execute", data.JobID, apiErr)
		}
	default:
		log.Printf("[hubspot_sync] unknown routing key %q; dropping", msg.RoutingKey)
		return c.inboxConsumer.Discard(ctx, "unknown routing key "+msg.RoutingKey)
	}
	return nil
}

// handleRunError decides whether a failed run is worth redelivering. Transient failures (HubSpot rate limits, 5xx, DB blips) are returned so the inbox retries with backoff — swallowing them would let a momentary hiccup permanently fail an entire backfill. Permanent failures are acked: the engine has already recorded them on the job, and redelivering could only reproduce them. The broker rejects to a dead-letter queue once retries are exhausted, so returning here cannot poison-loop the queue.
func (c *HubspotSyncConsumer) handleRunError(span trace.Span, phase, jobID string, apiErr *apierror.APIError) error {
	span.RecordError(apiErr)
	if apiErr.IsTransient {
		log.Printf("[hubspot_sync] %s hit a transient failure for job %s; retrying: %v", phase, jobID, apiErr)
		return apiErr
	}
	log.Printf("[hubspot_sync] %s permanently failed for job %s: %v", phase, jobID, apiErr)
	return nil
}
