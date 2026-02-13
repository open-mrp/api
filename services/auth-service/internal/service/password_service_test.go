package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	publishermock "github.com/augno/api/services/auth-service/internal/domain/mock/publisher"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PasswordSvcTestSuite struct {
	suite.Suite
	passwordSvc           domain.PasswordSvc
	repoFactory           *factorymock.MockRepoFactory
	passwordMed           *mediatormock.MockPasswordMed
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	userMed               *mediatormock.MockUserMed
	idempotencyMed        *mediatormock.MockIdempotencyMed
	ctrl                  *gomock.Controller
	notificationPublisher *publishermock.MockNotificationPublisher
}

func (suite *PasswordSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewRefreshTokenRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(nil).AnyTimes()

	suite.passwordMed = mediatormock.NewMockPasswordMed(suite.ctrl)
	suite.refreshTokenMed = mediatormock.NewMockRefreshTokenMed(suite.ctrl)
	suite.userMed = mediatormock.NewMockUserMed(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.notificationPublisher = publishermock.NewMockNotificationPublisher(suite.ctrl)

	passwordSvcConfig := &PasswordSvcConfig{
		Repos: suite.repoFactory,
		TxManager: stubTxManager{
			repoFactory: suite.repoFactory,
		},
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         suite.userMed,
			APIKey:       nil,
			Password:     suite.passwordMed,
			RefreshToken: suite.refreshTokenMed,
			Idempotency:  suite.idempotencyMed,
		}},
		NotificationPublisher: suite.notificationPublisher,
	}
	suite.passwordSvc = NewPasswordSvc(passwordSvcConfig)
}

func (suite *PasswordSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestPasswordSvcTestSuite(t *testing.T) {
	suite.Run(t, new(PasswordSvcTestSuite))
}

func (suite *PasswordSvcTestSuite) TestRequestPasswordReset_Success() {
	ctx := context.Background()
	identifier := "user@example.com"
	var accountSlug *string
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_123",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), nil).
		Return(idempotencyKey, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		RequestReset(gomock.Any(), identifier, accountSlug).
		Return(nil).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, struct{}{}).
		Return(nil).
		Times(1)

	apiErr := suite.passwordSvc.RequestPasswordReset(ctx, identifier, accountSlug)

	suite.Nil(apiErr)
}

func (suite *PasswordSvcTestSuite) TestResetPassword_Success() {
	ctx := context.Background()
	token := "reset-token"
	newPassword := "new-password"
	user := &types.User{ID: testutil.EntityIDUser}
	accessToken := "access-token"
	refreshToken := "refresh-token"
	refreshTokenModel := &domain.RefreshToken{Token: refreshToken}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_123",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), nil).
		Return(idempotencyKey, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		ValidatePasswordResetToken(gomock.Any(), token).
		Return(user, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		Update(gomock.Any(), user, newPassword).
		Return(nil).
		Times(1)
	suite.refreshTokenMed.EXPECT().
		Create(gomock.Any(), user.ID, nil).
		Return(refreshTokenModel, nil).
		Times(1)
	suite.userMed.EXPECT().
		GenAuthAccessToken(gomock.Any(), user.ID).
		Return(accessToken, nil).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, domain.LoginResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}).
		Return(nil).
		Times(1)

	result, apiErr := suite.passwordSvc.ResetPassword(ctx, token, newPassword)

	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(user, result.User)
	suite.Equal(accessToken, result.AccessToken)
	suite.Equal(refreshToken, result.RefreshToken)
}

func (suite *PasswordSvcTestSuite) TestUpdatePassword_Success() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	oldPassword := "old-password"
	newPassword := "new-password"
	user := &types.User{ID: userID}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_123",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), nil).
		Return(idempotencyKey, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		Validate(gomock.Any(), userID, oldPassword).
		Return(user, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		Update(gomock.Any(), user, newPassword).
		Return(nil).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, struct{}{}).
		Return(nil).
		Times(1)

	apiErr := suite.passwordSvc.UpdatePassword(ctx, userID, oldPassword, newPassword)

	suite.Nil(apiErr)
}

func (suite *PasswordSvcTestSuite) TestUpdatePassword_ValidateFails() {
	ctx := context.Background()
	userID := testutil.EntityIDUser
	oldPassword := "wrong-password"
	newPassword := "new-password"
	expectedErr := apierror.NewAuthenticationError("invalid password")
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_123",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), nil).
		Return(idempotencyKey, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		Validate(gomock.Any(), userID, oldPassword).
		Return(nil, expectedErr).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, expectedErr).
		Return(expectedErr).
		Times(1)

	apiErr := suite.passwordSvc.UpdatePassword(ctx, userID, oldPassword, newPassword)

	suite.NotNil(apiErr)
	suite.Equal(expectedErr, apiErr)
}
