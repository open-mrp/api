package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	publishermock "github.com/augno/api/services/auth-service/internal/domain/mock/publisher"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func tokenSvcIdentityCtx() context.Context {
	acctID := "acct_test"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: acctID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           testutil.EntityIDUser,
			AccountID:    &acctID,
		},
	})
}

type TokenSvcTestSuite struct {
	suite.Suite
	tokenSvc              domain.TokenSvc
	repoFactory           *factorymock.MockRepoFactory
	refreshTokenRepo      *repositorymock.MockRefreshTokenRepo
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	userMed               *mediatormock.MockUserMed
	idempotencyMed        *mediatormock.MockIdempotencyMed
	ctrl                  *gomock.Controller
	notificationPublisher *publishermock.MockNotificationPublisher
}

func (suite *TokenSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.refreshTokenRepo = repositorymock.NewMockRefreshTokenRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewRefreshTokenRepo().Return(suite.refreshTokenRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.refreshTokenMed = mediatormock.NewMockRefreshTokenMed(suite.ctrl)
	suite.userMed = mediatormock.NewMockUserMed(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.notificationPublisher = publishermock.NewMockNotificationPublisher(suite.ctrl)

	tokenSvcConfig := &TokenSvcConfig{
		Repos: suite.repoFactory,
		TxManager: stubTxManager{
			repoFactory: suite.repoFactory,
		},
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         suite.userMed,
			APIKey:       nil,
			Password:     nil,
			RefreshToken: suite.refreshTokenMed,
			Idempotency:  suite.idempotencyMed,
		}},
		NotificationPublisher: suite.notificationPublisher,
	}
	suite.tokenSvc = NewTokenSvc(tokenSvcConfig)
}

func (suite *TokenSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestTokenSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TokenSvcTestSuite))
}

func (suite *TokenSvcTestSuite) TestRefreshToken_Success() {
	ctx := context.Background()
	refreshToken := "refresh-token"
	userID := testutil.EntityIDUser
	accessToken := "access-token"

	suite.refreshTokenMed.EXPECT().
		Validate(gomock.Any(), refreshToken).
		Return(userID, nil).
		Times(1)
	suite.userMed.EXPECT().
		GenAuthAccessToken(gomock.Any(), userID).
		Return(accessToken, nil).
		Times(1)

	result, apiErr := suite.tokenSvc.RefreshToken(ctx, refreshToken)

	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(accessToken, result.AccessToken)
}

func (suite *TokenSvcTestSuite) TestRefreshToken_ValidateFails() {
	ctx := context.Background()
	refreshToken := "invalid-refresh"
	expectedErr := apierror.NewAuthenticationError("invalid refresh token")

	suite.refreshTokenMed.EXPECT().
		Validate(gomock.Any(), refreshToken).
		Return("", expectedErr).
		Times(1)

	result, apiErr := suite.tokenSvc.RefreshToken(ctx, refreshToken)

	suite.NotNil(apiErr)
	suite.Equal(expectedErr, apiErr)
	suite.Nil(result)
}

func (suite *TokenSvcTestSuite) TestRevokeRefreshToken_Success() {
	ctx := tokenSvcIdentityCtx()
	refreshToken := "refresh-token"
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_123",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), nil).
		Return(idempotencyKey, nil).
		Times(1)
	suite.refreshTokenRepo.EXPECT().
		Find(gomock.Any(), refreshToken).
		Return(&domain.RefreshToken{Token: refreshToken, UserID: testutil.EntityIDUser}, nil).
		Times(2)
	suite.refreshTokenMed.EXPECT().
		Revoke(gomock.Any(), refreshToken).
		Return(nil).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, struct{}{}).
		Return(nil).
		Times(1)

	apiErr := suite.tokenSvc.RevokeRefreshToken(ctx, refreshToken)

	suite.Nil(apiErr)
}

func (suite *TokenSvcTestSuite) TestRevokeRefreshToken_RevokeFails() {
	ctx := tokenSvcIdentityCtx()
	refreshToken := "refresh-token"
	expectedErr := apierror.NewAuthenticationError("token not found")
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_123",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), nil).
		Return(idempotencyKey, nil).
		Times(1)
	suite.refreshTokenRepo.EXPECT().
		Find(gomock.Any(), refreshToken).
		Return(&domain.RefreshToken{Token: refreshToken, UserID: testutil.EntityIDUser}, nil).
		Times(1)
	suite.refreshTokenMed.EXPECT().
		Revoke(gomock.Any(), refreshToken).
		Return(expectedErr).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, expectedErr).
		Return(expectedErr).
		Times(1)

	apiErr := suite.tokenSvc.RevokeRefreshToken(ctx, refreshToken)

	suite.NotNil(apiErr)
	suite.Equal(expectedErr, apiErr)
}
