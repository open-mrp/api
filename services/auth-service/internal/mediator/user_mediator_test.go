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

	identity, err := med.ValidateCredential(context.Background(), "", new("acct-123"), nil)
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

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_token", new("acct-1"), nil)
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

// TestValidateCredential_APIKeyCounterpartySideCarriesPermissions confirms a
// counterparty-side API key (e.g. a customer's key targeting a merchant) carries its
// own-account role permissions so downstream services can authorize customer-side
// capabilities (e.g. purchase_orders:create). RoleID/RoleType are cleared.
func TestValidateCredential_APIKeyCounterpartySideCarriesPermissions(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	apiKey := &apikey.APIKey{
		ID:             7,
		TypeID:         "apky_customer",
		KeyID:          "api-cust",
		Name:           "Customer Key",
		OwnerAccountID: "acct-customer",
		RoleID:         "role-cust",
		RoleType:       "user",
	}
	parsedKey := &apikey.ParsedAPIKey{AccountMode: constants.AccountModeProduction, ID: "api-cust", Secret: "secret", Checksum: "abc"}
	touchDone := make(chan struct{}, 1)

	apiKeyMed.EXPECT().FindAndValidate(gomock.Any(), "aug_sk_cust").Return(apiKey, nil)
	apiKeyMed.EXPECT().ParseKey(gomock.Any(), "aug_sk_cust").Return(parsedKey, nil)
	apiKeyMed.EXPECT().TouchIfNotRecent(gomock.Any(), apiKey).DoAndReturn(func(context.Context, *apikey.APIKey) *apierror.APIError {
		touchDone <- struct{}{}
		return nil
	})
	coreClient.EXPECT().GetAccountContext(gomock.Any(), "acct-merchant").Return(&domain.AccountContext{
		AccountID:   "acct-merchant",
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetAccountRelationByAPIKeyID(gomock.Any(), "acct-merchant", apiKey.ID).Return(&domain.AuthAccountRelation{
		ID:                      "rel-M-C",
		CounterpartyAccountID:   "acct-customer",
		AccountRelationRoleCode: types.IdentityRelationTypeCustomer,
		IsOwnerSide:             false,
	}, true, nil)
	// The new own-account permission lookup for the counterparty key.
	apiKeyMed.EXPECT().GetKeyAccountAccess(gomock.Any(), domain.APIKeyGetAccountAccessInput{
		AccountMode: constants.AccountModeProduction, APIKeyID: apiKey.ID, TargetAccountID: "acct-customer",
	}).Return(&domain.APIKeyAccountAccess{
		APIKeyID:    apiKey.TypeID,
		AccountID:   "acct-customer",
		RoleID:      &apiKey.RoleID,
		Permissions: map[string]bool{"purchase_orders:create": true},
	}, nil)

	med := &userMedImpl{repos: repoFactory, apiKeyMed: apiKeyMed, coreClient: coreClient, jwtSecret: testutil.JWTSecret}

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_cust", new("acct-merchant"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-touchDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected TouchIfNotRecent to be called")
	}

	if identity.Actor.RelationType != types.IdentityRelationTypeCustomer {
		t.Fatalf("expected customer relation actor, got %s", identity.Actor.RelationType)
	}
	if identity.Target == nil || identity.Target.AccountID != "acct-merchant" {
		t.Fatalf("expected target acct-merchant, got %+v", identity.Target)
	}
	if !identity.Actor.Permissions["purchase_orders:create"] {
		t.Fatalf("expected carried permission purchase_orders:create, got %+v", identity.Actor.Permissions)
	}
	if identity.Actor.RoleID != nil || identity.Actor.RoleType != nil {
		t.Fatalf("expected RoleID/RoleType cleared, got roleID=%v roleType=%v", identity.Actor.RoleID, identity.Actor.RoleType)
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

	identity, err := med.ValidateCredential(context.Background(), "aug_sk_missing", new("acct-target"), nil)
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
		RoleID:        new("role-99"),
		RoleType:      new("manager"),
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

	identity, err := med.ValidateCredential(context.Background(), expiredToken, new("acct-1"), nil)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	// An expired token surfaces as the distinct expired_token code (not the generic invalid_credentials) so request-log review can filter out routine token-rotation noise while keeping genuine auth failures visible.
	if err.PublicMessage != token.ErrExpiredJWT {
		t.Fatalf("expected expired JWT message, got %s", err.PublicMessage)
	}
	if err.Code != apierror.ErrorCodeExpiredToken {
		t.Fatalf("expected expired token code, got %s", err.Code)
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

	identity, err := med.ValidateCredential(context.Background(), "", new("acct-nonexistent"), nil)
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

	identity, err := med.ValidateCredential(context.Background(), validToken, new("acct-nonexistent"), nil)
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
	coreClient.EXPECT().GetAccountRelationByUserID(gomock.Any(), targetAccountID, "", userID).
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

// TestValidateCredential_UserToken_OwnerSideRelationConstrainedToActorAccount confirms that the
// owner-side relation lookup is bound to the validated actor account. A user who belongs to
// account A (which owns a relation to C) should be able to act on target C *only* when their
// supplied actor account is A. The mediator must forward the actor account to core-service so
// the relation can be matched against owner_account_id = actor_account_id.
func TestValidateCredential_UserToken_OwnerSideRelationConstrainedToActorAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-1"
	actorAccountID := "acct-A"
	targetAccountID := "acct-C"
	name := "Merchant User"
	userModel := &types.User{ID: userID, Name: &name}

	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	roleType := "manager"
	roleID := "role-99"
	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), actorAccountID).Return(&domain.AccountContext{
		AccountID:   actorAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, actorAccountID).Return(&domain.AccountUserAccess{
		AccountUserID: "acu-1",
		AccountID:     actorAccountID,
		RoleID:        &roleID,
		RoleType:      &roleType,
		Permissions:   map[string]bool{"product_line:read": true},
	}, true, nil)
	coreClient.EXPECT().MarkAccountUserUsed(gomock.Any(), "acu-1").Return(nil).AnyTimes()
	// The owner-side lookup MUST be called with actorAccountID — passing anything else
	// would let a user keep account B's permissions while targeting C via an A→C relation.
	coreClient.EXPECT().GetAccountRelationByUserID(gomock.Any(), targetAccountID, actorAccountID, userID).Return(&domain.AuthAccountRelation{
		ID:                      "rel-A-C",
		CounterpartyAccountID:   targetAccountID,
		AccountRelationRoleCode: types.IdentityRelationTypeCustomer,
		IsOwnerSide:             true,
	}, true, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, &targetAccountID, &actorAccountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity == nil {
		t.Fatal("expected identity, got nil")
	}
	if identity.Target == nil || identity.Target.AccountID != targetAccountID {
		t.Fatalf("expected target account %s, got %+v", targetAccountID, identity.Target)
	}
	if identity.Target.RelationType == nil || *identity.Target.RelationType != types.IdentityRelationTypeCustomer {
		t.Fatalf("expected target relation type customer, got %+v", identity.Target.RelationType)
	}
	// Actor permissions must remain intact — owner-side keeps the merchant's own role/perms
	// because the SQL constraint guaranteed the relation's owner == actor account.
	if identity.Actor == nil || identity.Actor.AccountID == nil || *identity.Actor.AccountID != actorAccountID {
		t.Fatalf("expected actor account %s, got %+v", actorAccountID, identity.Actor)
	}
	if !identity.Actor.Permissions["product_line:read"] {
		t.Fatalf("expected actor permissions to be preserved on owner-side, got %+v", identity.Actor.Permissions)
	}
}

// TestValidateCredential_UserToken_CounterpartySideCarriesPermissions confirms that a
// counterparty-side relation actor (e.g. a customer targeting a merchant) now KEEPS its
// own-account role permissions so downstream services can authorize customer-side
// capabilities (e.g. purchase_orders:create for a portal order). RoleID/RoleType are
// cleared so no admin bypass leaks across the relation.
func TestValidateCredential_UserToken_CounterpartySideCarriesPermissions(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-cust"
	actorAccountID := "acct-Customer"
	targetAccountID := "acct-Merchant"
	name := "Customer User"
	userModel := &types.User{ID: userID, Name: &name}

	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	roleType := "user"
	roleID := "role-cust"
	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), actorAccountID).Return(&domain.AccountContext{
		AccountID:   actorAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, actorAccountID).Return(&domain.AccountUserAccess{
		AccountUserID: "acu-cust",
		AccountID:     actorAccountID,
		RoleID:        &roleID,
		RoleType:      &roleType,
		Permissions:   map[string]bool{"purchase_orders:create": true},
	}, true, nil)
	coreClient.EXPECT().MarkAccountUserUsed(gomock.Any(), "acu-cust").Return(nil).AnyTimes()
	// Counterparty-side: the user belongs to the customer account; the merchant owns the relation.
	coreClient.EXPECT().GetAccountRelationByUserID(gomock.Any(), targetAccountID, actorAccountID, userID).Return(&domain.AuthAccountRelation{
		ID:                      "rel-M-C",
		CounterpartyAccountID:   actorAccountID,
		AccountRelationRoleCode: types.IdentityRelationTypeCustomer,
		IsOwnerSide:             false,
	}, true, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, &targetAccountID, &actorAccountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity == nil || identity.Actor == nil {
		t.Fatal("expected identity with actor, got nil")
	}
	if identity.Actor.RelationType != types.IdentityRelationTypeCustomer {
		t.Fatalf("expected customer relation actor, got %s", identity.Actor.RelationType)
	}
	if identity.Target == nil || identity.Target.AccountID != targetAccountID {
		t.Fatalf("expected target account %s, got %+v", targetAccountID, identity.Target)
	}
	// Permissions must be CARRIED (post-redesign), not stripped.
	if !identity.Actor.Permissions["purchase_orders:create"] {
		t.Fatalf("expected carried permission purchase_orders:create, got %+v", identity.Actor.Permissions)
	}
	// RoleID/RoleType cleared so IsAdmin()/IsRoleSet() stay false.
	if identity.Actor.RoleID != nil || identity.Actor.RoleType != nil {
		t.Fatalf("expected RoleID/RoleType cleared, got roleID=%v roleType=%v", identity.Actor.RoleID, identity.Actor.RoleType)
	}
}

// TestValidateCredential_UserToken_NoActorAccountRejectsOwnerSide guards the no-Augno-Actor-Account
// path. Without a validated actor account, an owner-side relation has no actor permissions to bind
// to and the related-user identity builder would incorrectly attribute Actor.AccountID to the
// counterparty — so we reject the request. The mediator passes "" as actorAccountID, which causes
// core-service to skip the owner-side fallback; this test additionally asserts the defense-in-depth
// rejection if a relation ever leaks through.
func TestValidateCredential_UserToken_NoActorAccountRejectsOwnerSide(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	repoFactory.EXPECT().NewUserRepo().Return(userRepo)

	apiKeyMed := mediatormock.NewMockAPIKeyMed(ctrl)
	coreClient := clientmock.NewMockAuthCoreClient(ctrl)

	userID := "user-1"
	targetAccountID := "acct-C"
	name := "Merchant User"
	userModel := &types.User{ID: userID, Name: &name}

	validToken, encErr := token.EncodeJWT(context.Background(), testutil.JWTSecret, userID, time.Hour, token.JWTTypeAccess)
	if encErr != nil {
		t.Fatalf("failed to encode test token: %v", encErr)
	}

	userRepo.EXPECT().Find(gomock.Any(), userID).Return(userModel, nil)
	coreClient.EXPECT().GetAccountContext(gomock.Any(), targetAccountID).Return(&domain.AccountContext{
		AccountID:   targetAccountID,
		AccountMode: constants.AccountModeProduction,
	}, nil)
	coreClient.EXPECT().GetUserAccountAccess(gomock.Any(), userID, targetAccountID).Return(nil, false, nil)
	// Mediator must pass "" for actor when no Augno-Actor-Account header is present.
	// Simulate the defense-in-depth path: core-service returns an owner-side relation anyway.
	coreClient.EXPECT().GetAccountRelationByUserID(gomock.Any(), targetAccountID, "", userID).Return(&domain.AuthAccountRelation{
		ID:                      "rel-A-C",
		CounterpartyAccountID:   targetAccountID,
		AccountRelationRoleCode: types.IdentityRelationTypeCustomer,
		IsOwnerSide:             true,
	}, true, nil)

	med := &userMedImpl{
		repos:      repoFactory,
		apiKeyMed:  apiKeyMed,
		coreClient: coreClient,
		jwtSecret:  testutil.JWTSecret,
	}

	identity, err := med.ValidateCredential(context.Background(), validToken, &targetAccountID, nil)
	if err == nil {
		t.Fatal("expected error for owner-side relation without actor account, got nil")
	}
	if identity != nil {
		t.Fatalf("expected nil identity on error, got %+v", identity)
	}
	if err.Code != apierror.ErrorCodeInsufficientPerms {
		t.Fatalf("expected insufficient_permissions error code, got %s", err.Code)
	}
}
