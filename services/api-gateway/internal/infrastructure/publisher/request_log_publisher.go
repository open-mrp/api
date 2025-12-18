package publisher

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	loggingpb "github.com/augno/api/shared/proto/logging"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var requestLogPublisherTracer = tracing.GetTracer("api-gateway.request_log_publisher")

type messagePublisher interface {
	PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error
}

type protoMarshaler interface {
	Marshal(proto.Message) ([]byte, error)
}

type requestLogPublisher struct {
	publisher messagePublisher
	marshaler protoMarshaler
}

func NewRequestLogPublisher(rabbitmq *messaging.RabbitMQ) domain.RequestLogPublisher {
	return &requestLogPublisher{
		publisher: rabbitmq,
		marshaler: protojson.MarshalOptions{},
	}
}

func NewRequestLogPublisherWithDeps(pub messagePublisher, marshaler protoMarshaler) domain.RequestLogPublisher {
	if marshaler == nil {
		marshaler = protojson.MarshalOptions{}
	}
	return &requestLogPublisher{
		publisher: pub,
		marshaler: marshaler,
	}
}

func (p *requestLogPublisher) Create(ctx context.Context, rl *domain.RequestLog) *contracts.APIError {
	ctx, span := requestLogPublisherTracer.Start(ctx, "publisher.request_log.publish")
	defer span.End()

	event := &loggingpb.RequestLogEvent{
		Id:                   rl.ID,
		Method:               rl.Method,
		Host:                 rl.Host,
		Path:                 rl.Path,
		NormalizedRoute:      rl.NormalizedRoute,
		QueryJson:            rl.QueryJSON,
		StatusCode:           int32(rl.StatusCode), // #nosec G115
		LatencyUs:            rl.Latency,
		AccountId:            rl.AccountID,
		ClientIp:             rl.ClientIP,
		ClientIpString:       rl.ClientIPString,
		UserAgent:            rl.UserAgent,
		Referrer:             rl.Referrer,
		ErrorCode:            rl.ErrorCode,
		ErrorMessage:         rl.ErrorMessage,
		OccurredAt:           timestamppb.New(rl.OccurredAt),
		IdempotencyKeyId:     rl.IdempotencyKey,
		ActorId:              rl.ActorID,
		ActorType:            rl.ActorType,
		InternalErrorMessage: rl.InternalErrorMessage,
		StackTrace:           rl.StackTrace,
		IdentityType:         rl.IdentityType,
	}

	payload, err := p.marshaler.Marshal(event)
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to marshal request log event."))
	}

	msg := contracts.AmqpMessage{
		UserID: rl.AccountID,
		Data:   payload,
	}

	if err := p.publisher.PublishMessage(ctx, contracts.LoggingEventRequestLogged, msg); err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to publish request log event."))
	}

	return nil
}
