package mediator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
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
	jwtSecret             string
	frontendURL           string
	refreshTokenMed       domain.RefreshTokenMed
	apiKeyMed             domain.APIKeyMed
	coreClient            domain.AuthCoreClient
	notificationPublisher domain.NotificationPublisher
}

type UserMedConfig struct {
	Repos                 domain.RepoFactory
	JWTSecret             string // #nosec G117 - Struct field, not a hardcoded credential
	FrontendURL           string
	RefreshTokenMed       domain.RefreshTokenMed
	APIKeyMed             domain.APIKeyMed
	CoreClient            domain.AuthCoreClient
	NotificationPublisher domain.NotificationPublisher
}

func (c *UserMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("user mediator: repos is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("user mediator: jwt secret is required")
	}
	if c.RefreshTokenMed == nil {
		return fmt.Errorf("user mediator: refresh token mediator is required")
	}
	if c.APIKeyMed == nil {
		return fmt.Errorf("user mediator: api key mediator is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("user mediator: core client is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("user mediator: notification publisher is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("user mediator: frontend url is required")
	}
	return nil
}

func NewUserMed(config *UserMedConfig) domain.UserMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &userMedImpl{
		repos:                 config.Repos,
		jwtSecret:             config.JWTSecret,
		frontendURL:           config.FrontendURL,
		refreshTokenMed:       config.RefreshTokenMed,
		apiKeyMed:             config.APIKeyMed,
		coreClient:            config.CoreClient,
		notificationPublisher: config.NotificationPublisher,
	}
}

// GenAuthAccessToken mints an access token that can be used to authenticate requests to the API.
func (s *userMedImpl) GenAuthAccessToken(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.gen_auth_access_token")
	defer span.End()

	return token.EncodeJWT(ctx, s.jwtSecret, userID, time.Hour, token.JWTTypeAccess)
}

// Register registers a new user with the given input.
//
// 1. Check if a user with the given email already exists; return a validation error if so.
// 2. Generate a unique user ID.
// 3. Create the user record in the repository.
// 4. Send a welcome email if the user has an email and name.
//
// Side effects:
//   - Sends a welcome email.
func (s *userMedImpl) Register(ctx context.Context, input domain.RegisterUserInput) (*types.User, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "domain.auth.register")
	defer span.End()

	userRepo := s.repos.NewUserRepo()

	existingUser, err := userRepo.Find(ctx, input.Email)
	if err != nil && !apierror.IsNotFound(err) { // These will be 500 errors
		return nil, err
	}

	// If the user already exists, return a validation error to not reveal that an email is taken.
	// The email is sent at the service layer outside the transaction so the outbox message is not rolled back.
	if existingUser != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Unable to process registration. If you already have an account, we will email you a magic login link."))
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
		publishCtx := event.WithRepos(ctx, s.repos)
		s.notificationPublisher.PublishSendEmail(
			publishCtx,
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

// ValidateMagicLoginToken validates a magic-login token and returns the associated user.
func (s *userMedImpl) ValidateMagicLoginToken(ctx context.Context, magicToken string) (*types.User, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.validate_magic_login_token")
	defer span.End()

	claims, err := token.DecodeJWT(ctx, s.jwtSecret, magicToken, token.JWTTypeMagicLogin)
	if err != nil {
		return nil, err
	}

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, claims.Subject)
	if err != nil {
		if apierror.IsNotFound(err) {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(token.ErrInvalidJWT))
		}
		return nil, err
	}

	return user, nil
}

// SendAlreadyRegisteredEmail generates a magic login token and sends the
// "already registered" email so the user can log in with one click.
// This must be called outside a transaction so the outbox message is not rolled back.
func (s *userMedImpl) SendAlreadyRegisteredEmail(ctx context.Context, user *types.User, accountSlug *string) {
	if user.Email == nil {
		return
	}

	magicToken, err := token.EncodeJWT(ctx, s.jwtSecret, user.ID, 15*time.Minute, token.JWTTypeMagicLogin)
	if err != nil {
		return
	}

	var loginURL string
	if accountSlug != nil && *accountSlug != "" {
		loginURL = fmt.Sprintf("%s/%s%s?t=%s", s.frontendURL, *accountSlug, constants.DashboardPathMagicLogin, magicToken)
	} else {
		loginURL = fmt.Sprintf("%s%s?t=%s", s.frontendURL, constants.DashboardPathMagicLogin, magicToken)
	}

	var userName string
	if user.Name != nil {
		userName = *user.Name
	}

	publishCtx := event.WithRepos(ctx, s.repos)
	s.notificationPublisher.PublishSendEmail(
		publishCtx,
		messaging.EmailSendData{
			To:         []string{*user.Email},
			Subject:    "You Already Have an Account",
			TemplateID: constants.EmailTemplateAlreadyRegistered,
			Params: map[string]any{
				"LoginURL": loginURL,
				"UserName": userName,
			},
			SentByID: &user.ID,
		},
	)
}

// ValidateCredential validates credentials provided by a request and returns an identity.
//
//  1. If authToken is empty, resolve the account mode for the target account (if provided)
//     and return an unauthenticated identity.
//  2. If authToken has the API key prefix, delegate to validateAPIKeyCredential.
//  3. Otherwise, delegate to validateUserCredential for JWT-based validation.
//
// Behavior:
//   - If authToken is empty, returns an unauthenticated identity.
//   - If authToken is an API key credential, validates it as an API key.
//   - Otherwise, validates it as a user credential (JWT).
func (s *userMedImpl) ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError) {
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
					return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(*targetAccountID)))
				}
				return nil, err
			}
			if accountCtx == nil {
				return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(*targetAccountID)))
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

	// Otherwise, validate it as a user credential.
	// User requests always include Augno-Actor-Account to identify the
	// user's own account. When it matches the target the user is internal;
	// when it differs the user is accessing cross-account (customer/supplier).
	if actorAccountID != nil {
		// Validate the user is a member of their actor account.
		identity, apiErr := s.validateUserCredential(ctx, span, authToken, actorAccountID, true)
		if apiErr != nil {
			return nil, apiErr
		}

		// Same account — internal user, target is already correct.
		if targetAccountID == nil || *actorAccountID == *targetAccountID {
			return identity, nil
		}

		// Cross-account — look up the relation to determine customer/supplier.
		// Pass the validated actor account so the owner-side lookup is constrained to it;
		// without this, any account the user belongs to that owns a relation to the target
		// would match and the caller could keep an unrelated actor's permissions.
		accountRelation, hasRelation, apiErr := s.coreClient.GetAccountRelationByUserID(ctx, *targetAccountID, *actorAccountID, identity.Actor.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		if !hasRelation {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(*targetAccountID)))
		}

		var actorType types.IdentityRelationType
		switch accountRelation.AccountRelationRoleCode {
		case types.IdentityRelationTypeCustomer:
			actorType = types.IdentityRelationTypeCustomer
		case types.IdentityRelationTypeSupplier:
			actorType = types.IdentityRelationTypeSupplier
		default:
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(*targetAccountID)))
		}

		// Owner-side: the user's actor account owns the relation (e.g. merchant targeting customer).
		// Keep the actor's own account permissions. The SQL constraint above guarantees the
		// relation's owner equals the supplied actor account, so retaining actor permissions is safe.
		if accountRelation.IsOwnerSide {
			relationType := accountRelation.AccountRelationRoleCode
			identity.Target.AccountID = *targetAccountID
			identity.Target.RelationType = &relationType
			return identity, nil
		}

		// Counterparty-side: the user belongs to the counterparty (e.g. customer user targeting merchant).
		// Strip actor permissions since they don't apply to the owner's account.
		identity.Actor.RelationType = actorType
		identity.Actor.RoleID = nil
		identity.Actor.RoleType = nil
		identity.Actor.Permissions = map[string]bool{}
		identity.Target.AccountID = *targetAccountID
		identity.Target.RelationType = &accountRelation.AccountRelationRoleCode

		return identity, nil
	}

	// No actor account — validate directly against the target.
	return s.validateUserCredential(ctx, span, authToken, targetAccountID, false)
}

func (s *userMedImpl) validateAPIKeyCredential(ctx context.Context, span trace.Span, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError) {
	// Find and validate the API key
	apiKeyModel, err := s.apiKeyMed.FindAndValidate(ctx, authToken)
	if err != nil {
		return nil, err
	}

	// Touch the API key to mark it as used
	go func() {
		_ = s.apiKeyMed.TouchIfNotRecent(context.Background(), apiKeyModel)
	}()

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

	// Fetch account context for subscription status
	accountCtx, err := s.coreClient.GetAccountContext(ctx, finalTargetAccountID)
	if err != nil {
		if err.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
		}
		return nil, err
	}

	// The request targets the account that owns the API key
	if apiKeyModel.OwnerAccountID == finalTargetAccountID {
		// Fetch the user account access (which includes permissions) from account service
		// For API keys, we use a special lookup that gets role permissions
		access, err := s.apiKeyMed.GetKeyAccountAccess(ctx, domain.APIKeyGetAccountAccessInput{
			AccountMode:     accountMode,
			APIKeyID:        apiKeyModel.ID,
			TargetAccountID: finalTargetAccountID,
		})
		if err != nil {
			if err.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
			}
			return nil, err
		}

		// If no access found via account service, the API key inherently has access to its owner account
		permissions := map[string]bool{}
		if access != nil {
			permissions = access.Permissions
		}

		return buildOwnedAPIKeyIdentity(apiKeyModel, finalTargetAccountID, permissions, accountMode, accountCtx.SubscriptionStatus), nil
	}

	// The request targets a different account, so we need to find the account relation
	accountRelation, hasRelation, err := s.coreClient.GetAccountRelationByAPIKeyID(ctx, finalTargetAccountID, apiKeyModel.ID)
	if err != nil {
		return nil, err
	}

	// These accounts have no relationship, request should fail
	if !hasRelation {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
	}

	// Owner-side: the API key's account owns the relation (e.g. merchant targeting customer).
	// Treat as internal actor with the API key's own account permissions.
	if accountRelation.IsOwnerSide {
		access, err := s.apiKeyMed.GetKeyAccountAccess(ctx, domain.APIKeyGetAccountAccessInput{
			AccountMode:     accountMode,
			APIKeyID:        apiKeyModel.ID,
			TargetAccountID: apiKeyModel.OwnerAccountID,
		})
		if err != nil {
			if err.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
			}
			return nil, err
		}

		permissions := map[string]bool{}
		if access != nil {
			permissions = access.Permissions
		}

		relationType := accountRelation.AccountRelationRoleCode
		return buildOwnedAPIKeyIdentityWithRelation(apiKeyModel, finalTargetAccountID, permissions, accountMode, accountCtx.SubscriptionStatus, &relationType), nil
	}

	// Counterparty-side: the API key's account is the counterparty (e.g. customer targeting merchant).
	var actorType types.IdentityRelationType
	switch accountRelation.AccountRelationRoleCode {
	case types.IdentityRelationTypeCustomer:
		actorType = types.IdentityRelationTypeCustomer
	case types.IdentityRelationTypeSupplier:
		actorType = types.IdentityRelationTypeSupplier
	default:
		return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Failed to find account relation."))
	}

	return buildRelatedAPIKeyIdentity(apiKeyModel, accountRelation, actorType, finalTargetAccountID, accountMode, accountCtx.SubscriptionStatus), nil
}

func (s *userMedImpl) validateUserCredential(ctx context.Context, span trace.Span, authToken string, targetAccountID *string, requireAccountUser bool) (*types.Identity, *apierror.APIError) {
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
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
		}
		return nil, err
	}
	accountMode := accountCtx.AccountMode

	// Get the user's access to this account from account-service
	access, hasAccess, err := s.coreClient.GetUserAccountAccess(ctx, userModel.ID, finalTargetAccountID)
	if err != nil {
		return nil, err
	}

	// When requireAccountUser is set (e.g. Augno-Actor-Account header), the user must be an account member.
	if requireAccountUser && !hasAccess {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(ErrActorAccountRequiresMember))
	}

	// This user isn't associated with the target account, but they may have a relationship with it
	if !hasAccess {
		// No actor account is supplied on this path, so owner-side relations must not match.
		// Passing "" as actorAccountID skips the owner-side fallback in core-service; we also
		// reject defensively below in case the contract ever changes.
		accountRelation, hasRelation, err := s.coreClient.GetAccountRelationByUserID(ctx, finalTargetAccountID, "", userModel.ID)
		if err != nil {
			return nil, err
		}

		// These accounts have no relationship, request should fail
		if !hasRelation {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
		}

		// Owner-side relations have no validated actor account to bind to; the related-user
		// identity builder would mis-attribute the actor account to the counterparty.
		if accountRelation.IsOwnerSide {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
		}

		// Determine the actor type based on the account relation role code
		var actorType types.IdentityRelationType
		switch accountRelation.AccountRelationRoleCode {
		case types.IdentityRelationTypeCustomer:
			actorType = types.IdentityRelationTypeCustomer
		case types.IdentityRelationTypeSupplier:
			actorType = types.IdentityRelationTypeSupplier
		default:
			return nil, tracing.Trace(span, apierror.NewAuthorizationError(errNoAccountAccess(finalTargetAccountID)))
		}

		return buildRelatedUserIdentity(userModel, accountRelation, actorType, finalTargetAccountID, accountMode, accountCtx.SubscriptionStatus), nil
	}

	// The user is associated with the target account, mark as used if not recent
	// Fire and forget - don't block on this
	go func() {
		_ = s.coreClient.MarkAccountUserUsed(context.Background(), access.AccountUserID)
	}()

	return buildAccountUserIdentity(userModel, access, finalTargetAccountID, accountMode, accountCtx.SubscriptionStatus), nil
}

func (s *userMedImpl) findUserByToken(ctx context.Context, accessToken string) (*types.User, *apierror.APIError) {
	ctx, span := userMedTracer.Start(ctx, "mediator.user.find_by_token")
	defer span.End()

	// Decode the access token into claims
	authToken, err := token.DecodeJWT(ctx, s.jwtSecret, accessToken, token.JWTTypeAccess)
	if err != nil {
		return nil, err
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
