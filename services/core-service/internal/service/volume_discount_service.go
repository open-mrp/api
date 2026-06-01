package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var volumeDiscountSvcTracer = tracing.GetTracer("core-service.volume_discount_service")

type volumeDiscountSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type VolumeDiscountSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *VolumeDiscountSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("volume discount service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("volume discount service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("volume discount service: tx manager is required")
	}
	return nil
}

func NewVolumeDiscountSvc(config *VolumeDiscountSvcConfig) domain.VolumeDiscountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &volumeDiscountSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *volumeDiscountSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *volumeDiscountSvcImpl) withTx(ctx context.Context, fn func(context.Context, *volumeDiscountSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &volumeDiscountSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *volumeDiscountSvcImpl) ListVolumeDiscounts(ctx context.Context, params domain.ListVolumeDiscountsParams) (*domain.ListVolumeDiscountsResult, *apierror.APIError) {
	ctx, span := volumeDiscountSvcTracer.Start(ctx, "service.volume_discount.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkVolumeDiscountReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		params.CustomerAccountID = actorAccountID
	}

	return s.repos.NewVolumeDiscountRepo().List(ctx, params)
}

func (s *volumeDiscountSvcImpl) GetVolumeDiscount(ctx context.Context, params domain.GetVolumeDiscountParams) (*domain.VolumeDiscount, *apierror.APIError) {
	ctx, span := volumeDiscountSvcTracer.Start(ctx, "service.volume_discount.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkVolumeDiscountReadPermission(identity); apiErr != nil {
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

	return s.repos.NewVolumeDiscountRepo().Get(ctx, params)
}

func (s *volumeDiscountSvcImpl) CreateVolumeDiscount(ctx context.Context, params domain.CreateVolumeDiscountParams) (*domain.VolumeDiscount, *apierror.APIError) {
	ctx, span := volumeDiscountSvcTracer.Start(ctx, "service.volume_discount.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	discountID, apiErr := id.GenID(id.QuantityDiscountIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Generate IDs for tiers
	for i := range params.Tiers {
		tierID, apiErr := id.GenID(id.QuantityDiscountTierIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.Tiers[i].ID = tierID
	}

	// Generate IDs for customer group associations
	for i := range params.CustomerGroups {
		cgID, apiErr := id.GenID(id.AccountGroupQuantityDiscountIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.CustomerGroups[i].ID = cgID
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.VolumeDiscount](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.VolumeDiscount
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *volumeDiscountSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewVolumeDiscountRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A volume discount with this name already exists.", "name")
			}

			created, apiErr := txRepo.Create(txCtx, discountID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeVolumeDiscount,
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

func (s *volumeDiscountSvcImpl) UpdateVolumeDiscount(ctx context.Context, params domain.UpdateVolumeDiscountParams) (*domain.VolumeDiscount, *apierror.APIError) {
	ctx, span := volumeDiscountSvcTracer.Start(ctx, "service.volume_discount.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Generate IDs for new tiers
	for i := range params.Tiers {
		if params.Tiers[i].ID == nil {
			tierID, apiErr := id.GenID(id.QuantityDiscountTierIDPrefix, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			params.Tiers[i].GeneratedID = tierID
		}
	}

	// Generate IDs for customer group associations
	for i := range params.CustomerGroups {
		cgID, apiErr := id.GenID(id.AccountGroupQuantityDiscountIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.CustomerGroups[i].ID = cgID
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.VolumeDiscount](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.VolumeDiscount
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *volumeDiscountSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewVolumeDiscountRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetVolumeDiscountParams{
				AccountID:        params.AccountID,
				VolumeDiscountID: params.VolumeDiscountID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.VolumeDiscountID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A volume discount with this name already exists.", "name")
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
				ResourceType: constants.ObjectTypeVolumeDiscount,
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

func (s *volumeDiscountSvcImpl) DeleteVolumeDiscount(ctx context.Context, volumeDiscountID string) *apierror.APIError {
	ctx, span := volumeDiscountSvcTracer.Start(ctx, "service.volume_discount.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	volumeDiscount, apiErr := s.repos.NewVolumeDiscountRepo().Get(ctx, domain.GetVolumeDiscountParams{
		AccountID:        accountID,
		VolumeDiscountID: volumeDiscountID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeVolumeDiscount, volumeDiscountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This volume discount has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *volumeDiscountSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeVolumeDiscount, volumeDiscount.ID, volumeDiscount); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewVolumeDiscountRepo().Delete(txCtx, domain.DeleteVolumeDiscountParams{
			AccountID:        accountID,
			VolumeDiscountID: volumeDiscountID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(volumeDiscount, (*domain.VolumeDiscount)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeVolumeDiscount,
			ResourceID:   volumeDiscount.ID,
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

// checkVolumeDiscountReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need discounts:read for their own account, or customers:read / suppliers:read for external accounts.
func checkVolumeDiscountReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead)
}
