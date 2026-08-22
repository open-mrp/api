package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/event"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/auth-service/internal/mediator"
	pwdutil "github.com/open-mrp/api/services/auth-service/internal/password"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var userSvcTracer = tracing.GetTracer("auth-service.user_service")

type userSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type UserSvcConfig struct {
	// Repos (required) is the repository factory for auth persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher

	// TxManager (optional; default: nil) wraps multi-step operations in database transactions. It is not validated at construction; transactional code paths panic at runtime if it is unset.
	TxManager TransactionManager
}

func (c *UserSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("user service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("user service: mediator factory is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("user service: notification publisher is required")
	}
	return nil
}

func NewUserSvc(config *UserSvcConfig) domain.UserSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &userSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func BuildUserSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) *UserSvcConfig {
	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &UserSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func (s *userSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *userSvcImpl) withTx(ctx context.Context, fn func(context.Context, *userSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil") // Should never happen
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &userSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// Login authenticates a user by identifier and password and returns auth tokens.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Validate the identifier/password combination via the password mediator.
// 3. Mint an access token for the authenticated user.
// 4. Create a refresh token inside a transaction.
// 5. Cache the success response and return the login result.
func (s *userSvcImpl) Login(ctx context.Context, identifier, password string) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.login")
	defer span.End()

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.LoginResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		user, apiErr := meds.Password.Validate(ctx, identifier, password)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		accessToken, apiErr := meds.User.GenAuthAccessToken(ctx, user.ID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var refreshToken string
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			refreshTokenModel, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
			if err != nil {
				return err
			}
			refreshToken = refreshTokenModel.Token

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &domain.LoginResult{
				User:         user,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			})
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return &domain.LoginResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// Register creates a new user account and returns auth tokens.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Hash the provided password.
// 3. Register the user via the user mediator inside a transaction.
// 4. Create a refresh token and mint an access token.
// 5. Cache the success response and return the login result.
//
// Side effects:
//   - Sends a welcome email (via the user mediator).
func (s *userSvcImpl) Register(ctx context.Context, input domain.RegisterInput) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.register")
	defer span.End()

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.LoginResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Check for existing user before the transaction so we can send the magic-login email outside the tx (outbox writes inside a rolled-back tx are lost).
		userRepo := s.repos.NewUserRepo()
		existingUser, findErr := userRepo.Find(ctx, input.Email)
		if findErr != nil && !apierror.IsNotFound(findErr) {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, findErr)
		}
		if existingUser != nil {
			publishCtx := event.WithRepos(ctx, s.repos)
			meds.User.SendAlreadyRegisteredEmail(publishCtx, existingUser, input.AccountSlug, input.PortalBaseURL)
			regErr := apierror.NewValidationError("Unable to process registration. If you already have an account, we will email you a magic login link.")
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, regErr)
		}

		hashedPassword, err := pwdutil.HashPassword(ctx, input.Password)
		if err != nil {
			apiErr := apierror.NewInternalError(err, "failed to hash password")
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var result *domain.LoginResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			user, regErr := txMeds.User.Register(txCtx, domain.RegisterUserInput{
				Name:           input.Name,
				Email:          input.Email,
				HashedPassword: hashedPassword,
				AccountSlug:    input.AccountSlug,
			})
			if regErr != nil {
				return regErr
			}

			refreshTokenModel, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
			if err != nil {
				return err
			}

			accessToken, tokenErr := txMeds.User.GenAuthAccessToken(txCtx, user.ID)
			if tokenErr != nil {
				return tokenErr
			}

			result = &domain.LoginResult{
				User:         user,
				AccessToken:  accessToken,
				RefreshToken: refreshTokenModel.Token,
			}

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// MagicLogin exchanges a magic-login token for auth tokens, logging the user in without a password.
//
//  1. Decode and validate the magic-login JWT.
//  2. Look up the user by the token's subject.
//  3. Mint an access token and create a refresh token.
func (s *userSvcImpl) MagicLogin(ctx context.Context, magicToken string) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.magic_login")
	defer span.End()

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.LoginResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Error != nil {
			return nil, cached.Error
		}
		return cached.Data, nil

	case domain.RecoveryPointStarted:
		user, apiErr := meds.User.ValidateMagicLoginToken(ctx, magicToken)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		accessToken, apiErr := meds.User.GenAuthAccessToken(ctx, user.ID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var refreshToken string
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			refreshTokenModel, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
			if err != nil {
				return err
			}
			refreshToken = refreshTokenModel.Token

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, domain.LoginResult{
				User:         user,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			})
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return &domain.LoginResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}
