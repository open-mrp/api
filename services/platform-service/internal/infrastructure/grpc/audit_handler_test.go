package grpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/platform-service/internal/domain"
	servicemock "github.com/augno/api/services/platform-service/internal/domain/mock/service"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	pb "github.com/augno/api/shared/proto/platform"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func auditHandlerIdentityCtx(targetAccountID string) context.Context {
	acctID := targetAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &acctID,
		},
	})
}

// ---------------------------------------------------------------------------
// suite
// ---------------------------------------------------------------------------

type AuditHandlerTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	auditSvc *servicemock.MockAuditEventSvc
	handler  *auditHandler
}

func (s *AuditHandlerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.auditSvc = servicemock.NewMockAuditEventSvc(s.ctrl)
	s.handler = &auditHandler{auditSvc: s.auditSvc}
}

func (s *AuditHandlerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ---------------------------------------------------------------------------
// CreateAuditEvent
// ---------------------------------------------------------------------------

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_NilRequest() {
	resp, err := s.handler.CreateAuditEvent(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_NilAuditEvent() {
	resp, err := s.handler.CreateAuditEvent(context.Background(), &pb.CreateAuditEventRequest{})
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_NilActor() {
	ctx := auditHandlerIdentityCtx("acct_1")
	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id: "aevt_1",
			// No Actor
		},
	})
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_NoIdentity() {
	resp, err := s.handler.CreateAuditEvent(context.Background(), &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:    "aevt_1",
			Actor: &pb.AuditActor{Id: "usr_1", Type: "internal", IdentityType: "user"},
		},
	})
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_NoTargetAccount() {
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
		},
	})

	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:    "aevt_1",
			Actor: &pb.AuditActor{Id: "usr_1", Type: "internal", IdentityType: "user"},
		},
	})
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_Success() {
	ctx := auditHandlerIdentityCtx("acct_1")
	now := timestamppb.Now()
	requestID := "req_1"
	serviceName := "core-service"

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.Equal("aevt_1", event.ID)
			s.Equal("usr_1", event.ActorID)
			s.Equal("internal", event.ActorType)
			s.Equal("user", event.IdentityType)
			s.Equal("acct_1", event.AccountID) // from identity context
			s.Equal(constants.AuditActionCreate, event.Action)
			s.Equal(constants.ObjectType("unit"), event.ResourceType)
			s.Equal("unit_1", event.ResourceID)
			s.Equal(&requestID, event.RequestID)
			s.Equal("core-service", event.ServiceName)
			return nil
		}).Times(1)

	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:           "aevt_1",
			Action:       "create",
			ResourceType: "unit",
			ResourceId:   "unit_1",
			Actor: &pb.AuditActor{
				Id:           "usr_1",
				Type:         "internal",
				IdentityType: "user",
			},
			ServiceName: &serviceName,
			RequestId:   &requestID,
			OccurredAt:  now,
		},
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_WithChanges() {
	ctx := auditHandlerIdentityCtx("acct_1")

	oldVal := `"old_name"`
	newVal := `"new_name"`

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.Len(event.Changes, 1)
			s.Equal("name", event.Changes[0].Field)
			s.Equal(json.RawMessage(`"old_name"`), event.Changes[0].OldValue)
			s.Equal(json.RawMessage(`"new_name"`), event.Changes[0].NewValue)
			return nil
		}).Times(1)

	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:           "aevt_1",
			Action:       "update",
			ResourceType: "unit",
			ResourceId:   "unit_1",
			Actor:        &pb.AuditActor{Id: "usr_1", Type: "internal", IdentityType: "user"},
			Changes: []*pb.AuditFieldChange{
				{Field: "name", OldValueJson: &oldVal, NewValueJson: &newVal},
			},
		},
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_WithMetadata() {
	ctx := auditHandlerIdentityCtx("acct_1")
	metadata := `{"reason":"bulk update"}`

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			s.Equal(json.RawMessage(`{"reason":"bulk update"}`), event.Metadata)
			return nil
		}).Times(1)

	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:           "aevt_1",
			Action:       "update",
			ResourceType: "unit",
			ResourceId:   "unit_1",
			Actor:        &pb.AuditActor{Id: "usr_1", Type: "internal", IdentityType: "user"},
			MetadataJson: &metadata,
		},
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_DefaultsOccurredAt() {
	ctx := auditHandlerIdentityCtx("acct_1")
	before := time.Now().UTC()

	s.auditSvc.EXPECT().
		SaveAuditEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *domain.AuditEvent) *apierror.APIError {
			after := time.Now().UTC()
			s.True(event.OccurredAt.After(before) || event.OccurredAt.Equal(before))
			s.True(event.OccurredAt.Before(after) || event.OccurredAt.Equal(after))
			return nil
		}).Times(1)

	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:           "aevt_1",
			Action:       "create",
			ResourceType: "unit",
			ResourceId:   "unit_1",
			Actor:        &pb.AuditActor{Id: "usr_1", Type: "internal", IdentityType: "user"},
			// OccurredAt is nil — should default to now
		},
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *AuditHandlerTestSuite) TestCreateAuditEvent_ServiceError() {
	ctx := auditHandlerIdentityCtx("acct_1")
	s.auditSvc.EXPECT().SaveAuditEvent(gomock.Any(), gomock.Any()).Return(apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.CreateAuditEvent(ctx, &pb.CreateAuditEventRequest{
		AuditEvent: &pb.AuditEventInfo{
			Id:           "aevt_1",
			Action:       "create",
			ResourceType: "unit",
			ResourceId:   "unit_1",
			Actor:        &pb.AuditActor{Id: "usr_1", Type: "internal", IdentityType: "user"},
		},
	})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// ListAuditEvents
// ---------------------------------------------------------------------------

func (s *AuditHandlerTestSuite) TestListAuditEvents_NilRequest() {
	resp, err := s.handler.ListAuditEvents(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestListAuditEvents_Success() {
	nextCursor := "cursor_next"
	s.auditSvc.EXPECT().
		ListAuditEvents(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.ListAuditEventsResult{
			AuditEvents: []*domain.AuditEventRead{
				{AuditEvent: domain.AuditEvent{ID: "aevt_1", Action: constants.AuditActionCreate, OccurredAt: time.Now().UTC()}},
			},
			PageInfo: pagination.PageInfo{HasNextPage: true, NextCursor: &nextCursor},
		}, nil).Times(1)

	resp, err := s.handler.ListAuditEvents(context.Background(), &pb.ListAuditEventsRequest{Limit: 10})
	s.NoError(err)
	s.Len(resp.AuditEvents, 1)
	s.True(resp.PageInfo.HasNextPage)
}

func (s *AuditHandlerTestSuite) TestListAuditEvents_WithDateFilters() {
	s.auditSvc.EXPECT().
		ListAuditEvents(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, filter *domain.ListAuditEventsFilter, _ []string) (*domain.ListAuditEventsResult, *apierror.APIError) {
			s.NotNil(filter.StartDate)
			s.NotNil(filter.EndDate)
			return &domain.ListAuditEventsResult{}, nil
		}).Times(1)

	_, err := s.handler.ListAuditEvents(context.Background(), &pb.ListAuditEventsRequest{
		StartDate: timestamppb.Now(),
		EndDate:   timestamppb.Now(),
		Limit:     10,
	})
	s.NoError(err)
}

func (s *AuditHandlerTestSuite) TestListAuditEvents_ServiceError() {
	s.auditSvc.EXPECT().ListAuditEvents(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.ListAuditEvents(context.Background(), &pb.ListAuditEventsRequest{Limit: 10})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// GetAuditEvent
// ---------------------------------------------------------------------------

func (s *AuditHandlerTestSuite) TestGetAuditEvent_NilRequest() {
	resp, err := s.handler.GetAuditEvent(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *AuditHandlerTestSuite) TestGetAuditEvent_Success() {
	s.auditSvc.EXPECT().
		GetAuditEvent(gomock.Any(), "aevt_1", []string{"actor"}).
		Return(&domain.AuditEventRead{
			AuditEvent: domain.AuditEvent{
				ID:           "aevt_1",
				Action:       constants.AuditActionCreate,
				ResourceType: "unit",
				ResourceID:   "unit_1",
				OccurredAt:   time.Now().UTC(),
			},
		}, nil).Times(1)

	resp, err := s.handler.GetAuditEvent(context.Background(), &pb.GetAuditEventRequest{
		Id:       "aevt_1",
		Includes: []string{"actor"},
	})
	s.NoError(err)
	s.Equal("aevt_1", resp.AuditEvent.Id)
	s.Equal("create", resp.AuditEvent.Action)
}

func (s *AuditHandlerTestSuite) TestGetAuditEvent_ServiceError() {
	s.auditSvc.EXPECT().GetAuditEvent(gomock.Any(), "aevt_1", gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("not found")).Times(1)

	resp, err := s.handler.GetAuditEvent(context.Background(), &pb.GetAuditEventRequest{Id: "aevt_1"})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// auditEventToProto mapping
// ---------------------------------------------------------------------------

func (s *AuditHandlerTestSuite) TestAuditEventToProto_NilEvent() {
	proto := auditEventToProto(nil)
	s.NotNil(proto) // should not panic
}

func (s *AuditHandlerTestSuite) TestAuditEventToProto_WithActor() {
	name := "Test User"
	handle := "test@example.com"
	ev := &domain.AuditEventRead{
		AuditEvent: domain.AuditEvent{
			ID:         "aevt_1",
			Action:     constants.AuditActionUpdate,
			OccurredAt: time.Now().UTC(),
		},
		Actor: &domain.AuditActor{
			ID:           "usr_1",
			ObjectType:   constants.ObjectTypeUser,
			Type:         "internal",
			IdentityType: "user",
			Name:         &name,
			Handle:       &handle,
		},
	}

	proto := auditEventToProto(ev)
	s.NotNil(proto.Actor)
	s.Equal("usr_1", proto.Actor.Id)
	s.Equal(string(constants.ObjectTypeUser), proto.Actor.ObjectType)
	s.Equal("internal", proto.Actor.Type)
	s.Equal(&name, proto.Actor.Name)
	s.Equal(&handle, proto.Actor.Handle)
}

func (s *AuditHandlerTestSuite) TestAuditEventToProto_WithChanges() {
	oldVal := json.RawMessage(`"old"`)
	newVal := json.RawMessage(`"new"`)
	ev := &domain.AuditEventRead{
		AuditEvent: domain.AuditEvent{
			ID:     "aevt_1",
			Action: constants.AuditActionUpdate,
			Changes: []domain.AuditFieldChange{
				{Field: "name", OldValue: oldVal, NewValue: newVal},
			},
			OccurredAt: time.Now().UTC(),
		},
	}

	proto := auditEventToProto(ev)
	s.Len(proto.Changes, 1)
	s.Equal("name", proto.Changes[0].Field)
	s.NotNil(proto.Changes[0].OldValueJson)
	s.Equal(`"old"`, *proto.Changes[0].OldValueJson)
	s.NotNil(proto.Changes[0].NewValueJson)
	s.Equal(`"new"`, *proto.Changes[0].NewValueJson)
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestAuditHandlerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AuditHandlerTestSuite))
}
