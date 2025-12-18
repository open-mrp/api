package mediator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	// Test models for API key service tests
	apiKeyValidTestModel    *domain.APIKey
	apiKeyValidProdModel    *domain.APIKey
	apiKeyExpiredModel      *domain.APIKey
	apiKeyBadSecretModel    *domain.APIKey
	apiKeyNeverExpiresModel *domain.APIKey
)

// APIKeyMedTestSuite provides a test suite for APIKeyMed tests
type APIKeyMedTestSuite struct {
	suite.Suite
	apiKeyMed   domain.APIKeyMed
	apiKeyUtils domain.APIKeyUtils
	apiKeyRepo  *repositorymock.MockAPIKeyRepo
	repoFactory *factorymock.MockRepoFactory
	ctrl        *gomock.Controller
}

// SetupSuite runs once before all tests in the suite
func (suite *APIKeyMedTestSuite) SetupSuite() {
	apiKeyConfig := apikey.DefaultAPIKeyConfig([]byte(testutil.Pepper))
	suite.apiKeyUtils = apikey.NewAPIKeyUtils(apiKeyConfig)
	suite.ctrl = gomock.NewController(suite.T())
	suite.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(suite.apiKeyRepo).AnyTimes()

	apiKeyMedConfig := APIKeyMedConfig{
		Repos:       suite.repoFactory,
		APIKeyUtils: suite.apiKeyUtils,
	}
	suite.apiKeyMed = NewAPIKeyMed(apiKeyMedConfig)

	// Parse API keys to extract secrets and generate hashes
	parsedTestKey, err := suite.apiKeyUtils.Parse(context.Background(), testutil.APIKeyValidSandboxMode)
	if err != nil {
		suite.T().Fatalf("Failed to parse test API key: %v", err)
	}
	secretHash, err := suite.apiKeyUtils.GenSecretHMAC(context.Background(), parsedTestKey.Secret)
	if err != nil {
		suite.T().Fatalf("Failed to generate secret HMAC: %v", err)
	}
	apiKeyValidTestModel = testutil.GetValidTestAPIKeyModel(secretHash)
	apiKeyValidTestModel.Name = "Test API Key"

	parsedProdKey, err := suite.apiKeyUtils.Parse(context.Background(), testutil.APIKeyValidProdMode)
	if err != nil {
		suite.T().Fatalf("Failed to parse prod API key: %v", err)
	}
	secretHash, err = suite.apiKeyUtils.GenSecretHMAC(context.Background(), parsedProdKey.Secret)
	if err != nil {
		suite.T().Fatalf("Failed to generate secret HMAC: %v", err)
	}
	apiKeyValidProdModel = testutil.GetValidProdAPIKeyModel(secretHash)
	apiKeyValidProdModel.Name = "Prod API Key"

	parsedExpiredKey, err := suite.apiKeyUtils.Parse(context.Background(), testutil.ApiKeyExpired)
	if err != nil {
		suite.T().Fatalf("Failed to parse expired API key: %v", err)
	}
	secretHash, err = suite.apiKeyUtils.GenSecretHMAC(context.Background(), parsedExpiredKey.Secret)
	if err != nil {
		suite.T().Fatalf("Failed to generate secret HMAC: %v", err)
	}
	apiKeyExpiredModel = testutil.GetExpiredAPIKeyModel(secretHash)
	apiKeyExpiredModel.Name = "Expired API Key"

	// Use a different secret hash to simulate bad secret (don't parse the actual key)
	secretHash, err = suite.apiKeyUtils.GenSecretHMAC(context.Background(), "wrong-secret")
	if err != nil {
		suite.T().Fatalf("Failed to generate secret HMAC: %v", err)
	}
	apiKeyBadSecretModel = testutil.GetBadSecretAPIKeyModel(secretHash)
	apiKeyBadSecretModel.Name = "Bad Secret API Key"

	parsedNeverExpiresKey, err := suite.apiKeyUtils.Parse(context.Background(), testutil.ApiKeyNeverExpires)
	if err != nil {
		suite.T().Fatalf("Failed to parse never expires API key: %v", err)
	}
	secretHash, err = suite.apiKeyUtils.GenSecretHMAC(context.Background(), parsedNeverExpiresKey.Secret)
	if err != nil {
		suite.T().Fatalf("Failed to generate secret HMAC: %v", err)
	}
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
	suite.Equal(testutil.EntityIDAPIKeyValidSandboxMode, apiKey.ID)
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
	suite.Equal(testutil.EntityIDAPIKeyValidProdMode, apiKey.ID)
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
	suite.Equal(testutil.EntityIDAPIKeyNeverExpires, apiKey.ID)
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
	suite.Equal(contracts.ErrorCodeExpiredAPIKey, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_BadSecret() {
	// Generate a valid API key for this test
	validKey, err := suite.apiKeyUtils.Gen(context.Background(), constants.AccountModeProduction)
	suite.Nil(err)
	keyString := validKey.String()

	// Create a model with a wrong secret hash (simulating bad secret in DB)
	wrongSecretHash, err := suite.apiKeyUtils.GenSecretHMAC(context.Background(), "wrong-secret")
	suite.Nil(err)
	badSecretModel := &domain.APIKey{
		ID:             validKey.ID,
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
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidAPIKey() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyInvalid).
		Return(nil, nil).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyInvalid)

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_RepoError() {
	ctx := context.Background()
	suite.apiKeyRepo.EXPECT().
		Find(gomock.Any(), testutil.EntityIDAPIKeyValidSandboxMode).
		Return(nil, contracts.NewInternalError(fmt.Errorf("database error"), "Database connection failed")).
		Times(1)

	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.APIKeyValidSandboxMode)

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInternalError, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidFormat() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, "not-an-api-key")

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_EmptyString() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, "")

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_TooShort() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyTooShort)

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_TooLong() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyTooLong)

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidPrefix() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyInvalidPrefix)

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_InvalidChecksum() {
	ctx := context.Background()
	apiKey, err := suite.apiKeyMed.FindAndValidate(ctx, testutil.ApiKeyInvalidChecksum)

	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestFind_VerifySecretError() {
	// Create a service with different pepper to cause verification failure
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	apiKeyRepo := repositorymock.NewMockAPIKeyRepo(ctrl)
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewAPIKeyRepo().Return(apiKeyRepo).AnyTimes()
	apiKeyUtils := apikey.NewAPIKeyUtils(apikey.DefaultAPIKeyConfig([]byte("different-pepper")))

	apiKeyMedConfig := APIKeyMedConfig{
		Repos:       repoFactory,
		APIKeyUtils: apiKeyUtils,
	}
	apiKeyMed := NewAPIKeyMed(apiKeyMedConfig)

	// Generate secret hash with different pepper
	secretHash, err := apiKeyUtils.GenSecretHMAC(context.Background(), "eR0LAkxYmlLllMxoTIwMcLls1Nvn1oIk1Z8pSrOhztciaRPPjK")
	suite.Nil(err)
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
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(apiKey)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_LastUsedAtIsNil() {
	ctx := context.Background()
	apiKey := &domain.APIKey{
		ID:         "api-key-id",
		LastUsedAt: nil,
	}

	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), "api-key-id").
		Return(nil).
		Times(1)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.Nil(err)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_LastUsedAtOlderThan24Hours() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-25 * time.Hour)
	apiKey := &domain.APIKey{
		ID:         "api-key-id",
		LastUsedAt: &lastUsedAt,
	}

	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), "api-key-id").
		Return(nil).
		Times(1)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.Nil(err)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_LastUsedAtLessThan24HoursAgo() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-2 * time.Hour)
	apiKey := &domain.APIKey{
		ID:         "api-key-id",
		LastUsedAt: &lastUsedAt,
	}

	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), "api-key-id").
		Times(0)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.Nil(err)
}

func (suite *APIKeyMedTestSuite) TestTouchIfNotRecent_TouchReturnsError() {
	ctx := context.Background()
	lastUsedAt := time.Now().UTC().Add(-25 * time.Hour)
	apiKey := &domain.APIKey{
		ID:         "api-key-id-error",
		LastUsedAt: &lastUsedAt,
	}

	expectedErr := contracts.NewInternalError(nil, "Failed to touch API key.")
	suite.apiKeyRepo.EXPECT().
		Touch(gomock.Any(), "api-key-id-error").
		Return(expectedErr).
		Times(1)

	err := suite.apiKeyMed.TouchIfNotRecent(ctx, apiKey)
	suite.NotNil(err)
	suite.Equal(expectedErr, err)
}
