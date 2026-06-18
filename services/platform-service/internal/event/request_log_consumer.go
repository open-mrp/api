package event

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	loggingpb "github.com/augno/api/shared/proto/platform"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
)

type RequestLogConsumer struct {
	rabbitmq      messaging.MessageBroker
	loggingSvc    domain.LoggingSvc
	tracer        trace.Tracer
	messageCodec  protojson.UnmarshalOptions
	inboxConsumer *messaging.InboxConsumer
}

func NewRequestLogConsumer(rabbitmq messaging.MessageBroker, loggingSvc domain.LoggingSvc, inboxRepo messaging.InboxRepo, tracer trace.Tracer) *RequestLogConsumer {
	return &RequestLogConsumer{
		rabbitmq:      rabbitmq,
		loggingSvc:    loggingSvc,
		tracer:        tracer,
		messageCodec:  protojson.UnmarshalOptions{DiscardUnknown: true},
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "platform-service"),
	}
}

// persistenceConsumerConcurrency is the worker count for the request-log and audit-event consumers. Each message is an independent insert (no cross-message ordering) deduplicated by the inbox pattern, so concurrent processing is safe; the gateway produces these in bursts proportional to HTTP traffic, and a serial consumer falls behind.
const persistenceConsumerConcurrency = 8

func (c *RequestLogConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(
		ctx,
		messaging.LoggingEventRequestLogQueue,
		c.inboxConsumer.Wrap("platform.request_log", c.handleMessage),
		messaging.WithConcurrency(persistenceConsumerConcurrency),
	)
}

func (c *RequestLogConsumer) handleMessage(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "event.request_log.consume")
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}

	var event loggingpb.RequestLog
	if err := c.messageCodec.Unmarshal(amqpMsg.Data, &event); err != nil {
		log.Printf("Failed to unmarshal request log payload: %v", err)
		return err
	}

	if apiErr := c.loggingSvc.SaveRequestLog(ctx, mapEventToDomain(&event)); apiErr != nil {
		return apiErr
	}

	return nil
}

func mapEventToDomain(event *loggingpb.RequestLog) *domain.RequestLog {
	occurredAt := time.Now().UTC()
	if ts := event.GetOccurredAt(); ts != nil {
		occurredAt = ts.AsTime()
	}

	createdAt := time.Now().UTC()
	if ts := event.GetCreatedAt(); ts != nil {
		createdAt = ts.AsTime()
	}

	return &domain.RequestLog{
		ID:                   event.GetId(),
		Method:               event.GetMethod(),
		Host:                 event.GetHost(),
		Path:                 event.GetPath(),
		NormalizedRoute:      event.GetNormalizedRoute(),
		QueryJSON:            event.QueryJson,
		StatusCode:           event.GetStatusCode(),
		LatencyUs:            event.GetLatencyUs(),
		AccountID:            event.AccountId,
		TargetAccountID:      event.TargetAccountId,
		ClientIP:             event.ClientIp,
		ClientIPString:       event.ClientIpString,
		UserAgent:            event.UserAgent,
		Referrer:             event.Referrer,
		ErrorCode:            event.ErrorCode,
		ErrorMessage:         event.ErrorMessage,
		CreatedAt:            createdAt,
		OccurredAt:           occurredAt,
		IdempotencyKeyTypeID: event.IdempotencyKeyId,
		ActorID:              event.ActorId,
		ActorType:            event.ActorType,
		InternalErrorMessage: event.InternalErrorMessage,
		StackTrace:           event.StackTrace,
		IdentityType:         event.IdentityType,
		APIVersion:           event.ApiVersion,
		TraceID:              event.TraceId,
		PublicEndpoint:       event.GetPublicEndpoint(),
		BodyJSON:             event.BodyJson,
		ResponseJSON:         event.ResponseJson,
	}
}
