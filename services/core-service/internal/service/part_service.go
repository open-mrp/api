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
	"github.com/shopspring/decimal"
)

var partSvcTracer = tracing.GetTracer("core-service.part_service")

type partSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type PartSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *PartSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("part service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("part service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("part service: tx manager is required")
	}
	return nil
}

func NewPartSvc(config *PartSvcConfig) domain.PartSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &partSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *partSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *partSvcImpl) withTx(ctx context.Context, fn func(context.Context, *partSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &partSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *partSvcImpl) ListParts(ctx context.Context, params domain.ListPartsParams) (*domain.ListPartsResult, *apierror.APIError) {
	ctx, span := partSvcTracer.Start(ctx, "service.part.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPartReadPermission(identity); apiErr != nil {
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

	return s.repos.NewPartRepo().List(ctx, params)
}

func (s *partSvcImpl) GetPart(ctx context.Context, params domain.GetPartParams) (*domain.Part, *apierror.APIError) {
	ctx, span := partSvcTracer.Start(ctx, "service.part.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPartReadPermission(identity); apiErr != nil {
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

	return s.repos.NewPartRepo().Get(ctx, domain.GetPartParams{
		AccountID: identity.Target.AccountID,
		PartID:    params.PartID,
		Includes:  params.Includes,
	})
}

func (s *partSvcImpl) CreatePart(ctx context.Context, params domain.CreatePartParams) (*domain.Part, *apierror.APIError) {
	ctx, span := partSvcTracer.Start(ctx, "service.part.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPartWritePermission(identity, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	partID, apiErr := id.GenID(id.PartIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	itemID, apiErr := id.GenID(id.ItemIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitValueRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitCostRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	burnRateRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Part](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Part
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *partSvcImpl) *apierror.APIError {
			txPartRepo := txSvc.repos.NewPartRepo()
			txItemRepo := txSvc.repos.NewItemRepo()

			// Check SKU uniqueness.
			exists, apiErr := txPartRepo.ExistsBySKU(txCtx, params.AccountID, params.SKU, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("An item with this SKU already exists.", "sku")
			}

			// Get base unit for rates from category.
			baseUnitID, apiErr := txItemRepo.GetCategoryBaseUnitID(txCtx, params.CategoryID)
			if apiErr != nil {
				return apiErr
			}

			// Insert rates for item (unit_value, unit_cost, burn_rate). Caller-supplied
			// inputs override the defaults; unit_price and unit_cost additionally enforce
			// the currency-numerator / non-currency-denominator rule.
			txUnitRepo := txSvc.repos.NewUnitRepo()

			unitValueValue, unitValueNum, unitValueDen := "0", baseUnitID, baseUnitID
			if params.UnitPrice != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitPrice.NumeratorUnitID, params.UnitPrice.DenominatorUnitID, "unit_price"); apiErr != nil {
					return apiErr
				}
				unitValueValue = params.UnitPrice.Value
				unitValueNum = params.UnitPrice.NumeratorUnitID
				unitValueDen = params.UnitPrice.DenominatorUnitID
			}
			if apiErr := txPartRepo.InsertRate(txCtx, unitValueRateID, unitValueValue, unitValueNum, unitValueDen); apiErr != nil {
				return apiErr
			}

			unitCostValue, unitCostNum, unitCostDen := "0", baseUnitID, baseUnitID
			if params.UnitCost != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitCost.NumeratorUnitID, params.UnitCost.DenominatorUnitID, "unit_cost"); apiErr != nil {
					return apiErr
				}
				unitCostValue = params.UnitCost.Value
				unitCostNum = params.UnitCost.NumeratorUnitID
				unitCostDen = params.UnitCost.DenominatorUnitID
			}
			if apiErr := txPartRepo.InsertRate(txCtx, unitCostRateID, unitCostValue, unitCostNum, unitCostDen); apiErr != nil {
				return apiErr
			}

			burnValue, burnNum, burnDen := "0", baseUnitID, baseUnitID
			if params.BurnRate != nil {
				burnValue = params.BurnRate.Value
				burnNum = params.BurnRate.NumeratorUnitID
				burnDen = params.BurnRate.DenominatorUnitID
			}
			if apiErr := txPartRepo.InsertRate(txCtx, burnRateRateID, burnValue, burnNum, burnDen); apiErr != nil {
				return apiErr
			}

			// Insert item.
			if apiErr := txPartRepo.InsertItem(txCtx, itemID, params, unitValueRateID, burnRateRateID, unitCostRateID); apiErr != nil {
				return apiErr
			}

			// Insert part record.
			created, apiErr := txPartRepo.Create(txCtx, partID, itemID, params)
			if apiErr != nil {
				return apiErr
			}

			// Link caller-supplied attributes to the new item (matches Dashboard behavior).
			for _, attrID := range params.AttributeIDs {
				if attrID == "" {
					continue
				}
				if apiErr := txItemRepo.AddAttribute(txCtx, domain.AddItemAttributeParams{
					AccountID:   params.AccountID,
					ItemID:      itemID,
					AttributeID: attrID,
				}); apiErr != nil {
					return apiErr
				}
			}

			result, apiErr = txPartRepo.Get(txCtx, domain.GetPartParams{AccountID: params.AccountID, PartID: created.ID, Includes: params.Includes})
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(nil, result.Item)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypePart,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Initialize inventory tracking with zero-quantity log and change log.
			txInvMutRepo := txSvc.repos.NewInventoryMutationRepo()
			zeroMeasure := decimal.Zero

			if apiErr := txInvMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
				AccountID: params.AccountID,
				ItemID:    itemID,
				Measure:   zeroMeasure,
				UnitID:    baseUnitID,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := txInvMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
				AccountID:  params.AccountID,
				ItemID:     itemID,
				Measure:    zeroMeasure,
				UnitID:     baseUnitID,
				ActionType: "user_action",
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

func (s *partSvcImpl) UpdatePart(ctx context.Context, params domain.UpdatePartParams) (*domain.Part, *apierror.APIError) {
	ctx, span := partSvcTracer.Start(ctx, "service.part.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPartWritePermission(identity, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Part](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Part
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *partSvcImpl) *apierror.APIError {
			txPartRepo := txSvc.repos.NewPartRepo()
			txItemRepo := txSvc.repos.NewItemRepo()

			// Fetch the part before update for audit diff.
			old, apiErr := txPartRepo.Get(txCtx, domain.GetPartParams{AccountID: params.AccountID, PartID: params.PartID})
			if apiErr != nil {
				return apiErr
			}

			// Check SKU uniqueness if being updated, excluding the current item.
			if params.SKU != nil {
				excludeItemID := old.ItemID
				exists, apiErr := txPartRepo.ExistsBySKU(txCtx, params.AccountID, *params.SKU, &excludeItemID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("An item with this SKU already exists.", "sku")
				}
			}

			// Update item fields (sku, description, notes).
			if apiErr := txItemRepo.Update(txCtx, domain.UpdateItemParams{
				AccountID:         params.AccountID,
				ItemID:            old.ItemID,
				SKU:               params.SKU,
				Description:       params.Description,
				UpdateDescription: params.UpdateDescription,
				Notes:             params.Notes,
				UpdateNotes:       params.UpdateNotes,
			}); apiErr != nil {
				return apiErr
			}

			// Touch part updated_at to match dashboard behavior.
			if apiErr := txPartRepo.TouchUpdatedAt(txCtx, params.PartID); apiErr != nil {
				return apiErr
			}

			// Fetch fresh part for response.
			updated, apiErr := txPartRepo.Get(txCtx, domain.GetPartParams{AccountID: params.AccountID, PartID: params.PartID, Includes: params.Includes})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old.Item, updated.Item)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePart,
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

func (s *partSvcImpl) DeletePart(ctx context.Context, partID string) (*domain.Part, *apierror.APIError) {
	ctx, span := partSvcTracer.Start(ctx, "service.part.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPartWritePermission(identity, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	accountID := identity.Target.AccountID

	// Fetch existing part before deletion.
	part, apiErr := s.repos.NewPartRepo().Get(ctx, domain.GetPartParams{AccountID: accountID, PartID: partID})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypePart, partID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This part has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	// Soft-delete within a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *partSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypePart, part.ID, part); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewPartRepo().Delete(txCtx, domain.DeletePartParams{
			AccountID: accountID,
			PartID:    partID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(part.Item, (*domain.Item)(nil))
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypePart,
			ResourceID:   part.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return part, nil
}

// checkPartReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need parts:read for their own account, or customers:read / suppliers:read for external accounts.
func checkPartReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainParts, types.ActionRead)
}

// checkPartWritePermission checks the appropriate write permission based on the identity context.
// Internal actors need parts:{action} for their own account, or customers:update / suppliers:update for external accounts.
func checkPartWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainParts, action)
}
