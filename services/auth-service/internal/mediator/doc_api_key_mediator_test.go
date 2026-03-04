package mediator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	clientmock "github.com/augno/api/services/auth-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DocAPIKeyMedTestSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	repoFactory   *factorymock.MockRepoFactory
	docAPIKeyRepo *repositorymock.MockDocAPIKeyRepo
	apiKeyRepo    *repositorymock.MockAPIKeyRepo
	coreClient    *clientmock.MockAuthCoreClient
	apiKeyMed     *mediatormock.MockAPIKeyMed
	encryptionKey []byte
	med           domain.DocAPIKeyMed
}

func (s *DocAPIKeyMedTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.docAPIKeyRepo = repositorymock.NewMockDocAPIKeyRepo(s.ctrl)
	s.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(s.ctrl)
	s.coreClient = clientmock.NewMockAuthCoreClient(s.ctrl)
	s.apiKeyMed = mediatormock.NewMockAPIKeyMed(s.ctrl)
	s.encryptionKey = []byte("01234567890123456789012345678901") // 32 bytes

	s.repoFactory.EXPECT().NewDocAPIKeyRepo().Return(s.docAPIKeyRepo).AnyTimes()
	s.repoFactory.EXPECT().NewAPIKeyRepo().Return(s.apiKeyRepo).AnyTimes()

	s.med = NewDocAPIKeyMed(&DocAPIKeyMedConfig{
		Repos:         s.repoFactory,
		EncryptionKey: s.encryptionKey,
		CoreClient:    s.coreClient,
		APIKeyMed:     s.apiKeyMed,
	})
}

func (s *DocAPIKeyMedTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestDocAPIKeyMedTestSuite(t *testing.T) {
	suite.Run(t, new(DocAPIKeyMedTestSuite))
}

func (s *DocAPIKeyMedTestSuite) TestResolve_ReturnExistingKey_HasRedactedValue() {
	ctx := context.Background()
	sandboxAccountID := "acc_sandbox123"
	apiKeyTypeID := "apikey_test123"
	secret := "my-secret-value"

	encrypted, err := crypto.EncryptAESGCM([]byte(secret), s.encryptionKey, []byte(apiKeyTypeID), docAPIKeyEncryptionKeyID)
	s.Require().NoError(err)

	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	existingDocKey := &apikey.DocAPIKey{
		ID:              1,
		TypeID:          "dak_test123",
		APIKeyID:        apiKeyTypeID,
		EncryptedSecret: encrypted,
		APIKeyExpiresAt: &expiresAt,
	}

	returnedAPIKey := &apikey.APIKey{
		ID:             1,
		TypeID:         apiKeyTypeID,
		KeyID:          "someKeyID",
		Name:           "Test Doc API Key",
		RedactedValue:  "aug_sk_test_****abcd",
		OwnerAccountID: sandboxAccountID,
		RoleID:         "rol_test123",
		ExpiresAt:      &expiresAt,
	}

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), sandboxAccountID).
		Return(existingDocKey, nil)

	s.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), apiKeyTypeID, gomock.Nil()).
		Return(returnedAPIKey, nil)

	result, apiErr := s.med.Resolve(ctx, sandboxAccountID)

	s.Nil(apiErr)
	s.Require().NotNil(result)
	s.Equal(secret, result.APIKeySecret)
	s.NotEmpty(result.APIKey.RedactedValue, "returned API key must have a RedactedValue")
	s.True(strings.HasSuffix(result.APIKey.RedactedValue, "abcd"),
		"RedactedValue should end with last four")
	s.True(strings.Contains(result.APIKey.RedactedValue, "****"),
		"RedactedValue should contain redaction mask")
}

func (s *DocAPIKeyMedTestSuite) TestResolve_CreateDocAPIKey_HasRedactedValue() {
	ctx := context.Background()
	sandboxAccountID := "acc_sandbox123"

	createdAPIKey := &apikey.APIKey{
		ID:            1,
		TypeID:        "apikey_new123",
		Name:          "Doc API Key [System Generated]",
		RedactedValue: "aug_sk_test_****wxyz",
	}

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), sandboxAccountID).
		Return(nil, apierror.NewResourceNotFoundError("not found"))

	s.coreClient.EXPECT().
		GetAdminRole(gomock.Any()).
		Return("rol_admin123", nil)

	s.apiKeyMed.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return("aug_sk_test_full_secret_wxyz", createdAPIKey, nil)

	s.docAPIKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(int64(1), nil)

	result, apiErr := s.med.Resolve(ctx, sandboxAccountID)

	s.Nil(apiErr)
	s.Require().NotNil(result)
	s.NotEmpty(result.APIKey.RedactedValue, "created API key must have a RedactedValue")
	s.True(strings.HasSuffix(result.APIKey.RedactedValue, "wxyz"),
		"RedactedValue should end with last four")
	s.True(strings.Contains(result.APIKey.RedactedValue, "****"),
		"RedactedValue should contain redaction mask")
}

func (s *DocAPIKeyMedTestSuite) TestResolve_RotateDocAPIKey_HasRedactedValue() {
	ctx := context.Background()
	sandboxAccountID := "acc_sandbox123"
	oldAPIKeyTypeID := "apikey_old123"

	expiredAt := time.Now().UTC().Add(-24 * time.Hour)
	existingDocKey := &apikey.DocAPIKey{
		ID:              1,
		TypeID:          "dak_old123",
		APIKeyID:        oldAPIKeyTypeID,
		EncryptedSecret: "old-encrypted",
		APIKeyExpiresAt: &expiredAt,
	}

	rotatedAPIKey := &apikey.APIKey{
		ID:            2,
		TypeID:        "apikey_rotated123",
		Name:          "Doc API Key [System Generated]",
		RedactedValue: "aug_sk_test_****qrst",
	}

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), sandboxAccountID).
		Return(existingDocKey, nil)

	s.docAPIKeyRepo.EXPECT().
		Delete(gomock.Any(), existingDocKey.ID).
		Return(nil)

	s.apiKeyMed.EXPECT().
		Rotate(gomock.Any(), gomock.Any()).
		Return("aug_sk_test_full_rotated_qrst", rotatedAPIKey, nil)

	s.docAPIKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(int64(2), nil)

	result, apiErr := s.med.Resolve(ctx, sandboxAccountID)

	s.Nil(apiErr)
	s.Require().NotNil(result)
	s.NotEmpty(result.APIKey.RedactedValue, "rotated API key must have a RedactedValue")
	s.True(strings.HasSuffix(result.APIKey.RedactedValue, "qrst"),
		"RedactedValue should end with last four")
	s.True(strings.Contains(result.APIKey.RedactedValue, "****"),
		"RedactedValue should contain redaction mask")
}

func (s *DocAPIKeyMedTestSuite) TestResolve_RevokedKey_StillFetchable() {
	ctx := context.Background()
	sandboxAccountID := "acc_sandbox123"

	// Simulate an API key revoked more than 30 days ago — outside the list filter window.
	revokedAt := time.Now().UTC().Add(-60 * 24 * time.Hour)
	existingDocKey := &apikey.DocAPIKey{
		ID:              1,
		TypeID:          "dak_revoked123",
		APIKeyID:        "apikey_revoked123",
		EncryptedSecret: "old-encrypted",
		APIKeyRevokedAt: &revokedAt,
	}

	s.docAPIKeyRepo.EXPECT().
		FindBySandboxAccountID(gomock.Any(), sandboxAccountID).
		Return(existingDocKey, nil)

	result, apiErr := s.med.Resolve(ctx, sandboxAccountID)

	// Resolve should return a validation error about revoked key, NOT a "not found" error.
	s.Nil(result)
	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	s.Contains(apiErr.PublicMessage, "revoked")
}
