package service

import (
	"context"
	"encoding/json"
	"fmt"

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

func (s *registrationSessionSvcImpl) UpdateSession(ctx context.Context, input domain.UpdateRegistrationSessionInput) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.update")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	if apiErr := s.validateSessionOwnership(ctx, input.SessionID, identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
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

func (s *registrationSessionSvcImpl) ListSessions(ctx context.Context, input domain.ListRegistrationSessionsInput) (*domain.ListRegistrationSessionsResult, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityTypeUser {
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

func (s *registrationSessionSvcImpl) CompleteRegistration(ctx context.Context, sessionID string) (*domain.CompleteRegistrationOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.complete_registration")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
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
			// recover the result instead of calling core-service again.
			if session.AccountID != nil {
				sandboxID, sandboxErr := s.coreClient.GetSandboxAccountByOwner(ctx, *session.AccountID)
				if sandboxErr != nil {
					return nil, tracing.Trace(span, sandboxErr)
				}
				result = &domain.CompleteRegistrationOutput{
					AccountID: *session.AccountID,
					SandboxID: sandboxID,
				}
				recoveryPoint = domain.RecoveryPointAccountsCreated
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

			// 4. Build core-service input
			coreInput := domain.CompleteAccountRegistrationInput{
				UserID:      *session.UserID,
				PlanCode:    session.PlanCode,
				AccountName: session.SessionData.AccountName,
			}
			if session.StripeCustomerID != nil {
				coreInput.StripeCustomerID = *session.StripeCustomerID
			}
			if session.StripeSubscriptionID != nil {
				coreInput.StripeSubscriptionID = *session.StripeSubscriptionID
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

			// 5. Call core-service (point of no return)
			accountResult, coreErr := s.coreClient.CompleteRegistration(ctx, coreInput)
			if coreErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, coreErr)
			}

			result = &domain.CompleteRegistrationOutput{
				AccountID: accountResult.AccountID,
				SandboxID: accountResult.SandboxID,
			}

			// 6. Atomically: mark session complete + advance recovery point.
			// Both must succeed together so that on retry we can trust the recovery point.
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
				// Transaction failed — session.account_id check (step 2) will catch retries.
				return nil, tracing.Trace(span, apiErr)
			}

			recoveryPoint = domain.RecoveryPointAccountsCreated

		case domain.RecoveryPointAccountsCreated:
			// Session is already complete (guaranteed by the transaction in RecoveryPointStarted).
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

func (s *registrationSessionSvcImpl) ConfirmPayment(ctx context.Context, input domain.ConfirmPaymentInput) (*domain.ConfirmPaymentOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.confirm_payment")
	defer span.End()

	// Validate session ownership
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}
	if apiErr := s.validateSessionOwnership(ctx, input.SessionID, identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 1. Load session
	regSessionRepo := s.repos.NewRegistrationSessionRepo()
	session, apiErr := regSessionRepo.GetByTypeID(ctx, input.SessionID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 2. Validate session state
	if session.CompletedAt != nil {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError("Registration session is already completed."))
	}

	// If payment is already confirmed, return success idempotently
	if session.PaymentCompleted {
		return &domain.ConfirmPaymentOutput{
			Status:           "complete",
			SubscriptionID:   derefOrEmpty(session.StripeSubscriptionID),
			StripeCustomerID: derefOrEmpty(session.StripeCustomerID),
		}, nil
	}

	// 3. Validate checkout session ID matches
	if session.StripeCheckoutSessionID == nil || *session.StripeCheckoutSessionID != input.CheckoutSessionID {
		return nil, tracing.Trace(span, apierror.NewValidationError("Checkout session ID does not match the registration session."))
	}

	// 4. Call billing service to get checkout status
	checkoutStatus, apiErr := s.billingClient.GetCheckoutSessionStatus(ctx, input.CheckoutSessionID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 5. If complete, update session in a transaction
	if checkoutStatus.Status == "complete" {
		txErr := s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
			txRegSessionRepo := txSvc.repos.NewRegistrationSessionRepo()
			var subID *string
			if checkoutStatus.SubscriptionID != "" {
				subID = &checkoutStatus.SubscriptionID
			}
			return txRegSessionRepo.UpdatePaymentCompleted(txCtx, session.ID, true, subID)
		})
		if txErr != nil {
			return nil, tracing.Trace(span, txErr)
		}
	}

	return &domain.ConfirmPaymentOutput{
		Status:           checkoutStatus.Status,
		SubscriptionID:   checkoutStatus.SubscriptionID,
		StripeCustomerID: checkoutStatus.CustomerID,
	}, nil
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *registrationSessionSvcImpl) CreateCheckout(ctx context.Context, input domain.CreateRegistrationCheckoutInput) (*domain.CreateRegistrationCheckoutOutput, *apierror.APIError) {
	ctx, span := registrationSessionSvcTracer.Start(ctx, "service.registration_session.create_checkout")
	defer span.End()

	meds := s.mediators()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity.Type != types.IdentityTypeUser {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("User authentication required."))
	}

	if apiErr := s.validateSessionOwnership(ctx, input.SessionID, identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recoveryPoint := idempotencyKey.RecoveryPoint
	var session *domain.RegistrationSession

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[domain.CreateRegistrationCheckoutOutput](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
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

			// If customer already exists from a previous checkout attempt, skip to phase 2
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
				return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Stripe customer ID is nil in checkout phase 2."))
			}

			// Create checkout session — use the request-level idempotency key so that
			// retries of the same request hit Stripe's cache, but a new request creates
			// a fresh checkout session.
			returnURL := fmt.Sprintf("%s"+string(constants.DashboardPathRegisterCheckoutReturn), s.frontendURL, session.TypeID)
			checkoutSession, checkoutErr := s.billingClient.CreateCheckoutSession(ctx, *session.StripeCustomerID, session.PlanCode, returnURL, idempotencyKey.IdempotencyKey+"_cs")
			if checkoutErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, checkoutErr)
			}

			result := &domain.CreateRegistrationCheckoutOutput{
				ClientSecret:     checkoutSession.ClientSecret,
				CheckoutID:       checkoutSession.ID,
				StripeCustomerID: *session.StripeCustomerID,
				PublishableKey:   checkoutSession.PublishableKey,
			}

			// Persist checkout session ID + cache response in a single transaction
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *registrationSessionSvcImpl) *apierror.APIError {
				txRegSessionRepo := txSvc.repos.NewRegistrationSessionRepo()
				if updateErr := txRegSessionRepo.UpdateStripeCustomer(txCtx, session.ID, session.StripeCustomerID, &checkoutSession.ID); updateErr != nil {
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
