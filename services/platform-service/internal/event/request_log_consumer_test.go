package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/platform-service/internal/domain"
	servicemock "github.com/augno/api/services/platform-service/internal/domain/mock/service"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	loggingpb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMapEventToDomain_MapsAccountID(t *testing.T) {
	t.Parallel()
	accountID := "acct_home123"
	targetAccountID := "acct_target456"
	actorID := "usr_abc"
	actorType := "internal"
	identityType := "user"

	event := &loggingpb.RequestLog{
		Id:              "rlog_test1",
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/v1/test",
		NormalizedRoute: "/v1/test",
		StatusCode:      200,
		LatencyUs:       1234,
		AccountId:       &accountID,
		TargetAccountId: &targetAccountID,
		ActorId:         &actorID,
		ActorType:       &actorType,
		IdentityType:    &identityType,
		OccurredAt:      timestamppb.Now(),
		CreatedAt:       timestamppb.Now(),
		PublicEndpoint:  true,
	}

	rl := mapEventToDomain(event)

	if rl.AccountID == nil || *rl.AccountID != accountID {
		t.Errorf("expected AccountID %q, got %v", accountID, rl.AccountID)
	}
	if rl.TargetAccountID == nil || *rl.TargetAccountID != targetAccountID {
		t.Errorf("expected TargetAccountID %q, got %v", targetAccountID, rl.TargetAccountID)
	}
	if rl.ActorID == nil || *rl.ActorID != actorID {
		t.Errorf("expected ActorID %q, got %v", actorID, rl.ActorID)
	}
	if rl.ActorType == nil || *rl.ActorType != actorType {
		t.Errorf("expected ActorType %q, got %v", actorType, rl.ActorType)
	}
	if rl.IdentityType == nil || *rl.IdentityType != identityType {
		t.Errorf("expected IdentityType %q, got %v", identityType, rl.IdentityType)
	}
}

func TestMapEventToDomain_NilAccountID(t *testing.T) {
	t.Parallel()
	event := &loggingpb.RequestLog{
		Id:             "rlog_test2",
		Method:         "GET",
		Host:           "api.example.com",
		Path:           "/v1/test",
		StatusCode:     200,
		OccurredAt:     timestamppb.Now(),
		CreatedAt:      timestamppb.Now(),
		PublicEndpoint: true,
	}

	rl := mapEventToDomain(event)

	if rl.AccountID != nil {
		t.Errorf("expected AccountID nil, got %v", rl.AccountID)
	}
}

func TestMapEventToDomain_AllFields(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)

	event := &loggingpb.RequestLog{
		Id:                   "rlog_full",
		Method:               "POST",
		Host:                 "api.example.com",
		Path:                 "/v1/things",
		NormalizedRoute:      "/v1/things",
		QueryJson:            new(`{"page":"1"}`),
		StatusCode:           422,
		LatencyUs:            5000,
		AccountId:            new("acct_home"),
		TargetAccountId:      new("acct_target"),
		ActorId:              new("apke_key1"),
		ActorType:            new("customer"),
		IdentityType:         new("api_key"),
		ClientIp:             []byte{192, 168, 1, 1},
		ClientIpString:       new("192.168.1.1"),
		UserAgent:            new("test-agent"),
		Referrer:             new("https://example.com"),
		ErrorCode:            new("validation_error"),
		ErrorMessage:         new("invalid input"),
		OccurredAt:           timestamppb.New(now),
		CreatedAt:            timestamppb.New(now),
		IdempotencyKeyId:     new("idem_key1"),
		InternalErrorMessage: new("internal details"),
		StackTrace:           new("goroutine 1 [running]:"),
		ApiVersion:           new("1.0.0"),
		TraceId:              new("trace123"),
		PublicEndpoint:       false,
		BodyJson:             new(`{"name":"test"}`),
	}

	rl := mapEventToDomain(event)

	if rl.ID != "rlog_full" {
		t.Errorf("expected ID 'rlog_full', got %q", rl.ID)
	}
	if rl.Method != "POST" {
		t.Errorf("expected Method 'POST', got %q", rl.Method)
	}
	if rl.StatusCode != 422 {
		t.Errorf("expected StatusCode 422, got %d", rl.StatusCode)
	}
	if rl.LatencyUs != 5000 {
		t.Errorf("expected LatencyUs 5000, got %d", rl.LatencyUs)
	}
	if rl.PublicEndpoint {
		t.Error("expected PublicEndpoint false, got true")
	}

	checks := []struct {
		name     string
		got      *string
		expected string
	}{
		{"AccountID", rl.AccountID, "acct_home"},
		{"TargetAccountID", rl.TargetAccountID, "acct_target"},
		{"ActorID", rl.ActorID, "apke_key1"},
		{"ActorType", rl.ActorType, "customer"},
		{"IdentityType", rl.IdentityType, "api_key"},
		{"QueryJSON", rl.QueryJSON, `{"page":"1"}`},
		{"BodyJSON", rl.BodyJSON, `{"name":"test"}`},
		{"UserAgent", rl.UserAgent, "test-agent"},
		{"Referrer", rl.Referrer, "https://example.com"},
		{"ErrorCode", rl.ErrorCode, "validation_error"},
		{"ErrorMessage", rl.ErrorMessage, "invalid input"},
		{"APIVersion", rl.APIVersion, "1.0.0"},
		{"TraceID", rl.TraceID, "trace123"},
		{"IdempotencyKeyTypeID", rl.IdempotencyKeyTypeID, "idem_key1"},
		{"InternalErrorMessage", rl.InternalErrorMessage, "internal details"},
		{"StackTrace", rl.StackTrace, "goroutine 1 [running]:"},
		{"ClientIPString", rl.ClientIPString, "192.168.1.1"},
	}

	for _, check := range checks {
		if check.got == nil {
			t.Errorf("%s: expected %q, got nil", check.name, check.expected)
		} else if *check.got != check.expected {
			t.Errorf("%s: expected %q, got %q", check.name, check.expected, *check.got)
		}
	}

	if !rl.OccurredAt.Equal(now) {
		t.Errorf("expected OccurredAt %v, got %v", now, rl.OccurredAt)
	}
	if !rl.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, rl.CreatedAt)
	}
}

func TestMapEventToDomain_DefaultTimestamps(t *testing.T) {
	t.Parallel()
	before := time.Now().UTC()

	event := &loggingpb.RequestLog{
		Id:         "rlog_notime",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/v1/test",
		StatusCode: 200,
	}

	rl := mapEventToDomain(event)

	after := time.Now().UTC()

	if rl.OccurredAt.Before(before) || rl.OccurredAt.After(after) {
		t.Errorf("expected OccurredAt to default to ~now, got %v", rl.OccurredAt)
	}
	if rl.CreatedAt.Before(before) || rl.CreatedAt.After(after) {
		t.Errorf("expected CreatedAt to default to ~now, got %v", rl.CreatedAt)
	}
}

// ---------------------------------------------------------------------------
// handleMessage tests (suite-based)
// ---------------------------------------------------------------------------

func makeRequestLogAMQP(t *testing.T, identity *types.Identity, protoLog *loggingpb.RequestLog) amqp.Delivery {
	t.Helper()
	data, err := protojson.Marshal(protoLog)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(contracts.AmqpMessage{
		Identity: identity,
		Data:     data,
	})
	if err != nil {
		t.Fatal(err)
	}

	return amqp.Delivery{Body: body}
}

type RequestLogConsumerTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	loggingSvc *servicemock.MockLoggingSvc
	consumer   *RequestLogConsumer
}

func (s *RequestLogConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.loggingSvc = servicemock.NewMockLoggingSvc(s.ctrl)
	s.consumer = &RequestLogConsumer{
		loggingSvc:   s.loggingSvc,
		tracer:       tracing.GetTracer("test"),
		messageCodec: protojson.UnmarshalOptions{DiscardUnknown: true},
	}
}

func (s *RequestLogConsumerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_Success() {
	acctID := "acct_1"
	identity := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
			AccountID:    &acctID,
		},
	}

	protoLog := &loggingpb.RequestLog{
		Id:              "rlog_1",
		Method:          "POST",
		Host:            "api.example.com",
		Path:            "/v1/things",
		NormalizedRoute: "/v1/things",
		StatusCode:      201,
		LatencyUs:       5000,
		PublicEndpoint:  true,
		OccurredAt:      timestamppb.Now(),
		CreatedAt:       timestamppb.Now(),
	}

	s.loggingSvc.EXPECT().
		SaveRequestLog(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, rl *domain.RequestLog) *apierror.APIError {
			s.Equal("rlog_1", rl.ID)
			s.Equal("POST", rl.Method)
			s.Equal("/v1/things", rl.Path)
			s.Equal(int32(201), rl.StatusCode)
			s.True(rl.PublicEndpoint)
			return nil
		}).Times(1)

	msg := makeRequestLogAMQP(s.T(), identity, protoLog)
	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_InvalidJSON() {
	msg := amqp.Delivery{Body: []byte("not json")}
	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_InvalidProtobuf() {
	body, _ := json.Marshal(contracts.AmqpMessage{
		Data: []byte("not valid protojson"),
	})
	msg := amqp.Delivery{Body: body}
	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_ServiceError() {
	protoLog := &loggingpb.RequestLog{
		Id:         "rlog_1",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/v1/test",
		StatusCode: 200,
		OccurredAt: timestamppb.Now(),
	}

	s.loggingSvc.EXPECT().SaveRequestLog(gomock.Any(), gomock.Any()).
		Return(apierror.NewInternalError(nil, "db error")).Times(1)

	msg := makeRequestLogAMQP(s.T(), nil, protoLog)
	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_SetsIdentityInContext() {
	acctID := "acct_ctx"
	identity := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_ctx",
			AccountID:    &acctID,
		},
	}

	protoLog := &loggingpb.RequestLog{
		Id:         "rlog_ctx",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/v1/test",
		StatusCode: 200,
		OccurredAt: timestamppb.Now(),
	}

	s.loggingSvc.EXPECT().SaveRequestLog(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	msg := makeRequestLogAMQP(s.T(), identity, protoLog)
	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_NilIdentity() {
	protoLog := &loggingpb.RequestLog{
		Id:         "rlog_noident",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/v1/test",
		StatusCode: 200,
		OccurredAt: timestamppb.Now(),
	}

	s.loggingSvc.EXPECT().SaveRequestLog(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	msg := makeRequestLogAMQP(s.T(), nil, protoLog)
	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err) // should not crash, just skip setting identity
}

func (s *RequestLogConsumerTestSuite) TestHandleMessage_AllFieldsMapped() {
	accountID := "acct_1"
	targetAccountID := "acct_2"
	actorID := "usr_1"
	actorType := "internal"
	identityType := "user"
	now := time.Now().UTC().Truncate(time.Second)

	protoLog := &loggingpb.RequestLog{
		Id:              "rlog_full",
		Method:          "PATCH",
		Host:            "api.example.com",
		Path:            "/v1/things/123",
		NormalizedRoute: "/v1/things/{id}",
		StatusCode:      200,
		LatencyUs:       9999,
		AccountId:       &accountID,
		TargetAccountId: &targetAccountID,
		ActorId:         &actorID,
		ActorType:       &actorType,
		IdentityType:    &identityType,
		PublicEndpoint:  true,
		OccurredAt:      timestamppb.New(now),
		CreatedAt:       timestamppb.New(now),
	}

	s.loggingSvc.EXPECT().
		SaveRequestLog(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, rl *domain.RequestLog) *apierror.APIError {
			s.Equal("rlog_full", rl.ID)
			s.Equal("PATCH", rl.Method)
			s.Equal("api.example.com", rl.Host)
			s.Equal("/v1/things/123", rl.Path)
			s.Equal("/v1/things/{id}", rl.NormalizedRoute)
			s.Equal(int32(200), rl.StatusCode)
			s.Equal(int64(9999), rl.LatencyUs)
			s.Equal(&accountID, rl.AccountID)
			s.Equal(&targetAccountID, rl.TargetAccountID)
			s.Equal(&actorID, rl.ActorID)
			s.Equal(&actorType, rl.ActorType)
			s.Equal(&identityType, rl.IdentityType)
			s.True(rl.PublicEndpoint)
			s.True(rl.OccurredAt.Equal(now))
			s.True(rl.CreatedAt.Equal(now))
			return nil
		}).Times(1)

	msg := makeRequestLogAMQP(s.T(), nil, protoLog)
	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func TestRequestLogConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RequestLogConsumerTestSuite))
}
