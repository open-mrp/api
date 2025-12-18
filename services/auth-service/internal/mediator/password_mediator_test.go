package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	publishermock "github.com/augno/api/services/auth-service/internal/domain/mock/publisher"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	emailpkg "github.com/augno/api/services/auth-service/internal/email"
	"github.com/augno/api/services/auth-service/internal/password"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// stubTemplateRenderer is a no-op implementation of TemplateRenderer for testing
type stubTemplateRenderer struct{}

func (s *stubTemplateRenderer) RenderWelcomeEmail(ctx context.Context, data emailpkg.WelcomeEmailData) (string, *contracts.APIError) {
	return "<html>Welcome</html>", nil
}

func (s *stubTemplateRenderer) RenderPasswordResetEmail(ctx context.Context, data emailpkg.PasswordResetEmailData) (string, *contracts.APIError) {
	return "<html>Password Reset</html>", nil
}

func (s *stubTemplateRenderer) RenderPasswordUpdatedEmail(ctx context.Context, data emailpkg.PasswordUpdatedEmailData) (string, *contracts.APIError) {
	return "<html>Password Updated</html>", nil
}

// PasswordMedTestSuite provides a test suite for PasswordMed tests
type PasswordMedTestSuite struct {
	suite.Suite
	passwordMed           domain.PasswordMed
	userRepo              *repositorymock.MockUserRepo
	repoFactory           *factorymock.MockRepoFactory
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	notificationPublisher *publishermock.MockNotificationPublisher
	templateRenderer      emailpkg.TemplateRenderer
	jwtUtils              domain.JWTUtils
	opaqueTokenUtils      domain.OpaqueTokenUtils
	ctrl                  *gomock.Controller
}

// SetupSuite runs once before all tests in the suite
func (suite *PasswordMedTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.userRepo = repositorymock.NewMockUserRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUserRepo().Return(suite.userRepo).AnyTimes()
	suite.refreshTokenMed = mediatormock.NewMockRefreshTokenMed(suite.ctrl)
	suite.notificationPublisher = publishermock.NewMockNotificationPublisher(suite.ctrl)
	suite.templateRenderer = &stubTemplateRenderer{}

	jwtConfig := token.DefaultJWTConfig(testutil.JWTSecret)
	suite.jwtUtils = token.NewJWTUtils(jwtConfig)
	suite.opaqueTokenUtils = token.NewOpaqueTokenUtils(token.DefaultOpaqueTokenConfig())

	passwordMedConfig := PasswordMedConfig{
		Repos:                 suite.repoFactory,
		RefreshTokenMed:       suite.refreshTokenMed,
		JWTUtils:              suite.jwtUtils,
		OpaqueTokenUtils:      suite.opaqueTokenUtils,
		NotificationPublisher: suite.notificationPublisher,
		TemplateRenderer:      suite.templateRenderer,
		FrontendURL:           "https://test.example.com",
	}
	suite.passwordMed = NewPasswordMed(passwordMedConfig)
}

// TearDownSuite runs once after all tests in the suite
func (suite *PasswordMedTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

// TestPasswordMedTestSuite runs the test suite
func TestPasswordMedTestSuite(t *testing.T) {
	suite.Run(t, new(PasswordMedTestSuite))
}

func (suite *PasswordMedTestSuite) TestUpdatePassword_Success() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	user := &types.User{ID: userID}

	suite.userRepo.EXPECT().
		UpdatePassword(gomock.Any(), userID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.refreshTokenMed.EXPECT().
		RevokeAll(gomock.Any(), userID).
		Return(nil).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.Nil(apiErr)
}

func (suite *PasswordMedTestSuite) TestUpdatePassword_UpdatePasswordError() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	user := &types.User{ID: userID}

	suite.userRepo.EXPECT().
		UpdatePassword(gomock.Any(), userID, gomock.Any()).
		Return(contracts.NewInternalError(nil, "Database error")).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.NotNil(apiErr)
	suite.Equal(contracts.ErrorCodeInternalError, apiErr.Code)
}

func (suite *PasswordMedTestSuite) TestUpdatePassword_RevokeAllError() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	user := &types.User{ID: userID}

	suite.userRepo.EXPECT().
		UpdatePassword(gomock.Any(), userID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.refreshTokenMed.EXPECT().
		RevokeAll(gomock.Any(), userID).
		Return(contracts.NewInternalError(nil, "Failed to revoke tokens")).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.NotNil(apiErr)
	suite.Equal(contracts.ErrorCodeInternalError, apiErr.Code)
}

func (suite *PasswordMedTestSuite) TestUpdatePassword_WithEmailNotification_Success() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	notifyEmail := "user@example.com"
	userName := "John Doe"

	user := &types.User{
		ID:             userID,
		Email:          stringPtr(notifyEmail),
		Name:           stringPtr(userName),
		HashedPassword: stringPtr("hashed"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		UpdatePassword(gomock.Any(), userID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.refreshTokenMed.EXPECT().
		RevokeAll(gomock.Any(), userID).
		Return(nil).
		Times(1)

	suite.notificationPublisher.EXPECT().
		PublishSendEmail(
			gomock.Any(),
			[]string{notifyEmail},
			"Password Updated",
			"<html>Password Updated</html>",
			true,
			nil,
			userID,
			nil,
		).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.Nil(apiErr)
}

func (suite *PasswordMedTestSuite) TestUpdatePassword_WithEmailNotification_UserNameNil() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	notifyEmail := "user@example.com"

	user := &types.User{
		ID:             userID,
		Email:          stringPtr(notifyEmail),
		Name:           nil,
		HashedPassword: stringPtr("hashed"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		UpdatePassword(gomock.Any(), userID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.refreshTokenMed.EXPECT().
		RevokeAll(gomock.Any(), userID).
		Return(nil).
		Times(1)

	suite.notificationPublisher.EXPECT().
		PublishSendEmail(
			gomock.Any(),
			[]string{notifyEmail},
			"Password Updated",
			"<html>Password Updated</html>",
			true,
			nil,
			userID,
			nil,
		).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.Nil(apiErr)
}

func (suite *PasswordMedTestSuite) TestUpdatePassword_WithEmailNotification_TemplateRendererError() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	notifyEmail := "user@example.com"
	userName := "John Doe"

	user := &types.User{
		ID:             userID,
		Email:          stringPtr(notifyEmail),
		Name:           stringPtr(userName),
		HashedPassword: stringPtr("hashed"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create a failing template renderer
	failingTemplateRenderer := &failingTemplateRenderer{}

	passwordMedConfig := PasswordMedConfig{
		Repos:                 suite.repoFactory,
		RefreshTokenMed:       suite.refreshTokenMed,
		JWTUtils:              suite.jwtUtils,
		OpaqueTokenUtils:      suite.opaqueTokenUtils,
		NotificationPublisher: suite.notificationPublisher,
		TemplateRenderer:      failingTemplateRenderer,
		FrontendURL:           "https://test.example.com",
	}
	passwordMed := NewPasswordMed(passwordMedConfig)

	suite.userRepo.EXPECT().
		UpdatePassword(gomock.Any(), userID, gomock.Any()).
		Return(nil).
		Times(1)

	suite.refreshTokenMed.EXPECT().
		RevokeAll(gomock.Any(), userID).
		Return(nil).
		Times(1)

	newPassword := "newPassword456"
	apiErr := passwordMed.Update(ctx, user, newPassword)

	suite.NotNil(apiErr)
	suite.Equal(contracts.ErrorCodeInternalError, apiErr.Code)
}

// failingTemplateRenderer is a template renderer that always fails
type failingTemplateRenderer struct{}

func (f *failingTemplateRenderer) RenderWelcomeEmail(ctx context.Context, data emailpkg.WelcomeEmailData) (string, *contracts.APIError) {
	return "", contracts.NewInternalError(nil, "Template renderer error")
}

func (f *failingTemplateRenderer) RenderPasswordResetEmail(ctx context.Context, data emailpkg.PasswordResetEmailData) (string, *contracts.APIError) {
	return "", contracts.NewInternalError(nil, "Template renderer error")
}

func (f *failingTemplateRenderer) RenderPasswordUpdatedEmail(ctx context.Context, data emailpkg.PasswordUpdatedEmailData) (string, *contracts.APIError) {
	return "", contracts.NewInternalError(nil, "Template renderer error")
}

func (suite *PasswordMedTestSuite) TestValidatePasswordResetToken_Success() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	hashedPassword, err := password.HashPassword(ctx, "initialPassword123")
	suite.Require().Nil(err)

	user := &types.User{
		ID:             userID,
		Email:          stringPtr("reset@example.com"),
		HashedPassword: &hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		Find(gomock.Any(), userID).
		Return(user, nil).
		Times(1)

	resetToken, err := suite.jwtUtils.Encode(ctx, userID, 15*time.Minute, domain.JWTTypePasswordReset)
	suite.Require().Nil(err)

	result, apiErr := suite.passwordMed.ValidatePasswordResetToken(ctx, resetToken)
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(userID, result.ID)
}

func (suite *PasswordMedTestSuite) TestValidatePasswordResetToken_RejectsAccessTokenType() {
	ctx := context.Background()
	userID := testutil.EntityIDUser

	accessToken, err := suite.jwtUtils.Encode(ctx, userID, time.Hour, domain.JWTTypeAccess)
	suite.Require().Nil(err)

	result, apiErr := suite.passwordMed.ValidatePasswordResetToken(ctx, accessToken)

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(contracts.ErrorTypeInvalidRequest, apiErr.Type)
	suite.Equal(token.ErrInvalidJWT, apiErr.PublicMessage)
}

func (suite *PasswordMedTestSuite) TestValidatePasswordResetToken_UserNotFound() {
	ctx := context.Background()
	userID := testutil.EntityIDUser

	resetToken, err := suite.jwtUtils.Encode(ctx, userID, 15*time.Minute, domain.JWTTypePasswordReset)
	suite.Require().Nil(err)

	suite.userRepo.EXPECT().
		Find(gomock.Any(), userID).
		Return(nil, contracts.NewResourceNotFoundError("User not found")).
		Times(1)

	result, apiErr := suite.passwordMed.ValidatePasswordResetToken(ctx, resetToken)

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(contracts.ErrorCodeResourceNotFound, apiErr.Code)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
