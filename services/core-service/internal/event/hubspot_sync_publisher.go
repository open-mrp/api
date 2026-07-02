package event

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var hubspotSyncPublisherTracer = tracing.GetTracer("core-service.hubspot_sync_publisher")

// outboxHubspotSyncPublisher writes HubSpot backfill commands to the outbox table so the command commits atomically with the job row in the same transaction.
type outboxHubspotSyncPublisher struct{}

// NewOutboxHubspotSyncPublisher creates a HubSpot sync command publisher that writes to the outbox table for reliable delivery.
func NewOutboxHubspotSyncPublisher() domain.HubspotSyncPublisher {
	return &outboxHubspotSyncPublisher{}
}

func (p *outboxHubspotSyncPublisher) PublishPreview(ctx context.Context, data messaging.HubspotSyncCommandData) *apierror.APIError {
	return p.publish(ctx, contracts.CoreCmdHubspotSyncPreview, data)
}

func (p *outboxHubspotSyncPublisher) PublishExecute(ctx context.Context, data messaging.HubspotSyncCommandData) *apierror.APIError {
	return p.publish(ctx, contracts.CoreCmdHubspotSyncExecute, data)
}

func (p *outboxHubspotSyncPublisher) publish(ctx context.Context, routingKey contracts.AmqpRoutingKey, data messaging.HubspotSyncCommandData) *apierror.APIError {
	ctx, span := hubspotSyncPublisherTracer.Start(ctx, "event.outbox_hubspot_sync_publisher.publish")
	defer span.End()

	repos, ok := GetReposFromContext(ctx)
	if !ok {
		return tracing.Trace(span, apierror.NewInternalError(nil, "RepoFactory not found in context for outbox publisher."))
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal HubSpot sync command data."))
	}

	msg := contracts.AmqpMessage{Data: dataJSON}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(routingKey),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(routingKey),
		Payload:     msg,
	}

	if _, err := repos.NewOutboxRepo().Create(ctx, outboxInput); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create outbox message for HubSpot sync command."))
	}
	return nil
}
