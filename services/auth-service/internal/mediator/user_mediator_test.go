package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	clientmock "github.com/augno/api/services/auth-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/auth-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/auth-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/auth-service/internal/domain/mock/repository"
	"github.com/augno/api/services/auth-service/internal/testutil"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"go.uber.org/mock/gomock"
)

func TestValidateCredential_EmptyToken(t *testing.T) {
	t.Parallel()
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

	identity, err := med.ValidateCredential(context.Background(), "", ptrString("acct-123"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity == nil || identity.Type != types.IdentityActorTypeUnauthenticated {
		t.Fatalf("expected unauthenticated identity, got %+v", identity)
	}
	if identity.Target == nil || identity.Target.AccountID != "acct-123" {
		t.Fatalf("expected target account acct-123, got %+v", identity.Target)
	}
}

func TestValidateCredential_APIKeyOwnedAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	apiKey := &apikey.APIKey{
		ID:             1,
		TypeID:         "apikey_sandbox",
		KeyID:          "api-1",
		Name:           "Key One",
		OwnerAccountID: "acct-1",
		RoleID:         "role-1",
		RoleType:       "admin",
	}

	parsedKey := &apikey.ParsedAPIKey{
		AccountMode: constants.AccountModeProduction,
		ID:          "api-1",
		Secret:      "secret",
		Checksum:    "abc",
	}
	touchDone := make(chan struct{}, 1)

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_token").Return(apiKey, nil)
	apiKeyMed.EXPECT().ParseKey(gomock.Any(), "aug_sk_token").Return(parsedKey, nil)
	apiKeyMed.EXPECT().TouchIfNotRecent(gomock.Any(), apiKey).DoAndReturn(func(context.Context, *apikey.APIKey) *apierror.APIError {
		touchDone <- struct{}{}
		return nil
	})
	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-1").Return(&domain.AccountContext{
		AccountID:   "acct-1",
		AccountMode: constants.AccountModeProduction,
	}, nil)
	apiKeyMed.EXPECT().GetKeyAccountAccess(gomock.Any(), domain.APIKeyGetAccountAccessInput{
		AccountMode: constants.AccountModeProduction, APIKeyID: apiKey.ID, TargetAccountID: "acct-1",
	}).Return(&domain.APIKeyAccountAccess{
		APIKeyID:    apiKey.TypeID,
		AccountID:   "acct-1",
		RoleID:      &apiKey.RoleID,
		Permissions: map[string]bool{"perm:view": true},
	}, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_token", ptrString("acct-1"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-touchDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected TouchIfNotRecent to be called")
	}

	if identity.Type != types.IdentityActorTypeAPIKey {
		t.Fatalf("expected API key identity, got %s", identity.Type)
	}
	if identity.Actor.RelationType != types.IdentityRelationTypeInternal {
		t.Fatalf("expected internal actor, got %s", identity.Actor.RelationType)
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
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	apiKey := &apikey.APIKey{
		ID:             2,
		TypeID:         "apikey_sandbox",
		KeyID:          "api-2",
		Name:           "Partner Key",
		OwnerAccountID: "acct-owner",
		RoleID:         "role-2",
		RoleType:       "partner",
	}

	parsedKey := &apikey.ParsedAPIKey{
		AccountMode: constants.AccountModeProduction,
		ID:          "api-2",
		Secret:      "secret",
		Checksum:    "abc",
	}
	touchDone := make(chan struct{}, 1)

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_missing").Return(apiKey, nil)
	apiKeyMed.EXPECT().ParseKey(gomock.Any(), "aug_sk_missing").Return(parsedKey, nil)
	apiKeyMed.EXPECT().TouchIfNotRecent(gomock.Any(), apiKey).DoAndReturn(func(context.Context, *apikey.APIKey) *apierror.APIError {
		touchDone <- struct{}{}
		return nil
	})
	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-target").Return(&domain.AccountContext{
		AccountID:   "acct-target",
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetAccountRelationByAPIKeyID(gomock.Any(), "acct-target", apiKey.ID).
		Return(nil, false, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_missing", ptrString("acct-target"), nil)
	select {
	case <-touchDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected TouchIfNotRecent to be called")
	}
	if err == nil {
		t.Fatal("expected error for missing relation, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.PublicMessage != errNoAccountAccess("acct-target") {
		t.Fatalf("expected no account access error, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserAccountUserPath(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)

	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-1"
	targetAccountID := "acct-22"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	// Create a real JWT token
	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), targetAccountID).Return(&domain.AccountContext{
		AccountID:   targetAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, targetAccountID).Return(&domain.AccountUserAccess{
		AccountUserID: "acu-1",
		AccountID:     targetAccountID,
		RoleID:        ptrString("role-99"),
		RoleType:      ptrString("manager"),
		Permissions:   map[string]bool{"perm:edit": true},
	}, true, nil)
	coreClient.EXPECT().MarkAccountUserUsed(gomock.Any(), "acu-1").Return(nil).AnyTimes()

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, &targetAccountID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.Type != types.IdentityActorTypeUser {
		t.Fatalf("expected user identity, got %s", identity.Type)
	}
	if identity.Actor.RelationType != types.IdentityRelationTypeInternal {
		t.Fatalf("expected internal user, got %s", identity.Actor.RelationType)
	}
	if identity.Actor.RoleID == nil || *identity.Actor.RoleID != "role-99" {
		t.Fatalf("expected role role-99, got %+v", identity.Actor.RoleID)
	}
	if identity.Actor.RoleType == nil || *identity.Actor.RoleType != "manager" {
		t.Fatalf("expected role type manager, got %+v", identity.Actor.RoleType)
	}
	if len(identity.Actor.Permissions) != 1 || !identity.Actor.Permissions["perm:edit"] {
		t.Fatalf("expected permission perm:edit, got %+v", identity.Actor.Permissions)
	}
}

func TestValidateCredential_UserExpiredToken(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	// Create a real expired JWT token
	expiredToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, "user-expired", -time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), expiredToken, ptrString("acct-1"), nil)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.PublicMessage != token.ErrInvalidJWT {
		t.Fatalf("expected invalid JWT message, got %s", err.PublicMessage)
	}
	if err.Code != apierror.ErrorCodeInvalidCredentials {
		t.Fatalf("expected invalid credentials code, got %s", err.Code)
	}
	if err.Type != apierror.ErrorTypeInvalidRequest {
		t.Fatalf("expected invalid request type, got %s", err.Type)
	}
}

func TestValidateCredential_EmptyToken_AccountNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-nonexistent").
		Return(nil, apierror.NewResourceNotFoundError("Account not found"))

	med := &userMedImpl{
		coreClient: coreClient,
	}

	identity, err := med.ValidateCredential(context.Background(), "", ptrString("acct-nonexistent"), nil)
	if err == nil {
		t.Fatal("expected error for nonexistent account, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
	if err.PublicMessage != errNoAccountAccess("acct-nonexistent") {
		t.Fatalf("expected access denied message, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserToken_AccountNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-1"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	// Create a real JWT token
	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-nonexistent").
		Return(nil, apierror.NewResourceNotFoundError("Account not found"))

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, ptrString("acct-nonexistent"), nil)
	if err == nil {
		t.Fatal("expected error for nonexistent account, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
	if err.PublicMessage != errNoAccountAccess("acct-nonexistent") {
		t.Fatalf("expected access denied message, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserToken_UserAccountAccessNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-1"
	targetAccountID := "acct-no-access"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	// Create a real JWT token
	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), targetAccountID).Return(&domain.AccountContext{
		AccountID:   targetAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, targetAccountID).
		Return(nil, false, nil)
	coreClient.EXPECT().GetAccountRelationByUserID(gomock.Any(), targetAccountID, userID).
		Return(nil, false, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, &targetAccountID, nil)
	if err == nil {
		t.Fatal("expected error for no account access, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
	if err.PublicMessage != errNoAccountAccess("acct-no-access") {
		t.Fatalf("expected no account access message, got %s", err.PublicMessage)
	}
}

func TestValidateCredential_UserToken_ActorAccountHeader_RequiresAccountUser(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-1"
	actorAccountID := "acct-actor"
	name := "Jane"
	userModel := &types.User{ID: userID, Name: &name}

	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), actorAccountID).Return(&domain.AccountContext{
		AccountID:   actorAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, actorAccountID).
		Return(nil, false, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, nil, &actorAccountID)
	if err == nil {
		t.Fatal("expected error when Augno-Actor-Account is set but user has no account-user relation, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInvalidCredentials {
		t.Fatalf("expected invalid_credentials (401), got %s", err.Code)
	}
	if err.PublicMessage != ErrActorAccountRequiresMember {
		t.Fatalf("expected actor account requires member message, got %s", err.PublicMessage)
	}
}

func ptrString(v string) *string {
	return &v
}
