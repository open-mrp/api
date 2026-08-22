package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func internalLoggingCtx(targetAccountID string) context.Context {
	acctID := targetAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &acctID,
			Permissions: map[string]bool{
				"request_logs:read": true,
			},
		},
	})
}

func customerIdentityCtx(targetAccountID string) context.Context {
	acctID := targetAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_cust123",
			AccountID:    &acctID,
			Permissions: map[string]bool{
				"request_logs:read": true,
			},
		},
	})
}

func noPermIdentityCtx(targetAccountID string) context.Context {
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

func noTargetAccountCtx() context.Context {
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

type LoggingServiceTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	requestLogRepo *repositorymock.MockRequestLogRepo
	svc            domain.LoggingSvc
}

func (s *LoggingServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.requestLogRepo = repositorymock.NewMockRequestLogRepo(s.ctrl)

	factory := factorymock.NewMockRepoFactory(s.ctrl)
	factory.EXPECT().NewRequestLogRepo().Return(s.requestLogRepo).Times(1)

	s.svc = NewLoggingSvc(&LoggingSvcConfig{Repos: factory})
}

func (s *LoggingServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ---------------------------------------------------------------------------
// SaveRequestLog
// ---------------------------------------------------------------------------

func (s *LoggingServiceTestSuite) TestSaveRequestLog_Success() {
	rl := &domain.RequestLog{ID: "rlog_1", Method: "GET", Path: "/v1/test"}
	s.requestLogRepo.EXPECT().Create(gomock.Any(), rl).Return(nil).Times(1)

	apiErr := s.svc.SaveRequestLog(context.Background(), rl)
	s.Nil(apiErr)
}

func (s *LoggingServiceTestSuite) TestSaveRequestLog_RepoError() {
	rl := &domain.RequestLog{ID: "rlog_2"}
	expected := apierror.NewInternalError(nil, "db failure")
	s.requestLogRepo.EXPECT().Create(gomock.Any(), rl).Return(expected).Times(1)

	apiErr := s.svc.SaveRequestLog(context.Background(), rl)
	s.NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
}

// ---------------------------------------------------------------------------
// GetRequestLog
// ---------------------------------------------------------------------------

func (s *LoggingServiceTestSuite) TestGetRequestLog_Success() {
	ctx := internalLoggingCtx("acct_123")
	expected := &domain.RequestLogRead{ID: "rlog_1", Method: "GET"}
	s.requestLogRepo.EXPECT().FindByID(gomock.Any(), "rlog_1", "acct_123", []string{"actor"}).Return(expected, nil).Times(1)

	result, apiErr := s.svc.GetRequestLog(ctx, "rlog_1", []string{"actor"})
	s.Nil(apiErr)
	s.Equal(expected.ID, result.ID)
}

func (s *LoggingServiceTestSuite) TestGetRequestLog_NoIdentity() {
	result, apiErr := s.svc.GetRequestLog(context.Background(), "rlog_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

func (s *LoggingServiceTestSuite) TestGetRequestLog_NotInternalActor() {
	ctx := customerIdentityCtx("acct_123")
	result, apiErr := s.svc.GetRequestLog(ctx, "rlog_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *LoggingServiceTestSuite) TestGetRequestLog_MissingPermission() {
	ctx := noPermIdentityCtx("acct_123")
	result, apiErr := s.svc.GetRequestLog(ctx, "rlog_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *LoggingServiceTestSuite) TestGetRequestLog_MissingTargetAccount() {
	ctx := noTargetAccountCtx()
	result, apiErr := s.svc.GetRequestLog(ctx, "rlog_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
}

func (s *LoggingServiceTestSuite) TestGetRequestLog_RepoError() {
	ctx := internalLoggingCtx("acct_123")
	expected := apierror.NewResourceNotFoundError("not found")
	s.requestLogRepo.EXPECT().FindByID(gomock.Any(), "rlog_1", "acct_123", gomock.Any()).Return(nil, expected).Times(1)

	result, apiErr := s.svc.GetRequestLog(ctx, "rlog_1", nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
}

// ---------------------------------------------------------------------------
// ListRequestLogs
// ---------------------------------------------------------------------------

func (s *LoggingServiceTestSuite) TestListRequestLogs_Success() {
	ctx := internalLoggingCtx("acct_123")
	expected := &domain.ListRequestLogsResult{
		RequestLogs: []*domain.RequestLogRead{{ID: "rlog_1"}},
		PageInfo:    pagination.PageInfo{HasNextPage: true},
	}
	s.requestLogRepo.EXPECT().List(gomock.Any(), "acct_123", gomock.Any(), gomock.Any()).Return(expected, nil).Times(1)

	filter := &domain.ListRequestLogsFilter{Limit: 10}
	result, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(apiErr)
	s.Len(result.RequestLogs, 1)
	s.True(result.PageInfo.HasNextPage)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_NilPublicEndpointPassedThrough() {
	ctx := internalLoggingCtx("acct_123")
	s.requestLogRepo.EXPECT().
		List(gomock.Any(), "acct_123", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, filter *domain.ListRequestLogsFilter, _ []string) (*domain.ListRequestLogsResult, *apierror.APIError) {
			s.Nil(filter.PublicEndpoint)
			return &domain.ListRequestLogsResult{}, nil
		}).Times(1)

	filter := &domain.ListRequestLogsFilter{Limit: 10}
	_, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(apiErr)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_PreservesExplicitPublicEndpoint() {
	ctx := internalLoggingCtx("acct_123")
	s.requestLogRepo.EXPECT().
		List(gomock.Any(), "acct_123", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, filter *domain.ListRequestLogsFilter, _ []string) (*domain.ListRequestLogsResult, *apierror.APIError) {
			s.NotNil(filter.PublicEndpoint)
			s.False(*filter.PublicEndpoint)
			return &domain.ListRequestLogsResult{}, nil
		}).Times(1)

	f := false
	filter := &domain.ListRequestLogsFilter{Limit: 10, PublicEndpoint: &f}
	_, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(apiErr)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_NoIdentity() {
	filter := &domain.ListRequestLogsFilter{Limit: 10}
	result, apiErr := s.svc.ListRequestLogs(context.Background(), filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_NotInternalActor() {
	ctx := customerIdentityCtx("acct_123")
	filter := &domain.ListRequestLogsFilter{Limit: 10}
	result, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_MissingPermission() {
	ctx := noPermIdentityCtx("acct_123")
	filter := &domain.ListRequestLogsFilter{Limit: 10}
	result, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_MissingTargetAccount() {
	ctx := noTargetAccountCtx()
	filter := &domain.ListRequestLogsFilter{Limit: 10}
	result, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
}

func (s *LoggingServiceTestSuite) TestListRequestLogs_RepoError() {
	ctx := internalLoggingCtx("acct_123")
	expected := apierror.NewInternalError(nil, "db error")
	s.requestLogRepo.EXPECT().List(gomock.Any(), "acct_123", gomock.Any(), gomock.Any()).Return(nil, expected).Times(1)

	filter := &domain.ListRequestLogsFilter{Limit: 10}
	result, apiErr := s.svc.ListRequestLogs(ctx, filter, nil)
	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(expected.Code, apiErr.Code)
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestLoggingServiceTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LoggingServiceTestSuite))
}
