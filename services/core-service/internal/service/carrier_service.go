package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/crypto"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/tracing"
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
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// ShippoFactory (required) builds Shippo shipping clients.
	ShippoFactory domain.ShippoClientFactory

	// EncryptionKey (optional; default: nil) encrypts sensitive fields at rest. It is not validated at construction.
	EncryptionKey []byte

	// AccountSvc (required) is the account service dependency.
	AccountSvc domain.AccountSvc
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
		return nil, apierror.NewValidationError("Shippo integration is misconfigured. Please reconnect Shippo.")
	}

	var creds domain.ShippoCredentials
	if err := json.Unmarshal([]byte(plaintext), &creds); err != nil {
		return nil, apierror.NewValidationError("Shippo integration is misconfigured. Please reconnect Shippo.")
	}

	return buildShippoClient(s.shippoFactory, s.repos, accountID, creds.APIKey)
}

func (s *carrierSvcImpl) ListCarriers(ctx context.Context, params domain.ListCarriersParams) (*domain.ListCarriersResult, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.list")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	carrierRepo := s.repos.NewCarrierRepo()

	result, apiErr := carrierRepo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "service_levels") {
		for _, carrier := range result.Carriers {
			options, apiErr := carrierRepo.ListOptionsByCarrierID(ctx, identity.Target.AccountID, carrier.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			carrier.ServiceLevels = options
		}
	}

	return result, nil
}

// BatchGetCarriersByIDs returns multiple carriers by ID with the same account scoping rules as GetCarrier. When serviceLevelsLimit > 0, each returned carrier is annotated with a preview of up to that many service_level IDs plus a has_more flag (these are populated as fields on the returned domain Carrier so the gRPC handler can mirror them into CarrierInfo).
func (s *carrierSvcImpl) BatchGetCarriersByIDs(ctx context.Context, ids []string, serviceLevelsLimit int32) ([]*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if identity.IsExternalTarget() {
		meds := s.mediators()
		// Counterparty-aware: this batch endpoint hydrates the customer-portal customer-detail
		// read (freight_preferences.carrier is the customer's own default carrier). A customer
		// relation actor reaches the target→actor direction of the relation, so the owner-side
		// CheckReadAccess would reject them. Results stay scoped to Target.AccountID (below) and
		// the IDs come from the caller's own already-authorized customer record, so this does not
		// widen access beyond the counterparty's own referenced carriers.
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	carrierRepo := s.repos.NewCarrierRepo()
	carriers, apiErr := carrierRepo.GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if serviceLevelsLimit <= 0 || len(carriers) == 0 {
		return carriers, nil
	}

	carrierIDs := make([]string, len(carriers))
	for i, c := range carriers {
		carrierIDs[i] = c.ID
	}
	idsByCarrier, apiErr := carrierRepo.ListOptionIDsForCarriers(ctx, identity.Target.AccountID, carrierIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	limit := int(serviceLevelsLimit)
	for _, c := range carriers {
		all := idsByCarrier[c.ID]
		if len(all) > limit {
			c.ServiceLevelIDsPreview = append([]string(nil), all[:limit]...)
			c.ServiceLevelsHasMore = true
		} else {
			c.ServiceLevelIDsPreview = append([]string(nil), all...)
			c.ServiceLevelsHasMore = false
		}
	}
	return carriers, nil
}

// BatchGetServiceLevelsByIDs returns service levels by ID, authorization follows the parent carrier's account scope (the repo enforces this via the inner JOIN on `carrier`).
func (s *carrierSvcImpl) BatchGetServiceLevelsByIDs(ctx context.Context, ids []string) ([]*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.batch_get_service_levels_by_ids")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if identity.IsExternalTarget() {
		meds := s.mediators()
		// Counterparty-aware: hydrates ?include=service_levels for a customer-portal
		// relation actor reading their own default carrier's options. Results are scoped
		// to Target.AccountID; the owner-side CheckReadAccess would wrongly reject them.
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	carrierRepo := s.repos.NewCarrierRepo()
	return carrierRepo.GetOptionsByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *carrierSvcImpl) GetCarrier(ctx context.Context, params domain.GetCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	carrierRepo := s.repos.NewCarrierRepo()

	carrier, apiErr := carrierRepo.Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "service_levels") {
		options, apiErr := carrierRepo.ListOptionsByCarrierID(ctx, identity.Target.AccountID, params.CarrierID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		carrier.ServiceLevels = options
	}

	return carrier, nil
}

func (s *carrierSvcImpl) CreateCarrier(ctx context.Context, params domain.CreateCarrierParams) (*domain.Carrier, *apierror.APIError) {
	ctx, span := carrierSvcTracer.Start(ctx, "service.carrier.create")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Carrier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		isShippo := params.Type.IsShippo()

		var shippoAccountID *string
		var shippoOptions []domain.CreateServiceLevelParams

		if isShippo {
			// Check sandbox — skip Shippo in sandbox mode
			accountCtx, apiErr := s.accountSvc.GetAccountContext(ctx, params.AccountID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			if !accountCtx.IsSandbox {
				code := string(params.Type)

				// UPS/USPS require an account number. Validate this pure input requirement before acquiring the Shippo client so a missing account number yields a specific 400 regardless of integration state.
				if params.Type != constants.CarrierTypeFedEx {
					if params.AccountNumber == nil || *params.AccountNumber == "" {
						return nil, tracing.Trace(span, apierror.NewValidationError("Account number is required for this carrier."))
					}
				}

				// Foreign mutation: call Shippo API
				shippoClient, apiErr := s.getShippoClient(ctx, params.AccountID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}

				var shippoAccount *domain.ShippoCarrierAccount

				if params.Type == constants.CarrierTypeFedEx {
					// FedEx uses OAuth — find or register account
					shippoAccount, apiErr = shippoClient.FindOrRegisterCarrierAccount(ctx, code)
					if apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}
				} else {
					// UPS/USPS — connect with account number
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
					shippoOptions = append(shippoOptions, domain.CreateServiceLevelParams{
						AccountID:       params.AccountID,
						CarrierID:       carrierID,
						Name:            level.Name,
						Code:            level.Token,
						Synced:          true,
						IsPortalEnabled: false,
						IsActive:        true,
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
				return apierror.NewConflictErrorWithParam("name", "A carrier with this name already exists.")
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Carrier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Carrier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *carrierSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewCarrierRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: params.CarrierID})
			if apiErr != nil {
				return apiErr
			}

			if old.AccountID == nil {
				return apierror.NewAuthorizationError("System-owned carriers cannot be updated.")
			}
			if *old.AccountID != params.AccountID {
				return apierror.NewAuthorizationError("This carrier is owned by another account and cannot be updated.")
			}

			if apiErr := txSvc.checkDefaultServiceLevel(txCtx, old, params.DefaultServiceLevelID); apiErr != nil {
				return apiErr
			}
			// The columns are assigned rather than merged so an explicit null can clear them, which makes carrying the existing values forward the meaning of leaving the fields out.
			params.SCAC = params.SCAC.BackfillUnsetPtr(old.SCAC)
			params.DefaultServiceLevelID = params.DefaultServiceLevelID.BackfillUnsetPtr(old.DefaultServiceLevelID)

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.CarrierID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("name", "A carrier with this name already exists.")
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: accountID, CarrierID: carrierID})
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeCarrier, carrierID, accountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewGoneError("This carrier has already been deleted and can no longer be modified."))
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

	// Checked before Shippo is told, so a carrier still in use keeps its Shippo account; checked again in the transaction below.
	if apiErr := domain.CarrierReferences.RequireUnused(ctx, s.repos.NewReferenceRepo(), carrier.ID); apiErr != nil {
		return tracing.Trace(span, apiErr)
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

		if apiErr := domain.CarrierReferences.RequireUnused(txCtx, txSvc.repos.NewReferenceRepo(), carrier.ID); apiErr != nil {
			return apiErr
		}
		// The carrier's own service levels go with it, so their references are probed too.
		levels, apiErr := txRepo.ListOptionsByCarrierID(txCtx, accountID, carrierID)
		if apiErr != nil {
			return apiErr
		}
		for _, level := range levels {
			labels, apiErr := txSvc.repos.NewReferenceRepo().InUseBy(txCtx, level.ID, domain.ServiceLevelReferences.Probes)
			if apiErr != nil {
				return apiErr
			}
			if len(labels) > 0 {
				return domain.CarrierReferences.InUseError(labels)
			}
		}

		if apiErr := txRepo.DeleteOptionsByCarrierID(txCtx, accountID, carrierID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeCarrier, carrier.ID, accountID, carrier); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.Delete(txCtx, accountID, carrierID); apiErr != nil {
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: accountID, CarrierID: carrierID})
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: accountID, CarrierID: carrierID})
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

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	carrier, apiErr := s.repos.NewCarrierRepo().Get(ctx, domain.GetCarrierParams{AccountID: accountID, CarrierID: carrierID})
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Carrier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
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

		existingOptions, apiErr := s.repos.NewCarrierRepo().ListOptionsByCarrierID(ctx, accountID, carrierID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Codes are unique per carrier, so a service the carrier offers is already present when any service level, synced or added by hand, holds its code.
		existingCodes := make(map[string]*domain.ServiceLevel, len(existingOptions))
		for _, opt := range existingOptions {
			existingCodes[opt.Code] = opt
		}

		var result *domain.Carrier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *carrierSvcImpl) *apierror.APIError {
			optionRepo := txSvc.repos.NewServiceLevelRepo()

			// Synced service levels the carrier no longer offers are archived, not deleted: orders and shipments may still use them. Ones added by hand are left alone.
			for _, opt := range existingOptions {
				if _, offered := shippoTokens[opt.Code]; offered || !opt.Synced || !opt.IsActive {
					continue
				}
				if apiErr := optionRepo.Archive(txCtx, accountID, opt.ID); apiErr != nil {
					return apiErr
				}
				archived := *opt
				archived.IsActive = false
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeServiceLevel,
					ResourceID:   opt.ID,
					Changes:      audit.ComputeChanges(opt, &archived),
				}); apiErr != nil {
					return apiErr
				}
			}

			for token, level := range shippoTokens {
				if _, exists := existingCodes[token]; exists {
					continue
				}
				optionID, apiErr := id.GenID(id.ServiceLevelIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				created, apiErr := optionRepo.Create(txCtx, optionID, domain.CreateServiceLevelParams{
					AccountID:       accountID,
					CarrierID:       carrierID,
					Name:            level.Name,
					Code:            token,
					Synced:          true,
					IsPortalEnabled: false,
					IsActive:        true,
				})
				if apiErr != nil {
					return apiErr
				}
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionCreate,
					ResourceType: constants.ObjectTypeServiceLevel,
					ResourceID:   created.ID,
					Changes:      audit.ComputeChanges(nil, created),
				}); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch carrier with updated options inside the transaction
			updatedCarrier, apiErr := txSvc.repos.NewCarrierRepo().Get(txCtx, domain.GetCarrierParams{AccountID: accountID, CarrierID: carrierID})
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

// checkDefaultServiceLevel refuses a default service level that isn't one of the carrier's own, or that is archived. One the carrier already has as its default is not checked again.
func (s *carrierSvcImpl) checkDefaultServiceLevel(ctx context.Context, carrier *domain.Carrier, next field.Clearable[string]) *apierror.APIError {
	const param = "default_service_level_id"
	id, ok := next.Value()
	if !ok || (carrier.DefaultServiceLevelID != nil && *carrier.DefaultServiceLevelID == id) {
		return nil
	}
	repo := s.repos.NewServiceLevelRepo()
	onCarrier, apiErr := repo.IsInCarrier(ctx, id, carrier.ID)
	if apiErr != nil {
		return apiErr
	}
	if !onCarrier {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "The default service level must be one of this carrier's service levels.")
	}
	level, apiErr := repo.Get(ctx, *carrier.AccountID, id)
	if apiErr != nil {
		return apiErr
	}
	return domain.RequireAssignable(param, "service level", constants.ArchiveStatusFromIsActive(level.IsActive))
}

// checkCarrierReadPermission checks the appropriate read permission based on the identity context. Internal actors need carriers:read for their own account, or customers:read / suppliers:read for external accounts.
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
