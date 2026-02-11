package service

import (
	"context"
	"errors"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type UserSvcTestSuite struct {
	suite.Suite
	userSvc               domain.UserSvc
	repoFactory           *factorymock.MockRepoFactory
	userMed               *mediatormock.MockUserMed
	passwordMed           *mediatormock.MockPasswordMed
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	idempotencyMed        *mediatormock.MockIdempotencyMed
	idempotencyKeyRepo    *repositorymock.MockIdempotencyKeyRepo
	ctrl                  *gomock.Controller
	notificationPublisher *publishermock.MockNotificationPublisher
}

func (suite *UserSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewRefreshTokenRepo().Return(nil).AnyTimes()
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()

	suite.idempotencyKeyRepo = repositorymock.NewMockIdempotencyKeyRepo(suite.ctrl)
	suite.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(suite.idempotencyKeyRepo).AnyTimes()

	suite.userMed = mediatormock.NewMockUserMed(suite.ctrl)
	suite.passwordMed = mediatormock.NewMockPasswordMed(suite.ctrl)
	suite.refreshTokenMed = mediatormock.NewMockRefreshTokenMed(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.notificationPublisher = publishermock.NewMockNotificationPublisher(suite.ctrl)

	userSvcConfig := UserSvcConfig{
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
	suite.userSvc = NewUserSvc(userSvcConfig)
}

func (suite *UserSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestUserSvcTestSuite(t *testing.T) {
	suite.Run(t, new(UserSvcTestSuite))
}

func (suite *UserSvcTestSuite) TestLogin_Success() {
	responseMeta := &appctx.IdempotencyResponseMetadata{}
	ctx := appctx.WithIdempotencyKey(context.Background(), "test-idempotency-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx,responseMeta)
	identifier := "user@example.com"
	password := "password"
	user := &types.User{ID: testutil.EntityIDUser}
	accessToken := "access-token"
	refreshToken := "refresh-token"
	refreshTokenModel := &domain.RefreshToken{Token: refreshToken}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		Validate(gomock.Any(), identifier, password).
		Return(user, nil).
		Times(1)
	suite.userMed.EXPECT().
		GenAuthAccessToken(gomock.Any(), user.ID).
		Return(accessToken, nil).
		Times(1)
	suite.refreshTokenMed.EXPECT().
		Create(gomock.Any(), user.ID, nil).
		Return(refreshTokenModel, nil).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		Return(nil).
		Times(1)

	result, apiErr := suite.userSvc.Login(ctx, identifier, password)

	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(user, result.User)
	suite.Equal(accessToken, result.AccessToken)
	suite.Equal(refreshToken, result.RefreshToken)
	suite.False(responseMeta.Replayed)
}

func (suite *UserSvcTestSuite) TestLogin_ValidateFails() {
	ctx := appctx.WithIdempotencyKey(context.Background(), "test-idempotency-key")
	identifier := "user@example.com"
	password := "wrong-password"
	expectedErr := apierror.NewAuthenticationError("invalid credentials")
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)
	suite.passwordMed.EXPECT().
		Validate(gomock.Any(), identifier, password).
		Return(nil, expectedErr).
		Times(1)
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, expectedErr).
		Return(expectedErr).
		Times(1)

	result, apiErr := suite.userSvc.Login(ctx, identifier, password)

	suite.NotNil(apiErr)
	suite.Equal(expectedErr, apiErr)
	suite.Nil(result)
}

func (suite *UserSvcTestSuite) TestRegister_Success() {
	ctx := appctx.WithIdempotencyKey(context.Background(), "test-idempotency-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx,&appctx.IdempotencyResponseMetadata{})
	input := domain.RegisterInput{
		Name:     "Test User",
		Email:    "user@example.com",
		Password: "password",
	}
	user := &types.User{ID: testutil.EntityIDUser}
	accessToken := "access-token"
	refreshToken := "refresh-token"
	refreshTokenModel := &domain.RefreshToken{Token: refreshToken}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(idempotencyKey, nil).
		Times(1)
	suite.userMed.EXPECT().
		Register(gomock.Any(), gomock.AssignableToTypeOf(domain.RegisterUserInput{})).
		Return(user, nil).
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
		CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).
		Return(nil).
		Times(1)

	result, apiErr := suite.userSvc.Register(ctx, input)

	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal(user, result.User)
	suite.Equal(accessToken, result.AccessToken)
	suite.Equal(refreshToken, result.RefreshToken)
}

type trackingTxManager struct {
	repoFactory     domain.RepoFactory
	BeginCalled     bool
	CommitCalled    bool
	RollbackCalled  bool
	ForceCommitFail bool
}

func (m *trackingTxManager) WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	m.BeginCalled = true
	apiErr := fn(ctx, m.repoFactory)
	if apiErr != nil {
		m.RollbackCalled = true
		return apiErr
	}
	if m.ForceCommitFail {
		m.RollbackCalled = true
		return apierror.NewInternalError(errors.New("commit failed"), "failed to commit transaction")
	}
	m.CommitCalled = true
	return nil
}

func (m *trackingTxManager) Reset() {
	m.BeginCalled = false
	m.CommitCalled = false
	m.RollbackCalled = false
	m.ForceCommitFail = false
}

func TestLogin_Success_RefreshTokenAndResponseInSameTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewRefreshTokenRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()

	idempotencyKeyRepo := repositorymock.NewMockIdempotencyKeyRepo(ctrl)
	repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(idempotencyKeyRepo).AnyTimes()

	userMed := mediatormock.NewMockUserMed(ctrl)
	passwordMed := mediatormock.NewMockPasswordMed(ctrl)
	refreshTokenMed := mediatormock.NewMockRefreshTokenMed(ctrl)
	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)

	txManager := &trackingTxManager{repoFactory: repoFactory}

	userSvcConfig := UserSvcConfig{
		Repos:     repoFactory,
		TxManager: txManager,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         userMed,
			Password:     passwordMed,
			RefreshToken: refreshTokenMed,
			Idempotency:  idempotencyMed,
		}},
	}
	userSvc := NewUserSvc(userSvcConfig)

	ctx := appctx.WithIdempotencyKey(context.Background(), "test-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx,&appctx.IdempotencyResponseMetadata{})

	user := &types.User{ID: testutil.EntityIDUser}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(idempotencyKey, nil)
	passwordMed.EXPECT().Validate(gomock.Any(), "user@example.com", "password").Return(user, nil)
	userMed.EXPECT().GenAuthAccessToken(gomock.Any(), user.ID).Return("access-token", nil)
	refreshTokenMed.EXPECT().Create(gomock.Any(), user.ID, nil).Return(&domain.RefreshToken{Token: "refresh-token"}, nil)
	idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).Return(nil)

	result, apiErr := userSvc.Login(ctx, "user@example.com", "password")

	assert.Nil(t, apiErr)
	assert.NotNil(t, result)
	assert.True(t, txManager.BeginCalled, "transaction should begin")
	assert.True(t, txManager.CommitCalled, "transaction should commit on success")
	assert.False(t, txManager.RollbackCalled, "transaction should not rollback on success")
}

func TestLogin_Success_ResponseFailsCausesRollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewRefreshTokenRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()

	idempotencyKeyRepo := repositorymock.NewMockIdempotencyKeyRepo(ctrl)
	repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(idempotencyKeyRepo).AnyTimes()

	userMed := mediatormock.NewMockUserMed(ctrl)
	passwordMed := mediatormock.NewMockPasswordMed(ctrl)
	refreshTokenMed := mediatormock.NewMockRefreshTokenMed(ctrl)
	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)

	txManager := &trackingTxManager{repoFactory: repoFactory}

	userSvcConfig := UserSvcConfig{
		Repos:     repoFactory,
		TxManager: txManager,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         userMed,
			Password:     passwordMed,
			RefreshToken: refreshTokenMed,
			Idempotency:  idempotencyMed,
		}},
	}
	userSvc := NewUserSvc(userSvcConfig)

	ctx := appctx.WithIdempotencyKey(context.Background(), "test-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx,&appctx.IdempotencyResponseMetadata{})

	user := &types.User{ID: testutil.EntityIDUser}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(idempotencyKey, nil)
	passwordMed.EXPECT().Validate(gomock.Any(), "user@example.com", "password").Return(user, nil)
	cacheErr := apierror.NewInternalError(errors.New("db error"), "failed to set response")
	userMed.EXPECT().GenAuthAccessToken(gomock.Any(), user.ID).Return("access-token", nil)
	refreshTokenMed.EXPECT().Create(gomock.Any(), user.ID, nil).Return(&domain.RefreshToken{Token: "refresh-token"}, nil)
	idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), idempotencyKey.TypeID, gomock.Any()).Return(cacheErr)
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, cacheErr).Return(cacheErr)

	result, apiErr := userSvc.Login(ctx, "user@example.com", "password")

	assert.NotNil(t, apiErr)
	assert.Nil(t, result)
	assert.True(t, txManager.BeginCalled, "transaction should begin")
	assert.False(t, txManager.CommitCalled, "transaction should not commit when CacheSuccessResponse fails")
	assert.True(t, txManager.RollbackCalled, "transaction should rollback when CacheSuccessResponse fails inside tx")
}

func TestLogin_RefreshTokenFails_NonTransientErrorCached(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewRefreshTokenRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()

	idempotencyKeyRepo := repositorymock.NewMockIdempotencyKeyRepo(ctrl)
	repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(idempotencyKeyRepo).AnyTimes()

	userMed := mediatormock.NewMockUserMed(ctrl)
	passwordMed := mediatormock.NewMockPasswordMed(ctrl)
	refreshTokenMed := mediatormock.NewMockRefreshTokenMed(ctrl)
	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)

	txManager := &trackingTxManager{repoFactory: repoFactory}

	userSvcConfig := UserSvcConfig{
		Repos:     repoFactory,
		TxManager: txManager,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         userMed,
			Password:     passwordMed,
			RefreshToken: refreshTokenMed,
			Idempotency:  idempotencyMed,
		}},
	}
	userSvc := NewUserSvc(userSvcConfig)

	ctx := appctx.WithIdempotencyKey(context.Background(), "test-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx,&appctx.IdempotencyResponseMetadata{})

	user := &types.User{ID: testutil.EntityIDUser}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	nonTransientError := apierror.NewInternalError(errors.New("db error"), "failed to create refresh token")
	nonTransientError.IsTransient = false

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(idempotencyKey, nil)
	passwordMed.EXPECT().Validate(gomock.Any(), "user@example.com", "password").Return(user, nil)
	userMed.EXPECT().GenAuthAccessToken(gomock.Any(), user.ID).Return("access-token", nil)
	refreshTokenMed.EXPECT().Create(gomock.Any(), user.ID, nil).Return(nil, nonTransientError)
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, nonTransientError).Return(nonTransientError)

	result, apiErr := userSvc.Login(ctx, "user@example.com", "password")

	assert.NotNil(t, apiErr)
	assert.Equal(t, nonTransientError, apiErr)
	assert.Nil(t, result)
	assert.True(t, txManager.RollbackCalled, "transaction should rollback on error")
}

func TestLogin_RefreshTokenFails_TransientErrorNotCached(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewRefreshTokenRepo().Return(nil).AnyTimes()
	repoFactory.EXPECT().NewAPIKeyRepo().Return(nil).AnyTimes()

	idempotencyKeyRepo := repositorymock.NewMockIdempotencyKeyRepo(ctrl)
	repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(idempotencyKeyRepo).AnyTimes()

	userMed := mediatormock.NewMockUserMed(ctrl)
	passwordMed := mediatormock.NewMockPasswordMed(ctrl)
	refreshTokenMed := mediatormock.NewMockRefreshTokenMed(ctrl)
	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)

	txManager := &trackingTxManager{repoFactory: repoFactory}

	userSvcConfig := UserSvcConfig{
		Repos:     repoFactory,
		TxManager: txManager,
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         userMed,
			Password:     passwordMed,
			RefreshToken: refreshTokenMed,
			Idempotency:  idempotencyMed,
		}},
	}
	userSvc := NewUserSvc(userSvcConfig)

	ctx := appctx.WithIdempotencyKey(context.Background(), "test-key")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx,&appctx.IdempotencyResponseMetadata{})

	user := &types.User{ID: testutil.EntityIDUser}
	idempotencyKey := &domain.IdempotencyKey{
		TypeID:        "idk_test",
		RecoveryPoint: domain.RecoveryPointStarted,
	}

	transientError := apierror.NewInternalError(errors.New("connection timeout"), "transient db error")
	transientError.IsTransient = true

	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(idempotencyKey, nil)
	passwordMed.EXPECT().Validate(gomock.Any(), "user@example.com", "password").Return(user, nil)
	userMed.EXPECT().GenAuthAccessToken(gomock.Any(), user.ID).Return("access-token", nil)
	refreshTokenMed.EXPECT().Create(gomock.Any(), user.ID, nil).Return(nil, transientError)
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), idempotencyKey.TypeID, transientError).Return(transientError)

	result, apiErr := userSvc.Login(ctx, "user@example.com", "password")

	assert.NotNil(t, apiErr)
	assert.Equal(t, transientError, apiErr)
	assert.Nil(t, result)
	assert.True(t, txManager.RollbackCalled, "transaction should rollback on error")
}
