package service

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/platform-service/internal/domain"
	factorymock "github.com/augno/api/services/platform-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/platform-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func internalAuditCtx(targetAccountID string) context.Context {
	acctID := targetAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &acctID,
			Permissions: map[string]bool{
				"audit_events:read": true,
			},
		},
	})
}

func customerAuditCtx(targetAccountID string) context.Context {
	acctID := targetAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_cust123",
			AccountID:    &acctID,
			Permissions: map[string]bool{
				"audit_events:read": true,
			},
		},
	})
}

func noPermAuditCtx(targetAccountID string) context.Context {
	acctID := targetAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &acctID,
			Permissions:  map[string]bool{},
		},
	})
}

func noTargetAuditCtx() context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
		},
	})
}

// ---------------------------------------------------------------------------
// suite
// ---------------------------------------------------------------------------

type AuditEventServiceTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	auditEventRepo *repositorymock.MockAuditEventRepo
	svc            domain.AuditEventSvc
}

func (s *AuditEventServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.auditEventRepo = repositorymock.NewMockAuditEventRepo(s.ctrl)

	factory := factorymock.NewMockRepoFactory(s.ctrl)
	factory.EXPECT().NewAuditEventRepo().Return(s.auditEventRepo).Times(1)

	s.svc = NewAuditEventSvc(&AuditEventSvcConfig{Repos: factory})
}

func (s *AuditEventServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ---------------------------------------------------------------------------
// SaveAuditEvent
// ---------------------------------------------------------------------------

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_Success() {
	event := &domain.AuditEvent{
		ID:           "aevt_1",
		ActorID:      "usr_1",
		ActorType:    "internal",
		IdentityType: "user",
		AccountID:    "acct_1",
		Action:       constants.AuditActionCreate,
		ResourceType: "unit",
		ResourceID:   "unit_1",
		OccurredAt:   time.Now().UTC(),
	}
	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(nil).Times(1)

	apiErr := s.svc.SaveAuditEvent(context.Background(), event)
	s.Nil(apiErr)
}

func (s *AuditEventServiceTestSuite) TestSaveAuditEvent_RepoError() {
	event := &domain.AuditEvent{ID: "aevt_2"}
	expected := apierror.NewInternalError(nil, "db failure")
	s.auditEventRepo.EXPECT().Create(gomock.Any(), event).Return(expected).Times(1)

	apiErr := s.svc.SaveAuditEvent(context.Background(), event)
	s.NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
}

// ---------------------------------------------------------------------------
// GetAuditEvent
// ---------------------------------------------------------------------------

func (s *AuditEventServiceTestSuite) TestGetAuditEvent_Success() {
	ctx := internalAuditCtx("acct_123")
	expected := &domain.AuditEventRead{
		AuditEvent: domain.AuditEvent{ID: "aevt_1"},
	}
	s.auditEventRepo.EXPECT().FindByID(gomock.Any(), "aevt_1", "acct_123", []string{"actor"}).Return(expected, nil).Times(1)

	result, apiErr := s.svc.GetAuditEvent(ctx, "aevt_1", []string{"actor"})
	s.Nil(apiErr)
	s.Equal(expected.ID, result.ID)
}

func (s *AuditEventServiceTestSuite) TestGetAuditEvent_NoIdentity() {
	result, apiErr := s.svc.GetAuditEvent(context.Background(), "aevt_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

func (s *AuditEventServiceTestSuite) TestGetAuditEvent_NotInternalActor() {
	ctx := customerAuditCtx("acct_123")
	result, apiErr := s.svc.GetAuditEvent(ctx, "aevt_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *AuditEventServiceTestSuite) TestGetAuditEvent_MissingPermission() {
	ctx := noPermAuditCtx("acct_123")
	result, apiErr := s.svc.GetAuditEvent(ctx, "aevt_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *AuditEventServiceTestSuite) TestGetAuditEvent_MissingTargetAccount() {
	ctx := noTargetAuditCtx()
	result, apiErr := s.svc.GetAuditEvent(ctx, "aevt_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
}

func (s *AuditEventServiceTestSuite) TestGetAuditEvent_RepoError() {
	ctx := internalAuditCtx("acct_123")
	expected := apierror.NewResourceNotFoundError("not found")
	s.auditEventRepo.EXPECT().FindByID(gomock.Any(), "aevt_1", "acct_123", gomock.Any()).Return(nil, expected).Times(1)

	result, apiErr := s.svc.GetAuditEvent(ctx, "aevt_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
}

// ---------------------------------------------------------------------------
// ListAuditEvents
// ---------------------------------------------------------------------------

func (s *AuditEventServiceTestSuite) TestListAuditEvents_Success() {
	ctx := internalAuditCtx("acct_123")
	expected := &domain.ListAuditEventsResult{
		AuditEvents: []*domain.AuditEventRead{{AuditEvent: domain.AuditEvent{ID: "aevt_1"}}},
		PageInfo:    pagination.PageInfo{HasNextPage: true},
	}
	s.auditEventRepo.EXPECT().List(gomock.Any(), "acct_123", gomock.Any(), gomock.Any()).Return(expected, nil).Times(1)

	filter := &domain.ListAuditEventsFilter{Limit: 10}
	result, apiErr := s.svc.ListAuditEvents(ctx, filter, nil)
	s.Nil(apiErr)
	s.Len(result.AuditEvents, 1)
	s.True(result.PageInfo.HasNextPage)
}

func (s *AuditEventServiceTestSuite) TestListAuditEvents_NoIdentity() {
	filter := &domain.ListAuditEventsFilter{Limit: 10}
	result, apiErr := s.svc.ListAuditEvents(context.Background(), filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

func (s *AuditEventServiceTestSuite) TestListAuditEvents_NotInternalActor() {
	ctx := customerAuditCtx("acct_123")
	filter := &domain.ListAuditEventsFilter{Limit: 10}
	result, apiErr := s.svc.ListAuditEvents(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *AuditEventServiceTestSuite) TestListAuditEvents_MissingPermission() {
	ctx := noPermAuditCtx("acct_123")
	filter := &domain.ListAuditEventsFilter{Limit: 10}
	result, apiErr := s.svc.ListAuditEvents(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *AuditEventServiceTestSuite) TestListAuditEvents_MissingTargetAccount() {
	ctx := noTargetAuditCtx()
	filter := &domain.ListAuditEventsFilter{Limit: 10}
	result, apiErr := s.svc.ListAuditEvents(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
}

func (s *AuditEventServiceTestSuite) TestListAuditEvents_RepoError() {
	ctx := internalAuditCtx("acct_123")
	expected := apierror.NewInternalError(nil, "db error")
	s.auditEventRepo.EXPECT().List(gomock.Any(), "acct_123", gomock.Any(), gomock.Any()).Return(nil, expected).Times(1)

	filter := &domain.ListAuditEventsFilter{Limit: 10}
	result, apiErr := s.svc.ListAuditEvents(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestAuditEventServiceTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AuditEventServiceTestSuite))
}
