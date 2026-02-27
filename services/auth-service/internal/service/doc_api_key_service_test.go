package service

import (
	"context"
	"encoding/json"
	"testing"

	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testOwnerAccountID   = "acct_owner123"
	testSandboxAccountID = "acct_sandbox123"
	testAdminRoleID      = "rl_admin123"
	testAPIKeyTypeID     = "apke_test123"
	testAPIKeySecret     = "aug_sk_test_abc123_secret" // #nosec G101
	testDocAPIKeyTypeID  = "apkedf_test123"
)

type DocAPIKeySvcTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	repoFactory    *factorymock.MockRepoFactory
	docAPIKeyMed   *mediatormock.MockDocAPIKeyMed
	idempotencyMed *mediatormock.MockIdempotencyMed
	svc            domain.DocAPIKeySvc
}

func (s *DocAPIKeySvcTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.docAPIKeyMed = mediatormock.NewMockDocAPIKeyMed(s.ctrl)
	s.idempotencyMed = mediatormock.NewMockIdempotencyMed(s.ctrl)

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)

	s.svc = NewDocAPIKeySvc(&DocAPIKeySvcConfig{
		Repos: s.repoFactory,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			DocAPIKey:   s.docAPIKeyMed,
			Idempotency: s.idempotencyMed,
		}},
		TxManager: stubTxManager{
			repoFactory: s.repoFactory,
		},
	})
}

func (s *DocAPIKeySvcTestSuite) TearDownSuite() {
	s.ctrl.Finish()
}

func TestDocAPIKeySvcTestSuite(t *testing.T) {
	suite.Run(t, new(DocAPIKeySvcTestSuite))
}

func (s *DocAPIKeySvcTestSuite) newAdminCtx() context.Context {
	identity := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: strPtr(testSandboxAccountID),
		AccountMode:     constants.AccountModeSandbox,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: strPtr("admin"),
		},
	}
	ctx := appctx.WithIdentity(context.Background(), identity)
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func strPtr(s string) *string {
	return &s
}

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_DelegatesToMediator() {
	ctx := s.newAdminCtx()

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	expectedResult := &domain.GetOrCreateDocAPIKeyResult{
		APIKeySecret: testAPIKeySecret,
		APIKey: &apikey.APIKey{
			TypeID: testAPIKeyTypeID,
		},
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	s.docAPIKeyMed.EXPECT().
		Resolve(gomock.Any(), testSandboxAccountID).
		Return(expectedResult, nil).
		Times(1)

	s.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		Return(nil).
		Times(1)

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(apiErr)
	s.NotNil(result)
	s.Equal(testAPIKeySecret, result.APIKeySecret)
	s.Equal(testAPIKeyTypeID, result.APIKey.TypeID)
}

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_CachedResponse() {
	ctx := s.newAdminCtx()

	cachedResult := &domain.GetOrCreateDocAPIKeyResult{
		APIKeySecret: testAPIKeySecret,
		APIKey: &apikey.APIKey{
			TypeID: testAPIKeyTypeID,
			Name:   "Documentation API Key",
		},
	}

	responseBody, err := json.Marshal(cachedResult)
	s.Require().NoError(err)

	statusCode := 200
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointFinished,
		ResponseCode:  &statusCode,
		ResponseBody:  responseBody,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(apiErr)
	s.NotNil(result)
	s.Equal(testAPIKeySecret, result.APIKeySecret)
	s.Equal(testAPIKeyTypeID, result.APIKey.TypeID)
}

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_ResolveError_CachesError() {
	ctx := s.newAdminCtx()

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	resolveErr := apierror.NewInternalError(nil, "resolve failed")

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	s.docAPIKeyMed.EXPECT().
		Resolve(gomock.Any(), testSandboxAccountID).
		Return(nil, resolveErr).
		Times(1)

	s.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(result)
	s.NotNil(apiErr)
}

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_RequiresTargetAccount() {
	identity := &types.Identity{
		Type:        types.IdentityTypeUser,
		AccountMode: constants.AccountModeSandbox,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: strPtr("admin"),
		},
	}
	ctx := appctx.WithIdentity(context.Background(), identity)

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(result)
	s.NotNil(apiErr)
}

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_RejectsProductionAccount() {
	identity := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: strPtr(testOwnerAccountID),
		AccountMode:     constants.AccountModeProduction,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: strPtr("admin"),
		},
	}
	ctx := appctx.WithIdentity(context.Background(), identity)

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(result)
	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	s.Contains(apiErr.PublicMessage, "sandbox")
}
