package event

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/augno/api/services/logging-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	loggingpb "github.com/augno/api/shared/proto/logging"
	"github.com/augno/api/shared/tracing"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
)

type RequestLogConsumer struct {
	rabbitmq     *messaging.RabbitMQ
	loggingSvc   domain.LoggingSvc
	tracer       trace.Tracer
	messageCodec protojson.UnmarshalOptions
}

func NewRequestLogConsumer(rabbitmq *messaging.RabbitMQ, loggingSvc domain.LoggingSvc) *RequestLogConsumer {
	return &RequestLogConsumer{
		rabbitmq:     rabbitmq,
		loggingSvc:   loggingSvc,
		tracer:       tracing.GetTracer("logging-service.request_log_consumer"),
		messageCodec: protojson.UnmarshalOptions{DiscardUnknown: true},
	}
}

func (c *RequestLogConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.LoggingEventRequestLogQueue, c.handleMessage)
}

func (c *RequestLogConsumer) handleMessage(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "event.request_log.consume")
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
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
		IdempotencyKeyID:     event.IdempotencyKeyId,
		ActorID:              event.ActorId,
		ActorType:            event.ActorType,
		InternalErrorMessage: event.InternalErrorMessage,
		StackTrace:           event.StackTrace,
		IdentityType:         event.IdentityType,
	}
}
