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
	"github.com/augno/api/services/auth-service/internal/password"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// PasswordMedTestSuite provides a test suite for PasswordMed tests
type PasswordMedTestSuite struct {
	suite.Suite
	passwordMed           domain.PasswordMed
	userRepo              *repositorymock.MockUserRepo
	repoFactory           *factorymock.MockRepoFactory
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	notificationPublisher *publishermock.MockNotificationPublisher
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

	passwordMedConfig := &PasswordMedConfig{
		Repos:                 suite.repoFactory,
		RefreshTokenMed:       suite.refreshTokenMed,
		JWTSecret:             testutil.JWTSecret,
		NotificationPublisher: suite.notificationPublisher,
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
	t.Parallel()
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
		Return(apierror.NewInternalError(nil, "Database error")).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
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
		Return(apierror.NewInternalError(nil, "Failed to revoke tokens")).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInternalError, apiErr.Code)
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
			gomock.Any(),
		).
		DoAndReturn(func(ctx context.Context, data messaging.EmailSendData) *apierror.APIError {
			suite.Equal([]string{notifyEmail}, data.To)
			suite.Equal("Password Updated", data.Subject)
			suite.Equal(constants.EmailTemplatePasswordUpdated, data.TemplateID)
			suite.Nil(data.SendAs)
			suite.Nil(data.AccountID)
			suite.Equal(&userID, data.SentByID)
			return nil
		}).
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
			gomock.Any(),
		).
		DoAndReturn(func(ctx context.Context, data messaging.EmailSendData) *apierror.APIError {
			suite.Equal([]string{notifyEmail}, data.To)
			suite.Equal("Password Updated", data.Subject)
			suite.Equal(constants.EmailTemplatePasswordUpdated, data.TemplateID)
			suite.Nil(data.SendAs)
			suite.Nil(data.AccountID)
			suite.Equal(&userID, data.SentByID)
			return nil
		}).
		Times(1)

	newPassword := "newPassword456"
	apiErr := suite.passwordMed.Update(ctx, user, newPassword)

	suite.Nil(apiErr)
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

	resetToken, err := token.EncodeJWT(ctx, testutil.JWTSecret, userID, 15*time.Minute, token.JWTTypePasswordReset)
	suite.Require().Nil(err)

	result, apiErr := suite.passwordMed.ValidatePasswordResetToken(ctx, resetToken)
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(userID, result.ID)
}

func (suite *PasswordMedTestSuite) TestValidatePasswordResetToken_RejectsAccessTokenType() {
	ctx := context.Background()
	userID := testutil.EntityIDUser

	accessToken, err := token.EncodeJWT(ctx, testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	suite.Require().Nil(err)

	result, apiErr := suite.passwordMed.ValidatePasswordResetToken(ctx, accessToken)

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorTypeInvalidRequest, apiErr.Type)
	suite.Equal(token.ErrInvalidJWT, apiErr.PublicMessage)
}

func (suite *PasswordMedTestSuite) TestValidatePasswordResetToken_UserNotFound() {
	ctx := context.Background()
	userID := testutil.EntityIDUser

	resetToken, err := token.EncodeJWT(ctx, testutil.JWTSecret, userID, 15*time.Minute, token.JWTTypePasswordReset)
	suite.Require().Nil(err)

	suite.userRepo.EXPECT().
		Find(gomock.Any(), userID).
		Return(nil, apierror.NewResourceNotFoundError("User not found")).
		Times(1)

	result, apiErr := suite.passwordMed.ValidatePasswordResetToken(ctx, resetToken)

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, apiErr.Code)
}

func (suite *PasswordMedTestSuite) TestValidate_Success() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	plainPassword := "correct-horse-battery-staple"
	hashedPassword, err := password.HashPassword(ctx, plainPassword)
	suite.Require().Nil(err)

	identifier := "user@example.com"
	user := &types.User{
		ID:             userID,
		Email:          stringPtr(identifier),
		HashedPassword: &hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		Find(gomock.Any(), identifier).
		Return(user, nil).
		Times(1)

	result, apiErr := suite.passwordMed.Validate(ctx, identifier, plainPassword)

	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(userID, result.ID)
}

func (suite *PasswordMedTestSuite) TestValidate_UserNotFound() {
	ctx := context.Background()
	identifier := "missing@example.com"

	suite.userRepo.EXPECT().
		Find(gomock.Any(), identifier).
		Return(nil, apierror.NewResourceNotFoundError("User not found")).
		Times(1)

	result, apiErr := suite.passwordMed.Validate(ctx, identifier, "anything")

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, apiErr.Code)
}

func (suite *PasswordMedTestSuite) TestValidate_NoHashedPassword_WithEmail_SendsResetEmail() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	identifier := "user@example.com"

	user := &types.User{
		ID:             userID,
		Email:          stringPtr(identifier),
		HashedPassword: nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		Find(gomock.Any(), identifier).
		Return(user, nil).
		Times(1)

	suite.notificationPublisher.EXPECT().
		PublishSendEmail(
			gomock.Any(),
			gomock.Any(),
		).
		DoAndReturn(func(ctx context.Context, data messaging.EmailSendData) *apierror.APIError {
			suite.Equal([]string{identifier}, data.To)
			suite.Equal("Password Reset Request", data.Subject)
			suite.Equal(constants.EmailTemplatePasswordReset, data.TemplateID)
			suite.Equal(&userID, data.SentByID)
			suite.NotNil(data.Params["ResetLink"])
			return nil
		}).
		Times(1)

	result, apiErr := suite.passwordMed.Validate(ctx, identifier, "anything")

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorTypeInvalidRequest, apiErr.Type)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *PasswordMedTestSuite) TestValidate_NoHashedPassword_WithoutEmail_NoEmailSent() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	identifier := userID

	user := &types.User{
		ID:             userID,
		Email:          nil,
		HashedPassword: nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		Find(gomock.Any(), identifier).
		Return(user, nil).
		Times(1)

	// No PublishSendEmail expectation: strict controller will fail if called.

	result, apiErr := suite.passwordMed.Validate(ctx, identifier, "anything")

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorTypeInvalidRequest, apiErr.Type)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *PasswordMedTestSuite) TestValidate_PasswordMismatch() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	hashedPassword, err := password.HashPassword(ctx, "correct-password")
	suite.Require().Nil(err)

	identifier := "user@example.com"
	user := &types.User{
		ID:             userID,
		Email:          stringPtr(identifier),
		HashedPassword: &hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.userRepo.EXPECT().
		Find(gomock.Any(), identifier).
		Return(user, nil).
		Times(1)

	result, apiErr := suite.passwordMed.Validate(ctx, identifier, "wrong-password")

	suite.Nil(result)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, apiErr.Code)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
