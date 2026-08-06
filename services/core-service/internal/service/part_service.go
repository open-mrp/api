package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/mediator"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
	"github.com/shopspring/decimal"
)

var partSvcTracer = tracing.GetTracer("core-service.part_service")

type partSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type PartSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service the async bulk upsert records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PartSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("part service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("part service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("part service: job service factory is required")
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
		jobSvcFactory:   config.JobSvcFactory,
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

	part, apiErr := s.repos.NewPartRepo().Get(ctx, domain.GetPartParams{
		AccountID: identity.Target.AccountID,
		PartID:    params.PartID,
		Includes:  params.Includes,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if part != nil && part.Item != nil {
		mediator.RefreshItemBurnRateAfterGet(ctx, s.repos, s.mediators(), identity.Target.AccountID, part.Item, params.Includes)
	}
	return part, nil
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
			created, apiErr := txSvc.createPartInTx(txCtx, params)
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

// createPartInTx inserts a part-type item, its rates, the part record, attributes,
// and opening inventory within an existing transaction, returning the fresh part.
// Shared by CreatePart (single) and BulkUpsertParts (batch); it does not handle the
// idempotency/permission envelope, which the callers own.
func (s *partSvcImpl) createPartInTx(txCtx context.Context, params domain.CreatePartParams) (*domain.Part, *apierror.APIError) {
	partID, apiErr := id.GenID(id.PartIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	itemID, apiErr := id.GenID(id.ItemIDPrefix, nil)
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

	txPartRepo := s.repos.NewPartRepo()
	txItemRepo := s.repos.NewItemRepo()

	// Check SKU uniqueness.
	exists, apiErr := txPartRepo.ExistsBySKU(txCtx, params.AccountID, params.SKU, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	if exists {
		return nil, apierror.NewConflictErrorWithParam(fmt.Sprintf("An item with the SKU '%s' already exists.", params.SKU), "sku")
	}

	// Get base unit for rates from category, and enforce that parts only use product
	// categories (same rule as the change-item-category endpoint).
	baseUnitID, categoryTypeCode, apiErr := txItemRepo.GetCategoryBaseUnitID(txCtx, params.CategoryID)
	if apiErr != nil {
		return nil, apiErr
	}
	if !categoryTypeMatchesItem(string(constants.ItemTypeCodePart), categoryTypeCode) {
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
	if apiErr := txPartRepo.InsertRate(txCtx, unitValueRateID, unitValueValue, unitValueNum, unitValueDen); apiErr != nil {
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
	if apiErr := txPartRepo.InsertRate(txCtx, unitCostRateID, unitCostValue, unitCostNum, unitCostDen); apiErr != nil {
		return nil, apiErr
	}

	// Burn rate is always initialized to "0" per day; it is recomputed
	// from inventory history by the burn-rate mediator.
	if apiErr := txPartRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, "day"); apiErr != nil {
		return nil, apiErr
	}

	// Insert item.
	if apiErr := txPartRepo.InsertItem(txCtx, itemID, params, unitValueRateID, burnRateRateID, unitCostRateID); apiErr != nil {
		return nil, apiErr
	}

	// Insert part record.
	created, apiErr := txPartRepo.Create(txCtx, partID, itemID, params)
	if apiErr != nil {
		return nil, apiErr
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
			return nil, apiErr
		}
	}

	result, apiErr := txPartRepo.Get(txCtx, domain.GetPartParams{AccountID: params.AccountID, PartID: created.ID, Includes: params.Includes})
	if apiErr != nil {
		return nil, apiErr
	}

	changes := audit.ComputeChanges(nil, result.Item)
	if apiErr := audit.NewPublisher().Publish(txCtx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionCreate,
		ResourceType: constants.ObjectTypePart,
		ResourceID:   result.ID,
		Changes:      changes,
	}); apiErr != nil {
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

	return result, nil
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
			updated, apiErr := txSvc.updatePartInTx(txCtx, params)
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

// updatePartInTx updates a part's item fields (sku, description, notes) within an
// existing transaction and returns the fresh part. Shared by UpdatePart (single) and
// BulkUpsertParts (batch); it does not own the idempotency/permission envelope.
func (s *partSvcImpl) updatePartInTx(txCtx context.Context, params domain.UpdatePartParams) (*domain.Part, *apierror.APIError) {
	txPartRepo := s.repos.NewPartRepo()

	// Fetch the part before update for audit diff (same includes as the
	// post-update fetch so include-only fields cannot produce false diffs).
	old, apiErr := txPartRepo.Get(txCtx, domain.GetPartParams{AccountID: params.AccountID, PartID: params.PartID, Includes: params.Includes})
	if apiErr != nil {
		return nil, apiErr
	}

	// Check SKU uniqueness if being updated, excluding the current item.
	if params.SKU != nil {
		excludeItemID := old.ItemID
		exists, apiErr := txPartRepo.ExistsBySKU(txCtx, params.AccountID, *params.SKU, &excludeItemID)
		if apiErr != nil {
			return nil, apiErr
		}
		if exists {
			return nil, apierror.NewConflictErrorWithParam(fmt.Sprintf("An item with the SKU '%s' already exists.", *params.SKU), "sku")
		}
	}

	if old.Item != nil {
		params.Description = params.Description.BackfillUnsetPtr(old.Item.Description)
		params.Notes = params.Notes.BackfillUnsetPtr(old.Item.Notes)
	}

	if apiErr := txPartRepo.UpdateItem(txCtx, domain.PartUpdateItemParams{
		AccountID:   params.AccountID,
		ItemID:      old.ItemID,
		SKU:         params.SKU,
		Description: params.Description,
		Notes:       params.Notes,
	}); apiErr != nil {
		return nil, apiErr
	}

	// Touch part updated_at to match dashboard behavior.
	if apiErr := txPartRepo.TouchUpdatedAt(txCtx, params.PartID); apiErr != nil {
		return nil, apiErr
	}

	// Fetch fresh part for response.
	updated, apiErr := txPartRepo.Get(txCtx, domain.GetPartParams{AccountID: params.AccountID, PartID: params.PartID, Includes: params.Includes})
	if apiErr != nil {
		return nil, apiErr
	}

	changes := audit.ComputeChanges(old.Item, updated.Item)
	if apiErr := audit.NewPublisher().Publish(txCtx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypePart,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return nil, apiErr
	}

	return updated, nil
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

func (s *partSvcImpl) BatchGetPartsByIDs(ctx context.Context, ids []string) ([]*domain.Part, *apierror.APIError) {
	ctx, span := partSvcTracer.Start(ctx, "service.part.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPartReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	parts, apiErr := s.repos.NewPartRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return parts, nil
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

// clearableFromPtr converts an optional pointer to a Clearable: a nil pointer is
// "unset" (leaves the existing value), a non-nil pointer sets the value.
func clearableFromPtr(v *string) field.Clearable[string] {
	if v == nil {
		return field.Unset[string]()
	}
	return field.Set(*v)
}
