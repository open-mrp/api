package publisher

import (
	"context"
	"log/slog"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var requestLogPublisherTracer = tracing.GetTracer("api-gateway.request_log_publisher")

type requestLogOutboxPublisher struct {
	outboxRepo messaging.OutboxRepo
}

func NewRequestLogOutboxPublisher(outboxRepo messaging.OutboxRepo) domain.RequestLogPublisher {
	return &requestLogOutboxPublisher{
		outboxRepo: outboxRepo,
	}
}

func (p *requestLogOutboxPublisher) Create(ctx context.Context, rl *appctx.RequestLog) error {
	ctx, span := requestLogPublisherTracer.Start(ctx, "publisher.request_log.create")
	defer span.End()

	pbLog := &pb.RequestLog{
		Id:                   rl.ID,
		Method:               rl.Method,
		Host:                 rl.Host,
		Path:                 rl.Path,
		NormalizedRoute:      rl.NormalizedRoute,
		QueryJson:            rl.QueryJSON,
		StatusCode:           int32(rl.StatusCode), // #nosec G115
		LatencyUs:            rl.LatencyUs,
		AccountId:            rl.AccountID,
		ClientIp:             rl.ClientIP,
		ClientIpString:       rl.ClientIPString,
		UserAgent:            rl.UserAgent,
		Referrer:             rl.Referrer,
		ErrorCode:            rl.ErrorCode,
		ErrorMessage:         rl.ErrorMessage,
		OccurredAt:           timestamppb.New(rl.OccurredAt),
		IdempotencyKeyId:     rl.IdempotencyKeyID,
		TargetAccountId:      rl.TargetAccountID,
		ActorId:              rl.ActorID,
		ActorType:            rl.ActorType,
		InternalErrorMessage: rl.InternalErrorMessage,
		StackTrace:           rl.StackTrace,
		IdentityType:         rl.IdentityType,
		CreatedAt:            timestamppb.Now(),
		ApiVersion:           rl.APIVersion,
		TraceId:              rl.TraceID,
	}

	_, marshalSpan := requestLogPublisherTracer.Start(ctx, "publisher.request_log.marshal")
	data, err := protojson.Marshal(pbLog)
	marshalSpan.End()
	if err != nil {
		slog.Error("Failed to marshal request log", "error", err, "request_id", rl.ID)
		return err
	}

	input := messaging.OutboxMessageInput{
		ServiceName: "api-gateway",
		MessageType: string(contracts.LoggingEventRequestLogged),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.LoggingEventRequestLogged),
		Payload: contracts.AmqpMessage{
			RequestID: rl.ID,
			Data:      data,
		},
	}

	// Save to outbox asynchronously - don't block the HTTP response
	// No tracing here since this runs after the request completes
	go func() {
		if _, err := p.outboxRepo.Create(context.Background(), input); err != nil {
			slog.Error("Failed to save request log to outbox", "error", err, "request_id", rl.ID)
		}
	}()

	return nil
}
