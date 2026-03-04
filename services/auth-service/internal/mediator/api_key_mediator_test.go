package mediator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	clientmock "github.com/augno/api/services/auth-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	// Test models for API key service tests
	apiKeyValidTestModel    *apikey.APIKey
	apiKeyValidProdModel    *apikey.APIKey
	apiKeyExpiredModel      *apikey.APIKey
	apiKeyBadSecretModel    *apikey.APIKey
	apiKeyNeverExpiresModel *apikey.APIKey
)

// APIKeyMedTestSuite provides a test suite for APIKeyMed tests
type APIKeyMedTestSuite struct {
	suite.Suite
	apiKeyMed   domain.APIKeyMed
	apiKeyRepo  *repositorymock.MockAPIKeyRepo
	repoFactory *factorymock.MockRepoFactory
	ctrl        *gomock.Controller
}

// SetupSuite runs once before all tests in the suite
func (suite *APIKeyMedTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(suite.apiKeyRepo).AnyTimes()

	coreClientMock := clientmock.NewMockAuthCoreClient(suite.ctrl)

	apiKeyMedConfig := &APIKeyMedConfig{
		Repos:      suite.repoFactory,
		Pepper:     []byte(testutil.Pepper),
		CoreClient: coreClientMock,
	}
	suite.apiKeyMed = NewAPIKeyMed(apiKeyMedConfig)

	pepper := []byte(testutil.Pepper)

	// Parse API keys to extract secrets and generate hashes
	parsedTestKey, err := apikey.ParseAPIKey(testutil.APIKeyValidSandboxMode)
	if err != nil {
		suite.T().Fatalf("Failed to parse test API key: %v", err)
	}
	secretHash := parsedTestKey.GenSecretHMAC(pepper)
	apiKeyValidTestModel = testutil.GetValidTestAPIKeyModel(secretHash)
	apiKeyValidTestModel.Name = "Test API Key"

	parsedProdKey, err := apikey.ParseAPIKey(testutil.APIKeyValidProdMode)
	if err != nil {
		suite.T().Fatalf("Failed to parse prod API key: %v", err)
	}
	secretHash = parsedProdKey.GenSecretHMAC(pepper)
	apiKeyValidProdModel = testutil.GetValidProdAPIKeyModel(secretHash)
	apiKeyValidProdModel.Name = "Prod API Key"

	parsedExpiredKey, err := apikey.ParseAPIKey(testutil.ApiKeyExpired)
	if err != nil {
		suite.T().Fatalf("Failed to parse expired API key: %v", err)
	}
	secretHash = parsedExpiredKey.GenSecretHMAC(pepper)
	apiKeyExpiredModel = testutil.GetExpiredAPIKeyModel(secretHash)
	apiKeyExpiredModel.Name = "Expired API Key"

	// Use a different secret hash to simulate bad secret (don't parse the actual key)
	secretHash = crypto.HMACSHA256(pepper, []byte("wrong-secret"))
	apiKeyBadSecretModel = testutil.GetBadSecretAPIKeyModel(secretHash)
	apiKeyBadSecretModel.Name = "Bad Secret API Key"

	parsedNeverExpiresKey, err := apikey.ParseAPIKey(testutil.ApiKeyNeverExpires)
	if err != nil {
		suite.T().Fatalf("Failed to parse never expires API key: %v", err)
	}
	secretHash = parsedNeverExpiresKey.GenSecretHMAC(pepper)
	apiKeyNeverExpiresModel = testutil.GetNeverExpiresAPIKeyModel(secretHash)
	apiKeyNeverExpiresModel.Name = "Never Expires API Key"
}

// TearDownSuite runs once after all tests in the suite
func (suite *APIKeyMedTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

// TestAPIKeyMedTestSuite runs the test suite
func TestAPIKeyMedTestSuite(t *testing.T) {
	suite.Run(t, new(APIKeyMedTestSuite))
}

func (suite *APIKeyMedTestSuite) TestFind_ValidSandboxAPIKey() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyValidSandboxMode).
		Return(apiKeyValidTestModel, nil).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.APIKeyValidSandboxMode)

	suite.Nil(err)
	suite.NotNil(apiKey)
	suite.Equal(testutil.EntityIDAPIKeyValidSandboxMode, apiKey.KeyID)
	suite.Equal(testutil.EntityIDAccount, apiKey.OwnerAccountID)
	suite.Equal(testutil.EntityIDRole, apiKey.RoleID)
}

func (suite *APIKeyMedTestSuite) TestFind_ValidProdAPIKey() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyValidProdMode).
		Return(apiKeyValidProdModel, nil).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.APIKeyValidProdMode)

	suite.Nil(err)
	suite.NotNil(apiKey)
	suite.Equal(testutil.EntityIDAPIKeyValidProdMode, apiKey.KeyID)
	suite.Equal(testutil.EntityIDAccount, apiKey.OwnerAccountID)
	suite.Equal(testutil.EntityIDRole, apiKey.RoleID)
}

func (suite *APIKeyMedTestSuite) TestFind_NeverExpires() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyNeverExpires).
		Return(apiKeyNeverExpiresModel, nil).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyNeverExpires)

	suite.Nil(err)
	suite.NotNil(apiKey)
	suite.Equal(testutil.EntityIDAPIKeyNeverExpires, apiKey.KeyID)
	suite.Nil(apiKey.ExpiresAt)
}

func (suite *APIKeyMedTestSuite) TestFind_ExpiredAPIKey() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyExpired).
		Return(apiKeyExpiredModel, nil).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyExpired)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeExpiredAPIKey, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_BadSecret() {
	// Generate a valid API key for this test
	validKey, err := apikey.GenParsedAPIKey(constants.AccountModeProduction, nil)
	suite.Nil(err)
	keyString := validKey.String()

	// Create a model with a wrong secret hash (simulating bad secret in DB)
	wrongSecretHash := crypto.HMACSHA256([]byte(testutil.Pepper), []byte("wrong-secret"))
	badSecretModel := &apikey.APIKey{
		ID:             1,
		TypeID:         "apikey_sandbox",
		KeyID:          validKey.ID,
		Name:           "Bad Secret Key",
		OwnerAccountID: testutil.EntityIDAccount,
		RoleID:         testutil.EntityIDRole,
		SecretHash:     wrongSecretHash,
		ExpiresAt:      nil, // Never expires
	}

	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), validKey.ID).
		Return(badSecretModel, nil).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, keyString)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidAPIKey() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyInvalid).
		Return(nil, apierror.NewResourceNotFoundError("API key not found")).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyInvalid)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_RepoError() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyValidSandboxMode).
		Return(nil, apierror.NewInternalError(fmt.Errorf("database error"), "Database connection failed")).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.APIKeyValidSandboxMode)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidFormat() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, "not-an-api-key")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidFormat, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_EmptyString() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, "")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidFormat, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_TooShort() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyTooShort)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidFormat, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_TooLong() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyTooLong)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidFormat, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidPrefix() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyInvalidPrefix)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidFormat, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidChecksum() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyInvalidChecksum)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidFormat, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_VerifySecretError() {
	// Create a service with different pepper to cause verification failure
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	apiKeyRepo := repositorymock.NewMockAPIKeyRepo(ctrl)
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewAPIKeyRepo().Return(apiKeyRepo).AnyTimes()

	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	apiKeyMedConfig := &APIKeyMedConfig{
		Repos:      repoFactory,
		Pepper:     []byte("different-pepper"),
		CoreClient: coreClient,
	}
	apiKeyMed := NewAPIKeyMed(apiKeyMedConfig)

	// Generate secret hash with different pepper
	secretHash := crypto.HMACSHA256([]byte("different-pepper"), []byte("eR0LAkxYmlLllMxoTIwMcLls1Nvn1oIk1Z8pSrOhztciaRPPjK"))
	apiKeyModel := testutil.GetValidTestAPIKeyModel(secretHash)
	apiKeyModel.Name = "Test API Key"

	ctx := context.Background()
	apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyValidSandboxMode).
		Return(apiKeyModel, nil).
		Times(1)

	// This should fail because we're using the original API key string which was generated with different pepper
	apiKey, err := apiKeyMed.FindAndValidate(ctx, testutil.APIKeyValidSandboxMode)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_LastUsedAtIsNil() {
	ctx := context.Background()
	apiKey := &apikey.APIKey{
		ID:         1,
		LastUsedAt: nil,
	}

	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), int64(1)).
		Return(nil).
		Times(1)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.Nil(err)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_LastUsedAtOlderThan24Hours() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-25 * time.Hour)
	apiKey := &apikey.APIKey{
		ID:         1,
		LastUsedAt: &lastUsedAt,
	}

	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), int64(1)).
		Return(nil).
		Times(1)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.Nil(err)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_LastUsedAtLessThan24HoursAgo() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-2 * time.Hour)
	apiKey := &apikey.APIKey{
		ID:         1,
		LastUsedAt: &lastUsedAt,
	}

	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), int64(1)).
		Times(0)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.Nil(err)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_TouchReturnsError() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-25 * time.Hour)
	apiKey := &apikey.APIKey{
		ID:         1,
		LastUsedAt: &lastUsedAt,
	}

	expectedErr := apierror.NewInternalError(nil, "Failed to touch API key.")
	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), int64(1)).
		Return(expectedErr).
		Times(1)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.NotNil(err)
	suite.Equal(expectedErr, err)
}

func (suite *APIKeyMedTestSuite) TestCreate_Success() {
	ctx := context.Background()
	accountMode := constants.AccountModeProduction
	ownerAccountID := "ac_123456789012"
	roleID := "rl_123456789012"
	name := "Test API Key"
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	var createdKey *apikey.APIKey
	suite.apiKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError) {
			suite.Equal(name, apiKey.Name)
			suite.Equal(ownerAccountID, apiKey.OwnerAccountID)
			suite.Equal(roleID, apiKey.RoleID)
			suite.Equal(&expiresAt, apiKey.ExpiresAt)
			suite.NotEmpty(apiKey.TypeID)
			suite.NotEmpty(apiKey.KeyID)
			suite.NotEmpty(apiKey.SecretHash)
			suite.NotEmpty(apiKey.RedactedValue)
			apiKey.ID = 123
			createdKey = apiKey
			return 123, nil
		}).
		Times(1)

	suite.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
			createdKey.RoleName = "Admin"
			createdKey.RoleTypeCode = "admin"
			return createdKey, nil
		}).
		Times(1)

	apiKeyString, apiKeyModel, err := suite.apiKeyMed.Create(ctx, domain.APIKeyCreateInput{
		AccountMode:    accountMode,
		OwnerAccountID: ownerAccountID,
		RoleID:         roleID,
		Name:           name,
		ExpiresAt:      &expiresAt,
	})

	suite.Nil(err)
	suite.NotEmpty(apiKeyString)
	suite.NotNil(apiKeyModel)
	suite.Equal(int64(123), apiKeyModel.ID)

	// Verify the generated key can be parsed
	parsedKey, parseErr := apikey.ParseAPIKey(apiKeyString)
	suite.Nil(parseErr)
	suite.Equal(accountMode, parsedKey.AccountMode)
	suite.Equal(apiKeyModel.KeyID, parsedKey.ID)

	// Verify the secret hash matches
	suite.True(parsedKey.VerifySecretHMAC([]byte(testutil.Pepper), apiKeyModel.SecretHash))
}

func (suite *APIKeyMedTestSuite) TestCreate_RedactedValueEndsWithLastFourOfFullKey() {
	ctx := context.Background()
	ownerAccountID := "ac_123456789012"
	roleID := "rl_123456789012"
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	var createdKey *apikey.APIKey
	suite.apiKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError) {
			apiKey.ID = 1
			createdKey = apiKey
			return 1, nil
		}).
		Times(1)

	suite.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
			return createdKey, nil
		}).
		Times(1)

	apiKeyString, apiKeyModel, err := suite.apiKeyMed.Create(ctx, domain.APIKeyCreateInput{
		AccountMode:    constants.AccountModeSandbox,
		OwnerAccountID: ownerAccountID,
		RoleID:         roleID,
		Name:           "Test Key",
		ExpiresAt:      &expiresAt,
	})

	suite.Nil(err)
	suite.NotNil(apiKeyModel)

	// RedactedValue must end with the actual last 4 characters of the full key string
	expectedLastFour := apiKeyString[len(apiKeyString)-4:]
	suite.Equal(expectedLastFour, apiKeyModel.RedactedValue[len(apiKeyModel.RedactedValue)-4:])
}

func (suite *APIKeyMedTestSuite) TestCreate_RedactedValueMatchesLastFour() {
	ctx := context.Background()
	ownerAccountID := "ac_123456789012"
	roleID := "rl_123456789012"
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	tests := []struct {
		name        string
		accountMode constants.AccountMode
	}{
		{"sandbox", constants.AccountModeSandbox},
		{"production", constants.AccountModeProduction},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var createdKey *apikey.APIKey
			suite.apiKeyRepo.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError) {
					apiKey.ID = 1
					createdKey = apiKey
					return 1, nil
				}).
				Times(1)

			suite.apiKeyRepo.EXPECT().
				FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
				DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
					return createdKey, nil
				}).
				Times(1)

			apiKeyString, apiKeyModel, err := suite.apiKeyMed.Create(ctx, domain.APIKeyCreateInput{
				AccountMode:    tt.accountMode,
				OwnerAccountID: ownerAccountID,
				RoleID:         roleID,
				Name:           "Test Key",
				ExpiresAt:      &expiresAt,
			})

			suite.Nil(err)
			suite.NotNil(apiKeyModel)

			// RedactedValue should end with the same last 4 characters as the full key
			expectedLastFour := apiKeyString[len(apiKeyString)-4:]
			suite.True(
				len(apiKeyModel.RedactedValue) > 4,
				"RedactedValue should not be empty",
			)
			suite.Equal(
				expectedLastFour,
				apiKeyModel.RedactedValue[len(apiKeyModel.RedactedValue)-4:],
				"RedactedValue should end with the actual last 4 characters of the full key",
			)

			// RedactedValue should contain the correct mode prefix
			expectedPrefix := "aug_sk_" + string(tt.accountMode) + "_****"
			suite.Equal(expectedPrefix, apiKeyModel.RedactedValue[:len(expectedPrefix)])
		})
	}
}

func (suite *APIKeyMedTestSuite) TestRotate_HasRedactedValue() {
	ctx := context.Background()
	ownerAccountID := "ac_123456789012"
	roleID := "rl_123456789012"
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	apiKeyTypeID := "apikey_old123"

	// The old key that Rotate will find
	oldKey := &apikey.APIKey{
		ID:             10,
		TypeID:         apiKeyTypeID,
		Name:           "Rotate Me",
		RedactedValue:  "aug_sk_test_****old1",
		OwnerAccountID: ownerAccountID,
		RoleID:         roleID,
		ExpiresAt:      &expiresAt,
	}

	suite.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), apiKeyTypeID, gomock.Nil()).
		Return(oldKey, nil).
		Times(1)

	suite.apiKeyRepo.EXPECT().
		Revoke(gomock.Any(), apiKeyTypeID).
		Return(nil).
		Times(1)

	var createdKey *apikey.APIKey
	suite.apiKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ak *apikey.APIKey) (int64, *apierror.APIError) {
			ak.ID = 1
			createdKey = ak
			return 1, nil
		}).
		Times(1)

	suite.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
			return createdKey, nil
		}).
		Times(1)

	_, rotatedModel, err := suite.apiKeyMed.Rotate(ctx, domain.APIKeyRotateInput{
		AccountMode:  constants.AccountModeSandbox,
		APIKeyTypeID: apiKeyTypeID,
	})

	suite.Nil(err)
	suite.Require().NotNil(rotatedModel)
	suite.NotEmpty(rotatedModel.RedactedValue, "rotated key must have a RedactedValue")
	suite.True(
		len(rotatedModel.RedactedValue) > 4,
		"RedactedValue should not be empty",
	)
	suite.True(
		strings.Contains(rotatedModel.RedactedValue, "****"),
		"RedactedValue should contain redaction mask",
	)
}

func (suite *APIKeyMedTestSuite) TestRotate_RevokesOldKey() {
	ctx := context.Background()
	apiKeyTypeID := "apikey_torevoke"
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	oldKey := &apikey.APIKey{
		ID:             10,
		TypeID:         apiKeyTypeID,
		Name:           "Old Key",
		OwnerAccountID: "ac_123456789012",
		RoleID:         "rl_123456789012",
		ExpiresAt:      &expiresAt,
	}

	suite.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), apiKeyTypeID, gomock.Nil()).
		Return(oldKey, nil).
		Times(1)

	suite.apiKeyRepo.EXPECT().
		Revoke(gomock.Any(), apiKeyTypeID).
		Return(nil).
		Times(1)

	var createdKey *apikey.APIKey
	suite.apiKeyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ak *apikey.APIKey) (int64, *apierror.APIError) {
			suite.Equal(oldKey.OwnerAccountID, ak.OwnerAccountID)
			suite.Equal(oldKey.RoleID, ak.RoleID)
			suite.Equal(oldKey.Name, ak.Name)
			ak.ID = 11
			createdKey = ak
			return 11, nil
		}).
		Times(1)

	suite.apiKeyRepo.EXPECT().
		FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
		DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
			return createdKey, nil
		}).
		Times(1)

	fullKey, newKey, err := suite.apiKeyMed.Rotate(ctx, domain.APIKeyRotateInput{
		AccountMode:  constants.AccountModeSandbox,
		APIKeyTypeID: apiKeyTypeID,
	})

	suite.Nil(err)
	suite.NotEmpty(fullKey)
	suite.Require().NotNil(newKey)
	suite.NotEqual(apiKeyTypeID, newKey.TypeID, "rotated key should have a new TypeID")
}

func (suite *APIKeyMedTestSuite) TestList_HasRedactedValue() {
	ctx := context.Background()
	ownerAccountID := "ac_123456789012"

	suite.apiKeyRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&domain.APIKeyListRepoResult{
			APIKeys: []*apikey.APIKey{
				{
					TypeID:        "apikey_1",
					Name:          "Key 1",
					RedactedValue: "aug_sk_test_****aaaa",
				},
				{
					TypeID:        "apikey_2",
					Name:          "Key 2",
					RedactedValue: "aug_sk_test_****bbbb",
				},
			},
		}, nil).
		Times(1)

	result, err := suite.apiKeyMed.List(ctx, domain.APIKeyListInput{
		OwnerAccountID: ownerAccountID,
		Limit:          10,
	})

	suite.Nil(err)
	suite.Require().NotNil(result)
	suite.Require().Len(result.APIKeys, 2)

	for _, ak := range result.APIKeys {
		suite.NotEmpty(ak.RedactedValue, "listed key %s must have a RedactedValue", ak.TypeID)
		suite.True(
			strings.Contains(ak.RedactedValue, "****"),
			"RedactedValue should contain redaction mask for key %s", ak.TypeID,
		)
	}
}

func (suite *APIKeyMedTestSuite) TestCreate_RedactedValueConsistentAcrossModes() {
	ctx := context.Background()
	ownerAccountID := "ac_123456789012"
	roleID := "rl_123456789012"
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	for _, mode := range []constants.AccountMode{constants.AccountModeSandbox, constants.AccountModeProduction} {
		var createdKey *apikey.APIKey
		suite.apiKeyRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError) {
				apiKey.ID = 1
				createdKey = apiKey
				return 1, nil
			}).
			Times(1)

		suite.apiKeyRepo.EXPECT().
			FindByTypeID(gomock.Any(), gomock.Any(), gomock.Nil()).
			DoAndReturn(func(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
				return createdKey, nil
			}).
			Times(1)

		apiKeyString, apiKeyModel, err := suite.apiKeyMed.Create(ctx, domain.APIKeyCreateInput{
			AccountMode:    mode,
			OwnerAccountID: ownerAccountID,
			RoleID:         roleID,
			Name:           "Test Key",
			ExpiresAt:      &expiresAt,
		})
		suite.Nil(err)

		lastFour := apiKeyString[len(apiKeyString)-4:]

		// 1. RedactedValue ends with the last 4 chars of the full key
		suite.Equal(lastFour, apiKeyModel.RedactedValue[len(apiKeyModel.RedactedValue)-4:])

		// 2. RedactedValue format is correct: aug_sk_{mode}_****{lastFour}
		expectedRedacted := "aug_sk_" + string(mode) + "_****" + lastFour
		suite.Equal(expectedRedacted, apiKeyModel.RedactedValue)
	}
}
