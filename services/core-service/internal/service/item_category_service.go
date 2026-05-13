package service

import (
	"context"
	"fmt"
	"slices"

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

var itemCategorySvcTracer = tracing.GetTracer("core-service.item_category_service")

type itemCategorySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ItemCategorySvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ItemCategorySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("item category service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("item category service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("item category service: tx manager is required")
	}
	return nil
}

func NewItemCategorySvc(config *ItemCategorySvcConfig) domain.ItemCategorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &itemCategorySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *itemCategorySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *itemCategorySvcImpl) withTx(ctx context.Context, fn func(context.Context, *itemCategorySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &itemCategorySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func loadItemCategoryFullTx(ctx context.Context, txRepo domain.ItemCategoryRepo, accountID, itemCategoryID string) (*domain.ItemCategoryFull, *apierror.APIError) {
	full, apiErr := txRepo.Get(ctx, domain.GetItemCategoryParams{
		AccountID:      accountID,
		ItemCategoryID: itemCategoryID,
	})
	if apiErr != nil {
		return nil, apiErr
	}
	props, apiErr := txRepo.GetProperties(ctx, itemCategoryID)
	if apiErr != nil {
		return nil, apiErr
	}
	full.Properties = props
	ug, apiErr := txRepo.GetUnitGroup(ctx, full.UnitGroupID, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	full.UnitGroup = ug
	return full, nil
}

func (s *itemCategorySvcImpl) ListItemCategories(ctx context.Context, params domain.ListItemCategoriesParams) (*domain.ListItemCategoriesResult, *apierror.APIError) {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewItemCategoryRepo()

	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, category := range result.ItemCategories {
		if slices.Contains(params.Includes, "properties") {
			properties, apiErr := repo.GetProperties(ctx, category.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			category.Properties = properties
		}

		if slices.Contains(params.Includes, "unit_group") {
			unitGroup, apiErr := repo.GetUnitGroup(ctx, category.UnitGroupID, params.Includes)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			category.UnitGroup = unitGroup
		}
	}

	return result, nil
}

func (s *itemCategorySvcImpl) GetItemCategory(ctx context.Context, params domain.GetItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewItemCategoryRepo()

	category, apiErr := repo.Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "properties") {
		properties, apiErr := repo.GetProperties(ctx, params.ItemCategoryID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		category.Properties = properties
	}

	if slices.Contains(params.Includes, "unit_group") {
		unitGroup, apiErr := repo.GetUnitGroup(ctx, category.UnitGroupID, params.Includes)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		category.UnitGroup = unitGroup
	}

	return category, nil
}

func (s *itemCategorySvcImpl) CreateItemCategory(ctx context.Context, params domain.CreateItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	categoryID, apiErr := id.GenID(id.ItemCategoryIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ItemCategoryFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ItemCategoryFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemCategorySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemCategoryRepo()

			created, apiErr := txRepo.Create(txCtx, categoryID, params)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "unit_group") {
				unitGroup, apiErr := txRepo.GetUnitGroup(txCtx, created.UnitGroupID, params.Includes)
				if apiErr != nil {
					return apiErr
				}
				created.UnitGroup = unitGroup
			}

			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeItemCategory,
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

func (s *itemCategorySvcImpl) UpdateItemCategory(ctx context.Context, params domain.UpdateItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultCategory(params.ItemCategoryID) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Default categories cannot be updated."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ItemCategoryFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ItemCategoryFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemCategorySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemCategoryRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetItemCategoryParams{
				AccountID:      params.AccountID,
				ItemCategoryID: params.ItemCategoryID,
			})
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "unit_group") {
				unitGroup, apiErr := txRepo.GetUnitGroup(txCtx, updated.UnitGroupID, params.Includes)
				if apiErr != nil {
					return apiErr
				}
				updated.UnitGroup = unitGroup
			}

			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItemCategory,
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

func (s *itemCategorySvcImpl) DeleteItemCategory(ctx context.Context, itemCategoryID string) *apierror.APIError {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultCategory(itemCategoryID) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Default categories cannot be deleted."))
	}

	accountID := identity.Target.AccountID

	category, apiErr := s.repos.NewItemCategoryRepo().Get(ctx, domain.GetItemCategoryParams{
		AccountID:      accountID,
		ItemCategoryID: itemCategoryID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeItemCategory, itemCategoryID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This item category has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemCategorySvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeItemCategory, category.ID, category); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewItemCategoryRepo().Delete(txCtx, domain.DeleteItemCategoryParams{
			AccountID:      accountID,
			ItemCategoryID: itemCategoryID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(category, (*domain.ItemCategoryFull)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeItemCategory,
			ResourceID:   category.ID,
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

func (s *itemCategorySvcImpl) AddItemCategoryProperty(ctx context.Context, params domain.AddItemCategoryPropertyParams) *apierror.APIError {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.add_property")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultCategory(params.ItemCategoryID) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Default categories cannot be updated."))
	}

	params.AccountID = identity.Target.AccountID
	repo := s.repos.NewItemCategoryRepo()

	isInAccount, apiErr := repo.IsInAccount(ctx, params.AccountID, params.ItemCategoryID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	isPropertyInAccount, apiErr := repo.IsPropertyInAccount(ctx, params.AccountID, params.PropertyID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isPropertyInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Property not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		_, err := idempotency.UnmarshalCachedResponse[any](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return nil

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemCategorySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemCategoryRepo()

			oldFull, apiErr := loadItemCategoryFullTx(txCtx, txRepo, params.AccountID, params.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}

			property, apiErr := txSvc.repos.NewPropertyRepo().Get(txCtx, domain.GetPropertyParams{
				PropertyID: params.PropertyID,
				AccountID:  params.AccountID,
			})
			if apiErr != nil {
				return apiErr
			}

			exists, apiErr := txRepo.PropertyExistsByNameInCategory(txCtx, params.AccountID, params.ItemCategoryID, property.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A property with this name already exists in this category.", "property_id")
			}

			if apiErr := txRepo.AddProperty(txCtx, params); apiErr != nil {
				return apiErr
			}

			newFull, apiErr := loadItemCategoryFullTx(txCtx, txRepo, params.AccountID, params.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(oldFull, newFull)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItemCategory,
				ResourceID:   params.ItemCategoryID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, nil)
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *itemCategorySvcImpl) RemoveItemCategoryProperty(ctx context.Context, params domain.RemoveItemCategoryPropertyParams) *apierror.APIError {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.remove_property")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultCategory(params.ItemCategoryID) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Default categories cannot be updated."))
	}

	params.AccountID = identity.Target.AccountID
	repo := s.repos.NewItemCategoryRepo()

	isInAccount, apiErr := repo.IsInAccount(ctx, params.AccountID, params.ItemCategoryID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	isPropertyInAccount, apiErr := repo.IsPropertyInAccount(ctx, params.AccountID, params.PropertyID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isPropertyInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Property not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		_, err := idempotency.UnmarshalCachedResponse[any](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return nil

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemCategorySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemCategoryRepo()

			oldFull, apiErr := loadItemCategoryFullTx(txCtx, txRepo, params.AccountID, params.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.RemoveProperty(txCtx, params); apiErr != nil {
				return apiErr
			}

			newFull, apiErr := loadItemCategoryFullTx(txCtx, txRepo, params.AccountID, params.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(oldFull, newFull)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItemCategory,
				ResourceID:   params.ItemCategoryID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, nil)
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *itemCategorySvcImpl) ChangeItemCategoryUnitGroup(ctx context.Context, params domain.ChangeItemCategoryUnitGroupParams) *apierror.APIError {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.change_unit_group")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultCategory(params.ItemCategoryID) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Default categories cannot be updated."))
	}

	params.AccountID = identity.Target.AccountID
	repo := s.repos.NewItemCategoryRepo()

	isInAccount, apiErr := repo.IsInAccount(ctx, params.AccountID, params.ItemCategoryID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item category not found."))
	}

	// Validate the new unit group has the same unit type as the current one.
	itemCategory, apiErr := repo.Get(ctx, domain.GetItemCategoryParams{
		AccountID:      params.AccountID,
		ItemCategoryID: params.ItemCategoryID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	unitGroupRepo := s.repos.NewUnitGroupRepo()

	currentUnitGroup, apiErr := unitGroupRepo.Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: itemCategory.UnitGroupID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	newUnitGroup, apiErr := unitGroupRepo.Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if currentUnitGroup.Type != newUnitGroup.Type {
		return tracing.Trace(span, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("The new unit group must have the same unit type as the current unit group (%s).", currentUnitGroup.Type),
			"unit_group_id",
		))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		_, err := idempotency.UnmarshalCachedResponse[any](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return nil

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemCategorySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemCategoryRepo()

			oldFull, apiErr := loadItemCategoryFullTx(txCtx, txRepo, params.AccountID, params.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.ChangeUnitGroup(txCtx, params); apiErr != nil {
				return apiErr
			}

			newFull, apiErr := loadItemCategoryFullTx(txCtx, txRepo, params.AccountID, params.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(oldFull, newFull)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItemCategory,
				ResourceID:   params.ItemCategoryID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, nil)
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}
