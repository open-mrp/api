package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/auth-service/internal/testutil"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// RefreshTokenMedTestSuite provides a test suite for RefreshTokenMed tests
type RefreshTokenMedTestSuite struct {
	suite.Suite
	refreshTokenMed  domain.RefreshTokenMed
	refreshTokenRepo *repositorymock.MockRefreshTokenRepo
	repoFactory      *factorymock.MockRepoFactory
	ctrl             *gomock.Controller
}

// SetupSuite runs once before all tests in the suite
func (suite *RefreshTokenMedTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.refreshTokenRepo = repositorymock.NewMockRefreshTokenRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewRefreshTokenRepo().Return(suite.refreshTokenRepo).AnyTimes()

	refreshTokenMedConfig := &RefreshTokenMedConfig{
		Repos: suite.repoFactory,
	}
	suite.refreshTokenMed = NewRefreshTokenMed(refreshTokenMedConfig)
}

// TearDownSuite runs once after all tests in the suite
func (suite *RefreshTokenMedTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

// TestRefreshTokenMedTestSuite runs the test suite
func TestRefreshTokenMedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RefreshTokenMedTestSuite))
}

func (suite *RefreshTokenMedTestSuite) TestValidateRefreshToken_ValidToken() {
	// Mock the refresh token repository to return a valid refresh token model
	expectedUserID := testutil.EntityIDUser
	expectedRefreshToken := &domain.RefreshToken{
		Token:     "valid-refresh-token",
		UserID:    expectedUserID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
	}
	suite.refreshTokenRepo.EXPECT().
		Find(gomock.Any(), "valid-refresh-token").
		Return(expectedRefreshToken, nil).
		Times(1)

	// Test with valid refresh token
	ctx := context.Background()
	userID, err := suite.refreshTokenMed.Validate(ctx, "valid-refresh-token")
	suite.Nil(err)
	suite.Equal(expectedUserID, userID)
}

func (suite *RefreshTokenMedTestSuite) TestValidateRefreshToken_InvalidToken() {
	// Mock the refresh token repository to return an error
	suite.refreshTokenRepo.EXPECT().
		Find(gomock.Any(), "invalid-refresh-token").
		Return(nil, apierror.NewAuthenticationError("Invalid refresh token")).
		Times(1)

	// Test with invalid refresh token
	ctx := context.Background()
	userID, err := suite.refreshTokenMed.Validate(ctx, "invalid-refresh-token")
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Equal("", userID)
}

func (suite *RefreshTokenMedTestSuite) TestCreateRefreshToken_ValidUserID() {
	userID := testutil.EntityIDUser

	// Mock the refresh token repository to create a token
	// The opaqueTokenUtils will generate a real token, so we need to accept any token string
	suite.refreshTokenRepo.EXPECT().
		Create(gomock.Any(), userID, gomock.Any(), 30).
		Return(&domain.RefreshToken{
			Token:     "new-refresh-token",
			UserID:    userID,
			ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour),
		}, nil).
		Times(1)

	// Test creating refresh token
	ctx := context.Background()
	expiresInDays := 30
	refreshToken, err := suite.refreshTokenMed.Create(ctx, userID, &expiresInDays)
	suite.Nil(err)
	suite.NotNil(refreshToken)
	suite.Equal(userID, refreshToken.UserID)
	suite.NotEmpty(refreshToken.Token)
}

func (suite *RefreshTokenMedTestSuite) TestCreateRefreshToken_EmptyUserID() {
	// Test with empty user ID - the opaqueTokenUtils will still generate a token,
	// but the repo Create will be called. Let's test that it fails appropriately
	ctx := context.Background()
	expiresInDays := 30

	// Mock the repo to return an error for empty userID
	suite.refreshTokenRepo.EXPECT().
		Create(gomock.Any(), "", gomock.Any(), 30).
		Return(nil, apierror.NewValidationError("User ID cannot be empty")).
		Times(1)

	refreshToken, err := suite.refreshTokenMed.Create(ctx, "", &expiresInDays)
	suite.NotNil(err)
	suite.Nil(refreshToken)
}
