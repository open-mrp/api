package publisher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/messaging"
	pb "github.com/augno/api/shared/proto/platform"
	"google.golang.org/protobuf/encoding/protojson"
)

type capturingOutboxRepo struct {
	mu    sync.Mutex
	input *messaging.OutboxMessageInput
	done  chan struct{}
}

func newCapturingOutboxRepo() *capturingOutboxRepo {
	return &capturingOutboxRepo{done: make(chan struct{}, 1)}
}

func (r *capturingOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	r.mu.Lock()
	r.input = &input
	r.mu.Unlock()
	r.done <- struct{}{}
	return 1, nil
}

func (r *capturingOutboxRepo) waitForCreate(t *testing.T) *messaging.OutboxMessageInput {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for outbox Create call")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.input
}

func TestPublisher_MapsAccountID(t *testing.T) {
	repo := newCapturingOutboxRepo()
	pub := NewRequestLogOutboxPublisher(repo, "")

	accountID := "acct_home123"
	targetAccountID := "acct_target456"
	actorID := "usr_abc"
	actorType := "internal"
	identityType := "user"

	rl := &appctx.RequestLog{
		ID:              "rlog_test123",
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/v1/test",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       1234,
		AccountID:       &accountID,
		TargetAccountID: &targetAccountID,
		ActorID:         &actorID,
		ActorType:       &actorType,
		IdentityType:    &identityType,
		OccurredAt:      time.Now().UTC(),
		PublicEndpoint:  true,
	}

	err := pub.Create(context.Background(), rl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := repo.waitForCreate(t)

	var event pb.RequestLog
	if err := protojson.Unmarshal(input.Payload.Data, &event); err != nil {
		t.Fatalf("failed to unmarshal proto: %v", err)
	}

	if event.AccountId == nil || *event.AccountId != accountID {
		t.Errorf("expected AccountId %q, got %v", accountID, event.AccountId)
	}
	if event.TargetAccountId == nil || *event.TargetAccountId != targetAccountID {
		t.Errorf("expected TargetAccountId %q, got %v", targetAccountID, event.TargetAccountId)
	}
	if event.ActorId == nil || *event.ActorId != actorID {
		t.Errorf("expected ActorId %q, got %v", actorID, event.ActorId)
	}
	if event.ActorType == nil || *event.ActorType != actorType {
		t.Errorf("expected ActorType %q, got %v", actorType, event.ActorType)
	}
	if event.IdentityType == nil || *event.IdentityType != identityType {
		t.Errorf("expected IdentityType %q, got %v", identityType, event.IdentityType)
	}
}

func TestPublisher_NilAccountID(t *testing.T) {
	repo := newCapturingOutboxRepo()
	pub := NewRequestLogOutboxPublisher(repo, "")

	rl := &appctx.RequestLog{
		ID:              "rlog_test456",
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/v1/test",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       100,
		AccountID:       nil,
		OccurredAt:      time.Now().UTC(),
		PublicEndpoint:  true,
	}

	err := pub.Create(context.Background(), rl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := repo.waitForCreate(t)

	var event pb.RequestLog
	if err := protojson.Unmarshal(input.Payload.Data, &event); err != nil {
		t.Fatalf("failed to unmarshal proto: %v", err)
	}

	if event.AccountId != nil {
		t.Errorf("expected AccountId to be nil, got %v", event.AccountId)
	}
}

func TestPublisher_MapsAllFields(t *testing.T) {
	repo := newCapturingOutboxRepo()
	pub := NewRequestLogOutboxPublisher(repo, "")

	accountID := "acct_home"
	targetAccountID := "acct_target"
	actorID := "apke_key1"
	actorType := "customer"
	identityType := "api_key"
	queryJSON := `{"page":"1"}`
	bodyJSON := `{"name":"test"}`
	userAgent := "test-agent"
	referrer := "https://example.com"
	errorCode := "validation_error"
	errorMessage := "invalid input"
	apiVersion := "1.0.0"
	traceID := "trace123"
	idempotencyKeyID := "idem_key1"
	internalErrorMessage := "internal details"
	stackTrace := "goroutine 1 [running]:"
	clientIPString := "192.168.1.1"

	rl := &appctx.RequestLog{
		ID:                   "rlog_full",
		Method:               "POST",
		Host:                 "api.example.com",
		Path:                 "/v1/things",
		NormalizedRoute:      "/v1/things",
		QueryJSON:            &queryJSON,
		StatusCode:           422,
		LatencyUs:            5000,
		AccountID:            &accountID,
		TargetAccountID:      &targetAccountID,
		ActorID:              &actorID,
		ActorType:            &actorType,
		IdentityType:         &identityType,
		ClientIP:             []byte{192, 168, 1, 1},
		ClientIPString:       &clientIPString,
		UserAgent:            &userAgent,
		Referrer:             &referrer,
		ErrorCode:            &errorCode,
		ErrorMessage:         &errorMessage,
		OccurredAt:           time.Now().UTC(),
		IdempotencyKeyID:     &idempotencyKeyID,
		InternalErrorMessage: &internalErrorMessage,
		StackTrace:           &stackTrace,
		APIVersion:           &apiVersion,
		TraceID:              &traceID,
		PublicEndpoint:       false,
		BodyJSON:             &bodyJSON,
	}

	err := pub.Create(context.Background(), rl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := repo.waitForCreate(t)

	var event pb.RequestLog
	if err := protojson.Unmarshal(input.Payload.Data, &event); err != nil {
		t.Fatalf("failed to unmarshal proto: %v", err)
	}

	checks := []struct {
		name     string
		got      *string
		expected string
	}{
		{"AccountId", event.AccountId, accountID},
		{"TargetAccountId", event.TargetAccountId, targetAccountID},
		{"ActorId", event.ActorId, actorID},
		{"ActorType", event.ActorType, actorType},
		{"IdentityType", event.IdentityType, identityType},
		{"QueryJson", event.QueryJson, queryJSON},
		{"BodyJson", event.BodyJson, bodyJSON},
		{"UserAgent", event.UserAgent, userAgent},
		{"Referrer", event.Referrer, referrer},
		{"ErrorCode", event.ErrorCode, errorCode},
		{"ErrorMessage", event.ErrorMessage, errorMessage},
		{"ApiVersion", event.ApiVersion, apiVersion},
		{"TraceId", event.TraceId, traceID},
		{"IdempotencyKeyId", event.IdempotencyKeyId, idempotencyKeyID},
		{"InternalErrorMessage", event.InternalErrorMessage, internalErrorMessage},
		{"StackTrace", event.StackTrace, stackTrace},
		{"ClientIpString", event.ClientIpString, clientIPString},
	}

	for _, check := range checks {
		if check.got == nil {
			t.Errorf("%s: expected %q, got nil", check.name, check.expected)
		} else if *check.got != check.expected {
			t.Errorf("%s: expected %q, got %q", check.name, check.expected, *check.got)
		}
	}

	if event.GetId() != "rlog_full" {
		t.Errorf("expected ID 'rlog_full', got %q", event.GetId())
	}
	if event.GetMethod() != "POST" {
		t.Errorf("expected Method 'POST', got %q", event.GetMethod())
	}
	if event.GetStatusCode() != 422 {
		t.Errorf("expected StatusCode 422, got %d", event.GetStatusCode())
	}
	if event.GetPublicEndpoint() {
		t.Error("expected PublicEndpoint false, got true")
	}
}
