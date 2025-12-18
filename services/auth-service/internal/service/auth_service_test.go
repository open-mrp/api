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
	"github.com/augno/api/shared/contracts"
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

// stubTxManager executes the callback without a real transaction.
type stubTxManager struct {
	repoFactory domain.RepoFactory
}

func (m stubTxManager) WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *contracts.APIError) *contracts.APIError {
	return fn(ctx, m.repoFactory)
}

// MiddlewareSvcTestSuite provides a test suite for MiddlewareSvc tests
type MiddlewareSvcTestSuite struct {
	suite.Suite
	authSvc               domain.AuthSvc
	repoFactory           *factorymock.MockRepoFactory
	userRepo              *repositorymock.MockUserRepo
	apiKeyRepo            *repositorymock.MockAPIKeyRepo
	accountRelationRepo   *repositorymock.MockAccountRelationRepo
	accountUserRepo       *repositorymock.MockAccountUserRepo
	rolePermissionRepo    *repositorymock.MockRolePermissionRepo
	refreshTokenRepo      *repositorymock.MockRefreshTokenRepo
	userMed               *mediatormock.MockUserMed
	apiKeyMed             *mediatormock.MockAPIKeyMed
	accountUserMed        *mediatormock.MockAccountUserMed
	passwordMed           *mediatormock.MockPasswordMed
	refreshTokenMed       *mediatormock.MockRefreshTokenMed
	jwtUtils              domain.JWTUtils
	ctrl                  *gomock.Controller
	notificationPublisher *publishermock.MockNotificationPublisher
}

// SetupSuite runs once before all tests in the suite
func (suite *MiddlewareSvcTestSuite) SetupSuite() {
	jwtConfig := token.DefaultJWTConfig(testutil.JWTSecret)
	suite.jwtUtils = token.NewJWTUtils(jwtConfig)

	suite.ctrl = gomock.NewController(suite.T())
	suite.userRepo = repositorymock.NewMockUserRepo(suite.ctrl)
	suite.accountRelationRepo = repositorymock.NewMockAccountRelationRepo(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.rolePermissionRepo = repositorymock.NewMockRolePermissionRepo(suite.ctrl)
	suite.refreshTokenRepo = repositorymock.NewMockRefreshTokenRepo(suite.ctrl)
	suite.apiKeyRepo = repositorymock.NewMockAPIKeyRepo(suite.ctrl)

	suite.userMed = mediatormock.NewMockUserMed(suite.ctrl)
	suite.apiKeyMed = mediatormock.NewMockAPIKeyMed(suite.ctrl)
	suite.accountUserMed = mediatormock.NewMockAccountUserMed(suite.ctrl)
	suite.passwordMed = mediatormock.NewMockPasswordMed(suite.ctrl)
	suite.refreshTokenMed = mediatormock.NewMockRefreshTokenMed(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUserRepo().Return(suite.userRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewRefreshTokenRepo().Return(suite.refreshTokenRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountRelationRepo().Return(suite.accountRelationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAPIKeyRepo().Return(suite.apiKeyRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewRolePermissionRepo().Return(suite.rolePermissionRepo).AnyTimes()

	suite.notificationPublisher = publishermock.NewMockNotificationPublisher(suite.ctrl)

	authSvcConfig := AuthSvcConfig{
		Repos: suite.repoFactory,
		TxManager: stubTxManager{
			repoFactory: suite.repoFactory,
		},
		MediatorFactory: staticMediatorFactory{mediators: domain.Mediators{
			User:         suite.userMed,
			APIKey:       suite.apiKeyMed,
			AccountUser:  suite.accountUserMed,
			Password:     suite.passwordMed,
			RefreshToken: suite.refreshTokenMed,
		}},
		NotificationPublisher: suite.notificationPublisher,
	}
	suite.authSvc = NewAuthSvc(authSvcConfig)
}

// TearDownSuite runs once after all tests in the suite
func (suite *MiddlewareSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

// TestMiddlewareSvcTestSuite runs the test suite
func TestMiddlewareSvcTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareSvcTestSuite))
}

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_NoAuthHeader() {
	ctx := context.Background()

	expectedIdentity := types.GetUnauthenticatedIdentity("")
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), "", "").
		Return(expectedIdentity, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, "", "")

	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeUnauthenticated, identity.Type)
	suite.Nil(identity.TargetAccountID)
	suite.Nil(identity.Actor)
	suite.Equal(constants.AccountModeProduction, identity.AccountMode)
}

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_APIKey_Unassigned() {
	ctx := context.Background()

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: nil,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeUnassigned,
			ID:           testutil.EntityIDAPIKeyValidProdMode,
			Name:         ptrutil.String("Test API Key"),
			AccountID:    ptrutil.String(testutil.EntityIDAccount),
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, "").
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, "")
	suite.Nil(err)
	suite.NotNil(identity)
	suite.Equal(types.IdentityTypeAPIKey, identity.Type)
	suite.Nil(identity.TargetAccountID)
	suite.NotNil(identity.Actor)
	suite.Equal(types.IdentityActorTypeUnassigned, identity.Actor.Type)
	suite.Equal(testutil.EntityIDAPIKeyValidProdMode, identity.Actor.ID)
	suite.NotNil(identity.Actor.AccountID)
	suite.Equal(testutil.EntityIDAccount, *identity.Actor.AccountID)
	suite.Nil(identity.Actor.RoleID)
	suite.Equal(map[string]bool{}, identity.Actor.Permissions)
}

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_APIKey_Internal() {
	ctx := context.Background()

	permissions := map[string]bool{
		"orders:read":  true,
		"orders:write": true,
	}

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: ptrutil.String(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           testutil.EntityIDAPIKeyValidProdMode,
			Name:         ptrutil.String("Test API Key"),
			AccountID:    ptrutil.String(testutil.EntityIDAccount),
			RoleID:       ptrutil.String(testutil.EntityIDRole),
			RoleTypeCode: ptrutil.String("admin"),
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, testutil.EntityIDAccount).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, testutil.EntityIDAccount)
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

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_APIKey_Customer() {
	ctx := context.Background()

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: ptrutil.String(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeCustomer,
			ID:           testutil.EntityIDAPIKeyValidProdMode,
			Name:         ptrutil.String("Test API Key"),
			AccountID:    ptrutil.String("acc_customer123"),
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, testutil.EntityIDAccount).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, testutil.EntityIDAccount)
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

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_JWT_Unassigned() {
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
			Name:         ptrutil.String("Test User"),
			AccountID:    nil,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, "").
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, "")
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

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_JWT_Internal() {
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
		TargetAccountID: ptrutil.String(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           userID,
			Name:         ptrutil.String("Test User"),
			AccountID:    ptrutil.String(testutil.EntityIDAccount),
			RoleID:       ptrutil.String(testutil.EntityIDRole),
			RoleTypeCode: ptrutil.String("admin"),
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, testutil.EntityIDAccount).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, testutil.EntityIDAccount)
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

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_JWT_Customer() {
	ctx := context.Background()

	userID := testutil.EntityIDUser
	token, err := suite.jwtUtils.Encode(context.Background(), userID, time.Hour, domain.JWTTypeAccess)
	suite.Nil(err)

	identityResponse := &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: ptrutil.String(testutil.EntityIDAccount),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeCustomer,
			ID:           userID,
			Name:         ptrutil.String("Test User"),
			AccountID:    ptrutil.String("acc_customer123"),
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, testutil.EntityIDAccount).
		Return(identityResponse, nil).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, testutil.EntityIDAccount)
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

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_JWT_Expired() {
	userID := testutil.EntityIDUser
	token, err := suite.jwtUtils.Encode(context.Background(), userID, -time.Hour, domain.JWTTypeAccess) // Expired token
	suite.Nil(err)

	ctx := context.Background()
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, testutil.EntityIDAccount).
		Return(nil, contracts.NewAuthenticationError("Access token has expired.")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, testutil.EntityIDAccount)
	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_JWT_Invalid() {
	invalidToken := "invalid.jwt.token"
	ctx := context.Background()
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), invalidToken, testutil.EntityIDAccount).
		Return(nil, contracts.NewAuthenticationError("Invalid token")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, invalidToken, testutil.EntityIDAccount)
	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_JWT_UserNotFound() {
	userID := "usr_notfound123"
	token, err := suite.jwtUtils.Encode(context.Background(), userID, time.Hour, domain.JWTTypeAccess)
	suite.Nil(err)

	ctx := context.Background()
	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), token, testutil.EntityIDAccount).
		Return(nil, contracts.NewAuthenticationError("Invalid token")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, token, testutil.EntityIDAccount)
	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}

func (suite *MiddlewareSvcTestSuite) TestAuthenticateRequest_APIKey_Invalid() {
	ctx := context.Background()

	suite.userMed.EXPECT().
		ValidateCredential(gomock.Any(), testutil.APIKeyValidProdMode, testutil.EntityIDAccount).
		Return(nil, contracts.NewAuthenticationError("Invalid API key")).
		Times(1)

	identity, err := suite.authSvc.ValidateCredential(ctx, testutil.APIKeyValidProdMode, testutil.EntityIDAccount)
	suite.NotNil(err)
	suite.Equal(contracts.ErrorCodeInvalidCredentials, err.Code)
	suite.Nil(identity)
}
