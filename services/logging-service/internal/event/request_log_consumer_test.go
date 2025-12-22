package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/augno/api/services/logging-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	loggingpb "github.com/augno/api/shared/proto/logging"

	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stubLoggingSvc struct {
	received *domain.RequestLog
	err      *contracts.APIError
}

func (s *stubLoggingSvc) SaveRequestLog(ctx context.Context, rl *domain.RequestLog) *contracts.APIError {
	s.received = rl
	return s.err
}

func TestRequestLogConsumer_Success(t *testing.T) {
	now := time.Now().UTC()
	event := &loggingpb.RequestLog{
		Id:                   "req_1",
		Method:               "GET",
		Host:                 "example.com",
		Path:                 "/healthz",
		NormalizedRoute:      "/healthz",
		StatusCode:           200,
		LatencyUs:            100,
		AccountId:            proto.String("acct_1"),
		ClientIp:             []byte{127, 0, 0, 1},
		ClientIpString:       proto.String("127.0.0.1"),
		UserAgent:            proto.String("tester"),
		Referrer:             proto.String("ref"),
		ErrorCode:            proto.String(""),
		ErrorMessage:         proto.String(""),
		OccurredAt:           timestamppb.New(now),
		IdempotencyKeyId:     proto.String("idem"),
		ActorId:              proto.String("actor"),
		ActorType:            proto.String("user"),
		InternalErrorMessage: proto.String(""),
		StackTrace:           proto.String(""),
		IdentityType:         proto.String("user"),
		CreatedAt:            timestamppb.New(now),
	}

	payload, err := protojson.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	amqpBody, err := json.Marshal(contracts.AmqpMessage{Data: payload})
	if err != nil {
		t.Fatalf("failed to marshal amqp message: %v", err)
	}

	svc := &stubLoggingSvc{}
	consumer := NewRequestLogConsumer(nil, svc)

	msg := amqp091.Delivery{Body: amqpBody}
	if err := consumer.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if svc.received == nil {
		t.Fatalf("expected request log to be saved")
	}
	if svc.received.ID != event.Id || svc.received.Method != event.Method {
		t.Fatalf("received request log does not match event")
	}
	if svc.received.OccurredAt.UTC() != now {
		t.Fatalf("expected occurred_at %v, got %v", now, svc.received.OccurredAt)
	}
}

func TestRequestLogConsumer_InvalidMessage(t *testing.T) {
	svc := &stubLoggingSvc{}
	consumer := NewRequestLogConsumer(nil, svc)

	msg := amqp091.Delivery{Body: []byte("not-json")}
	if err := consumer.handleMessage(context.Background(), msg); err == nil {
		t.Fatalf("expected error for invalid message")
	}
}

func TestRequestLogConsumer_InvalidPayload(t *testing.T) {
	svc := &stubLoggingSvc{}
	consumer := NewRequestLogConsumer(nil, svc)

	amqpBody, _ := json.Marshal(contracts.AmqpMessage{Data: []byte("not-json")})
	msg := amqp091.Delivery{Body: amqpBody}
	if err := consumer.handleMessage(context.Background(), msg); err == nil {
		t.Fatalf("expected error for invalid payload")
	}
}

func TestRequestLogConsumer_ServiceError(t *testing.T) {
	svc := &stubLoggingSvc{err: contracts.NewInternalError(errors.New("boom"), "fail")}
	consumer := NewRequestLogConsumer(nil, svc)

	payload, _ := protojson.Marshal(&loggingpb.RequestLog{
		Id:         "req_err",
		OccurredAt: timestamppb.Now(),
	})
	amqpBody, _ := json.Marshal(contracts.AmqpMessage{Data: payload})

	msg := amqp091.Delivery{Body: amqpBody}
	if err := consumer.handleMessage(context.Background(), msg); err == nil {
		t.Fatalf("expected error from service")
	}
}
