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

var unitGroupSvcTracer = tracing.GetTracer("core-service.unit_group_service")

type unitGroupSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type UnitGroupSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *UnitGroupSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("unit group service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("unit group service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("unit group service: tx manager is required")
	}
	return nil
}

func NewUnitGroupSvc(config *UnitGroupSvcConfig) domain.UnitGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitGroupSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *unitGroupSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *unitGroupSvcImpl) withTx(ctx context.Context, fn func(context.Context, *unitGroupSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &unitGroupSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *unitGroupSvcImpl) ListUnitGroups(ctx context.Context, params domain.ListUnitGroupsParams) (*domain.ListUnitGroupsResult, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewUnitGroupRepo().List(ctx, params)
}

func (s *unitGroupSvcImpl) GetUnitGroup(ctx context.Context, unitGroupID string) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	return s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
		AccountID:   identity.Target.AccountID,
		UnitGroupID: unitGroupID,
	})
}

func (s *unitGroupSvcImpl) CreateUnitGroup(ctx context.Context, params domain.CreateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitGroupID, apiErr := id.GenID(id.UnitGroupIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.UnitGroupFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.UnitGroupFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitGroupSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewUnitGroupRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A unit group with this name already exists.", "name")
			}

			// Validate all unit conversions match the group's type
			if apiErr := s.validateUnitConversionTypes(txCtx, params.AccountID, params.Type, params.UnitConversions); apiErr != nil {
				return apiErr
			}

			if _, apiErr := txRepo.Create(txCtx, unitGroupID, params); apiErr != nil {
				return apiErr
			}

			// Create unit conversions
			for _, uc := range params.UnitConversions {
				ucID, apiErr := id.GenID(id.UnitGroupsUnitsIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				_, apiErr = txRepo.UpsertUnitGroupUnit(txCtx, ucID, domain.UpsertUnitGroupUnitParams{
					AccountID:          params.AccountID,
					UnitGroupID:        unitGroupID,
					UnitGroupUnitID:    ucID,
					UnitID:             uc.UnitID,
					DiscountPercentage: uc.DiscountPercentage,
					DiscountFixed:      uc.DiscountFixed,
					IsVisible:          uc.IsVisible,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch with conversions
			created, apiErr := txRepo.Get(txCtx, domain.GetUnitGroupParams{
				AccountID:   params.AccountID,
				UnitGroupID: unitGroupID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeUnitGroup,
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

func (s *unitGroupSvcImpl) UpdateUnitGroup(ctx context.Context, params domain.UpdateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.UnitGroupFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.UnitGroupFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitGroupSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewUnitGroupRepo()

			existing, apiErr := txRepo.Get(txCtx, domain.GetUnitGroupParams{
				AccountID:   params.AccountID,
				UnitGroupID: params.UnitGroupID,
			})
			if apiErr != nil {
				return apiErr
			}
			if existing.AccountID == nil {
				return apierror.NewValidationError("System unit groups cannot be modified.")
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.UnitGroupID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A unit group with this name already exists.", "name")
				}
			}

			// Validate unit conversions match the group's type if provided
			if params.UnitConversions != nil {
				if apiErr := s.validateUnitConversionTypes(txCtx, params.AccountID, existing.Type, *params.UnitConversions); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			// If unit conversions are provided, upsert them (additive — existing conversions not in the list are preserved)
			if params.UnitConversions != nil {
				for _, uc := range *params.UnitConversions {
					ucID, apiErr := id.GenID(id.UnitGroupsUnitsIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					_, apiErr = txRepo.UpsertUnitGroupUnit(txCtx, ucID, domain.UpsertUnitGroupUnitParams{
						AccountID:          params.AccountID,
						UnitGroupID:        params.UnitGroupID,
						UnitGroupUnitID:    ucID,
						UnitID:             uc.UnitID,
						DiscountPercentage: uc.DiscountPercentage,
						DiscountFixed:      uc.DiscountFixed,
						IsVisible:          uc.IsVisible,
					})
					if apiErr != nil {
						return apiErr
					}
				}

				// Re-fetch with updated conversions
				updated, apiErr = txRepo.Get(txCtx, domain.GetUnitGroupParams{
					AccountID:   params.AccountID,
					UnitGroupID: params.UnitGroupID,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			result = updated

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeUnitGroup,
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

func (s *unitGroupSvcImpl) DeleteUnitGroup(ctx context.Context, unitGroupID string) *apierror.APIError {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	existing, apiErr := s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
		AccountID:   accountID,
		UnitGroupID: unitGroupID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeUnitGroup, unitGroupID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This unit group has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if existing.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("System unit groups cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitGroupSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewUnitGroupRepo()

		// Cascade delete all unit_group_unit records first
		if apiErr := txRepo.DeleteAllUnitGroupUnits(txCtx, accountID, unitGroupID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeUnitGroup, existing.ID, existing); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.Delete(txCtx, domain.DeleteUnitGroupParams{
			AccountID:   accountID,
			UnitGroupID: unitGroupID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(existing, (*domain.UnitGroupFull)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeUnitGroup,
			ResourceID:   existing.ID,
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

func (s *unitGroupSvcImpl) UpsertUnitGroupUnit(ctx context.Context, params domain.UpsertUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.upsert_unit")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Generate an ID if one was not provided (create vs update).
	if params.UnitGroupUnitID == "" {
		genID, apiErr := id.GenID(id.UnitGroupsUnitsIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.UnitGroupUnitID = genID
	}

	// Verify unit group exists and is owned by the account
	existing, apiErr := s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if existing.AccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("System unit groups cannot be modified."))
	}

	// For updates on an existing record, fetch current values and merge so the
	// upsert INSERT clause has all required columns populated.
	isUpdate := params.UnitID == "" || params.DiscountPercentage == "" || params.DiscountFixed == ""
	if isUpdate {
		for _, uc := range existing.UnitConversions {
			if uc.ID == params.UnitGroupUnitID {
				if params.UnitID == "" {
					params.UnitID = uc.UnitID
				}
				if params.DiscountPercentage == "" {
					params.DiscountPercentage = uc.DiscountPercentage
				}
				if params.DiscountFixed == "" {
					params.DiscountFixed = uc.DiscountFixed
				}
				break
			}
		}
	}

	// Validate unit type matches (only when unit_id is being set/changed).
	if params.UnitID != "" {
		unit, apiErr := s.repos.NewUnitQueryRepo().Find(ctx, params.AccountID, params.UnitID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if unit.Type != existing.Type {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unit type does not match the unit group type.", "unit_id"))
		}
	}

	var result *domain.UnitGroupUnit
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitGroupSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewUnitGroupRepo()

		upserted, apiErr := txRepo.UpsertUnitGroupUnit(txCtx, params.UnitGroupUnitID, params)
		if apiErr != nil {
			return apiErr
		}
		result = upserted
		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *unitGroupSvcImpl) DeleteUnitGroupUnit(ctx context.Context, params domain.DeleteUnitGroupUnitParams) *apierror.APIError {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.delete_unit")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Verify unit group exists
	existing, apiErr := s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if existing.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("System unit groups cannot be modified."))
	}

	// Find the unit group unit to be deleted for audit
	var oldUnit *domain.UnitGroupUnit
	for _, uc := range existing.UnitConversions {
		if uc.ID == params.UnitGroupUnitID {
			oldUnit = uc
			break
		}
	}

	if oldUnit == nil {
		wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeUnitGroupUnit, params.UnitGroupUnitID)
		if deletedCheckErr != nil {
			return tracing.Trace(span, deletedCheckErr)
		}
		if wasDeleted {
			return tracing.Trace(span, apierror.NewAlreadyDeletedError("This unit group unit has already been deleted and can no longer be modified."))
		}
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitGroupSvcImpl) *apierror.APIError {
		if oldUnit != nil {
			if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeUnitGroupUnit, oldUnit.ID, oldUnit); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.repos.NewUnitGroupRepo().DeleteUnitGroupUnit(txCtx, params); apiErr != nil {
			return apiErr
		}

		if oldUnit != nil {
			changes := audit.ComputeChanges(oldUnit, (*domain.UnitGroupUnit)(nil))

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionDelete,
				ResourceType: constants.ObjectTypeUnitGroupUnit,
				ResourceID:   oldUnit.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (s *unitGroupSvcImpl) ListUnitGroupUnits(ctx context.Context, unitGroupID string) ([]*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.list_units")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	// Verify unit group exists
	_, apiErr := s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
		AccountID:   identity.Target.AccountID,
		UnitGroupID: unitGroupID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewUnitGroupRepo().ListUnits(ctx, unitGroupID)
}

func (s *unitGroupSvcImpl) GetUnitGroupUnit(ctx context.Context, params domain.GetUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(ctx, "service.unit_group.get_unit")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainUnitGroups, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	params.AccountID = identity.Target.AccountID

	// Verify unit group exists
	_, apiErr := s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewUnitGroupRepo().GetUnit(ctx, params)
}

// validateUnitConversionTypes checks that all unit conversions have units matching the group's type.
func (s *unitGroupSvcImpl) validateUnitConversionTypes(ctx context.Context, accountID, groupType string, conversions []domain.CreateUnitGroupUnitParams) *apierror.APIError {
	for _, uc := range conversions {
		unit, apiErr := s.repos.NewUnitQueryRepo().Find(ctx, accountID, uc.UnitID)
		if apiErr != nil {
			return apiErr
		}
		if unit.Type != groupType {
			return apierror.NewValidationErrorWithParam("Unit type does not match the unit group type.", "unit_conversions")
		}
	}
	return nil
}
