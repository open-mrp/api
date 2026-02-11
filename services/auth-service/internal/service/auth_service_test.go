package service

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	publishermock "github.com/augno/api/services/auth-service/internal/domain/mock/publisher"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/ptrutil"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type staticMediatorFactory struct {
	mediators domain.Mediators
}

func (f staticMediatorFactory) Build(domain.RepoFactory) domain.Mediators {
	return f.mediators
}

type stubTxManager struct {
	repoFactory domain.RepoFactory
}

func (m stubTxManager) WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.repoFactory)
}

type AuthSvcTestSuite struct {
	suite.Suite
	authSvc               domain.AuthSvc
	repoFactory           *factorymock.MockRepoFactory
	userRepo              *repositorymock.MockUserRepo
	apiKeyRepo            *repositorymock.MockAPIKeyRepo
	refreshTokenRepo      *repositorymock.MockRefreshTokenRepo
	idempotencyKeyRepo    *repositorymock.MockIdempotencyKeyRepo
	userMed               *mediatormock.MockUserMed
	apiKeyMed             *mediatormock.MockAPIKeyMed
	passwordMed           *mediatormock.MockPasswordMed
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	idempotencyMed        *mediatormock.MockIdempotencyMed
	jwtUtils              domain.JWTUtils
	ctrl                  *gomock.Controller
	notificationPublisher *publishermock.MockNotificationPublisher
}

func (suite *AuthSvcTestSuite) SetupSuite() {
	suite.jwtUtils = token.NewJWTUtils(&token.JWTConfig{Secret: testutil.JWTSecret})

	suite.ctrl = gomock.NewController(suite.T())
	suite.userRepo = repositorymock.NewMockUserRepo(suite.ctrl)
	suite.refreshTokenRepo = repositorymock.NewMockRefreshTokenRepo(suite.ctrl)
	suite.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(suite.ctrl)
	suite.idempotencyKeyRepo = repositorymock.NewMockIdempotencyKeyRepo(suite.ctrl)

	suite.userMed = mediatormock.NewMockUserMed(suite.ctrl)
	suite.apiKeyMed = mediatormock.NewMockAPIKeyMed(suite.ctrl)
	suite.passwordMed = mediatormock.NewMockPasswordMed(suite.ctrl)
	suite.refreshTokenMed = mediatormock.NewMockRefreshTokenMed(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUserRepo().Return(suite.userRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewRefreshTokenRepo().Return(suite.refreshTokenRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(suite.apiKeyRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(suite.idempotencyKeyRepo).AnyTimes()

	suite.notificationPublisher = publishermock.NewMockNotificationPublisher(suite.ctrl)

	authSvcConfig := AuthSvcConfig{
		Repos: suite.repoFactory,
		TxManager: stubTxManager{
			repoFactory: suite.repoFactory,
		},
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         suite.userMed,
			APIKey:       suite.apiKeyMed,
			Password:     suite.passwordMed,
			RefreshToken: suite.refreshTokenMed,
			Idempotency:  suite.idempotencyMed,
		}},
		NotificationPublisher: suite.notificationPublisher,
	}
	suite.authSvc = NewAuthSvc(authSvcConfig)
}

func (suite *AuthSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestAuthSvcTestSuite(t *testing.T) {
	suite.Run(t, new(AuthSvcTestSuite))
}

func (suite *AuthSvcTestSuite) TestValidateCredential_NoAuthHeader() {
	ctx := context.Background()

	expectedIdentity := types.GetUnauthenticatedIdentity(nil)
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), "", (*string)(nil)).
		Return(expectedIdentity, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, "", nil)

	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeUnauthenticated, identity.Type)
	suite.Nil(identity.TargetAccountID)
	suite.Nil(identity.Actor)
	suite.Equal(constants.AccountModeProduction, identity.AccountMode)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_APIKey_DefaultsToOwnerAccount() {
	ctx := context.Background()

	permissions := map[string]bool{
		"orders:read":  true,
		"orders:write": true,
	}

	// When no target account is specified, the API key defaults to its owner account
	identityResponse := &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: ptrutil.Ptr(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           testutil.EntityIDAPIKeyValidProdMode,
			Name:         ptrutil.Ptr("Test API Key"),
			AccountID:    ptrutil.Ptr(testutil.EntityIDAccount),
			RoleID:       ptrutil.Ptr(testutil.EntityIDRole),
			RoleTypeCode: ptrutil.Ptr("admin"),
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, (*string)(nil)).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, nil)
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeAPIKey, identity.Type)
	suite.NotNil(identity.TargetAccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeInternal, identity.Actor.Type)
	suite.Equal(testutil.EntityIDAPIKeyValidProdMode, identity.Actor.ID)
	suite.NotNil(identity.Actor.AccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.Actor.AccountID)
	suite.NotNil(identity.Actor.RoleID)
	suite.Equal(testutil.EntityIDRole, *identity.Actor.RoleID)
	suite.Equal(permissions, identity.Actor.Permissions)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_APIKey_Internal() {
	ctx := context.Background()

	permissions := map[string]bool{
		"orders:read":  true,
		"orders:write": true,
	}

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: ptrutil.Ptr(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           testutil.EntityIDAPIKeyValidProdMode,
			Name:         ptrutil.Ptr("Test API Key"),
			AccountID:    ptrutil.Ptr(testutil.EntityIDAccount),
			RoleID:       ptrutil.Ptr(testutil.EntityIDRole),
			RoleTypeCode: ptrutil.Ptr("admin"),
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeAPIKey, identity.Type)
	suite.NotNil(identity.TargetAccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeInternal, identity.Actor.Type)
	suite.Equal(testutil.EntityIDAPIKeyValidProdMode, identity.Actor.ID)
	suite.NotNil(identity.Actor.AccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.Actor.AccountID)
	suite.NotNil(identity.Actor.RoleID)
	suite.Equal(testutil.EntityIDRole, *identity.Actor.RoleID)
	suite.Equal(permissions, identity.Actor.Permissions)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_APIKey_Customer() {
	ctx := context.Background()

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: ptrutil.Ptr(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeCustomer,
			ID:           testutil.EntityIDAPIKeyValidProdMode,
			Name:         ptrutil.Ptr("Test API Key"),
			AccountID:    ptrutil.Ptr("acc_customer123"),
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeAPIKey, identity.Type)
	suite.NotNil(identity.TargetAccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeCustomer, identity.Actor.Type)
	suite.Equal(testutil.EntityIDAPIKeyValidProdMode, identity.Actor.ID)
	suite.NotNil(identity.Actor.AccountID)
	suite.Equal("acc_customer123", *identity.Actor.AccountID)
	suite.Nil(identity.Actor.RoleID)
	suite.Equal(map[string]bool{}, identity.Actor.Permissions)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_JWT_Unassigned() {
	ctx := context.Background()

	userID := testutil.EntityIDUser
	token, err := suite.jwtUtils.Encode(context.Background(), userID, time.Hour, domain.JWTTypeAccess)
	suite.Nil(err)

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: nil,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeUnassigned,
			ID:           userID,
			Name:         ptrutil.Ptr("Test User"),
			AccountID:    nil,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, (*string)(nil)).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, nil)
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeUser, identity.Type)
	suite.Nil(identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeUnassigned, identity.Actor.Type)
	suite.Equal(userID, identity.Actor.ID)
	suite.Nil(identity.Actor.AccountID)
	suite.Nil(identity.Actor.RoleID)
	suite.Equal(map[string]bool{}, identity.Actor.Permissions)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_JWT_Internal() {
	ctx := context.Background()

	userID := testutil.EntityIDUser
	token, err := suite.jwtUtils.Encode(context.Background(), userID, time.Hour, domain.JWTTypeAccess)
	suite.Nil(err)

	permissions := map[string]bool{
		"orders:read":  true,
		"orders:write": true,
	}

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: ptrutil.Ptr(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           userID,
			Name:         ptrutil.Ptr("Test User"),
			AccountID:    ptrutil.Ptr(testutil.EntityIDAccount),
			RoleID:       ptrutil.Ptr(testutil.EntityIDRole),
			RoleTypeCode: ptrutil.Ptr("admin"),
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeUser, identity.Type)
	suite.NotNil(identity.TargetAccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeInternal, identity.Actor.Type)
	suite.Equal(userID, identity.Actor.ID)
	suite.NotNil(identity.Actor.AccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.Actor.AccountID)
	suite.NotNil(identity.Actor.RoleID)
	suite.Equal(testutil.EntityIDRole, *identity.Actor.RoleID)
	suite.Equal(permissions, identity.Actor.Permissions)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_JWT_Customer() {
	ctx := context.Background()

	userID := testutil.EntityIDUser
	token, err := suite.jwtUtils.Encode(context.Background(), userID, time.Hour, domain.JWTTypeAccess)
	suite.Nil(err)

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: ptrutil.Ptr(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeCustomer,
			ID:           userID,
			Name:         ptrutil.Ptr("Test User"),
			AccountID:    ptrutil.Ptr("acc_customer123"),
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeUser, identity.Type)
	suite.NotNil(identity.TargetAccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeCustomer, identity.Actor.Type)
	suite.Equal(userID, identity.Actor.ID)
	suite.NotNil(identity.Actor.AccountID)
	suite.Equal("acc_customer123", *identity.Actor.AccountID)
	suite.Nil(identity.Actor.RoleID)
	suite.Equal(map[string]bool{}, identity.Actor.Permissions)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_JWT_Expired() {
	userID := testutil.EntityIDUser
	token, err := suite.jwtUtils.Encode(context.Background(), userID, -time.Hour, domain.JWTTypeAccess) // Expired token
	suite.Nil(err)

	ctx := context.Background()
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(nil, apierror.NewAuthenticationError("Access token has expired.")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_JWT_Invalid() {
	invalidToken := "invalid.jwt.token"
	ctx := context.Background()
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), invalidToken, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(nil, apierror.NewAuthenticationError("Invalid token")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, invalidToken, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_JWT_UserNotFound() {
	userID := "usr_notfound123"
	token, err := suite.jwtUtils.Encode(context.Background(), userID, time.Hour, domain.JWTTypeAccess)
	suite.Nil(err)

	ctx := context.Background()
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(nil, apierror.NewAuthenticationError("Invalid token")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}

func (suite *AuthSvcTestSuite) TestValidateCredential_APIKey_Invalid() {
	ctx := context.Background()

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, ptrutil.Ptr(testutil.EntityIDAccount)).
		Return(nil, apierror.NewAuthenticationError("Invalid API key")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, ptrutil.Ptr(testutil.EntityIDAccount))
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}
