package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	servicemock "github.com/augno/api/services/platform-service/internal/domain/mock/service"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	pb "github.com/augno/api/shared/proto/platform"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------------------------------------------------------------------------
// suite
// ---------------------------------------------------------------------------

type LoggingHandlerTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	loggingSvc *servicemock.MockLoggingSvc
	handler    *loggingHandler
}

func (s *LoggingHandlerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.loggingSvc = servicemock.NewMockLoggingSvc(s.ctrl)
	s.handler = &loggingHandler{loggingSvc: s.loggingSvc}
}

func (s *LoggingHandlerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ---------------------------------------------------------------------------
// CreateRequestLog
// ---------------------------------------------------------------------------

func (s *LoggingHandlerTestSuite) TestCreateRequestLog_NilRequest() {
	resp, err := s.handler.CreateRequestLog(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *LoggingHandlerTestSuite) TestCreateRequestLog_NilRequestLog() {
	resp, err := s.handler.CreateRequestLog(context.Background(), &pb.CreateRequestLogRequest{})
	s.Nil(resp)
	s.Error(err)
}

func (s *LoggingHandlerTestSuite) TestCreateRequestLog_Success() {
	now := timestamppb.Now()
	accountID := "acct_1"
	actorID := "usr_1"
	actorType := "internal"

	s.loggingSvc.EXPECT().
		SaveRequestLog(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, rl *domain.RequestLog) *apierror.APIError {
			s.Equal("rlog_1", rl.ID)
			s.Equal("POST", rl.Method)
			s.Equal("api.example.com", rl.Host)
			s.Equal("/v1/things", rl.Path)
			s.Equal("/v1/things", rl.NormalizedRoute)
			s.Equal(int32(201), rl.StatusCode)
			s.Equal(int64(5000), rl.LatencyUs)
			s.Equal(&accountID, rl.AccountID)
			s.Equal(&actorID, rl.ActorID)
			s.Equal(&actorType, rl.ActorType)
			s.True(rl.PublicEndpoint)
			return nil
		}).Times(1)

	resp, err := s.handler.CreateRequestLog(context.Background(), &pb.CreateRequestLogRequest{
		RequestLog: &pb.RequestLog{
			Id:              "rlog_1",
			Method:          "POST",
			Host:            "api.example.com",
			Path:            "/v1/things",
			NormalizedRoute: "/v1/things",
			StatusCode:      201,
			LatencyUs:       5000,
			AccountId:       &accountID,
			ActorId:         &actorID,
			ActorType:       &actorType,
			PublicEndpoint:  true,
			OccurredAt:      now,
			CreatedAt:       now,
		},
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *LoggingHandlerTestSuite) TestCreateRequestLog_ServiceError() {
	s.loggingSvc.EXPECT().SaveRequestLog(gomock.Any(), gomock.Any()).Return(apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.CreateRequestLog(context.Background(), &pb.CreateRequestLogRequest{
		RequestLog: &pb.RequestLog{
			Id:         "rlog_1",
			Method:     "GET",
			OccurredAt: timestamppb.Now(),
		},
	})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// ListRequestLogs
// ---------------------------------------------------------------------------

func (s *LoggingHandlerTestSuite) TestListRequestLogs_NilRequest() {
	resp, err := s.handler.ListRequestLogs(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *LoggingHandlerTestSuite) TestListRequestLogs_Success() {
	nextCursor := "cursor_next"
	s.loggingSvc.EXPECT().
		ListRequestLogs(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.ListRequestLogsResult{
			RequestLogs: []*domain.RequestLogRead{
				{ID: "rlog_1", Method: "GET", OccurredAt: time.Now().UTC(), CreatedAt: time.Now().UTC()},
				{ID: "rlog_2", Method: "POST", OccurredAt: time.Now().UTC(), CreatedAt: time.Now().UTC()},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage: true,
				NextCursor:  &nextCursor,
			},
		}, nil).Times(1)

	resp, err := s.handler.ListRequestLogs(context.Background(), &pb.ListRequestLogsRequest{
		Limit: 10,
	})
	s.NoError(err)
	s.Len(resp.RequestLogs, 2)
	s.True(resp.PageInfo.HasNextPage)
	s.NotNil(resp.PageInfo.NextCursor)
	s.Equal("cursor_next", *resp.PageInfo.NextCursor)
}

func (s *LoggingHandlerTestSuite) TestListRequestLogs_WithAllFilters() {
	startDate := timestamppb.Now()
	endDate := timestamppb.Now()
	statusCode := int32(500)
	exactMatch := true

	s.loggingSvc.EXPECT().
		ListRequestLogs(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, filter *domain.ListRequestLogsFilter, _ []string) (*domain.ListRequestLogsResult, *apierror.APIError) {
			s.NotNil(filter.StartDate)
			s.NotNil(filter.EndDate)
			s.NotNil(filter.StatusCode)
			s.Equal(int32(500), *filter.StatusCode)
			s.True(filter.ExactMatch)
			return &domain.ListRequestLogsResult{}, nil
		}).Times(1)

	query := "test"
	method := "POST"
	_, err := s.handler.ListRequestLogs(context.Background(), &pb.ListRequestLogsRequest{
		Query:      &query,
		Method:     &method,
		StatusCode: &statusCode,
		StartDate:  startDate,
		EndDate:    endDate,
		ExactMatch: &exactMatch,
		Limit:      25,
	})
	s.NoError(err)
}

func (s *LoggingHandlerTestSuite) TestListRequestLogs_ServiceError() {
	s.loggingSvc.EXPECT().ListRequestLogs(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.ListRequestLogs(context.Background(), &pb.ListRequestLogsRequest{Limit: 10})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// GetRequestLog
// ---------------------------------------------------------------------------

func (s *LoggingHandlerTestSuite) TestGetRequestLog_NilRequest() {
	resp, err := s.handler.GetRequestLog(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *LoggingHandlerTestSuite) TestGetRequestLog_Success() {
	s.loggingSvc.EXPECT().
		GetRequestLog(gomock.Any(), "rlog_1", []string{"actor"}).
		Return(&domain.RequestLogRead{
			ID:         "rlog_1",
			Method:     "GET",
			Host:       "api.example.com",
			Path:       "/v1/things",
			StatusCode: 200,
			OccurredAt: time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
		}, nil).Times(1)

	resp, err := s.handler.GetRequestLog(context.Background(), &pb.GetRequestLogRequest{
		Id:       "rlog_1",
		Includes: []string{"actor"},
	})
	s.NoError(err)
	s.Equal("rlog_1", resp.RequestLog.Id)
	s.Equal("GET", resp.RequestLog.Method)
}

func (s *LoggingHandlerTestSuite) TestGetRequestLog_ServiceError() {
	s.loggingSvc.EXPECT().GetRequestLog(gomock.Any(), "rlog_1", gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("not found")).Times(1)

	resp, err := s.handler.GetRequestLog(context.Background(), &pb.GetRequestLogRequest{Id: "rlog_1"})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// requestLogToProto mapping
// ---------------------------------------------------------------------------

func (s *LoggingHandlerTestSuite) TestRequestLogToProto_SanitizesInvalidUTF8() {
	invalidUTF8 := "hello\xff\xfeworld"
	rl := &domain.RequestLogRead{
		ID:         "rlog_1",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       invalidUTF8,
		StatusCode: 200,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
		BodyJSON:   &invalidUTF8,
	}

	proto := requestLogToProto(rl)
	s.NotContains(proto.Path, "\xff")
	s.NotContains(*proto.BodyJson, "\xff")
}

func (s *LoggingHandlerTestSuite) TestRequestLogToProto_WithActor() {
	roleName := "Admin"
	roleID := "role_1"
	name := "Test User"
	email := "test@example.com"
	roleTypeCode := "admin"

	rl := &domain.RequestLogRead{
		ID:         "rlog_1",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/test",
		StatusCode: 200,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
		Actor: &domain.RequestLogActor{
			ID:           "usr_1",
			ActorType:    constants.ActorTypeUser,
			Name:         &name,
			Email:        &email,
			RoleID:       &roleID,
			RoleName:     &roleName,
			RoleTypeCode: &roleTypeCode,
		},
	}

	proto := requestLogToProto(rl)
	s.NotNil(proto.Actor)
	s.Equal("usr_1", proto.Actor.Id)
	s.Equal(string(constants.ActorTypeUser), proto.Actor.ActorType)
	s.Equal(&name, proto.Actor.Name)
	s.Equal(&email, proto.Actor.Email)
	s.Equal(&roleID, proto.Actor.RoleId)
	s.Equal(&roleName, proto.Actor.RoleName)
}

func (s *LoggingHandlerTestSuite) TestRequestLogToProto_WithoutActor() {
	rl := &domain.RequestLogRead{
		ID:         "rlog_1",
		Method:     "GET",
		Host:       "api.example.com",
		Path:       "/test",
		StatusCode: 200,
		OccurredAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	}

	proto := requestLogToProto(rl)
	s.Nil(proto.Actor)
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestLoggingHandlerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LoggingHandlerTestSuite))
}
