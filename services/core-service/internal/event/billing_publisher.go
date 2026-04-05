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

var billingPublisherTracer = tracing.GetTracer("core-service.billing_publisher")

// outboxBillingPublisher writes billing commands to the outbox table instead of
// publishing directly to RabbitMQ.
type outboxBillingPublisher struct{}

// NewOutboxBillingPublisher creates a billing publisher that writes
// to the outbox table for reliable message delivery.
func NewOutboxBillingPublisher() domain.BillingPublisher {
	return &outboxBillingPublisher{}
}

func (p *outboxBillingPublisher) PublishReportSeatChange(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := billingPublisherTracer.Start(ctx, "event.outbox_billing_publisher.publish_report_seat_change")
	defer span.End()

	repos, ok := GetReposFromContext(ctx)
	if !ok {
		return tracing.Trace(span, apierror.NewInternalError(nil, "RepoFactory not found in context for outbox publisher."))
	}

	data := messaging.SeatChangeReportData{
		AccountID: accountID,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal seat change report data."))
	}

	msg := contracts.AmqpMessage{
		Data: dataJSON,
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.BillingCmdReportSeatChange),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.BillingCmdReportSeatChange),
		Payload:     msg,
	}

	outboxRepo := repos.NewOutboxRepo()
	_, err = outboxRepo.Create(ctx, outboxInput)

	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create outbox message."))
	}

	return nil
}

func (p *outboxBillingPublisher) PublishSyncSeats(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := billingPublisherTracer.Start(ctx, "event.outbox_billing_publisher.publish_sync_seats")
	defer span.End()

	repos, ok := GetReposFromContext(ctx)
	if !ok {
		return tracing.Trace(span, apierror.NewInternalError(nil, "RepoFactory not found in context for outbox publisher."))
	}

	data := messaging.SeatSyncData{
		AccountID: accountID,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal seat sync data."))
	}

	msg := contracts.AmqpMessage{
		Data: dataJSON,
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.BillingCmdSyncSeats),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.BillingCmdSyncSeats),
		Payload:     msg,
	}

	outboxRepo := repos.NewOutboxRepo()
	_, err = outboxRepo.Create(ctx, outboxInput)

	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create outbox message."))
	}

	return nil
}
