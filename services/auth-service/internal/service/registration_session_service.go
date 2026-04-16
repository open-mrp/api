package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var registrationSessionSvcTracer = tracing.GetTracer("auth-service.registration_session_service")

type registrationSessionSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
	billingClient         domain.AuthBillingClient
	coreClient            domain.AuthCoreClient
	frontendURL           string
}

type RegistrationSessionSvcConfig struct {
	Repos                 domain.RepoFactory
	MediatorFactory       domain.MediatorFactory
	NotificationPublisher domain.NotificationPublisher
	TxManager             TransactionManager
	BillingClient         domain.AuthBillingClient
	CoreClient            domain.AuthCoreClient
	FrontendURL           string
}

func (c *RegistrationSessionSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("registration session service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("registration session service: mediator factory is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("registration session service: notification publisher is required")
	}
	return nil
}

func NewRegistrationSessionSvc(config *RegistrationSessionSvcConfig) domain.RegistrationSessionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &registrationSessionSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
		billingClient:         config.BillingClient,
		coreClient:            config.CoreClient,
		frontendURL:           config.FrontendURL,
	}
}

func (c *RegistrationSessionSvcConfig) WithDefaults(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient, billingClient domain.AuthBillingClient) *RegistrationSessionSvcConfig {
	if c == nil {
		c = &RegistrationSessionSvcConfig{}
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

	return &RegistrationSessionSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
		BillingClient:         billingClient,
		CoreClient:            coreClient,
		FrontendURL:           frontendURL,
	}
}

func (s *registrationSessionSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *registrationSessionSvcImpl) withTx(ctx context.Context, fn func(context.Context, *registrationSessionSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &registrationSessionSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
			billingClient:         s.billingClient,
			coreClient:            s.coreClient,
			frontendURL:           s.frontendURL,
		}
		return fn(txCtx, txSvc)
	})
}

// CreateSession creates a new registration session or returns an existing active session
// for the given email (idempotent).
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Delegate to the registration mediator inside a transaction.
// 3. Cache the success response and return the session ID.
//
// Side effects:
//   - Sends a verification email to the user.
func (s *registrationSessionSvcImpl) CreateSession(ctx context.Context, input domain.CreateRegistrationSessionInput) (*domain.CreateRegistrationSessionResult, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.create")
	defer span.End()

	meds := s.mediators()

	// Unauthenticated endpoint — no identity
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CreateRegistrationSessionResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.CreateRegistrationSessionResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			var createErr *apierror.APIError
			result, createErr = txMeds.Registration.CreateSession(txCtx, input)
			if createErr != nil {
				return createErr
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

// GetIncompleteByUserID returns the most recent incomplete registration session
// for the given user, or (nil, nil) if none exists.
func (s *registrationSessionSvcImpl) GetIncompleteByUserID(ctx context.Context, userID string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.get_incomplete_by_user_id")
	defer span.End()

	session, apiErr := s.repos.NewRegistrationSessionRepo().GetIncompleteByUserID(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return session, nil
}

// GetSession returns the registration session for the given type ID.
//
// 1. Look up the session by its type ID in the repository.
func (s *registrationSessionSvcImpl) GetSession(ctx context.Context, sessionID string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.get")
	defer span.End()

	regSessionRepo := s.repos.NewRegistrationSessionRepo()
	session, apiErr := regSessionRepo.GetByTypeID(ctx, sessionID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return session, nil
}

// CreateUserForSession creates or resolves a user for the registration session and returns
// the user ID with auth tokens.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Delegate to the registration mediator inside a transaction.
// 3. Cache the success response and return the user ID, access token, and refresh token.
//
// Behavior:
//   - If the session already has a user, tokens are generated for the existing user (idempotent).
//
// Side effects:
//   - May create a new user record.
//   - Associates the user with the session.
//   - Advances the session step to account_details.
func (s *registrationSessionSvcImpl) CreateUserForSession(ctx context.Context, input domain.CreateUserForRegistrationInput) (*domain.CreateUserForRegistrationOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.create_user")
	defer span.End()

	meds := s.mediators()

	// Unauthenticated endpoint — no identity
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CreateUserForRegistrationOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.CreateUserForRegistrationOutput
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			var createErr *apierror.APIError
			result, createErr = txMeds.Registration.CreateUserForSession(txCtx, input)
			if createErr != nil {
				return createErr
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

// VerifyToken verifies the email verification token and marks the session as email-verified.
//
// 1. Delegate to the registration mediator's VerifyToken inside a transaction.
// 2. Return the updated registration session.
//
// Behavior:
//   - Advances the session step to user_details.
//   - Idempotent: if the session is already verified, returns the current session without error.
func (s *registrationSessionSvcImpl) VerifyToken(ctx context.Context, token string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.verify_token")
	defer span.End()

	var result *domain.RegistrationSession
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
		txMeds := txSvc.mediators()
		var verifyErr *apierror.APIError
		result, verifyErr = txMeds.Registration.VerifyToken(txCtx, token)
		return verifyErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *registrationSessionSvcImpl) validateSessionOwnership(ctx context.Context, sessionID string, identity *types.Identity) *apierror.APIError {
	regSessionRepo := s.repos.NewRegistrationSessionRepo()
	session, apiErr := regSessionRepo.GetByTypeID(ctx, sessionID)
	if apiErr != nil {
		return apiErr
	}
	if session.UserID != nil && *session.UserID != identity.Actor.ID {
		return apierror.NewAuthorizationError("Session does not belong to the authenticated user.")
	}
	return nil
}

// UpdateSession updates an in-progress registration session's step and form data.
//
// 1. Extract the identity from the context and require user authentication.
// 2. Validate that the session belongs to the authenticated user.
// 3. Upsert an idempotency key; return the cached response if already finished.
// 4. Delegate to the registration mediator inside a transaction.
// 5. Cache the success response and return the refreshed session.
func (s *registrationSessionSvcImpl) UpdateSession(ctx context.Context, input domain.UpdateRegistrationSessionInput) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.update")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityActorTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	if apiErr := s.validateSessionOwnership(ctx, input.SessionID, identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
		TargetAccountID: func() *string {
			if identity.Target != nil {
				return &identity.Target.AccountID
			}
			return nil
		}(),
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.RegistrationSession](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.RegistrationSession
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			var updateErr *apierror.APIError
			result, updateErr = txMeds.Registration.UpdateSession(txCtx, input.SessionID, input.Step, input.SessionData)
			if updateErr != nil {
				return updateErr
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

// ListSessions returns a paginated list of registration sessions for the authenticated user.
//
// 1. Extract the identity from the context and require user authentication.
// 2. Query the repository for sessions belonging to the authenticated user.
func (s *registrationSessionSvcImpl) ListSessions(ctx context.Context, input domain.ListRegistrationSessionsInput) (*domain.ListRegistrationSessionsResult, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityActorTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	regSessionRepo := s.repos.NewRegistrationSessionRepo()
	sessions, pageInfo, apiErr := regSessionRepo.ListByUserID(ctx, identity.Actor.ID, input.Cursor, input.Limit)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ListRegistrationSessionsResult{
		Sessions: sessions,
		PageInfo: pageInfo,
	}, nil
}

// ResendVerificationEmail regenerates the verification token and resends the verification email.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Delegate to the registration mediator inside a transaction.
// 3. Cache the success response.
//
// Side effects:
//   - Rotates the verification token.
//   - Sends a new verification email to the user.
func (s *registrationSessionSvcImpl) ResendVerificationEmail(ctx context.Context, sessionID string) *apierror.APIError {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.resend_verification_email")
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
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			if resendErr := txMeds.Registration.ResendVerificationEmail(txCtx, sessionID); resendErr != nil {
				return resendErr
			}
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &struct{}{})
		})
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// CompleteRegistration finalizes a registration session by creating the account via core-service.
//
// 1. Require user authentication and upsert an idempotency key.
// 2. Fetch the session and recover if accounts were already created on a prior attempt.
// 3. Validate session state (user exists, payment completed for paid plans, etc.).
// 4. Build the core-service input and call CompleteRegistration (point of no return).
// 5. Atomically mark the session complete and advance the recovery point.
// 6. Cache the final success response and return the account and sandbox IDs.
//
// Behavior:
//   - Uses a multi-phase recovery-point loop to safely resume after partial failures.
//   - If the registration limit is hit, queues the registration and sends an admin alert.
func (s *registrationSessionSvcImpl) CompleteRegistration(ctx context.Context, sessionID string) (*domain.CompleteRegistrationOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.complete_registration")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityActorTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
		TargetAccountID: func() *string {
			if identity.Target != nil {
				return &identity.Target.AccountID
			}
			return nil
		}(),
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recoveryPoint := idempotencyKey.RecoveryPoint
	var result *domain.CompleteRegistrationOutput

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[domain.CompleteRegistrationOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
			}
			return cached.Data, cached.Error

		case domain.RecoveryPointStarted:
			// 1. Fetch session
			session, getErr := meds.Registration.GetSession(ctx, sessionID)
			if getErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, getErr)
			}

			// 2. If accounts were already created for this session (by a prior request),
			// skip directly to billing phase.
			if session.AccountID != nil {
				sandboxID, sandboxErr := s.coreClient.GetSandboxAccountByOwner(ctx, *session.AccountID)
				if sandboxErr != nil {
					return nil, tracing.Trace(span, sandboxErr)
				}
				result = &domain.CompleteRegistrationOutput{
					AccountID: *session.AccountID,
					SandboxID: sandboxID,
				}
				recoveryPoint = domain.RecoveryPointCoreAccountCreated
				continue
			}

			if session.CompletedAt != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewResourceConflictError("Registration session is already completed."))
			}

			// 3. Validate session state
			if session.UserID == nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationError("User must be created before completing registration."))
			}
			if *session.UserID != identity.Actor.ID {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewAuthorizationError("Session does not belong to the authenticated user."))
			}
			if session.SessionData.AccountName == "" {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
					apierror.NewValidationError("Account name is required."))
			}

			isFreePlan := session.PlanCode == string(constants.PlanCodeFree)
			if !isFreePlan {
				if session.StripeCustomerID == nil {
					return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
						apierror.NewValidationError("Stripe customer must be created before completing registration."))
				}
				if !session.PaymentCompleted {
					return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
						apierror.NewValidationError("Payment must be completed before completing registration."))
				}
			}

			// 4. Pre-validate Stripe pricing plan before account creation
			if !isFreePlan {
				if validateErr := s.billingClient.ValidateStripePricingPlan(ctx, session.PlanCode); validateErr != nil {
					return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, validateErr)
				}
			}

			// 5. Build core-service input
			coreInput := domain.CompleteAccountRegistrationInput{
				UserID:      *session.UserID,
				PlanCode:    session.PlanCode,
				AccountName: session.SessionData.AccountName,
			}
			if session.StripeCustomerID != nil {
				coreInput.StripeCustomerID = *session.StripeCustomerID
			}
			if session.SessionData.BillingAddressLine1 != "" || session.SessionData.BillingAddressCountry != "" {
				coreInput.BusinessAddress = &domain.RegistrationAddress{
					Line1:      session.SessionData.BillingAddressLine1,
					Line2:      session.SessionData.BillingAddressLine2,
					City:       session.SessionData.BillingAddressCity,
					State:      session.SessionData.BillingAddressState,
					PostalCode: session.SessionData.BillingAddressPostalCode,
					Country:    session.SessionData.BillingAddressCountry,
				}
			}

			// 6. Call core-service to create the account (point of no return)
			accountResult, coreErr := s.coreClient.CompleteRegistration(ctx, coreInput)
			if coreErr != nil {
				if coreErr.Code == apierror.ErrorCodeRegistrationClosed {
					s.handleRegistrationLimitHit(ctx, session)
				}
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, coreErr)
			}

			result = &domain.CompleteRegistrationOutput{
				AccountID: accountResult.AccountID,
				SandboxID: accountResult.SandboxID,
			}

			// 7. Checkpoint: persist account_id to session and advance recovery point.
			// This prevents duplicate account creation on retry if billing fails.
			resultBody, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal account result."))
			}
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
				regRepo := txSvc.repos.NewRegistrationSessionRepo()
				if updateErr := regRepo.UpdateAccountID(txCtx, session.ID, result.AccountID); updateErr != nil {
					return updateErr
				}
				return txSvc.repos.NewIdempotencyKeyRepo().SetResponse(txCtx, idempotencyKey.TypeID, 200, resultBody, domain.RecoveryPointCoreAccountCreated)
			})
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			recoveryPoint = domain.RecoveryPointCoreAccountCreated

		case domain.RecoveryPointCoreAccountCreated:
			// Recover result from idempotency key cached response if entering from a retry.
			if result == nil {
				cached, err := idempotency.UnmarshalCachedResponse[domain.CompleteRegistrationOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
				if err != nil {
					return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling recovery point response."))
				}
				result = cached.Data
			}

			// Stamp the newly created account onto the identity so downstream
			// service calls (billing → core UpdateAccountSubscription → audit
			// publisher) see a Target and don't reject the request.
			if identity, ok := appctx.GetIdentityFromContext(ctx); ok && identity != nil {
				identity.Target = &types.IdentityTarget{AccountID: result.AccountID}
				ctx = appctx.WithIdentity(ctx, identity)
			}

			// Re-fetch session for billing fields.
			session, getErr := meds.Registration.GetSession(ctx, sessionID)
			if getErr != nil {
				return nil, tracing.Trace(span, getErr)
			}

			isFreePlan := session.PlanCode == string(constants.PlanCodeFree)

			// Set up billing profile and subscribe for paid accounts.
			// Stripe calls use idempotency keys so retries are safe.
			if !isFreePlan && session.StripeCustomerID != nil {
				if _, profileErr := s.billingClient.SetupBillingProfile(ctx, result.AccountID); profileErr != nil {
					return nil, tracing.Trace(span, profileErr)
				}
				if subErr := s.billingClient.SubscribeToPricingPlan(ctx, *session.StripeCustomerID, session.PlanCode); subErr != nil {
					return nil, tracing.Trace(span, subErr)
				}
			}

			// Atomically: mark session complete + advance recovery point.
			resultBody, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal account result."))
			}
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
				if completeErr := txSvc.mediators().Registration.CompleteSession(txCtx, sessionID, result.AccountID); completeErr != nil {
					return completeErr
				}
				return txSvc.repos.NewIdempotencyKeyRepo().SetResponse(txCtx, idempotencyKey.TypeID, 200, resultBody, domain.RecoveryPointAccountsCreated)
			})
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			recoveryPoint = domain.RecoveryPointAccountsCreated

		case domain.RecoveryPointAccountsCreated:
			// Session is already complete (guaranteed by the transaction in RecoveryPointCoreAccountCreated).
			// Recover the result if entering this phase from a retry.
			if result == nil {
				cached, err := idempotency.UnmarshalCachedResponse[domain.CompleteRegistrationOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
				if err != nil {
					return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling recovery point response."))
				}
				result = cached.Data
			}

			// 7. Cache final response.
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})
			if apiErr != nil {
				// Don't cache error — this is a transient failure. Retry will resume from AccountsCreated.
				return nil, tracing.Trace(span, apiErr)
			}

			return result, nil

		default:
			return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
		}
	}
}

func (s *registrationSessionSvcImpl) handleRegistrationLimitHit(ctx context.Context, session *domain.RegistrationSession) {
	limits := constants.GetRegistrationLimits(constants.PlanCode(session.PlanCode))

	// Best-effort: insert into registration_queue
	if queueErr := s.repos.NewRegistrationQueueRepo().Create(ctx, session.Email, session.SessionData.AccountName, session.PlanCode, session.TypeID); queueErr != nil {
		slog.ErrorContext(ctx, "failed to insert registration queue entry",
			"error", queueErr.PublicMessage,
			"email", session.Email,
			"plan_code", session.PlanCode,
		)
	}

	// Best-effort: send admin alert email
	publishCtx := event.WithRepos(ctx, s.repos)
	if emailErr := s.notificationPublisher.PublishSendEmail(publishCtx, messaging.EmailSendData{
		To:         []string{"dev@augno.com"},
		Subject:    fmt.Sprintf("[Registration Limit] %s plan at capacity", session.PlanCode),
		TemplateID: constants.EmailTemplateRegistrationLimitAlert,
		Params: map[string]any{
			"Email":       session.Email,
			"PlanCode":    session.PlanCode,
			"AccountName": session.SessionData.AccountName,
			"PublicLimit": limits.PublicLimit,
		},
	}); emailErr != nil {
		slog.ErrorContext(ctx, "failed to publish registration limit alert email",
			"error", emailErr.PublicMessage,
			"email", session.Email,
			"plan_code", session.PlanCode,
		)
	}
}

// ConfirmPayment verifies that a Setup Intent succeeded and marks the
// registration session's payment as completed.
func (s *registrationSessionSvcImpl) ConfirmPayment(ctx context.Context, input domain.ConfirmPaymentInput) (*domain.ConfirmPaymentOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.confirm_payment")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityActorTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	if apiErr := s.validateSessionOwnership(ctx, input.SessionID, identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
		TargetAccountID: func() *string {
			if identity.Target != nil {
				return &identity.Target.AccountID
			}
			return nil
		}(),
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ConfirmPaymentOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		regSessionRepo := s.repos.NewRegistrationSessionRepo()
		session, getErr := regSessionRepo.GetByTypeID(ctx, input.SessionID)
		if getErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, getErr)
		}

		// Idempotent: if payment already completed, return success
		if session.PaymentCompleted {
			result := &domain.ConfirmPaymentOutput{Status: "succeeded"}
			if cacheErr := meds.Idempotency.CacheSuccessResponse(ctx, idempotencyKey.TypeID, result); cacheErr != nil {
				return nil, cacheErr
			}
			return result, nil
		}

		// Validate Setup Intent ID matches the one stored during SetupBilling
		if session.StripeCheckoutSessionID == nil || *session.StripeCheckoutSessionID != input.SetupIntentID {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError("Setup Intent ID does not match the session."))
		}

		// Verify Setup Intent status via billing service
		siResult, siErr := s.billingClient.GetSetupIntentStatus(ctx, input.SetupIntentID)
		if siErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, siErr)
		}

		if siResult.Status != "succeeded" {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID,
				apierror.NewValidationError(fmt.Sprintf("Setup Intent status is '%s', expected 'succeeded'.", siResult.Status)))
		}

		// Mark payment as completed
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
			txRegSessionRepo := txSvc.repos.NewRegistrationSessionRepo()
			if updateErr := txRegSessionRepo.UpdatePaymentCompleted(txCtx, session.ID, true, nil); updateErr != nil {
				return updateErr
			}
			txMeds := txSvc.mediators()
			result := &domain.ConfirmPaymentOutput{
				Status:          siResult.Status,
				PaymentMethodID: siResult.PaymentMethodID,
			}
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return &domain.ConfirmPaymentOutput{
			Status:          siResult.Status,
			PaymentMethodID: siResult.PaymentMethodID,
		}, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// SetupBilling creates a Stripe customer and billing profile for a registration
// session's selected plan.
//
//  1. Require user authentication and validate session ownership.
//  2. Upsert an idempotency key; return the cached response if already finished.
//  3. Load the session; skip customer creation if a Stripe customer already exists.
//  4. Create a Stripe customer via the billing service and persist the customer ID
//     atomically with a recovery point advance.
//  5. Set up a billing profile via the billing service and mark payment as ready.
//  6. Cache the success response.
//
// Behavior:
//   - Uses a multi-phase recovery-point loop to safely resume after partial failures.
func (s *registrationSessionSvcImpl) SetupBilling(ctx context.Context, input domain.SetupBillingInput) (*domain.SetupBillingOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.setup_billing")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityActorTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	if apiErr := s.validateSessionOwnership(ctx, input.SessionID, identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
		TargetAccountID: func() *string {
			if identity.Target != nil {
				return &identity.Target.AccountID
			}
			return nil
		}(),
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recoveryPoint := idempotencyKey.RecoveryPoint
	var session *domain.RegistrationSession

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[domain.SetupBillingOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
			}
			return cached.Data, cached.Error

		case domain.RecoveryPointStarted:
			regSessionRepo := s.repos.NewRegistrationSessionRepo()
			session, apiErr = regSessionRepo.GetByTypeID(ctx, input.SessionID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			if session.CompletedAt != nil {
				return nil, tracing.Trace(span, apierror.NewResourceConflictError("Registration session is already completed."))
			}

			// If customer already exists from a previous attempt, skip to phase 2
			if session.StripeCustomerID != nil {
				recoveryPoint = domain.RecoveryPointCustomerCreated
				continue
			}

			// Create Stripe customer
			customerName := session.Email
			if session.SessionData.UserName != "" {
				customerName = session.SessionData.UserName
			}

			customer, customerErr := s.billingClient.CreateCustomer(ctx, session.Email, customerName, idempotencyKey.IdempotencyKey+"_cust", map[string]string{
				"registration_email": session.Email,
			})
			if customerErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, customerErr)
			}

			// Persist customer ID + advance recovery point in a single transaction
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
				txRegSessionRepo := txSvc.repos.NewRegistrationSessionRepo()
				if updateErr := txRegSessionRepo.UpdateStripeCustomer(txCtx, session.ID, &customer.ID, nil); updateErr != nil {
					return updateErr
				}
				txIdempotencyRepo := txSvc.repos.NewIdempotencyKeyRepo()
				return txIdempotencyRepo.AdvanceRecoveryPoint(txCtx, idempotencyKey.TypeID, domain.RecoveryPointCustomerCreated)
			})
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}

			session.StripeCustomerID = &customer.ID
			recoveryPoint = domain.RecoveryPointCustomerCreated

		case domain.RecoveryPointCustomerCreated:
			if session == nil {
				regSessionRepo := s.repos.NewRegistrationSessionRepo()
				session, apiErr = regSessionRepo.GetByTypeID(ctx, input.SessionID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}

			if session.StripeCustomerID == nil {
				return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Stripe customer ID is nil in billing setup phase 2."))
			}

			// Create a Setup Intent to collect the payment method
			siResult, siErr := s.billingClient.CreateSetupIntent(ctx, *session.StripeCustomerID, idempotencyKey.IdempotencyKey+"_si")
			if siErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, siErr)
			}

			// Store the Setup Intent ID and cache the response atomically.
			result := &domain.SetupBillingOutput{
				StripeCustomerID: *session.StripeCustomerID,
				ClientSecret:     siResult.ClientSecret,
				PublishableKey:   siResult.PublishableKey,
			}

			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
				txRegSessionRepo := txSvc.repos.NewRegistrationSessionRepo()
				if updateErr := txRegSessionRepo.UpdateStripeCustomer(txCtx, session.ID, session.StripeCustomerID, &siResult.SetupIntentID); updateErr != nil {
					return updateErr
				}
				txMeds := txSvc.mediators()
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
}
