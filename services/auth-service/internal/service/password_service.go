package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var passwordSvcTracer = tracing.GetTracer("auth-service.password_service")

type passwordSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type PasswordSvcConfig struct {
	// Repos (required) is the repository factory for auth persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher

	// TxManager (optional; default: nil) wraps multi-step operations in database transactions. It is not validated at construction; transactional code paths panic at runtime if it is unset.
	TxManager TransactionManager
}

func (c *PasswordSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("password service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("password service: mediator factory is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("password service: notification publisher is required")
	}
	return nil
}

func NewPasswordSvc(config *PasswordSvcConfig) domain.PasswordSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &passwordSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func (c *PasswordSvcConfig) WithDefaults(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) *PasswordSvcConfig {
	if c == nil {
		c = &PasswordSvcConfig{}
	}

	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &PasswordSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func (s *passwordSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *passwordSvcImpl) withTx(ctx context.Context, fn func(context.Context, *passwordSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil") // Should never happen
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &passwordSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// RequestPasswordReset initiates a password reset flow for the given identifier.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Delegate to the password mediator's RequestReset inside a transaction.
// 3. Cache the success response.
//
// Side effects:
//   - Sends a password reset email if the identifier matches a known user.
func (s *passwordSvcImpl) RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError {
	ctx, span := passwordSvcTracer.Start(ctx, "service.password.request_password_reset")
	defer span.End()

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *passwordSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			if err := txMeds.Password.RequestReset(txCtx, identifier, accountSlug); err != nil {
				return err
			}
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// ResetPassword validates a password reset token and sets a new password, returning auth tokens.
//
//  1. Upsert an idempotency key; return the cached response if already finished.
//  2. Validate the password reset token and retrieve the associated user.
//  3. Mint an access token for the user.
//  4. Update the password, revoke all existing refresh tokens, and create a new refresh token
//     inside a transaction.
//  5. Cache the success response and return the login result.
//
// Side effects:
//   - Revokes all existing refresh tokens for the user.
//   - Sends a password updated email.
func (s *passwordSvcImpl) ResetPassword(ctx context.Context, token, newPassword string) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := passwordSvcTracer.Start(ctx, "service.password.reset_password")
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
		user, apiErr := meds.Password.ValidatePasswordResetToken(ctx, token)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		accessToken, apiErr := meds.User.GenAuthAccessToken(ctx, user.ID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var refreshToken string
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *passwordSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			err := txMeds.Password.Update(txCtx, user, newPassword)
			if err != nil {
				return err
			}

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

		result := &domain.LoginResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// UpdatePassword changes a user's password after verifying the old password.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Validate the old password via the password mediator.
// 3. Update to the new password inside a transaction.
// 4. Cache the success response.
//
// Side effects:
//   - Revokes all existing refresh tokens for the user.
//   - Sends a password updated email.
func (s *passwordSvcImpl) UpdatePassword(ctx context.Context, oldPassword, newPassword string) *apierror.APIError {
	ctx, span := passwordSvcTracer.Start(ctx, "service.password.update_password")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	// Account-agnostic: users without an assigned account (e.g. mid
	// registration) may still change their password, so require an
	// authenticated user actor rather than an assigned account.
	if apiErr := identity.CheckHasUserActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		user, apiErr := meds.Password.Validate(ctx, identity.Actor.ID, oldPassword)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *passwordSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			if err := txMeds.Password.Update(txCtx, user, newPassword); err != nil {
				return err
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeUser,
				ResourceID:   identity.Actor.ID,
				Metadata:     map[string]any{"password_rotated": true},
			}); apiErr != nil {
				return apiErr
			}

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}
