package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	clientmock "github.com/augno/api/services/auth-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	utilsmock "github.com/augno/api/services/auth-service/internal/domain/mock/utils"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"
)

func TestValidateCredential_EmptyToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	// For unauthenticated identity, we check account context to determine mode
	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-123").Return(&domain.AccountContext{
		AccountID:   "acct-123",
		AccountMode: constants.AccountModeProduction,
	}, nil)

	med := &userMedImpl{
		coreClient: coreClient,
	}

	identity, err := med.ValidateCredential(context.Background(), "", ptrString("acct-123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity == nil || identity.Type != types.IdentityTypeUnauthenticated {
		t.Fatalf("expected unauthenticated identity, got %+v", identity)
	}
	if identity.TargetAccountID == nil || *identity.TargetAccountID != "acct-123" {
		t.Fatalf("expected target account acct-123, got %+v", identity.TargetAccountID)
	}
}

func TestValidateCredential_APIKeyOwnedAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	apiKey := &domain.APIKey{
		ID:             1,
		TypeID:         "apikey_sandbox",
		KeyID:          "api-1",
		Name:           "Key One",
		OwnerAccountID: "acct-1",
		RoleID:         "role-1",
		RoleTypeCode:   "admin",
	}

	parsedKey := &domain.ParsedAPIKey{
		AccountMode: constants.AccountModeProduction,
		ID:          "api-1",
		Secret:      "secret",
		Checksum:    "abc",
	}

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_token").Return(apiKey, nil)
	apiKeyMed.EXPECT().ParseKey(gomock.Any(), "aug_sk_token").Return(parsedKey, nil)
	apiKeyMed.EXPECT().TouchIfNotRecent(gomock.Any(), apiKey).Return(nil)
	apiKeyMed.EXPECT().GetKeyAccountAccess(gomock.Any(), constants.AccountModeProduction, apiKey.ID, "acct-1").Return(&domain.APIKeyAccountAccess{
		APIKeyID:    apiKey.TypeID,
		AccountID:   "acct-1",
		RoleID:      &apiKey.RoleID,
		Permissions: map[string]bool{"perm:view": true},
	}, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtUtils:   utilsmock.NewMockJWTUtils(ctrl),
	}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_token", ptrString("acct-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.Type != types.IdentityTypeAPIKey {
		t.Fatalf("expected API key identity, got %s", identity.Type)
	}
	if identity.Actor.Type != types.IdentityActorTypeInternal {
		t.Fatalf("expected internal actor, got %s", identity.Actor.Type)
	}
	if identity.Actor.AccountID == nil || *identity.Actor.AccountID != "acct-1" {
		t.Fatalf("expected owner account acct-1, got %+v", identity.Actor.AccountID)
	}
	if identity.Actor.RoleID == nil || *identity.Actor.RoleID != "role-1" {
		t.Fatalf("expected role role-1, got %+v", identity.Actor.RoleID)
	}
	if len(identity.Actor.Permissions) != 1 || !identity.Actor.Permissions["perm:view"] {
		t.Fatalf("expected permissions to include perm:view, got %+v", identity.Actor.Permissions)
	}
}

func TestValidateCredential_APIKeyRelationMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	apiKey := &domain.APIKey{
		ID:             2,
		TypeID:         "apikey_sandbox",
		KeyID:          "api-2",
		Name:           "Partner Key",
		OwnerAccountID: "acct-owner",
		RoleID:         "role-2",
		RoleTypeCode:   "partner",
	}

	parsedKey := &domain.ParsedAPIKey{
		AccountMode: constants.AccountModeProduction,
		ID:          "api-2",
		Secret:      "secret",
		Checksum:    "abc",
	}

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_missing").Return(apiKey, nil)
	apiKeyMed.EXPECT().ParseKey(gomock.Any(), "aug_sk_missing").Return(parsedKey, nil)
	coreClient.EXPECT().GetAccountRelationByAPIKeyID(gomock.Any(), "acct-target", apiKey.ID).
		Return(nil, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtUtils:   utilsmock.NewMockJWTUtils(ctrl),
	}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_missing", ptrString("acct-target"))
	if err == nil {
		t.Fatal("expected error for missing relation, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.PublicMessage != token.ErrInvalidJWT {
		t.Fatalf("expected invalid JWT error, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserAccountUserPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)

	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)
	accessTokenUtils := utilsmock.NewMockJWTUtils(ctrl)

	userID := "user-1"
	targetAccountID := "acct-22"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	claims := &domain.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TokenType: domain.JWTTypeAccess,
	}

	accessTokenUtils.EXPECT().Decode(gomock.Any(), "valid-token", domain.JWTTypeAccess).Return(claims, nil)
	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), targetAccountID).Return(&domain.AccountContext{
		AccountID:   targetAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, targetAccountID).Return(&domain.AccountUserAccess{
		AccountUserID: "acu-1",
		AccountID:     targetAccountID,
		RoleID:        ptrString("role-99"),
		RoleTypeCode:  ptrString("manager"),
		Permissions:   map[string]bool{"perm:edit": true},
	}, nil)
	coreClient.EXPECT().MarkAccountUserUsed(gomock.Any(), "acu-1").Return(nil).AnyTimes()

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtUtils:   accessTokenUtils,
	}

	identity, err := med.ValidateCredential(context.Background(), "valid-token", &targetAccountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.Type != types.IdentityTypeUser {
		t.Fatalf("expected user identity, got %s", identity.Type)
	}
	if identity.Actor.Type != types.IdentityActorTypeInternal {
		t.Fatalf("expected internal user, got %s", identity.Actor.Type)
	}
	if identity.Actor.RoleID == nil || *identity.Actor.RoleID != "role-99" {
		t.Fatalf("expected role role-99, got %+v", identity.Actor.RoleID)
	}
	if identity.Actor.RoleTypeCode == nil || *identity.Actor.RoleTypeCode != "manager" {
		t.Fatalf("expected role type manager, got %+v", identity.Actor.RoleTypeCode)
	}
	if len(identity.Actor.Permissions) != 1 || !identity.Actor.Permissions["perm:edit"] {
		t.Fatalf("expected permission perm:edit, got %+v", identity.Actor.Permissions)
	}
}

func TestValidateCredential_UserExpiredToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)
	accessTokenUtils := utilsmock.NewMockJWTUtils(ctrl)

	expiredClaims := &domain.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-expired",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
		TokenType: domain.JWTTypeAccess,
	}

	accessTokenUtils.EXPECT().Decode(gomock.Any(), "expired-token", domain.JWTTypeAccess).Return(expiredClaims, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtUtils:   accessTokenUtils,
	}

	identity, err := med.ValidateCredential(context.Background(), "expired-token", ptrString("acct-1"))
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.PublicMessage != ErrAccessTokenExpired {
		t.Fatalf("expected expired token message, got %s", err.PublicMessage)
	}
	if err.Code != apierror.ErrorCodeInvalidCredentials {
		t.Fatalf("expected invalid credentials code, got %s", err.Code)
	}
	if err.Type != apierror.ErrorTypeInvalidRequest {
		t.Fatalf("expected invalid request type, got %s", err.Type)
	}
}

func TestValidateCredential_EmptyToken_AccountNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-nonexistent").
		Return(nil, apierror.NewResourceNotFoundError("Account not found"))

	med := &userMedImpl{
		coreClient: coreClient,
	}

	identity, err := med.ValidateCredential(context.Background(), "", ptrString("acct-nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent account, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
	if err.PublicMessage != ErrNoAccountAccess {
		t.Fatalf("expected access denied message, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserToken_AccountNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)
	accessTokenUtils := utilsmock.NewMockJWTUtils(ctrl)

	userID := "user-1"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	claims := &domain.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TokenType: domain.JWTTypeAccess,
	}

	accessTokenUtils.EXPECT().Decode(gomock.Any(), "valid-token", domain.JWTTypeAccess).Return(claims, nil)
	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-nonexistent").
		Return(nil, apierror.NewResourceNotFoundError("Account not found"))

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtUtils:   accessTokenUtils,
	}

	identity, err := med.ValidateCredential(context.Background(), "valid-token", ptrString("acct-nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent account, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
	if err.PublicMessage != ErrNoAccountAccess {
		t.Fatalf("expected access denied message, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserToken_UserAccountAccessNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)
	accessTokenUtils := utilsmock.NewMockJWTUtils(ctrl)

	userID := "user-1"
	targetAccountID := "acct-no-access"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	claims := &domain.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TokenType: domain.JWTTypeAccess,
	}

	accessTokenUtils.EXPECT().Decode(gomock.Any(), "valid-token", domain.JWTTypeAccess).Return(claims, nil)
	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), targetAccountID).Return(&domain.AccountContext{
		AccountID:   targetAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, targetAccountID).
		Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtUtils:   accessTokenUtils,
	}

	identity, err := med.ValidateCredential(context.Background(), "valid-token", &targetAccountID)
	if err == nil {
		t.Fatal("expected error for no account access, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
	if err.PublicMessage != ErrNoAccountAccess {
		t.Fatalf("expected access denied message, got %s", err.PublicMessage)
	}
}

func ptrString(v string) *string {
	return &v
}
