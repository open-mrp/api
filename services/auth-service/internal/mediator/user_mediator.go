package mediator

import (
	"context"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	tracing "github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

var userMedTracer = tracing.GetTracer("auth-service.user_mediator")

type userMedImpl struct {
	repos                 domain.RepoFactory
	jwtUtils              domain.JWTUtils
	refreshTokenMed       domain.RefreshTokenMed
	apiKeyMed             domain.APIKeyMed
	coreClient            domain.AuthCoreClient
	notificationPublisher domain.NotificationPublisher
}

type UserMedConfig struct {
	Repos                 domain.RepoFactory
	JWTSecret             string // #nosec G117 - Struct field, not a hardcoded credential
	RefreshTokenMed       domain.RefreshTokenMed
	APIKeyMed             domain.APIKeyMed
	CoreClient            domain.AuthCoreClient
	NotificationPublisher domain.NotificationPublisher
}

func NewUserMed(config UserMedConfig) domain.UserMed {
	if config.JWTSecret == "" {
		panic("JWTSecret is not set in the config.")
	}

	if config.NotificationPublisher == nil {
		panic("NotificationPublisher is not set in the config.")
	}

	return &userMedImpl{
		repos:                 config.Repos,
		jwtUtils:              token.NewJWTUtils(&token.JWTConfig{Secret: config.JWTSecret}),
		refreshTokenMed:       config.RefreshTokenMed,
		apiKeyMed:             config.APIKeyMed,
		coreClient:            config.CoreClient,
		notificationPublisher: config.NotificationPublisher,
	}
}

// GenAuthAccessToken mints an access token that can be utilized to authenticate requests to the API.
func (s *userMedImpl) GenAuthAccessToken(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.gen_auth_access_token")
	defer span.End()

	return s.jwtUtils.Encode(ctx, userID, time.Hour, domain.JWTTypeAccess)
}

// GenPasswordResetAccessToken mints an access token that can be utilized to reset the password for the given user ID.
func (s *userMedImpl) GenPasswordResetAccessToken(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.gen_password_reset_access_token")
	defer span.End()

	return s.jwtUtils.Encode(ctx, userID, 15*time.Minute, domain.JWTTypePasswordReset)
}

func (s *userMedImpl) Register(ctx context.Context, input domain.RegisterUserInput) (*types.User, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "domain.auth.register")
	defer span.End()

	userRepo := s.repos.NewUserRepo()

	existingUser, err := userRepo.Find(ctx, input.Email)
	if err != nil && !apierror.IsNotFound(err) {
		return nil, err
	}
	if existingUser != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Unable to process registration."))
	}

	userID, err := id.GenID(id.UserIDPrefix, nil)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "failed to generate user id"))
	}

	user, err := userRepo.Create(ctx, userID, input.Email, input.Name, input.HashedPassword)
	if err != nil {
		return nil, err
	}

	if user.Email != nil && user.Name != nil {
		s.notificationPublisher.PublishSendEmail(
			ctx,
			messaging.EmailSendData{
				To:         []string{*user.Email},
				Subject:    "Welcome!",
				TemplateID: constants.EmailTemplateWelcome,
				Params: map[string]any{
					"UserName":  *user.Name,
					"UserEmail": *user.Email,
				},
				SentByID: &user.ID,
			},
		)
	}

	return user, nil
}

// ValidateCredential validates a credentials provided by a request and returns an identity.
func (s *userMedImpl) ValidateCredential(ctx context.Context, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.validate_credential")
	defer span.End()

	// If the auth token is empty, we return an unauthenticated identity
	if authToken == "" {
		// For unauthenticated requests, check if target is a sandbox
		accountMode := constants.AccountModeProduction
		if targetAccountID != nil {
			accountCtx, err := s.coreClient.GetAccountContext(ctx, *targetAccountID)
			if err != nil {
				if err.Code == apierror.ErrorCodeResourceNotFound {
					return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
				}
				return nil, err
			}
			if accountCtx == nil {
				return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
			}

			accountMode = accountCtx.AccountMode
		}

		return types.GetUnauthenticatedIdentityWithMode(targetAccountID, accountMode), nil
	}

	// If the auth token has the API key prefix, validate it as such
	// API keys already encode their mode (prod/test) in the key prefix
	if strings.HasPrefix(authToken, string(types.APIKeyPrefixSecretKey)) {
		return s.validateAPIKeyCredential(ctx, span, authToken, targetAccountID)
	}

	// Otherwise, validate it as a user credential
	return s.validateUserCredential(ctx, span, authToken, targetAccountID)
}

func (s *userMedImpl) validateAPIKeyCredential(ctx context.Context, span trace.Span, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError) {
	// Find and validate the API key
	apiKeyModel, err := s.apiKeyMed.FindAndValidate(ctx, authToken)
	if err != nil {
		return nil, err
	}

	// Parse the API key to get the account mode (embedded in the key)
	parsedKey, err := s.apiKeyMed.ParseKey(ctx, authToken)
	if err != nil {
		return nil, err
	}
	accountMode := parsedKey.AccountMode

	var finalTargetAccountID string
	// If no target account specified, default to the API key's owner account
	if targetAccountID == nil {
		finalTargetAccountID = apiKeyModel.OwnerAccountID
	} else {
		finalTargetAccountID = *targetAccountID
	}

	// The request targets the account that owns the API key
	if apiKeyModel.OwnerAccountID == finalTargetAccountID {
		// Touch the API key to mark it as used
		err := s.apiKeyMed.TouchIfNotRecent(ctx, apiKeyModel)
		if err != nil {
			return nil, err
		}

		// Fetch the user account access (which includes permissions) from account service
		// For API keys, we use a special lookup that gets role permissions
		access, err := s.apiKeyMed.GetKeyAccountAccess(ctx, accountMode, apiKeyModel.ID, finalTargetAccountID)
		if err != nil {
			if err.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
			}
			return nil, err
		}

		// If no access found via account service, the API key inherently has access to its owner account
		permissions := map[string]bool{}
		if access != nil {
			permissions = access.Permissions
		}

		return buildOwnedAPIKeyIdentity(apiKeyModel, finalTargetAccountID, permissions, accountMode), nil
	}

	// The request targets a different account, so we need to find the account relation
	accountRelation, err := s.coreClient.GetAccountRelationByAPIKeyID(ctx, finalTargetAccountID, apiKeyModel.ID)
	if err != nil {
		if err.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
		}
		return nil, err
	}

	// These accounts have no relationship, request should fail
	if accountRelation == nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(token.ErrInvalidJWT))
	}

	// Determine the actor type based on the account relation role code
	var actorType types.IdentityActorType
	switch accountRelation.AccountRelationRoleCode {
	case types.IdentityActorTypeCustomer:
		actorType = types.IdentityActorTypeCustomer
	case types.IdentityActorTypeSupplier:
		actorType = types.IdentityActorTypeSupplier
	default:
		return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Failed to find account relation."))
	}

	return buildRelatedAPIKeyIdentity(apiKeyModel, accountRelation, actorType, finalTargetAccountID, accountMode), nil
}

func (s *userMedImpl) validateUserCredential(ctx context.Context, span trace.Span, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError) {
	// Find the user by the token
	userModel, err := s.findUserByToken(ctx, authToken)
	if err != nil {
		return nil, err
	}

	var finalTargetAccountID string
	if targetAccountID == nil {
		return buildUnassignedUserIdentity(userModel), nil
	} else {
		finalTargetAccountID = *targetAccountID
	}

	// Get the account context to determine if this is a sandbox (cached)
	accountCtx, err := s.coreClient.GetAccountContext(ctx, finalTargetAccountID)
	if err != nil {
		if err.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
		}
		return nil, err
	}
	accountMode := accountCtx.AccountMode

	// Get the user's access to this account from account-service
	access, err := s.coreClient.GetUserAccountAccess(ctx, userModel.ID, finalTargetAccountID)
	if err != nil {
		if err.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
		}
		return nil, err
	}

	// This user isn't associated with the target account, but they may have a relationship with it
	if access == nil {
		// Find the account relation by the owner account and user
		accountRelation, err := s.coreClient.GetAccountRelationByUserID(ctx, finalTargetAccountID, userModel.ID)
		if err != nil {
			if err.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewAuthorizationError(ErrNoAccountAccess))
			}
			return nil, err
		}

		// These accounts have no relationship, request should fail
		if accountRelation == nil {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(token.ErrInvalidJWT))
		}

		// Determine the actor type based on the account relation role code
		var actorType types.IdentityActorType
		switch accountRelation.AccountRelationRoleCode {
		case types.IdentityActorTypeCustomer:
			actorType = types.IdentityActorTypeCustomer
		case types.IdentityActorTypeSupplier:
			actorType = types.IdentityActorTypeSupplier
		default:
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(token.ErrInvalidJWT))
		}

		return buildRelatedUserIdentity(userModel, accountRelation, actorType, finalTargetAccountID, accountMode), nil
	}

	// The user is associated with the target account, mark as used if not recent
	// Fire and forget - don't block on this
	go func() {
		_ = s.coreClient.MarkAccountUserUsed(context.Background(), access.AccountUserID)
	}()

	return buildAccountUserIdentity(userModel, access, finalTargetAccountID, accountMode), nil
}

func (s *userMedImpl) findUserByToken(ctx context.Context, accessToken string) (*types.User, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.find_by_token")
	defer span.End()

	// Decode the access token into claims
	authToken, err := s.jwtUtils.Decode(ctx, accessToken, domain.JWTTypeAccess)
	if err != nil {
		return nil, err
	}

	// Validate the token is not expired
	if authToken.ExpiresAt.Before(time.Now().UTC()) {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrAccessTokenExpired))
	}

	// Get the user
	userRepo := s.repos.NewUserRepo()
	userModel, err := userRepo.Find(ctx, authToken.Subject)

	if err != nil {
		if apierror.IsNotFound(err) {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrAccessTokenInvalid))
		}
		return nil, tracing.Trace(span, err)
	}

	return userModel, nil
}
