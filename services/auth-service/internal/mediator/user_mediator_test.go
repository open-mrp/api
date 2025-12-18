package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	utilsmock "github.com/augno/api/services/auth-service/internal/domain/mock/utils"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"
)

func TestValidateCredential_EmptyToken(t *testing.T) {
	med := &userMedImpl{}

	identity, err := med.ValidateCredential(context.Background(), "", "acct-123")
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
	accountRelationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	rolePermissionRepo := repositorymock.NewMockRolePermissionRepo(ctrl)

	repoFactory.EXPECT().NewAccountRelationRepo().Return(accountRelationRepo)
	repoFactory.EXPECT().NewAccountUserRepo().Return(accountUserRepo)
	repoFactory.EXPECT().NewRolePermissionRepo().Return(rolePermissionRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	accountUserMed := mediatormock.NewMockAccountUserMed(ctrl)

	apiKey := &domain.APIKey{
		ID:             "api-1",
		Name:           "Key One",
		OwnerAccountID: "acct-1",
		RoleID:         "role-1",
		RoleTypeCode:   "admin",
	}

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_token").Return(apiKey, nil)
	apiKeyMed.EXPECT().TouchIfNotRecent(gomock.Any(), apiKey).Return(nil)
	rolePermissionRepo.EXPECT().FindByRoleID(gomock.Any(), apiKey.RoleID).Return(map[string]bool{"perm:view": true}, nil)

	med := &userMedImpl{
		repos:          repoFactory,
		apiKeyMed:      apiKeyMed,
		accountUserMed: accountUserMed,
		jwtUtils:       utilsmock.NewMockJWTUtils(ctrl),
	}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_token", "acct-1")
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
	accountRelationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	rolePermissionRepo := repositorymock.NewMockRolePermissionRepo(ctrl)

	repoFactory.EXPECT().NewAccountRelationRepo().Return(accountRelationRepo)
	repoFactory.EXPECT().NewAccountUserRepo().Return(accountUserRepo)
	repoFactory.EXPECT().NewRolePermissionRepo().Return(rolePermissionRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	accountUserMed := mediatormock.NewMockAccountUserMed(ctrl)

	apiKey := &domain.APIKey{
		ID:             "api-2",
		Name:           "Partner Key",
		OwnerAccountID: "acct-owner",
		RoleID:         "role-2",
		RoleTypeCode:   "partner",
	}

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_missing").Return(apiKey, nil)
	accountRelationRepo.EXPECT().FindByOwnerAccountAndUserID(gomock.Any(), "acct-target", apiKey.ID).
		Return(nil, nil)

	med := &userMedImpl{
		repos:          repoFactory,
		apiKeyMed:      apiKeyMed,
		accountUserMed: accountUserMed,
		jwtUtils:       utilsmock.NewMockJWTUtils(ctrl),
	}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_missing", "acct-target")
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
	accountRelationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	rolePermissionRepo := repositorymock.NewMockRolePermissionRepo(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)

	repoFactory.EXPECT().NewAccountRelationRepo().Return(accountRelationRepo)
	repoFactory.EXPECT().NewAccountUserRepo().Return(accountUserRepo)
	repoFactory.EXPECT().NewRolePermissionRepo().Return(rolePermissionRepo)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	accountUserMed := mediatormock.NewMockAccountUserMed(ctrl)
	accessTokenUtils := utilsmock.NewMockJWTUtils(ctrl)

	userID := "user-1"
	targetAccountID := "acct-22"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}
	accountUser := &domain.AccountUser{
		ID:           "acu-1",
		UserID:       userID,
		AccountID:    targetAccountID,
		RoleID:       ptrString("role-99"),
		RoleTypeCode: ptrString("manager"),
	}

	claims := &domain.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TokenType: domain.JWTTypeAccess,
	}

	accessTokenUtils.EXPECT().Decode(gomock.Any(), "valid-token", domain.JWTTypeAccess).Return(claims, nil)
	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	accountUserRepo.EXPECT().FindByAccountAndUserID(gomock.Any(), userID, targetAccountID).Return(accountUser, nil)
	accountUserMed.EXPECT().MarkUsedIfNotRecent(gomock.Any(), accountUser).Return(nil)
	rolePermissionRepo.EXPECT().FindByRoleID(gomock.Any(), *accountUser.RoleID).Return(map[string]bool{"perm:edit": true}, nil)

	med := &userMedImpl{
		repos:          repoFactory,
		apiKeyMed:      apiKeyMed,
		accountUserMed: accountUserMed,
		jwtUtils:       accessTokenUtils,
	}

	identity, err := med.ValidateCredential(context.Background(), "valid-token", targetAccountID)
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
	accountRelationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	rolePermissionRepo := repositorymock.NewMockRolePermissionRepo(ctrl)

	repoFactory.EXPECT().NewAccountRelationRepo().Return(accountRelationRepo)
	repoFactory.EXPECT().NewAccountUserRepo().Return(accountUserRepo)
	repoFactory.EXPECT().NewRolePermissionRepo().Return(rolePermissionRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	accountUserMed := mediatormock.NewMockAccountUserMed(ctrl)
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
		repos:          repoFactory,
		apiKeyMed:      apiKeyMed,
		accountUserMed: accountUserMed,
		jwtUtils:       accessTokenUtils,
	}

	identity, err := med.ValidateCredential(context.Background(), "expired-token", "acct-1")
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.PublicMessage != ErrAccessTokenExpired {
		t.Fatalf("expected expired token message, got %s", err.PublicMessage)
	}
	if err.Code != contracts.ErrorCodeInvalidCredentials {
		t.Fatalf("expected invalid credentials code, got %s", err.Code)
	}
	if err.Type != contracts.ErrorTypeInvalidRequest {
		t.Fatalf("expected invalid request type, got %s", err.Type)
	}
}

func ptrString(v string) *string {
	return &v
}
