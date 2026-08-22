package grpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/platform-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/platform-service/pkg/idempotency"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/platform"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// suite
// ---------------------------------------------------------------------------

type IdempotencyHandlerTestSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	repo    *repositorymock.MockIdempotencyKeyRepo
	handler *gRPCHandler
}

func (s *IdempotencyHandlerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.repo = repositorymock.NewMockIdempotencyKeyRepo(s.ctrl)
	s.handler = &gRPCHandler{idempotencyRepo: s.repo}
}

func (s *IdempotencyHandlerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func validProcessRequest() *pb.ProcessIdempotencyKeyRequest {
	actorID := "usr_1"
	targetAccountID := "acct_1"
	return &pb.ProcessIdempotencyKeyRequest{
		IdempotencyKey:  "idem_abc",
		ActorId:         &actorID,
		TargetAccountId: &targetAccountID,
		IdentityType:    "user",
		RequestMethod:   "POST",
		NormalizedRoute: "/v1/things",
		ScopeHash:       "hash123",
		RequestBodyHash: "bodyhash",
		RequestParams:   []byte(`{"name":"test"}`),
	}
}

// ---------------------------------------------------------------------------
// ProcessIdempotencyKey
// ---------------------------------------------------------------------------

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_NilRequest() {
	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_NewKey() {
	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key *domain.IdempotencyKey) (*domain.UpsertAndLockResult, *apierror.APIError) {
			s.Equal("idem_abc", key.IdempotencyKey)
			s.Equal("POST", key.RequestMethod)
			s.Equal("/v1/things", key.NormalizedRoute)
			s.Equal("hash123", key.ScopeHash)
			s.Equal("bodyhash", key.RequestBodyHash)
			s.Equal(idempotency.RecoveryPointStarted.String(), key.RecoveryPoint)
			return &domain.UpsertAndLockResult{
				Key:     key,
				Created: true,
				Locked:  true,
			}, nil
		}).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.NoError(err)
	s.Equal(pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_NEW, resp.Result)
	s.NotEmpty(resp.IdempotencyKeyId)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_Replay() {
	responseCode := 200
	responseBody := json.RawMessage(`{"id":"thing_1"}`)
	responseHeaders := json.RawMessage(`{"content-type":"application/json"}`)

	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		Return(&domain.UpsertAndLockResult{
			Key: &domain.IdempotencyKey{
				ID:              "ikey_existing",
				ResponseCode:    &responseCode,
				ResponseBody:    responseBody,
				ResponseHeaders: responseHeaders,
			},
			Created: false,
			Locked:  false,
		}, nil).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.NoError(err)
	s.Equal(pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_REPLAY, resp.Result)
	s.Equal("ikey_existing", resp.IdempotencyKeyId)
	s.NotNil(resp.ResponseCode)
	s.Equal(int32(200), *resp.ResponseCode)
	s.Equal([]byte(responseBody), resp.ResponseBody)
	s.Equal([]byte(responseHeaders), resp.ResponseHeaders)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_InProgress_Locked() {
	lockExpires := time.Now().Add(5 * time.Minute)
	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		Return(&domain.UpsertAndLockResult{
			Key: &domain.IdempotencyKey{
				ID:            "ikey_locked",
				LockExpiresAt: &lockExpires,
			},
			Created: false,
			Locked:  false, // we didn't acquire the lock
		}, nil).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.NoError(err)
	s.Equal(pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_IN_PROGRESS, resp.Result)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_InProgress_ErrorCode() {
	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewIdempotencyInProgressError("idem_abc")).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.NoError(err)
	s.Equal(pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_IN_PROGRESS, resp.Result)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_HashMismatch() {
	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewIdempotencyHashMismatchError("idem_abc")).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.NoError(err)
	s.Equal(pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_HASH_MISMATCH, resp.Result)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_RepoError() {
	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(nil, "db error")).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.Nil(resp)
	s.Error(err)
}

func (s *IdempotencyHandlerTestSuite) TestProcessIdempotencyKey_UnlockedExisting() {
	// Existing key, no response, not locked — should return NEW (re-locked)
	s.repo.EXPECT().
		UpsertAndLock(gomock.Any(), gomock.Any()).
		Return(&domain.UpsertAndLockResult{
			Key: &domain.IdempotencyKey{
				ID: "ikey_unlocked",
				// no ResponseCode, no lock
			},
			Created: false,
			Locked:  true,
		}, nil).Times(1)

	resp, err := s.handler.ProcessIdempotencyKey(context.Background(), validProcessRequest())
	s.NoError(err)
	s.Equal(pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_NEW, resp.Result)
	s.Equal("ikey_unlocked", resp.IdempotencyKeyId)
}

// ---------------------------------------------------------------------------
// SetIdempotencyKeyResponse
// ---------------------------------------------------------------------------

func (s *IdempotencyHandlerTestSuite) TestSetIdempotencyKeyResponse_NilRequest() {
	resp, err := s.handler.SetIdempotencyKeyResponse(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *IdempotencyHandlerTestSuite) TestSetIdempotencyKeyResponse_Success() {
	s.repo.EXPECT().
		SetResponse(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.SetResponseParams) *apierror.APIError {
			s.Equal("ikey_1", params.ID)
			s.Equal(201, params.StatusCode)
			s.Equal(idempotency.RecoveryPointFinished.String(), params.RecoveryPoint)
			s.Equal(json.RawMessage(`{"id":"new"}`), params.Body)
			return nil
		}).Times(1)

	resp, err := s.handler.SetIdempotencyKeyResponse(context.Background(), &pb.SetIdempotencyKeyResponseRequest{
		IdempotencyKeyId: "ikey_1",
		ResponseCode:     201,
		ResponseBody:     []byte(`{"id":"new"}`),
		ResponseHeaders:  []byte(`{}`),
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *IdempotencyHandlerTestSuite) TestSetIdempotencyKeyResponse_WithTTL() {
	ttl := int32(3600)
	s.repo.EXPECT().
		SetResponse(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.SetResponseParams) *apierror.APIError {
			s.NotNil(params.TTLSeconds)
			s.Equal(int32(3600), *params.TTLSeconds)
			return nil
		}).Times(1)

	resp, err := s.handler.SetIdempotencyKeyResponse(context.Background(), &pb.SetIdempotencyKeyResponseRequest{
		IdempotencyKeyId: "ikey_1",
		ResponseCode:     200,
		TtlSeconds:       &ttl,
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *IdempotencyHandlerTestSuite) TestSetIdempotencyKeyResponse_RepoError() {
	s.repo.EXPECT().SetResponse(gomock.Any(), gomock.Any()).Return(apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.SetIdempotencyKeyResponse(context.Background(), &pb.SetIdempotencyKeyResponseRequest{
		IdempotencyKeyId: "ikey_1",
		ResponseCode:     200,
	})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// ReleaseIdempotencyKey
// ---------------------------------------------------------------------------

func (s *IdempotencyHandlerTestSuite) TestReleaseIdempotencyKey_NilRequest() {
	resp, err := s.handler.ReleaseIdempotencyKey(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *IdempotencyHandlerTestSuite) TestReleaseIdempotencyKey_Success() {
	s.repo.EXPECT().ReleaseLock(gomock.Any(), "ikey_1").Return(nil).Times(1)

	resp, err := s.handler.ReleaseIdempotencyKey(context.Background(), &pb.ReleaseIdempotencyKeyRequest{
		IdempotencyKeyId: "ikey_1",
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *IdempotencyHandlerTestSuite) TestReleaseIdempotencyKey_RepoError() {
	s.repo.EXPECT().ReleaseLock(gomock.Any(), "ikey_1").Return(apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.ReleaseIdempotencyKey(context.Background(), &pb.ReleaseIdempotencyKeyRequest{
		IdempotencyKeyId: "ikey_1",
	})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// AdvanceRecoveryPoint
// ---------------------------------------------------------------------------

func (s *IdempotencyHandlerTestSuite) TestAdvanceRecoveryPoint_NilRequest() {
	resp, err := s.handler.AdvanceRecoveryPoint(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *IdempotencyHandlerTestSuite) TestAdvanceRecoveryPoint_Success() {
	stepData := json.RawMessage(`{"stripe_sub":"sub_123"}`)
	s.repo.EXPECT().
		AdvanceRecoveryPoint(gomock.Any(), domain.AdvanceRecoveryPointParams{
			ID:            "ikey_1",
			RecoveryPoint: "stripe_subscribed",
			StepData:      stepData,
		}).
		Return(nil).Times(1)

	resp, err := s.handler.AdvanceRecoveryPoint(context.Background(), &pb.AdvanceRecoveryPointRequest{
		IdempotencyKeyId: "ikey_1",
		RecoveryPoint:    "stripe_subscribed",
		StepData:         stepData,
	})
	s.NoError(err)
	s.True(resp.Success)
}

func (s *IdempotencyHandlerTestSuite) TestAdvanceRecoveryPoint_RepoError() {
	s.repo.EXPECT().AdvanceRecoveryPoint(gomock.Any(), gomock.Any()).Return(apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.AdvanceRecoveryPoint(context.Background(), &pb.AdvanceRecoveryPointRequest{
		IdempotencyKeyId: "ikey_1",
		RecoveryPoint:    "step2",
	})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// GetRecoveryPoint
// ---------------------------------------------------------------------------

func (s *IdempotencyHandlerTestSuite) TestGetRecoveryPoint_NilRequest() {
	resp, err := s.handler.GetRecoveryPoint(context.Background(), nil)
	s.Nil(resp)
	s.Error(err)
}

func (s *IdempotencyHandlerTestSuite) TestGetRecoveryPoint_Success() {
	stepData := json.RawMessage(`{"stripe_sub":"sub_123"}`)
	s.repo.EXPECT().
		GetRecoveryPoint(gomock.Any(), "ikey_1").
		Return(&domain.GetRecoveryPointResult{
			RecoveryPoint: "stripe_subscribed",
			StepData:      stepData,
		}, nil).Times(1)

	resp, err := s.handler.GetRecoveryPoint(context.Background(), &pb.GetRecoveryPointRequest{
		IdempotencyKeyId: "ikey_1",
	})
	s.NoError(err)
	s.Equal("stripe_subscribed", resp.RecoveryPoint)
	s.Equal([]byte(stepData), resp.StepData)
}

func (s *IdempotencyHandlerTestSuite) TestGetRecoveryPoint_RepoError() {
	s.repo.EXPECT().GetRecoveryPoint(gomock.Any(), "ikey_1").Return(nil, apierror.NewInternalError(nil, "fail")).Times(1)

	resp, err := s.handler.GetRecoveryPoint(context.Background(), &pb.GetRecoveryPointRequest{
		IdempotencyKeyId: "ikey_1",
	})
	s.Nil(resp)
	s.Error(err)
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestIdempotencyHandlerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IdempotencyHandlerTestSuite))
}
