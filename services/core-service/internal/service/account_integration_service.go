package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

var accountIntegrationSvcTracer = tracing.GetTracer("core-service.account_integration_service")

type accountIntegrationSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	encryptionKey   []byte
	encryptionKeyID string
}

type AccountIntegrationSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
	EncryptionKey   []byte
	EncryptionKeyID string
}

func (c *AccountIntegrationSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account integration service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("account integration service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("account integration service: tx manager is required")
	}
	if len(c.EncryptionKey) == 0 {
		return fmt.Errorf("account integration service: encryption key is required")
	}
	if c.EncryptionKeyID == "" {
		return fmt.Errorf("account integration service: encryption key ID is required")
	}
	return nil
}

func NewAccountIntegrationSvc(config *AccountIntegrationSvcConfig) domain.AccountIntegrationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountIntegrationSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		encryptionKey:   config.EncryptionKey,
		encryptionKeyID: config.EncryptionKeyID,
	}
}

func (s *accountIntegrationSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *accountIntegrationSvcImpl) withTx(ctx context.Context, fn func(context.Context, *accountIntegrationSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &accountIntegrationSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			encryptionKey:   s.encryptionKey,
			encryptionKeyID: s.encryptionKeyID,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *accountIntegrationSvcImpl) ListAccountIntegrations(ctx context.Context, params domain.ListAccountIntegrationsParams) (*domain.ListAccountIntegrationsResult, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAccountIntegrationRepo().List(ctx, params)
}

func (s *accountIntegrationSvcImpl) CreateAccountIntegration(ctx context.Context, params domain.CreateAccountIntegrationParams) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Validate integration code
	if !params.IntegrationCode.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Invalid integration code: %s. Must be one of: %s", params.IntegrationCode, strings.Join(params.IntegrationCode.EnumValues(), ", ")),
			"integration_code",
		))
	}

	// Validate and parse credentials
	if apiErr := s.validateCredentials(ctx, span, params); apiErr != nil {
		return nil, apiErr
	}

	// Encrypt credentials
	encryptedCredentials, err := crypto.EncryptAESGCM([]byte(params.Credentials), s.encryptionKey, []byte(params.AccountID), s.encryptionKeyID)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to encrypt credentials."))
	}

	integrationID, apiErr := id.GenID(id.AccountIntegrationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountIntegration](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountIntegration
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountIntegrationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountIntegrationRepo()

			// Check if an integration with this code already exists (upsert)
			existing, existErr := txRepo.FindByCode(txCtx, params.AccountID, params.IntegrationCode)
			if existErr != nil {
				// If not found, create new
				if apierror.IsNotFound(existErr) {
					created, createErr := txRepo.Create(txCtx, integrationID, params, encryptedCredentials)
					if createErr != nil {
						return createErr
					}
					result = created

					changes := audit.ComputeChanges(nil, created)

					if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
						ServiceName:  domain.ServiceName,
						Action:       constants.AuditActionCreate,
						ResourceType: constants.ObjectTypeAccountIntegration,
						ResourceID:   created.ID,
						Changes:      changes,
					}); apiErr != nil {
						return apiErr
					}

					return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
				}
				return existErr
			}

			// Existing found — upsert: update credentials and name
			updated, updateErr := txRepo.UpdateCredentials(txCtx, params.AccountID, existing.ID, params.Name, encryptedCredentials)
			if updateErr != nil {
				return updateErr
			}
			result = updated

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAccountIntegration,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *accountIntegrationSvcImpl) validateCredentials(ctx context.Context, span trace.Span, params domain.CreateAccountIntegrationParams) *apierror.APIError {
	// Get account context to determine sandbox/production
	accountCtx, apiErr := s.repos.NewAccountRepo().GetAccountContext(ctx, params.AccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch params.IntegrationCode {
	case constants.IntegrationCodeStripe:
		return s.validateStripeCredentials(span, params.Credentials, accountCtx.IsSandbox)
	case constants.IntegrationCodeShippo:
		return s.validateShippoCredentials(span, params.Credentials, accountCtx.IsSandbox)
	default:
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Unsupported integration code.", "integration_code"))
	}
}

func (s *accountIntegrationSvcImpl) validateStripeCredentials(span trace.Span, credentialsJSON string, isSandbox bool) *apierror.APIError {
	var creds domain.StripeCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid credentials JSON format.", "credentials"))
	}

	if !strings.HasPrefix(creds.PrivateKey, "sk_") {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Stripe private key must start with 'sk_'.", "credentials"))
	}
	if !strings.HasPrefix(creds.PublishableKey, "pk_") {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Stripe publishable key must start with 'pk_'.", "credentials"))
	}
	if !strings.HasPrefix(creds.WebhookSecret, "whsec_") {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Stripe webhook secret must start with 'whsec_'.", "credentials"))
	}

	if isSandbox {
		if !strings.HasPrefix(creds.PrivateKey, "sk_test_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Sandbox accounts must use test Stripe keys (sk_test_).", "credentials"))
		}
		if !strings.HasPrefix(creds.PublishableKey, "pk_test_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Sandbox accounts must use test Stripe keys (pk_test_).", "credentials"))
		}
	} else {
		if !strings.HasPrefix(creds.PrivateKey, "sk_live_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Production accounts must use live Stripe keys (sk_live_).", "credentials"))
		}
		if !strings.HasPrefix(creds.PublishableKey, "pk_live_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Production accounts must use live Stripe keys (pk_live_).", "credentials"))
		}
	}

	return nil
}

func (s *accountIntegrationSvcImpl) validateShippoCredentials(span trace.Span, credentialsJSON string, isSandbox bool) *apierror.APIError {
	var creds domain.ShippoCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid credentials JSON format.", "credentials"))
	}

	if !strings.HasPrefix(creds.APIKey, "shippo_live_") && !strings.HasPrefix(creds.APIKey, "shippo_test_") {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Shippo API key must start with 'shippo_live_' or 'shippo_test_'.", "credentials"))
	}

	if isSandbox {
		if !strings.HasPrefix(creds.APIKey, "shippo_test_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Sandbox accounts must use test Shippo keys (shippo_test_).", "credentials"))
		}
	} else {
		if !strings.HasPrefix(creds.APIKey, "shippo_live_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Production accounts must use live Shippo keys (shippo_live_).", "credentials"))
		}
	}

	return nil
}

func (s *accountIntegrationSvcImpl) UpdateAccountIntegration(ctx context.Context, params domain.UpdateAccountIntegrationParams) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountIntegration](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountIntegration
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountIntegrationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountIntegrationRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.ID)
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAccountIntegration,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *accountIntegrationSvcImpl) DeleteAccountIntegration(ctx context.Context, params domain.DeleteAccountIntegrationParams) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	integration, apiErr := s.repos.NewAccountIntegrationRepo().Get(ctx, params.AccountID, params.ID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAccountIntegration, params.ID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This account integration has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.AccountIntegration
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountIntegrationSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAccountIntegration, integration.ID, integration); apiErr != nil {
			return apiErr
		}

		deleted, apiErr := txSvc.repos.NewAccountIntegrationRepo().Delete(txCtx, params)
		if apiErr != nil {
			return apiErr
		}
		result = deleted

		changes := audit.ComputeChanges(deleted, (*domain.AccountIntegration)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAccountIntegration,
			ResourceID:   deleted.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *accountIntegrationSvcImpl) GetStripePublishableKey(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.get_stripe_publishable_key")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return "", tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	// Only internal actors and customer actors are allowed; supplier actors are rejected.
	if !identity.IsInternalActor() && !identity.IsCustomerUser() {
		return "", tracing.Trace(span, apierror.NewAuthorizationError("User does not have permission to access this resource."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return "", tracing.Trace(span, apiErr)
		}
	}

	repo := s.repos.NewAccountIntegrationRepo()
	encryptedCreds, isActive, apiErr := repo.GetEncryptedCredentials(ctx, identity.Target.AccountID, constants.IntegrationCodeStripe)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	if !isActive {
		return "", tracing.Trace(span, apierror.NewValidationError("Stripe integration is inactive."))
	}

	plaintext, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, []byte(identity.Target.AccountID))
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to decrypt credentials."))
	}

	var creds domain.StripeCredentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse credentials."))
	}

	return creds.PublishableKey, nil
}

func (s *accountIntegrationSvcImpl) HasStripeIntegration(ctx context.Context) (bool, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.has_stripe_integration")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return false, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return false, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	// Customer actor read access check
	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
	}

	return s.repos.NewAccountIntegrationRepo().HasIntegration(ctx, identity.Target.AccountID, constants.IntegrationCodeStripe)
}
