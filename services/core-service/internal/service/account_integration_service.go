package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"

	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/trace"
)

// maxMonthlySpendLimit bounds a spend cap to what the column's decimal(19,4) can hold with room to spare.
var maxMonthlySpendLimit = decimal.NewFromInt(1_000_000_000)

var accountIntegrationSvcTracer = tracing.GetTracer("core-service.account_integration_service")

type accountIntegrationSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	encryptionKey   []byte
	encryptionKeyID string
	platformMode    constants.PlatformMode
	openRouter      domain.OpenRouterClient
}

type AccountIntegrationSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// EncryptionKey (required) encrypts sensitive fields at rest.
	EncryptionKey []byte

	// EncryptionKeyID (required) identifies the active encryption key version.
	EncryptionKeyID string

	// PlatformMode is the deployment mode. In development it relaxes the live-key requirement for Stripe credentials so local, production-type accounts can integrate against Stripe's test environment; outside production it requires Shippo test keys, so no other deployment can buy real labels.
	PlatformMode constants.PlatformMode

	// OpenRouter (required) reads an OpenRouter integration's spend for the usage endpoint and agent-run spend checks.
	OpenRouter domain.OpenRouterClient
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
	if c.OpenRouter == nil {
		return fmt.Errorf("account integration service: openrouter client is required")
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
		platformMode:    config.PlatformMode,
		openRouter:      config.OpenRouter,
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
			platformMode:    s.platformMode,
			openRouter:      s.openRouter,
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

// BatchGetAccountIntegrationsByIDs returns account integrations matching the input IDs that the caller's account is authorized to read. Mirrors the admin-only access check from List.
func (s *accountIntegrationSvcImpl) BatchGetAccountIntegrationsByIDs(ctx context.Context, ids []string) ([]*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.batch_get_by_ids")
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
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewAccountIntegrationRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
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
	if params.MonthlySpendLimit != nil {
		limit, apiErr := normalizeMonthlySpendLimit(params.IntegrationCode, params.MonthlySpendLimit)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.MonthlySpendLimit = limit
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
			updated, updateErr := txRepo.UpdateCredentials(txCtx, params.AccountID, existing.ID, params.Name, encryptedCredentials, params.MonthlySpendLimit)
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
	case constants.IntegrationCodeHubspot:
		return s.validateHubspotCredentials(span, params.Credentials)
	case constants.IntegrationCodeOpenRouter:
		return s.validateOpenRouterCredentials(span, params.Credentials)
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

	// Local development accounts are seeded as production-type (non-sandbox) accounts, which would otherwise force live keys. In development mode accept either test or live keys so integrations can be exercised against Stripe's test environment.
	if s.platformMode.IsDevelopment() {
		return nil
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

	// Legacy's rule: only a production deployment spends with a live key, and never for a sandbox.
	switch {
	case isSandbox:
		if !strings.HasPrefix(creds.APIKey, "shippo_test_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Sandbox accounts must use test Shippo keys (shippo_test_).", "credentials"))
		}
	case s.platformMode.IsProduction():
		if !strings.HasPrefix(creds.APIKey, "shippo_live_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Production accounts must use live Shippo keys (shippo_live_).", "credentials"))
		}
	default:
		if !strings.HasPrefix(creds.APIKey, "shippo_test_") {
			return tracing.Trace(span, apierror.NewValidationErrorWithParam("Live Shippo keys (shippo_live_) are only accepted in production; use a test key (shippo_test_).", "credentials"))
		}
	}

	return nil
}

// validateHubspotCredentials validates a HubSpot Private App access token. HubSpot tokens carry no sandbox/production distinction in their format, so the same prefix check applies to every account.
func (s *accountIntegrationSvcImpl) validateHubspotCredentials(span trace.Span, credentialsJSON string) *apierror.APIError {
	var creds domain.HubspotCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid credentials JSON format.", "credentials"))
	}

	if !strings.HasPrefix(creds.AccessToken, "pat-") {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("HubSpot access token must start with 'pat-'.", "credentials"))
	}

	return nil
}

// validateOpenRouterCredentials checks an OpenRouter API key's format. OpenRouter keys carry no sandbox/production distinction.
func (s *accountIntegrationSvcImpl) validateOpenRouterCredentials(span trace.Span, credentialsJSON string) *apierror.APIError {
	var creds domain.OpenRouterCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid credentials JSON format.", "credentials"))
	}

	if !strings.HasPrefix(creds.APIKey, "sk-or-") {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam("OpenRouter API key must start with 'sk-or-'.", "credentials"))
	}

	return nil
}

// normalizeMonthlySpendLimit validates a spend cap and returns it with its amount in canonical decimal form.
func normalizeMonthlySpendLimit(code constants.IntegrationCode, limit *domain.Money) (*domain.Money, *apierror.APIError) {
	if code != constants.IntegrationCodeOpenRouter {
		return nil, apierror.NewValidationErrorWithParam("Only OpenRouter integrations take a monthly spend limit.", "monthly_spend_limit")
	}
	if limit.Currency != "usd" {
		return nil, apierror.NewValidationErrorWithParam("OpenRouter bills in US dollars, so the monthly spend limit's currency must be 'usd'.", "monthly_spend_limit.currency")
	}
	amount, err := decimal.NewFromString(limit.Amount)
	if err != nil {
		return nil, apierror.NewValidationErrorWithParam("Monthly spend limit amount must be a decimal number.", "monthly_spend_limit.amount")
	}
	if !amount.IsPositive() || amount.GreaterThan(maxMonthlySpendLimit) || amount.Exponent() < -4 {
		return nil, apierror.NewValidationErrorWithParam("Monthly spend limit amount must be greater than 0, at most 1000000000, and have at most 4 decimal places.", "monthly_spend_limit.amount")
	}
	return &domain.Money{Amount: amount.String(), Currency: limit.Currency}, nil
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

			if params.SetMonthlySpendLimit && params.MonthlySpendLimit != nil {
				limit, apiErr := normalizeMonthlySpendLimit(old.IntegrationCode, params.MonthlySpendLimit)
				if apiErr != nil {
					return apiErr
				}
				params.MonthlySpendLimit = limit
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
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeAccountIntegration, params.ID, params.AccountID)
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
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeAccountIntegration, integration.ID, params.AccountID, integration); apiErr != nil {
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
		return "", tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	// Only internal actors and customer actors are allowed; supplier actors are rejected.
	if !identity.IsInternalActor() && !identity.IsCustomerUser() {
		return "", tracing.Trace(span, apierror.NewAuthorizationError("User does not have permission to access this resource."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
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
		return false, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	// Customer actor read access check
	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
	}

	return s.repos.NewAccountIntegrationRepo().HasIntegration(ctx, identity.Target.AccountID, constants.IntegrationCodeStripe)
}

// GetOpenRouterUsage returns what the caller's account has spent through its OpenRouter integration, alongside its monthly cap.
func (s *accountIntegrationSvcImpl) GetOpenRouterUsage(ctx context.Context) (*domain.OpenRouterUsage, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.get_openrouter_usage")
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

	stored, apiKey, apiErr := s.openRouterCredentials(ctx, identity.Target.AccountID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("This account has no OpenRouter integration."))
		}
		return nil, tracing.Trace(span, apiErr)
	}

	usage, apiErr := s.openRouter.GetKeyUsage(ctx, apiKey)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := &domain.OpenRouterUsage{Key: *usage, MonthlySpendLimit: stored.MonthlySpendLimit}
	if limit := stored.MonthlySpendLimit; limit != nil {
		result.MonthlySpendRemaining = monthlySpendRemaining(limit, usage.SpentThisMonth)
	}
	return result, nil
}

// GetOpenRouterAccess returns the OpenRouter key an agent run in the caller's target account bills to, and whether its monthly cap is spent. A sandbox bills to the account that owns it.
func (s *accountIntegrationSvcImpl) GetOpenRouterAccess(ctx context.Context) (*domain.OpenRouterAccess, *apierror.APIError) {
	ctx, span := accountIntegrationSvcTracer.Start(ctx, "service.account_integration.get_openrouter_access")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAgent(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID
	accountCtx, apiErr := s.repos.NewAccountRepo().GetAccountContext(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if accountCtx.IsSandbox && accountCtx.OwnerAccountID != nil {
		accountID = *accountCtx.OwnerAccountID
	}

	stored, apiKey, apiErr := s.openRouterCredentials(ctx, accountID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return &domain.OpenRouterAccess{}, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	access := &domain.OpenRouterAccess{Configured: true, IsActive: stored.IsActive, APIKey: apiKey}
	if !stored.IsActive || stored.MonthlySpendLimit == nil {
		return access, nil
	}

	// The cap is a guard on top of OpenRouter's own key limit, so an unreadable spend lets the run go ahead rather than stopping every agent while OpenRouter's key endpoint is down.
	usage, apiErr := s.openRouter.GetKeyUsage(ctx, apiKey)
	if apiErr != nil {
		slog.WarnContext(ctx, "openrouter spend check skipped", "account_id", accountID, "error", apiErr.Error())
		return access, nil
	}
	access.MonthlySpendLimit = stored.MonthlySpendLimit
	access.SpentThisMonth = &usage.SpentThisMonth
	access.LimitReached = monthlySpendReached(usage.SpentThisMonth, stored.MonthlySpendLimit.Amount)
	return access, nil
}

// monthlySpendRemaining is what is left of limit after spent, floored at zero; an unparseable figure leaves it unknown.
func monthlySpendRemaining(limit *domain.Money, spent string) *domain.Money {
	limitAmount, err := decimal.NewFromString(limit.Amount)
	if err != nil {
		return nil
	}
	spentAmount, err := decimal.NewFromString(spent)
	if err != nil {
		return nil
	}
	left := decimal.Max(limitAmount.Sub(spentAmount), decimal.Zero)
	return &domain.Money{Amount: left.String(), Currency: limit.Currency}
}

func monthlySpendReached(spent, limit string) bool {
	spentAmount, err := decimal.NewFromString(spent)
	if err != nil {
		return false
	}
	limitAmount, err := decimal.NewFromString(limit)
	if err != nil {
		return false
	}
	return spentAmount.GreaterThanOrEqual(limitAmount)
}

func (s *accountIntegrationSvcImpl) openRouterCredentials(ctx context.Context, accountID string) (*domain.StoredIntegrationCredentials, string, *apierror.APIError) {
	stored, apiErr := s.repos.NewAccountIntegrationRepo().GetStoredCredentials(ctx, accountID, constants.IntegrationCodeOpenRouter)
	if apiErr != nil {
		return nil, "", apiErr
	}

	plaintext, err := crypto.DecryptAESGCM(stored.Encrypted, s.encryptionKey, []byte(accountID))
	if err != nil {
		return nil, "", apierror.NewInternalError(err, "Failed to decrypt credentials.")
	}

	var creds domain.OpenRouterCredentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, "", apierror.NewInternalError(err, "Failed to parse credentials.")
	}

	return stored, creds.APIKey, nil
}
