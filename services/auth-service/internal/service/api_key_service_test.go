package service

import (
	"context"
	"testing"
	"time"

	factorymock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/repository"

	"github.com/open-mrp/api/services/auth-service/internal/apikey"
	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type APIKeySvcTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	repoFactory    *factorymock.MockRepoFactory
	apiKeyRepo     *repositorymock.MockAPIKeyRepo
	apiKeyMed      *mediatormock.MockAPIKeyMed
	docAPIKeyMed   *mediatormock.MockDocAPIKeyMed
	idempotencyMed *mediatormock.MockIdempotencyMed
	svc            domain.APIKeySvc
}

func (s *APIKeySvcTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.apiKeyMed = mediatormock.NewMockAPIKeyMed(s.ctrl)
	s.docAPIKeyMed = mediatormock.NewMockDocAPIKeyMed(s.ctrl)
	s.idempotencyMed = mediatormock.NewMockIdempotencyMed(s.ctrl)

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(s.ctrl)
	s.repoFactory.EXPECT().NewAPIKeyRepo().Return(s.apiKeyRepo).AnyTimes()
	s.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	s.svc = NewAPIKeySvc(&APIKeySvcConfig{
		Repos: s.repoFactory,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			APIKey:      s.apiKeyMed,
			DocAPIKey:   s.docAPIKeyMed,
			Idempotency: s.idempotencyMed,
		}},
		TxManager: stubTxManager{
			repoFactory: s.repoFactory,
		},
	})
}

func (s *APIKeySvcTestSuite) TearDownSuite() {
	s.ctrl.Finish()
}

func TestAPIKeySvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(APIKeySvcTestSuite))
}

func (s *APIKeySvcTestSuite) newAdminCtx() context.Context {
	identity := &types.Identity{
		Type:        types.IdentityActorTypeUser,
		Target:      &types.IdentityTarget{AccountID: testOwnerAccountID},
		AccountMode: constants.AccountModeProduction,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    new(testOwnerAccountID),
			RoleType:     new("admin"),
		},
	}
	ctx := appctx.WithIdentity(context.Background(), identity)
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func (s *APIKeySvcTestSuite) TestRotateAPIKey_Success_NoDocAPIKey() {
	ctx := s.newAdminCtx()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	newAPIKey := &apikey.APIKey{
		ID:             2,
		TypeID:         "apke_new123",
		Name:           "Test API Key",
		OwnerAccountID: testOwnerAccountID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	s.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), testAPIKeyTypeID, gomock.Nil()).
		Return(&apikey.APIKey{
			TypeID:         testAPIKeyTypeID,
			OwnerAccountID: testOwnerAccountID,
		}, nil).
		Times(2)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), gomock.Any()).
		Return(testAPIKeySecret, newAPIKey, nil)

	s.docAPIKeyMed.EXPECT().
		SyncRotatedAPIKey(gomock.Any(), domain.DocAPIKeySyncInput{
			OldAPIKeyID: testAPIKeyTypeID,
			NewSecret:   testAPIKeySecret,
			NewAPIKey:   newAPIKey,
		}).
		Return(nil)

	s.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		Return(nil)

	result, apiErr := s.svc.RotateAPIKey(ctx, domain.RotateAPIKeyInput{
		APIKeyID:  testAPIKeyTypeID,
		ExpiresAt: &expiresAt,
	})

	s.Nil(apiErr)
	s.NotNil(result)
	s.Equal(testAPIKeySecret, result.APIKeySecret)
	s.Equal("apke_new123", result.APIKey.TypeID)
}

// TestRevokeAPIKey_CrossTenant_ReturnsNotFound asserts that a caller targeting
// account A cannot revoke an API key owned by account B; the service must
// return resource_not_found without leaking the key's existence or invoking
// the revoke mediator.
func (s *APIKeySvcTestSuite) TestRevokeAPIKey_CrossTenant_ReturnsNotFound() {
	ctx := s.newAdminCtx()

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_revoke_cross",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	// The key exists, but its owner is a different account than the caller's target.
	s.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), testAPIKeyTypeID, gomock.Nil()).
		Return(&apikey.APIKey{
			TypeID:         testAPIKeyTypeID,
			OwnerAccountID: "acct_otherTenant",
		}, nil).
		Times(1)

	// APIKey.Revoke must not be invoked when the ownership check fails.
	// No expectation set; the mock will fail the test if it is called.

	s.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		})

	apiErr := s.svc.RevokeAPIKey(ctx, domain.RevokeAPIKeyInput{
		APIKeyID: testAPIKeyTypeID,
	})

	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

// TestRotateAPIKey_CrossTenant_ReturnsNotFound asserts that a caller targeting
// account A cannot rotate an API key owned by account B; the service must
// return resource_not_found without invoking the rotate mediator (which would
// return a fresh secret for the victim key).
func (s *APIKeySvcTestSuite) TestRotateAPIKey_CrossTenant_ReturnsNotFound() {
	ctx := s.newAdminCtx()

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_rotate_cross",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	s.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), testAPIKeyTypeID, gomock.Nil()).
		Return(&apikey.APIKey{
			TypeID:         testAPIKeyTypeID,
			OwnerAccountID: "acct_otherTenant",
		}, nil).
		Times(1)

	// APIKey.Rotate and DocAPIKey.SyncRotatedAPIKey must not be invoked.

	s.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		})

	result, apiErr := s.svc.RotateAPIKey(ctx, domain.RotateAPIKeyInput{
		APIKeyID: testAPIKeyTypeID,
	})

	s.Nil(result)
	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

func (s *APIKeySvcTestSuite) TestRotateAPIKey_SyncDocAPIKeyError() {
	ctx := s.newAdminCtx()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	newAPIKey := &apikey.APIKey{
		ID:             2,
		TypeID:         "apke_new123",
		Name:           "Test API Key",
		OwnerAccountID: testOwnerAccountID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	s.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), testAPIKeyTypeID, gomock.Nil()).
		Return(&apikey.APIKey{
			TypeID:         testAPIKeyTypeID,
			OwnerAccountID: testOwnerAccountID,
		}, nil).
		Times(1)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), gomock.Any()).
		Return(testAPIKeySecret, newAPIKey, nil)

	s.docAPIKeyMed.EXPECT().
		SyncRotatedAPIKey(gomock.Any(), domain.DocAPIKeySyncInput{
			OldAPIKeyID: testAPIKeyTypeID,
			NewSecret:   testAPIKeySecret,
			NewAPIKey:   newAPIKey,
		}).
		Return(apierror.NewInternalError(nil, "sync failed"))

	s.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		})

	result, apiErr := s.svc.RotateAPIKey(ctx, domain.RotateAPIKeyInput{
		APIKeyID:  testAPIKeyTypeID,
		ExpiresAt: &expiresAt,
	})

	s.Nil(result)
	s.NotNil(apiErr)
}
