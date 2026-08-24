package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var materialSvcTracer = tracing.GetTracer("core-service.material_service")

type materialSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type MaterialSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service the async bulk upsert records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *MaterialSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("material service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("material service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("material service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("material service: tx manager is required")
	}
	return nil
}

func NewMaterialSvc(config *MaterialSvcConfig) domain.MaterialSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &materialSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *materialSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *materialSvcImpl) withTx(ctx context.Context, fn func(context.Context, *materialSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &materialSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *materialSvcImpl) ListMaterials(ctx context.Context, params domain.ListMaterialsParams) (*domain.ListMaterialsResult, *apierror.APIError) {
	ctx, span := materialSvcTracer.Start(ctx, "service.material.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkMaterialReadPermission(identity); apiErr != nil {
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

	return s.repos.NewMaterialRepo().List(ctx, params)
}

// GetMaterial retrieves a single material by material ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and items:read permission.
// 2. Require the OpenMRP-Account header.
// 3. Fetch the material from the repository by account ID and item ID.
func (s *materialSvcImpl) GetMaterial(ctx context.Context, params domain.GetMaterialParams) (*domain.Material, *apierror.APIError) {
	ctx, span := materialSvcTracer.Start(ctx, "service.material.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkMaterialReadPermission(identity); apiErr != nil {
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

	material, apiErr := s.repos.NewMaterialRepo().GetByID(ctx, domain.GetMaterialParams{
		AccountID:  identity.Target.AccountID,
		MaterialID: params.MaterialID,
		Includes:   params.Includes,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if material != nil && material.Item != nil {
		mediator.RefreshItemBurnRateAfterGet(ctx, s.repos, s.mediators(), identity.Target.AccountID, material.Item, params.Includes)
	}
	return material, nil
}

// CreateMaterial creates a new material with its associated item, rates, and quantities, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and materials:create permission.
// 2. Generate unique IDs for the material, item, rates, and quantities.
// 3. Upsert an idempotency key; if already finished, return the cached response.
// 4. Within a transaction, check for duplicate SKU, create rates, item, quantities, and material.
// 5. Fetch the created material and cache the success response.
// 6. On error, cache the error response for idempotent replay.
func (s *materialSvcImpl) CreateMaterial(ctx context.Context, params domain.CreateMaterialParams) (*domain.Material, *apierror.APIError) {
	ctx, span := materialSvcTracer.Start(ctx, "service.material.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkMaterialWritePermission(identity, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Material](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Material
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *materialSvcImpl) *apierror.APIError {
			created, apiErr := txSvc.createMaterialInTx(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = created
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

// createMaterialInTx inserts a material-type item, its rates, attributes, order-point
// and lead-time quantities, the material record, and opening inventory within an
// existing transaction, returning the fresh material. Shared by CreateMaterial (single)
// and BulkUpsertMaterials (batch); it does not own the idempotency/permission envelope,
// and expects params.AccountID set.
func (s *materialSvcImpl) createMaterialInTx(txCtx context.Context, params domain.CreateMaterialParams) (*domain.Material, *apierror.APIError) {
	materialID, apiErr := id.GenID(id.MaterialIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	itemID, apiErr := id.GenID(id.ItemIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	orderPointQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	leadTimeQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	unitValueRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	unitCostRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	burnRateRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	txMaterialRepo := s.repos.NewMaterialRepo()
	txItemRepo := s.repos.NewItemRepo()

	// Check SKU uniqueness.
	exists, apiErr := txItemRepo.CheckSKUExists(txCtx, params.AccountID, params.SKU, "")
	if apiErr != nil {
		return nil, apiErr
	}
	if exists {
		return nil, apierror.NewConflictErrorWithParam(fmt.Sprintf("An item with the SKU '%s' already exists.", params.SKU), "sku")
	}

	// Get base unit for rates from category, and enforce that materials only use
	// material categories (same rule as the change-item-category endpoint).
	baseUnitID, categoryTypeCode, apiErr := txItemRepo.GetCategoryBaseUnitID(txCtx, params.CategoryID)
	if apiErr != nil {
		// A category that does not exist is a bad caller-supplied reference, so it is reported against the field like the other foreign keys on this request rather than as a bare not-found for the material itself.
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewValidationErrorWithParam("The category does not exist.", "category_id")
		}
		return nil, apiErr
	}
	if !categoryTypeMatchesItem(string(constants.ItemTypeCodeMaterial), categoryTypeCode) {
		return nil, apierror.NewValidationErrorWithParam("This category type cannot be assigned to this item type.", "category_id")
	}

	// Insert rates for item (unit_value, unit_cost, burn_rate). Caller-supplied
	// inputs override the defaults; unit_price and unit_cost additionally enforce
	// the currency-numerator / non-currency-denominator rule.
	txUnitRepo := s.repos.NewUnitRepo()

	unitValueValue, unitValueNum, unitValueDen := "0", baseUnitID, baseUnitID
	if params.UnitPrice != nil {
		if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitPrice.NumeratorUnitID, params.UnitPrice.DenominatorUnitID, "unit_price"); apiErr != nil {
			return nil, apiErr
		}
		unitValueValue = params.UnitPrice.Value
		unitValueNum = params.UnitPrice.NumeratorUnitID
		unitValueDen = params.UnitPrice.DenominatorUnitID
	}
	if apiErr := txMaterialRepo.InsertRate(txCtx, unitValueRateID, unitValueValue, unitValueNum, unitValueDen); apiErr != nil {
		return nil, apiErr
	}

	unitCostValue, unitCostNum, unitCostDen := "0", baseUnitID, baseUnitID
	if params.UnitCost != nil {
		if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitCost.NumeratorUnitID, params.UnitCost.DenominatorUnitID, "unit_cost"); apiErr != nil {
			return nil, apiErr
		}
		unitCostValue = params.UnitCost.Value
		unitCostNum = params.UnitCost.NumeratorUnitID
		unitCostDen = params.UnitCost.DenominatorUnitID
	}
	if apiErr := txMaterialRepo.InsertRate(txCtx, unitCostRateID, unitCostValue, unitCostNum, unitCostDen); apiErr != nil {
		return nil, apiErr
	}

	// Burn rate is always initialized to "0" per day; it is recomputed
	// from inventory history by the burn-rate mediator.
	if apiErr := txMaterialRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, "day"); apiErr != nil {
		return nil, apiErr
	}

	// Insert item.
	if apiErr := txMaterialRepo.InsertItem(txCtx, domain.InsertMaterialItemParams{
		ItemID:          itemID,
		AccountID:       params.AccountID,
		SKU:             params.SKU,
		Description:     params.Description,
		Notes:           params.Notes,
		CategoryID:      params.CategoryID,
		UnitValueRateID: unitValueRateID,
		UnitCostRateID:  unitCostRateID,
		BurnRateRateID:  burnRateRateID,
	}); apiErr != nil {
		return nil, apiErr
	}

	// Link caller-supplied attributes to the new item (matches Dashboard behavior).
	if apiErr := attachItemAttributesInTx(txCtx, s.repos, params.AccountID, params.CategoryID, itemID, params.AttributeIDs); apiErr != nil {
		return nil, apiErr
	}

	// Insert order point quantity.
	opValue := "0"
	opUnitID := baseUnitID
	if params.OrderPoint != nil {
		opValue = params.OrderPoint.Value
		opUnitID = params.OrderPoint.UnitID
	}
	if apiErr := txMaterialRepo.InsertQuantity(txCtx, orderPointQtyID, opValue, opUnitID); apiErr != nil {
		return nil, apiErr
	}

	// Insert lead time quantity.
	ltValue := "0"
	ltUnitID := baseUnitID
	if params.LeadTime != nil {
		ltValue = params.LeadTime.Value
		ltUnitID = params.LeadTime.UnitID
	}
	if apiErr := txMaterialRepo.InsertQuantity(txCtx, leadTimeQtyID, ltValue, ltUnitID); apiErr != nil {
		return nil, apiErr
	}

	// Insert material record.
	if apiErr := txMaterialRepo.Create(txCtx, materialID, itemID, orderPointQtyID, leadTimeQtyID); apiErr != nil {
		return nil, apiErr
	}

	// Initialize inventory tracking with zero-quantity log and change log.
	txInvMutRepo := s.repos.NewInventoryMutationRepo()
	zeroMeasure := decimal.Zero

	if apiErr := txInvMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
		AccountID: params.AccountID,
		ItemID:    itemID,
		Measure:   zeroMeasure,
		UnitID:    baseUnitID,
	}); apiErr != nil {
		return nil, apiErr
	}

	if apiErr := txInvMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
		AccountID:  params.AccountID,
		ItemID:     itemID,
		Measure:    zeroMeasure,
		UnitID:     baseUnitID,
		ActionType: "user_action",
	}); apiErr != nil {
		return nil, apiErr
	}

	// Fetch fresh material for response.
	created, apiErr := txMaterialRepo.GetByID(txCtx, domain.GetMaterialParams{AccountID: params.AccountID, MaterialID: materialID, Includes: params.Includes})
	if apiErr != nil {
		return nil, apiErr
	}

	changes := audit.ComputeChanges(nil, created)

	if apiErr := audit.NewPublisher().Publish(txCtx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionCreate,
		ResourceType: constants.ObjectTypeMaterial,
		ResourceID:   created.ID,
		Changes:      changes,
	}); apiErr != nil {
		return nil, apiErr
	}

	return created, nil
}

// UpdateMaterial partially updates an existing material, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and materials:update permission.
// 2. Upsert an idempotency key; if already finished, return the cached response.
// 3. Within a transaction, verify the material exists, check for duplicate SKU if changed.
// 4. Update item fields, quantities, and material timestamp, then cache the success response.
// 5. On error, cache the error response for idempotent replay.
func (s *materialSvcImpl) UpdateMaterial(ctx context.Context, params domain.UpdateMaterialParams) (*domain.Material, *apierror.APIError) {
	ctx, span := materialSvcTracer.Start(ctx, "service.material.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkMaterialWritePermission(identity, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Material](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Material
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *materialSvcImpl) *apierror.APIError {
			updated, apiErr := txSvc.updateMaterialInTx(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated
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

// updateMaterialInTx updates a material's item fields (sku/description/notes), order-point
// and lead-time quantities, and unit_cost within an existing transaction, returning the
// fresh material. Shared by UpdateMaterial (single) and BulkUpsertMaterials (batch); it
// does not own the idempotency/permission envelope, and expects params.AccountID set.
func (s *materialSvcImpl) updateMaterialInTx(txCtx context.Context, params domain.UpdateMaterialParams) (*domain.Material, *apierror.APIError) {
	txMaterialRepo := s.repos.NewMaterialRepo()
	txItemRepo := s.repos.NewItemRepo()

	// Verify the material exists. Load with the same includes as the
	// post-update fetch so include-only fields cannot produce false audit diffs.
	existing, apiErr := txMaterialRepo.GetByID(txCtx, domain.GetMaterialParams{AccountID: params.AccountID, MaterialID: params.MaterialID, Includes: params.Includes})
	if apiErr != nil {
		return nil, apiErr
	}

	// Check SKU uniqueness if being updated, excluding the current item.
	if params.SKU != nil {
		exists, apiErr := txItemRepo.CheckSKUExists(txCtx, params.AccountID, *params.SKU, existing.ItemID)
		if apiErr != nil {
			return nil, apiErr
		}
		if exists {
			return nil, apierror.NewConflictErrorWithParam(fmt.Sprintf("An item with the SKU '%s' already exists.", *params.SKU), "sku")
		}
	}

	if params.UnitCost != nil {
		txUnitRepo := s.repos.NewUnitRepo()
		if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.UnitCost.NumeratorUnitID, params.UnitCost.DenominatorUnitID, "unit_cost"); apiErr != nil {
			return nil, apiErr
		}
		if existing.Item == nil || existing.Item.UnitCostID == "" {
			return nil, apierror.NewInvariantViolationError("Material item or unit cost rate is missing.")
		}
		if apiErr := txItemRepo.UpdateRate(txCtx, existing.Item.UnitCostID, *params.UnitCost); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := txItemRepo.ClearItemDirtyFlag(txCtx, params.AccountID, existing.ItemID); apiErr != nil {
			return nil, apiErr
		}
	}

	// Update item fields (sku, description, notes).
	if apiErr := txMaterialRepo.UpdateItem(txCtx, params); apiErr != nil {
		return nil, apiErr
	}

	// Update order point quantity if provided.
	if params.OrderPoint != nil && existing.OrderPoint != nil {
		if apiErr := txMaterialRepo.UpdateQuantity(txCtx, existing.OrderPoint.ID, params.OrderPoint.Value, params.OrderPoint.UnitID); apiErr != nil {
			return nil, apiErr
		}
	}

	// Update lead time quantity if provided.
	if params.LeadTime != nil && existing.LeadTime != nil {
		if apiErr := txMaterialRepo.UpdateQuantity(txCtx, existing.LeadTime.ID, params.LeadTime.Value, params.LeadTime.UnitID); apiErr != nil {
			return nil, apiErr
		}
	}

	// Update material timestamp.
	if apiErr := txMaterialRepo.Update(txCtx, params); apiErr != nil {
		return nil, apiErr
	}

	// Fetch fresh material for response.
	updated, apiErr := txMaterialRepo.GetByID(txCtx, domain.GetMaterialParams{AccountID: params.AccountID, MaterialID: params.MaterialID, Includes: params.Includes})
	if apiErr != nil {
		return nil, apiErr
	}

	changes := audit.ComputeChanges(existing, updated)

	if apiErr := audit.NewPublisher().Publish(txCtx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeMaterial,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return nil, apiErr
	}

	return updated, nil
}

// DeleteMaterial soft-deletes a material by its material ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and materials:delete permission.
// 2. Fetch the material to verify it exists.
// 3. Within a transaction, soft-delete the material.
// 4. Return the material as it was before deletion.
func (s *materialSvcImpl) DeleteMaterial(ctx context.Context, materialID string) (*domain.Material, *apierror.APIError) {
	ctx, span := materialSvcTracer.Start(ctx, "service.material.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkMaterialWritePermission(identity, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	accountID := identity.Target.AccountID

	// Fetch existing material before deletion.
	material, apiErr := s.repos.NewMaterialRepo().GetByID(ctx, domain.GetMaterialParams{AccountID: accountID, MaterialID: materialID})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeMaterial, materialID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This material has already been deleted and can no longer be modified."),
				)
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	// Soft-delete within a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *materialSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeMaterial, material.ID, material); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewMaterialRepo().DeleteByID(txCtx, accountID, materialID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(material, (*domain.Material)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeMaterial,
			ResourceID:   material.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return material, nil
}

// checkMaterialReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need materials:read for their own account, or customers:read / suppliers:read for external accounts.
func (s *materialSvcImpl) BatchGetMaterialsByIDs(ctx context.Context, ids []string) ([]*domain.Material, *apierror.APIError) {
	ctx, span := materialSvcTracer.Start(ctx, "service.material.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkMaterialReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	materials, apiErr := s.repos.NewMaterialRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return materials, nil
}

func checkMaterialReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainMaterials, types.ActionRead)
}

// checkMaterialWritePermission checks the appropriate write permission based on the identity context.
// Internal actors need materials:{action} for their own account, or customers:update / suppliers:update for external accounts.
func checkMaterialWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainMaterials, action)
}
