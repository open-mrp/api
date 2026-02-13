package service

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type APIKeySvcTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	repoFactory    *factorymock.MockRepoFactory
	docAPIKeyRepo  *repositorymock.MockDocAPIKeyRepo
	apiKeyMed      *mediatormock.MockAPIKeyMed
	idempotencyMed *mediatormock.MockIdempotencyMed
	encryptionKey  []byte
	svc            domain.APIKeySvc
}

func (s *APIKeySvcTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.docAPIKeyRepo = repositorymock.NewMockDocAPIKeyRepo(s.ctrl)
	s.apiKeyMed = mediatormock.NewMockAPIKeyMed(s.ctrl)
	s.idempotencyMed = mediatormock.NewMockIdempotencyMed(s.ctrl)

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewDocAPIKeyRepo().Return(s.docAPIKeyRepo).AnyTimes()

	s.encryptionKey = make([]byte, 32)
	_, err := rand.Read(s.encryptionKey)
	s.Require().NoError(err)

	s.svc = NewAPIKeySvc(&APIKeySvcConfig{
		Repos: s.repoFactory,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			APIKey:      s.apiKeyMed,
			Idempotency: s.idempotencyMed,
		}},
		TxManager: stubTxManager{
			repoFactory: s.repoFactory,
		},
		EncryptionKey: s.encryptionKey,
	})
}

func (s *APIKeySvcTestSuite) TearDownSuite() {
	s.ctrl.Finish()
}

func TestAPIKeySvcTestSuite(t *testing.T) {
	suite.Run(t, new(APIKeySvcTestSuite))
}

func (s *APIKeySvcTestSuite) newAdminCtx() context.Context {
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
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func (s *APIKeySvcTestSuite) TestRotateAPIKey_Success_NoDocAPIKey() {
	ctx := s.newAdminCtx()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	newAPIKey := &domain.APIKey{
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

	s.docAPIKeyRepo.EXPECT().
		FindByAPIKeyID(gomock.Any(), testAPIKeyTypeID).
		Return(nil, nil)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), constants.AccountModeProduction, testAPIKeyTypeID, &expiresAt).
		Return(testAPIKeySecret, newAPIKey, nil)

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

func (s *APIKeySvcTestSuite) TestRotateAPIKey_Success_UpdatesDocAPIKey() {
	ctx := s.newAdminCtx()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	newAPIKey := &domain.APIKey{
		ID:             2,
		TypeID:         "apke_new123",
		Name:           "Test API Key",
		OwnerAccountID: testOwnerAccountID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	encrypted, err := crypto.EncryptAESGCM([]byte("old_secret"), s.encryptionKey)
	s.Require().NoError(err)

	existingDocKey := &domain.DocAPIKey{
		ID:              1,
		TypeID:          testDocAPIKeyTypeID,
		APIKeyID:        testAPIKeyTypeID,
		EncryptedSecret: encrypted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	s.docAPIKeyRepo.EXPECT().
		FindByAPIKeyID(gomock.Any(), testAPIKeyTypeID).
		Return(existingDocKey, nil)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), constants.AccountModeProduction, testAPIKeyTypeID, &expiresAt).
		Return(testAPIKeySecret, newAPIKey, nil)

	s.docAPIKeyRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, docKey *domain.DocAPIKey) *apierror.APIError {
			s.Equal(int64(1), docKey.ID)
			s.Equal("apke_new123", docKey.APIKeyID)
			s.NotEmpty(docKey.EncryptedSecret)
			s.NotEqual(encrypted, docKey.EncryptedSecret)

			decrypted, decErr := crypto.DecryptAESGCM(docKey.EncryptedSecret, s.encryptionKey)
			s.Require().NoError(decErr)
			s.Equal(testAPIKeySecret, string(decrypted))

			return nil
		})

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

func (s *APIKeySvcTestSuite) TestRotateAPIKey_FindDocAPIKeyError() {
	ctx := s.newAdminCtx()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	s.docAPIKeyRepo.EXPECT().
		FindByAPIKeyID(gomock.Any(), testAPIKeyTypeID).
		Return(nil, apierror.NewInternalError(nil, "db error"))

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

func (s *APIKeySvcTestSuite) TestRotateAPIKey_UpdateDocAPIKeyError() {
	ctx := s.newAdminCtx()

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	newAPIKey := &domain.APIKey{
		ID:             2,
		TypeID:         "apke_new123",
		Name:           "Test API Key",
		OwnerAccountID: testOwnerAccountID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	encrypted, err := crypto.EncryptAESGCM([]byte("old_secret"), s.encryptionKey)
	s.Require().NoError(err)

	existingDocKey := &domain.DocAPIKey{
		ID:              1,
		TypeID:          testDocAPIKeyTypeID,
		APIKeyID:        testAPIKeyTypeID,
		EncryptedSecret: encrypted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil)

	s.docAPIKeyRepo.EXPECT().
		FindByAPIKeyID(gomock.Any(), testAPIKeyTypeID).
		Return(existingDocKey, nil)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), constants.AccountModeProduction, testAPIKeyTypeID, &expiresAt).
		Return(testAPIKeySecret, newAPIKey, nil)

	s.docAPIKeyRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(apierror.NewInternalError(nil, "update failed"))

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
