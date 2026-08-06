package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type ItemCategorySvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service the async bulk upsert records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ItemCategorySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("item category service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("item category service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("item category service: job service factory is required")
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
		jobSvcFactory:   config.JobSvcFactory,
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

// linkRowPropertiesToCategoryInTx links each of a bulk-upsert row's properties to the
// item's category (idempotent), so a newly-imported attribute renders in the
// category-driven detail UI. Without this the attribute is attached to the item but its
// property is not on the item's category, so the row never shows. propIDByName is the
// map returned by resolvePropertyAttributesInTx. Shared by the per-type item bulk upserts.
func linkRowPropertiesToCategoryInTx(txCtx context.Context, repos domain.RepoFactory, categoryID string, properties []domain.UpsertItemPropertyParams, propIDByName map[string]string) *apierror.APIError {
	if categoryID == "" || len(properties) == 0 {
		return nil
	}
	catRepo := repos.NewItemCategoryRepo()
	seen := make(map[string]struct{}, len(properties))
	for _, p := range properties {
		if p.Name == "" {
			continue
		}
		propID, ok := propIDByName[strings.ToLower(p.Name)]
		if !ok || propID == "" {
			continue
		}
		if _, dup := seen[propID]; dup {
			continue
		}
		seen[propID] = struct{}{}
		if apiErr := catRepo.UpsertProperty(txCtx, categoryID, propID); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// bulkCategoryRow pairs a bulk-upsert row index with the category it will be created
// under, for up-front category validation.
type bulkCategoryRow struct {
	Index      int
	CategoryID string
}

// validateBulkCreateCategoriesInTx verifies that every category referenced by a create
// row has a base unit and has the category type the item type requires (materials →
// material_category; parts and products → product_category), collecting ALL offences
// into a single validation error instead of failing the batch with an opaque "Resource
// not found." The identifiers carry category IDs already resolved by the caller. fieldName is
// the request array name (e.g. "materials") and refField the input field (e.g.
// "category"), together building params like "materials[2].category"; itemTypeCode is
// the item type being upserted. Only create rows should be passed — category is
// create-only on bulk upsert. Shared by the per-type item bulk upserts.
func validateBulkCreateCategoriesInTx(txCtx context.Context, repos domain.RepoFactory, fieldName, refField, itemTypeCode string, identifiers []bulkCategoryRow) *apierror.APIError {
	if len(identifiers) == 0 {
		return nil
	}

	ids := make([]string, 0, len(identifiers))
	seen := make(map[string]struct{}, len(identifiers))
	for _, r := range identifiers {
		if r.CategoryID == "" {
			continue
		}
		if _, ok := seen[r.CategoryID]; ok {
			continue
		}
		seen[r.CategoryID] = struct{}{}
		ids = append(ids, r.CategoryID)
	}

	categories, apiErr := repos.NewItemRepo().GetCategoryBaseUnitIDs(txCtx, ids)
	if apiErr != nil {
		return apiErr
	}

	var rowErrs apierror.RowErrors
	for _, r := range identifiers {
		param := fmt.Sprintf("%s[%d].%s", fieldName, r.Index, refField)
		switch category, exists := categories[r.CategoryID]; {
		case r.CategoryID == "" || !exists:
			rowErrs.AddValidation(r.Index, param, fmt.Sprintf("category %q was not found", r.CategoryID))
		case !categoryTypeMatchesItem(itemTypeCode, category.ItemCategoryTypeCode):
			rowErrs.AddValidation(r.Index, param, fmt.Sprintf("category %q is a %s and cannot be assigned to a %s", r.CategoryID, category.ItemCategoryTypeCode, itemTypeCode))
		case category.BaseUnitID == "":
			rowErrs.AddValidation(r.Index, param, fmt.Sprintf("category %q has no base unit; assign a base unit to its unit group", r.CategoryID))
		}
	}

	return rowErrs.Summary("categories")
}

func (s *itemCategorySvcImpl) BatchGetItemCategoriesByIDs(ctx context.Context, ids []string) ([]*domain.ItemCategoryFull, *apierror.APIError) {
	ctx, span := itemCategorySvcTracer.Start(ctx, "service.item_category.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	meds := s.mediators()
	if apiErr := authorizeCatalogBatchRead(ctx, identity, span, meds, func() *apierror.APIError {
		return identity.CheckHasPermission(types.PermissionDomainCategories, types.ActionRead)
	}); apiErr != nil {
		return nil, apiErr
	}
	if len(ids) == 0 {
		return nil, nil
	}

	repo := s.repos.NewItemCategoryRepo()

	categories, apiErr := repo.GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	allIncludes := []string{"unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"}

	for _, cat := range categories {
		properties, apiErr := repo.GetProperties(ctx, cat.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		cat.Properties = properties

		unitGroup, apiErr := repo.GetUnitGroup(ctx, cat.UnitGroupID, allIncludes)
		if apiErr != nil {
			// An orphaned unit_group reference (the unit group was deleted after the category was created) must not fail the whole batch and 404 the entire list. Surface the category with a nil unit_group instead.
			if apierror.IsNotFound(apiErr) {
				continue
			}
			return nil, tracing.Trace(span, apiErr)
		}
		cat.UnitGroup = unitGroup
	}

	return categories, nil
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

	// Validate the referenced unit group exists (and belongs to this account)
	// before creating. Vitess does not enforce this FK at the DB level, so an
	// unvalidated unit_group_id would otherwise be persisted silently and the
	// category would render with unit_group=null instead of rejecting the request.
	if params.UnitGroupID != "" {
		if _, apiErr := s.repos.NewUnitGroupRepo().Get(ctx, domain.GetUnitGroupParams{
			AccountID:   params.AccountID,
			UnitGroupID: params.UnitGroupID,
		}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

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
