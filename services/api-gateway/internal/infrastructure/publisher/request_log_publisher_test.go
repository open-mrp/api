package publisher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/shared/contracts"
	loggingpb "github.com/augno/api/shared/proto/logging"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stubPublisher struct {
	routingKey string
	message    contracts.AmqpMessage
	err        error
}

func (s *stubPublisher) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	s.routingKey = routingKey
	s.message = message
	return s.err
}

type stubMarshaler struct {
	payload []byte
	err     error
}

func (m stubMarshaler) Marshal(proto.Message) ([]byte, error) {
	return m.payload, m.err
}

func TestRequestLogPublisher_PublishSuccess(t *testing.T) {
	stubPub := &stubPublisher{}
	now := time.Now().UTC()
	p := NewRequestLogPublisherWithDeps(stubPub, protojson.MarshalOptions{})

	strPtr := func(s string) *string { return &s }

	reqLog := &domain.RequestLog{
		ID:                   "req_123",
		Method:               "GET",
		Host:                 "example.com",
		Path:                 "/healthz",
		NormalizedRoute:      "/healthz",
		QueryJSON:            strPtr(`{"q":"1"}`),
		StatusCode:           200,
		LatencyUs:            1234,
		AccountID:            strPtr("acct_1"),
		ClientIP:             []byte{127, 0, 0, 1},
		ClientIPString:       strPtr("127.0.0.1"),
		UserAgent:            strPtr("tester"),
		Referrer:             strPtr("ref"),
		ErrorCode:            strPtr(""),
		ErrorMessage:         strPtr(""),
		OccurredAt:           now,
		IdempotencyKeyID:     strPtr("idem"),
		ActorID:              strPtr("actor"),
		ActorType:            strPtr("user"),
		InternalErrorMessage: strPtr(""),
		StackTrace:           strPtr(""),
		IdentityType:         strPtr("user"),
	}

	if err := p.Create(context.Background(), reqLog); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stubPub.routingKey != contracts.LoggingEventRequestLogged {
		t.Fatalf("unexpected routing key: %s", stubPub.routingKey)
	}

	var event loggingpb.RequestLog
	if err := protojson.Unmarshal(stubPub.message.Data, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.Id != reqLog.ID || event.Method != reqLog.Method || event.Path != reqLog.Path {
		t.Fatalf("event fields did not match request log")
	}

	if event.LatencyUs != reqLog.LatencyUs {
		t.Fatalf("expected latency %d, got %d", reqLog.LatencyUs, event.LatencyUs)
	}

	if ts := event.GetOccurredAt().AsTime().UTC(); !ts.Equal(now) {
		t.Fatalf("expected occurred_at %v, got %v", now, ts)
	}
}

func TestRequestLogPublisher_MarshalError(t *testing.T) {
	stubPub := &stubPublisher{}
	p := NewRequestLogPublisherWithDeps(stubPub, stubMarshaler{err: errors.New("marshal fail")})

	err := p.Create(context.Background(), &domain.RequestLog{OccurredAt: time.Now().UTC()})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	apiErr := err
	if apiErr.Code != contracts.ErrorCodeInternalError {
		t.Fatalf("expected internal error code, got %s", apiErr.Code)
	}
}

func TestRequestLogPublisher_PublishError(t *testing.T) {
	stubPub := &stubPublisher{err: errors.New("publish fail")}
	now := time.Now().UTC()
	marshaled, _ := protojson.Marshal(&loggingpb.RequestLog{
		OccurredAt: timestamppb.New(now),
	})
	p := NewRequestLogPublisherWithDeps(stubPub, stubMarshaler{payload: marshaled})

	err := p.Create(context.Background(), &domain.RequestLog{OccurredAt: now})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	apiErr := err
	if apiErr.Code != contracts.ErrorCodeInternalError {
		t.Fatalf("expected internal error code, got %s", apiErr.Code)
	}
}
