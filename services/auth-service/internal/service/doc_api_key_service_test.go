package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	clientmock "github.com/augno/api/services/auth-service/internal/domain/mock/client"
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
	docAPIKeyRepo  *repositorymock.MockDocAPIKeyRepo
	apiKeyRepo     *repositorymock.MockAPIKeyRepo
	apiKeyMed      *mediatormock.MockAPIKeyMed
	idempotencyMed *mediatormock.MockIdempotencyMed
	coreClient     *clientmock.MockAuthCoreClient
	encryptionKey  []byte
	svc            domain.DocAPIKeySvc
}

func (s *DocAPIKeySvcTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.docAPIKeyRepo = repositorymock.NewMockDocAPIKeyRepo(s.ctrl)
	s.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(s.ctrl)
	s.apiKeyMed = mediatormock.NewMockAPIKeyMed(s.ctrl)
	s.idempotencyMed = mediatormock.NewMockIdempotencyMed(s.ctrl)
	s.coreClient = clientmock.NewMockAuthCoreClient(s.ctrl)

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewDocAPIKeyRepo().Return(s.docAPIKeyRepo).AnyTimes()
	s.repoFactory.EXPECT().NewAPIKeyRepo().Return(s.apiKeyRepo).AnyTimes()

	s.encryptionKey = make([]byte, 32)
	_, err := rand.Read(s.encryptionKey)
	s.Require().NoError(err)

	s.svc = NewDocAPIKeySvc(&DocAPIKeySvcConfig{
		Repos: s.repoFactory,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			APIKey:      s.apiKeyMed,
			Idempotency: s.idempotencyMed,
		}},
		TxManager: stubTxManager{
			repoFactory: s.repoFactory,
		},
		CoreClient:    s.coreClient,
		EncryptionKey: s.encryptionKey,
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
		TargetAccountID: strPtr(testOwnerAccountID),
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

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_CreateNew() {
	ctx := s.newAdminCtx()

	apiKeyModel := &domain.APIKey{
		ID:             1,
		TypeID:         testAPIKeyTypeID,
		Name:           "Documentation API Key",
		OwnerAccountID: testSandboxAccountID,
		RoleID:         testAdminRoleID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	// No existing doc API key
	s.coreClient.EXPECT().
		GetSandboxAccountByOwner(gomock.Any(), testOwnerAccountID).
		Return(testSandboxAccountID, nil)

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), testSandboxAccountID).
		Return(nil, nil)

	s.coreClient.EXPECT().
		GetAdminRole(gomock.Any()).
		Return(testAdminRoleID, nil)

	s.apiKeyMed.EXPECT().
		Create(gomock.Any(), constants.AccountModeSandbox, testSandboxAccountID, testAdminRoleID, "Documentation API Key [System Generated]", gomock.Any()).
		Return(testAPIKeySecret, apiKeyModel, nil)

	s.docAPIKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, docKey *domain.DocAPIKey) (int64, *apierror.APIError) {
			s.NotEmpty(docKey.TypeID)
			s.Equal(testAPIKeyTypeID, docKey.APIKeyID)
			s.NotEmpty(docKey.EncryptedSecret)
			return 1, nil
		})

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

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_ReturnExisting() {
	ctx := s.newAdminCtx()

	encrypted, err := crypto.EncryptAESGCM([]byte(testAPIKeySecret), s.encryptionKey)
	s.Require().NoError(err)

	futureTime := time.Now().Add(7 * 24 * time.Hour)
	existing := &domain.DocAPIKey{
		ID:              1,
		TypeID:          testDocAPIKeyTypeID,
		APIKeyID:        testAPIKeyTypeID,
		EncryptedSecret: encrypted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		APIKeyExpiresAt: &futureTime,
	}

	apiKeyModel := &domain.APIKey{
		ID:             1,
		TypeID:         testAPIKeyTypeID,
		Name:           "Documentation API Key",
		OwnerAccountID: testSandboxAccountID,
		RoleID:         testAdminRoleID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ExpiresAt:      &futureTime,
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	s.coreClient.EXPECT().
		GetSandboxAccountByOwner(gomock.Any(), testOwnerAccountID).
		Return(testSandboxAccountID, nil)

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), testSandboxAccountID).
		Return(existing, nil)

	s.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), testAPIKeyTypeID).
		Return(apiKeyModel, nil)

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

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_RotateExpired() {
	ctx := s.newAdminCtx()

	encrypted, err := crypto.EncryptAESGCM([]byte("old_secret"), s.encryptionKey)
	s.Require().NoError(err)

	expiredTime := time.Now().Add(-1 * time.Hour)
	existing := &domain.DocAPIKey{
		ID:              1,
		TypeID:          testDocAPIKeyTypeID,
		APIKeyID:        "apke_old123",
		EncryptedSecret: encrypted,
		CreatedAt:       time.Now().Add(-31 * 24 * time.Hour),
		UpdatedAt:       time.Now().Add(-31 * 24 * time.Hour),
		APIKeyExpiresAt: &expiredTime,
	}

	newAPIKeyModel := &domain.APIKey{
		ID:             2,
		TypeID:         testAPIKeyTypeID,
		Name:           "Documentation API Key",
		OwnerAccountID: testSandboxAccountID,
		RoleID:         testAdminRoleID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	s.coreClient.EXPECT().
		GetSandboxAccountByOwner(gomock.Any(), testOwnerAccountID).
		Return(testSandboxAccountID, nil)

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), testSandboxAccountID).
		Return(existing, nil)

	s.docAPIKeyRepo.EXPECT().
		DeleteAllBySandboxAccountID(gomock.Any(), testSandboxAccountID).
		Return(nil)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), constants.AccountModeSandbox, "apke_old123", gomock.Any()).
		Return(testAPIKeySecret, newAPIKeyModel, nil)

	s.docAPIKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, docKey *domain.DocAPIKey) (int64, *apierror.APIError) {
			s.NotEmpty(docKey.TypeID)
			s.Equal(testAPIKeyTypeID, docKey.APIKeyID)
			s.NotEmpty(docKey.EncryptedSecret)
			return 2, nil
		})

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

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_RotateRevoked() {
	ctx := s.newAdminCtx()

	encrypted, err := crypto.EncryptAESGCM([]byte("old_secret"), s.encryptionKey)
	s.Require().NoError(err)

	revokedTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(7 * 24 * time.Hour)
	existing := &domain.DocAPIKey{
		ID:              1,
		TypeID:          testDocAPIKeyTypeID,
		APIKeyID:        "apke_old123",
		EncryptedSecret: encrypted,
		CreatedAt:       time.Now().Add(-10 * 24 * time.Hour),
		UpdatedAt:       time.Now().Add(-10 * 24 * time.Hour),
		APIKeyExpiresAt: &futureTime,
		APIKeyRevokedAt: &revokedTime,
	}

	newAPIKeyModel := &domain.APIKey{
		ID:             2,
		TypeID:         testAPIKeyTypeID,
		Name:           "Documentation API Key",
		OwnerAccountID: testSandboxAccountID,
		RoleID:         testAdminRoleID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	s.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)

	s.coreClient.EXPECT().
		GetSandboxAccountByOwner(gomock.Any(), testOwnerAccountID).
		Return(testSandboxAccountID, nil)

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), testSandboxAccountID).
		Return(existing, nil)

	s.docAPIKeyRepo.EXPECT().
		DeleteAllBySandboxAccountID(gomock.Any(), testSandboxAccountID).
		Return(nil)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), constants.AccountModeSandbox, "apke_old123", gomock.Any()).
		Return(testAPIKeySecret, newAPIKeyModel, nil)

	s.docAPIKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, docKey *domain.DocAPIKey) (int64, *apierror.APIError) {
			s.NotEmpty(docKey.TypeID)
			s.Equal(testAPIKeyTypeID, docKey.APIKeyID)
			s.NotEmpty(docKey.EncryptedSecret)
			return 2, nil
		})

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
		APIKey: &domain.APIKey{
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

	// No business logic mocks should be called

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(apiErr)
	s.NotNil(result)
	s.Equal(testAPIKeySecret, result.APIKeySecret)
	s.Equal(testAPIKeyTypeID, result.APIKey.TypeID)
}

func (s *DocAPIKeySvcTestSuite) TestGetOrCreateDocAPIKey_RequiresInternalAdmin() {
	// Non-admin identity
	identity := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: strPtr(testOwnerAccountID),
		AccountMode:     constants.AccountModeSandbox,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: strPtr("member"),
		},
	}
	ctx := appctx.WithIdentity(context.Background(), identity)

	result, apiErr := s.svc.GetOrCreateDocAPIKey(ctx)

	s.Nil(result)
	s.NotNil(apiErr)
	s.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
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
