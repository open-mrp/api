package mediator

import (
	"context"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	emailpkg "github.com/augno/api/services/auth-service/internal/email"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	tracing "github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

var userMedTracer = tracing.GetTracer("auth-service.user_mediator")

type userMedImpl struct {
	repos                 domain.RepoFactory
	jwtUtils              domain.JWTUtils
	refreshTokenMed       domain.RefreshTokenMed
	apiKeyMed             domain.APIKeyMed
	accountUserMed        domain.AccountUserMed
	notificationPublisher domain.NotificationPublisher
	templateRenderer      emailpkg.TemplateRenderer
}

type UserMedConfig struct {
	Repos                 domain.RepoFactory
	JWTSecret             string
	RefreshTokenMed       domain.RefreshTokenMed
	APIKeyMed             domain.APIKeyMed
	AccountUserMed        domain.AccountUserMed
	NotificationPublisher domain.NotificationPublisher
	TemplateRenderer      emailpkg.TemplateRenderer
}

func NewUserMed(config UserMedConfig) domain.UserMed {
	if config.JWTSecret == "" {
		panic("JWTSecret is not set in the config.")
	}

	if config.TemplateRenderer == nil {
		panic("TemplateRenderer is not set in the config.")
	}

	if config.NotificationPublisher == nil {
		panic("NotificationPublisher is not set in the config.")
	}

	return &userMedImpl{
		repos:                 config.Repos,
		jwtUtils:              token.NewJWTUtils(token.DefaultJWTConfig(config.JWTSecret)),
		refreshTokenMed:       config.RefreshTokenMed,
		apiKeyMed:             config.APIKeyMed,
		accountUserMed:        config.AccountUserMed,
		notificationPublisher: config.NotificationPublisher,
		templateRenderer:      config.TemplateRenderer,
	}
}

func DefaultUserMedConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte) UserMedConfig {
	repoFactory := repository.NewRepoFactory(queries)
	refreshTokenMed := NewRefreshTokenMed(RefreshTokenMedConfig{
		Repos:            repoFactory,
		JWTUtils:         token.NewJWTUtils(token.DefaultJWTConfig(jwtSecret)),
		OpaqueTokenUtils: token.NewOpaqueTokenUtils(token.DefaultOpaqueTokenConfig()),
	})
	apiKeyMed := NewAPIKeyMed(APIKeyMedConfig{
		Repos:       repoFactory,
		APIKeyUtils: apikey.NewAPIKeyUtils(apikey.DefaultAPIKeyConfig(pepper)),
	})
	accountUserMed := NewAccountUserMed(AccountUserMedConfig{
		Repos: repoFactory,
	})

	return UserMedConfig{
		Repos:           repoFactory,
		JWTSecret:       jwtSecret,
		RefreshTokenMed: refreshTokenMed,
		APIKeyMed:       apiKeyMed,
		AccountUserMed:  accountUserMed,
	}
}

func NewDefaultUserMed(queries *sqlc.Queries, jwtSecret string, pepper []byte) domain.UserMed {
	return NewUserMed(DefaultUserMedConfig(queries, jwtSecret, pepper))
}

// GenAuthAccessToken mints an access token that can be utilized to authenticate requests to the API.
func (s *userMedImpl) GenAuthAccessToken(ctx context.Context, userID string) (string, *contracts.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.gen_auth_access_token")
	defer span.End()

	return s.jwtUtils.Encode(ctx, userID, time.Hour, domain.JWTTypeAccess)
}

// GenPasswordResetAccessToken mints an access token that can be utilized to reset the password for the given user ID.
func (s *userMedImpl) GenPasswordResetAccessToken(ctx context.Context, userID string) (string, *contracts.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.gen_password_reset_access_token")
	defer span.End()

	return s.jwtUtils.Encode(ctx, userID, 15*time.Minute, domain.JWTTypePasswordReset)
}

func (s *userMedImpl) Register(ctx context.Context, name, email, hashedPassword string) (*types.User, *contracts.APIError) {
	ctx, span := userMedTracer.Start(ctx, "domain.auth.register")
	defer span.End()

	userRepo := s.repos.NewUserRepo()

	existingUser, err := userRepo.Find(ctx, email)
	if err == nil && existingUser != nil {
		return nil, tracing.Trace(span, contracts.NewValidationError("Unable to process registration."))
	}

	if err != nil {
		return nil, err
	}

	userID, err := id.GenID(id.UserIDPrefix, nil)
	if err != nil {
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "failed to generate user id"))
	}

	user, err := userRepo.Create(ctx, userID, email, name, hashedPassword)
	if err != nil {
		return nil, err
	}

	if user.Email != nil && user.Name != nil {
		body, err := s.templateRenderer.RenderWelcomeEmail(ctx, emailpkg.WelcomeEmailData{
			UserName:  *user.Name,
			UserEmail: *user.Email,
		})
		if err != nil {
			return nil, tracing.Trace(span, err)
		}

		s.notificationPublisher.PublishSendEmail(
			ctx,
			[]string{*user.Email},
			"Welcome!",
			body,
			true,
			nil,
			user.ID,
			nil,
		)
	}

	return user, nil
}

// ValidateCredential validates a credentials provided by a request and returns an identity.
func (s *userMedImpl) ValidateCredential(ctx context.Context, authToken string, targetAccountID string) (*types.Identity, *contracts.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.validate_credential")
	defer span.End()

	// If the auth token is empty, we return an unauthenticated identity
	if authToken == "" {
		return types.GetUnauthenticatedIdentity(targetAccountID), nil
	}

	deps := credentialValidationDeps{
		accountRelationRepo: s.repos.NewAccountRelationRepo(),
		accountUserRepo:     s.repos.NewAccountUserRepo(),
		rolePermissionRepo:  s.repos.NewRolePermissionRepo(),
	}

	// If the auth token has the API key prefix, validate it as such
	if strings.HasPrefix(authToken, string(types.APIKeyPrefixSecretKey)) {
		return s.validateAPIKeyCredential(ctx, span, authToken, targetAccountID, deps)
	}

	// Otherwise, validate it as a user credential
	return s.validateUserCredential(ctx, span, authToken, targetAccountID, deps)
}

func (s *userMedImpl) validateAPIKeyCredential(ctx context.Context, span trace.Span, authToken, targetAccountID string, deps credentialValidationDeps) (*types.Identity, *contracts.APIError) {
	// Find and validate the API key
	apiKeyModel, err := s.apiKeyMed.FindAndValidate(ctx, authToken)
	if err != nil {
		return nil, err
	}

	// This request is not targeting an account, so mark as unassigned
	if targetAccountID == "" {
		return buildUnassignedAPIKeyIdentity(apiKeyModel), nil
	}

	// The request targets the account that owns the API key
	if apiKeyModel.OwnerAccountID == targetAccountID {
		// Touch the API key to mark it as used
		err := s.apiKeyMed.TouchIfNotRecent(ctx, apiKeyModel)
		if err != nil {
			return nil, err
		}

		// Fetch the permissions for the role that the API key is associated with
		rolePermissions, err := deps.rolePermissionRepo.FindByRoleID(ctx, apiKeyModel.RoleID)
		if err != nil {
			return nil, err
		}

		return buildOwnedAPIKeyIdentity(apiKeyModel, targetAccountID, rolePermissions), nil
	}

	// The request targets a different account, so we need to find the account relation
	accountRelation, err := deps.accountRelationRepo.FindByOwnerAccountAndUserID(ctx, targetAccountID, apiKeyModel.ID)
	if err != nil {
		return nil, err
	}

	// These accounts have no relationship, request should fail
	if accountRelation == nil {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(token.ErrInvalidJWT))
	}

	// Determine the actor type based on the account relation role code
	var actorType types.IdentityActorType
	switch accountRelation.AccountRelationRoleCode {
	case types.IdentityActorTypeCustomer:
		actorType = types.IdentityActorTypeCustomer
	case types.IdentityActorTypeSupplier:
		actorType = types.IdentityActorTypeSupplier
	default:
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "Failed to find account relation."))
	}

	return buildRelatedAPIKeyIdentity(apiKeyModel, accountRelation, actorType, targetAccountID), nil
}

func (s *userMedImpl) validateUserCredential(ctx context.Context, span trace.Span, authToken, targetAccountID string, deps credentialValidationDeps) (*types.Identity, *contracts.APIError) {
	// Find the user by the token
	userModel, err := s.findUserByToken(ctx, authToken)
	if err != nil {
		return nil, err
	}

	// This request is not targeting an account, so mark as unassigned
	if targetAccountID == "" {
		return buildUnassignedUserIdentity(userModel), nil
	}

	// Find the account user by the user and target account
	accountUser, err := deps.accountUserRepo.FindByAccountAndUserID(ctx, userModel.ID, targetAccountID)
	if err != nil {
		return nil, err
	}

	// This user isn't associated with the target account, but they may have a relationship with it
	if accountUser == nil {
		// Find the account relation by the owner account and user
		accountRelation, err := deps.accountRelationRepo.FindByOwnerAccountAndUserID(ctx, targetAccountID, userModel.ID)
		if err != nil {
			return nil, err
		}

		// These accounts have no relationship, request should fail
		if accountRelation == nil {
			return nil, tracing.Trace(span, contracts.NewAuthenticationError(token.ErrInvalidJWT))
		}

		// Determine the actor type based on the account relation role code
		var actorType types.IdentityActorType
		switch accountRelation.AccountRelationRoleCode {
		case types.IdentityActorTypeCustomer:
			actorType = types.IdentityActorTypeCustomer
		case types.IdentityActorTypeSupplier:
			actorType = types.IdentityActorTypeSupplier
		default:
			return nil, tracing.Trace(span, contracts.NewAuthenticationError(token.ErrInvalidJWT))
		}

		return buildRelatedUserIdentity(userModel, accountRelation, actorType, targetAccountID), nil
	}

	// The user is associated with the target account, proceed
	err = s.accountUserMed.MarkUsedIfNotRecent(ctx, accountUser)
	if err != nil {
		return nil, err
	}

	// Fetch the permissions for the role that the user is associated with
	permissions := map[string]bool{}
	if accountUser.RoleID != nil {
		rolePermissions, err := deps.rolePermissionRepo.FindByRoleID(ctx, *accountUser.RoleID)
		if err != nil {
			return nil, err
		}
		permissions = rolePermissions
	}

	return buildAccountUserIdentity(userModel, accountUser, permissions, targetAccountID), nil
}

func (s *userMedImpl) findUserByToken(ctx context.Context, accessToken string) (*types.User, *contracts.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.find_by_token")
	defer span.End()

	// Decode the access token into claims
	authToken, err := s.jwtUtils.Decode(ctx, accessToken, domain.JWTTypeAccess)
	if err != nil {
		return nil, err
	}

	// Validate the token is not expired
	if authToken.ExpiresAt.Before(time.Now().UTC()) {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrAccessTokenExpired))
	}

	// Get the user
	userRepo := s.repos.NewUserRepo()
	userModel, err := userRepo.Find(ctx, authToken.Subject)

	if err != nil || userModel == nil {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(ErrAccessTokenInvalid))
	}

	return userModel, nil
}

type credentialValidationDeps struct {
	accountRelationRepo domain.AccountRelationRepo
	accountUserRepo     domain.AccountUserRepo
	rolePermissionRepo  domain.RolePermissionRepo
}
