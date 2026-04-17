package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var carrierSvcTracer = tracing.GetTracer("core-service.carrier_service")

type carrierSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	shippoFactory   domain.ShippoClientFactory
	encryptionKey   []byte
	accountSvc      domain.AccountSvc
}

type CarrierSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
	ShippoFactory   domain.ShippoClientFactory
	EncryptionKey   []byte
	AccountSvc      domain.AccountSvc
}

func (c *CarrierSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("carrier service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("carrier service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("carrier service: tx manager is required")
	}
	if c.ShippoFactory == nil {
		return fmt.Errorf("carrier service: shippo factory is required")
	}
	if c.AccountSvc == nil {
		return fmt.Errorf("carrier service: account svc is required")
	}
	return nil
}

func NewCarrierSvc(config *CarrierSvcConfig) domain.CarrierSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &carrierSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		shippoFactory:   config.ShippoFactory,
		encryptionKey:   config.EncryptionKey,
		accountSvc:      config.AccountSvc,
	}
}

func (s *carrierSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *carrierSvcImpl) withTx(ctx context.Context, fn func(context.Context, *carrierSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &carrierSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			shippoFactory:   s.shippoFactory,
			encryptionKey:   s.encryptionKey,
			accountSvc:      s.accountSvc,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *carrierSvcImpl) getShippoClient(ctx context.Context, accountID string) (domain.ShippoClient, *apierror.APIError) {
	repo := s.repos.NewAccountIntegrationRepo()
	encryptedCreds, isActive, apiErr := repo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, apiErr
	}
	if !isActive {
		return nil, apierror.NewValidationError("Shippo integration is not active. Please configure Shippo credentials first.")
	}

	plaintext, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, []byte(accountID))
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to decrypt Shippo credentials.")
	}

	var creds domain.ShippoCredentials
	if err := json.Unmarshal([]byte(plaintext), &creds); err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse Shippo credentials.")
	}

	return s.shippoFactory.Build(creds.APIKey), nil
}

func (s *carrierSvcImpl) ListCarriers(ctx context.Context, params domain.ListCarriersParams) (*domain.ListCarriersResult, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCarrierReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewCarrierRepo().List(ctx, params)
}

func (s *carrierSvcImpl) GetCarrier(ctx context.Context, carrierID string) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCarrierReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	carrierRepo := s.repos.NewCarrierRepo()

	carrier, apiErr := carrierRepo.Get(ctx, identity.Target.AccountID, carrierID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	options, apiErr := carrierRepo.ListOptionsByCarrierID(ctx, identity.Target.AccountID, carrierID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	carrier.ServiceLevels = options

	return carrier, nil
}

func (s *carrierSvcImpl) CreateCarrier(ctx context.Context, params domain.CreateCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	carrierID, apiErr := id.GenID(id.CarrierIDPrefix, nil)
	if apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Carrier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Check if this is a Shippo-managed carrier
		isShippo := constants.IsShippoCarrier(params.Code)

		var shippoAccountID *string
		var shippoOptions []domain.CreateServiceLevelParams

		if isShippo {
			// Check sandbox — skip Shippo in sandbox mode
			accountCtx, apiErr := s.accountSvc.GetAccountContext(ctx, params.AccountID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			if !accountCtx.IsSandbox {
				// Foreign mutation: call Shippo API
				shippoClient, apiErr := s.getShippoClient(ctx, params.AccountID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}

				code := *params.Code
				var shippoAccount *domain.ShippoCarrierAccount

				if code == string(constants.CarrierCodeFedEx) {
					// FedEx uses OAuth — find or register account
					shippoAccount, apiErr = shippoClient.FindOrRegisterCarrierAccount(ctx, code)
					if apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}
				} else {
					// UPS/USPS — connect with account number
					if params.AccountNumber == nil || *params.AccountNumber == "" {
						return nil, tracing.Trace(span, apierror.NewValidationError("Account number is required for this carrier."))
					}
					connectParams := map[string]string{
						"account_number": *params.AccountNumber,
					}
					shippoAccount, apiErr = shippoClient.ConnectCarrierAccount(ctx, code, *params.AccountNumber, connectParams)
					if apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}
				}

				shippoAccountID = &shippoAccount.ObjectID

				// Fetch service levels and create default options
				levels, apiErr := shippoClient.GetCarrierServiceLevels(ctx, shippoAccount.ObjectID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}

				for _, level := range levels {
					token := level.Token
					shippoOptions = append(shippoOptions, domain.CreateServiceLevelParams{
						AccountID:         params.AccountID,
						CarrierID:         carrierID,
						Name:              level.Name,
						Code:              level.Token,
						ServiceLevelToken: &token,
						IsPortalEnabled:   false,
						IsDefault:         false,
					})
				}
			}
		}

		params.ShippoCarrierAccountID = shippoAccountID

		var result *domain.Carrier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *carrierSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewCarrierRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A carrier with this name already exists.", "name")
			}

			created, apiErr := txRepo.Create(txCtx, carrierID, params)
			if apiErr != nil {
				return apiErr
			}

			// Create Shippo-synced options
			optionRepo := txSvc.repos.NewServiceLevelRepo()
			var createdOptions []*domain.ServiceLevel
			for _, optParams := range shippoOptions {
				optionID, apiErr := id.GenID(id.ServiceLevelIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				opt, apiErr := optionRepo.Create(txCtx, optionID, optParams)
				if apiErr != nil {
					return apiErr
				}
				createdOptions = append(createdOptions, opt)
			}

			created.ServiceLevels = createdOptions
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeCarrier,
				ResourceID:   created.ID,
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

func (s *carrierSvcImpl) UpdateCarrier(ctx context.Context, params domain.UpdateCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Carrier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Carrier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *carrierSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewCarrierRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.CarrierID)
			if apiErr != nil {
				return apiErr
			}

			if old.AccountID == nil {
				return apierror.NewAuthorizationError("System-owned carriers cannot be updated.")
			}
			if *old.AccountID != params.AccountID {
				return apierror.NewAuthorizationError("This carrier is owned by another account and cannot be updated.")
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.CarrierID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A carrier with this name already exists.", "name")
				}
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
				ResourceType: constants.ObjectTypeCarrier,
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

func (s *carrierSvcImpl) DeleteCarrier(ctx context.Context, carrierID string) *apierror.APIError {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, accountID, carrierID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeCarrier, carrierID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This carrier has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	if carrier.AccountID == nil {
		return tracing.Trace(span, apierror.NewAuthorizationError("System-owned carriers cannot be deleted."))
	}
	if *carrier.AccountID != accountID {
		return tracing.Trace(span, apierror.NewAuthorizationError("This carrier is owned by another account and cannot be deleted."))
	}

	// Deactivate on Shippo if applicable
	if carrier.ShippoCarrierAccountID != nil {
		accountCtx, apiErr := s.accountSvc.GetAccountContext(ctx, accountID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		if !accountCtx.IsSandbox {
			shippoClient, apiErr := s.getShippoClient(ctx, accountID)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}

			if apiErr := shippoClient.DeactivateCarrierAccount(ctx, *carrier.ShippoCarrierAccountID); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *carrierSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewCarrierRepo()

		if apiErr := txRepo.DeleteOptionsByCarrierID(txCtx, accountID, carrierID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeCarrier, carrier.ID, carrier); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.SoftDelete(txCtx, accountID, carrierID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(carrier, (*domain.Carrier)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeCarrier,
			ResourceID:   carrier.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (s *carrierSvcImpl) InitiateOAuth(ctx context.Context, carrierID, redirectURI string, state *string) (string, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.initiate_oauth")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionUpdate); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	accountCtx, apiErr := s.accountSvc.GetAccountContext(ctx, accountID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if accountCtx.IsSandbox {
		return "", tracing.Trace(span, apierror.NewValidationError("OAuth is not available in sandbox mode."))
	}

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, accountID, carrierID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	if carrier.ShippoCarrierAccountID == nil {
		return "", tracing.Trace(span, apierror.NewValidationError("This carrier does not have a Shippo carrier account."))
	}

	shippoClient, apiErr := s.getShippoClient(ctx, accountID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	oauthURL, apiErr := shippoClient.InitiateOAuth(ctx, *carrier.ShippoCarrierAccountID, redirectURI, state)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return oauthURL, nil
}

func (s *carrierSvcImpl) GetOAuthStatus(ctx context.Context, carrierID string) (string, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.get_oauth_status")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionRead); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	accountCtx, apiErr := s.accountSvc.GetAccountContext(ctx, accountID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if accountCtx.IsSandbox {
		return "disconnected", nil
	}

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, accountID, carrierID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	if carrier.ShippoCarrierAccountID == nil {
		return "disconnected", nil
	}

	shippoClient, apiErr := s.getShippoClient(ctx, accountID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	account, apiErr := shippoClient.GetCarrierAccount(ctx, *carrier.ShippoCarrierAccountID)
	if apiErr != nil {
		return "disconnected", nil
	}

	if account.IsShippoAccount {
		return "authorization_pending", nil
	}

	return "connected", nil
}

func (s *carrierSvcImpl) SyncOptions(ctx context.Context, carrierID string) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.sync_options")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	accountCtx, apiErr := s.accountSvc.GetAccountContext(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if accountCtx.IsSandbox {
		return nil, tracing.Trace(span, apierror.NewValidationError("Syncing options is not available in sandbox mode."))
	}

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, accountID, carrierID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if carrier.ShippoCarrierAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("This carrier does not have a Shippo carrier account."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Carrier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Foreign mutation: fetch service levels from Shippo
		shippoClient, apiErr := s.getShippoClient(ctx, accountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		levels, apiErr := shippoClient.GetCarrierServiceLevels(ctx, *carrier.ShippoCarrierAccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Build a set of tokens from Shippo
		shippoTokens := make(map[string]domain.ShippoServiceLevel)
		for _, level := range levels {
			shippoTokens[level.Token] = level
		}

		// Get existing options for this carrier
		existingOptions, apiErr := s.repos.NewCarrierRepo().ListOptionsByCarrierID(ctx, accountID, carrierID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Build set of existing tokens
		existingTokens := make(map[string]*domain.ServiceLevel)
		for _, opt := range existingOptions {
			if opt.ServiceLevelToken != nil {
				existingTokens[*opt.ServiceLevelToken] = opt
			}
		}

		var result *domain.Carrier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *carrierSvcImpl) *apierror.APIError {
			optionRepo := txSvc.repos.NewServiceLevelRepo()

			// Remove options whose tokens no longer exist in Shippo
			for token, opt := range existingTokens {
				if _, exists := shippoTokens[token]; !exists {
					if apiErr := optionRepo.Delete(txCtx, accountID, opt.ID); apiErr != nil {
						return apiErr
					}
				}
			}

			// Add new options for tokens that don't exist yet
			for token, level := range shippoTokens {
				if _, exists := existingTokens[token]; !exists {
					optionID, apiErr := id.GenID(id.ServiceLevelIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					t := token
					_, apiErr = optionRepo.Create(txCtx, optionID, domain.CreateServiceLevelParams{
						AccountID:         accountID,
						CarrierID:         carrierID,
						Name:              level.Name,
						Code:              level.Token,
						ServiceLevelToken: &t,
						IsPortalEnabled:   false,
						IsDefault:         false,
					})
					if apiErr != nil {
						return apiErr
					}
				}
			}

			// Re-fetch carrier with updated options inside the transaction
			updatedCarrier, apiErr := txSvc.repos.NewCarrierRepo().Get(txCtx, accountID, carrierID)
			if apiErr != nil {
				return apiErr
			}

			options, apiErr := txSvc.repos.NewCarrierRepo().ListOptionsByCarrierID(txCtx, accountID, carrierID)
			if apiErr != nil {
				return apiErr
			}
			updatedCarrier.ServiceLevels = options
			result = updatedCarrier

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

// checkCarrierReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need carriers:read for their own account, or customers:read / suppliers:read for external accounts.
func checkCarrierReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionRead)
}
