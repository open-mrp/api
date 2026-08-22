package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	servicemock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeAuditAMQPMessage(t *testing.T, identity *types.Identity, payload audit.PublishedEvent, requestID string) amqp.Delivery {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(contracts.AmqpMessage{
		Identity:  identity,
		Data:      data,
		RequestID: requestID,
	})
	if err != nil {
		t.Fatal(err)
	}

	return amqp.Delivery{Body: body}
}

func validAuditIdentity() *types.Identity {
	acctID := "acct_1"
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
			AccountID:    &acctID,
		},
	}
}

func validAuditPayload() audit.PublishedEvent {
	return audit.PublishedEvent{
		TypeID:       "aevt_1",
		Action:       constants.AuditActionCreate,
		ResourceType: "unit",
		ResourceID:   "unit_1",
		ServiceName:  "core-service",
		OccurredAt:   time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// suite
// ---------------------------------------------------------------------------

type AuditEventConsumerTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	auditSvc *servicemock.MockAuditEventSvc
	consumer *AuditEventConsumer
}

func (s *AuditEventConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.auditSvc = servicemock.NewMockAuditEventSvc(s.ctrl)
	s.consumer = &AuditEventConsumer{
		auditSvc: s.auditSvc,
		tracer:   tracing.GetTracer("test"),
	}
}

func (s *AuditEventConsumerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ---------------------------------------------------------------------------
// handleMessage
// ---------------------------------------------------------------------------

func (s *AuditEventConsumerTestSuite) TestHandleMessage_Success() {
	identity := validAuditIdentity()
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "req_1")

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.Equal("aevt_1", event.ID)
			s.Equal("usr_1", event.ActorID)
			s.Equal(string(types.IdentityRelationTypeInternal), event.ActorType)
			s.Equal(string(types.IdentityActorTypeUser), event.IdentityType)
			s.Equal("acct_1", event.AccountID)
			s.NotNil(event.TargetAccountID)
			s.Equal("acct_1", *event.TargetAccountID)
			s.Equal(constants.AuditActionCreate, event.Action)
			s.Equal(constants.ObjectType("unit"), event.ResourceType)
			s.Equal("unit_1", event.ResourceID)
			s.Equal("core-service", event.ServiceName)
			s.NotNil(event.RequestID)
			s.Equal("req_1", *event.RequestID)
			return nil
		}).Times(1)

	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_InvalidJSON() {
	msg := amqp.Delivery{Body: []byte("not json")}

	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_InvalidPayloadData() {
	body, _ := json.Marshal(contracts.AmqpMessage{
		Identity: validAuditIdentity(),
		Data:     []byte("not valid audit json"),
	})
	msg := amqp.Delivery{Body: body}

	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_NoIdentity() {
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), nil, payload, "")

	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_NoTargetAccount() {
	identity := &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
		},
	}
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "")

	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_NoActor() {
	acctID := "acct_1"
	identity := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		// Actor is nil
	}
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "")

	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_MapsChanges() {
	identity := validAuditIdentity()
	payload := validAuditPayload()
	payload.Changes = []audit.FieldChange{
		{Field: "name", OldValue: json.RawMessage(`"old"`), NewValue: json.RawMessage(`"new"`)},
		{Field: "qty", OldValue: json.RawMessage(`1`), NewValue: json.RawMessage(`5`)},
	}
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "")

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.Len(event.Changes, 2)
			s.Equal("name", event.Changes[0].Field)
			s.Equal(json.RawMessage(`"old"`), event.Changes[0].OldValue)
			s.Equal(json.RawMessage(`"new"`), event.Changes[0].NewValue)
			s.Equal("qty", event.Changes[1].Field)
			return nil
		}).Times(1)

	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_MapsMetadata() {
	identity := validAuditIdentity()
	payload := validAuditPayload()
	payload.Metadata = map[string]any{"reason": "bulk update", "source": "api"}
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "")

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.NotNil(event.Metadata)
			var meta map[string]any
			s.NoError(json.Unmarshal(event.Metadata, &meta))
			s.Equal("bulk update", meta["reason"])
			s.Equal("api", meta["source"])
			return nil
		}).Times(1)

	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_MapsRequestID() {
	identity := validAuditIdentity()
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "req_abc123")

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.NotNil(event.RequestID)
			s.Equal("req_abc123", *event.RequestID)
			return nil
		}).Times(1)

	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_EmptyRequestID() {
	identity := validAuditIdentity()
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "")

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.Nil(event.RequestID)
			return nil
		}).Times(1)

	err := s.consumer.handleMessage(context.Background(), msg)
	s.NoError(err)
}

func (s *AuditEventConsumerTestSuite) TestHandleMessage_ServiceError() {
	identity := validAuditIdentity()
	payload := validAuditPayload()
	msg := makeAuditAMQPMessage(s.T(), identity, payload, "")

	s.auditSvc.EXPECT().SaveAuditEvent(gomock.Any(), gomock.Any()).
		Return(apierror.NewInternalError(nil, "save failed")).Times(1)

	err := s.consumer.handleMessage(context.Background(), msg)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestAuditEventConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AuditEventConsumerTestSuite))
}
