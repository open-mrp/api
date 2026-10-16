package service

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

// internalIdentity is the caller, who must belong to the account acting.
func internalIdentity(ctx context.Context) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// requireSupplier reads the owner's supplier the path names; any other is not found.
func requireSupplier(ctx context.Context, repos domain.RepoFactory, ownerAccountID, supplierID string) (*domain.Supplier, *apierror.APIError) {
	supplier, apiErr := repos.NewSupplierRepo().Get(ctx, domain.GetSupplierParams{OwnerAccountID: ownerAccountID, SupplierID: supplierID})
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			return nil, apierror.NewNotFoundError("No supplier found with the provided ID.").WithParam("supplier_id")
		}
		return nil, apiErr
	}
	return supplier, nil
}

func (s *supplierMaterialSvcImpl) ListSupplierItems(ctx context.Context, params domain.ListSupplierItemsParams) (*domain.ListSupplierItemsResult, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_item.list")
	defer span.End()

	identity, apiErr := internalIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.OwnerAccountID = identity.Target.AccountID
	if _, apiErr := requireSupplier(ctx, s.repos, params.OwnerAccountID, params.SupplierAccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewSupplierMaterialRepo().ListItems(ctx, params)
}

func (s *supplierMaterialSvcImpl) GetSupplierItem(ctx context.Context, supplierAccountID, supplierItemID string) (*domain.SupplierItem, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_item.get")
	defer span.End()

	identity, apiErr := internalIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewSupplierMaterialRepo().GetItem(ctx, identity.Target.AccountID, supplierAccountID, supplierItemID)
}

// CreateSupplierItem records that a supplier provides an item. The supplier and the item must be the caller's and
// active, the item must have a material section, and a supplier holds an item at most once.
func (s *supplierMaterialSvcImpl) CreateSupplierItem(ctx context.Context, params domain.CreateSupplierItemParams) (*domain.SupplierItem, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_item.create")
	defer span.End()

	identity, apiErr := internalIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.ArchiveStatus == "" {
		params.ArchiveStatus = constants.ArchiveStatusActive
	}
	if !params.ArchiveStatus.IsValid() {
		return nil, tracing.Trace(span, apierror.NewFieldError("status", apierror.CodeInvalidFormat, "status must be one of active or archived."))
	}
	if apiErr := checkSupplierDescription(params.SupplierDescription); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.OwnerAccountID = identity.Target.AccountID

	supplierItemID, apiErr := id.GenID(id.SupplierItemIDPrefix, nil)
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.SupplierItem](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SupplierItem
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierMaterialSvcImpl) *apierror.APIError {
			supplier, apiErr := requireSupplier(txCtx, txSvc.repos, params.OwnerAccountID, params.SupplierAccountID)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := domain.RequireAssignable("supplier_id", "supplier", supplier.ArchiveStatus); apiErr != nil {
				return apiErr
			}

			states, apiErr := txSvc.repos.NewItemRepo().GetAssignability(txCtx, params.OwnerAccountID, []string{params.ItemID})
			if apiErr != nil {
				return apiErr
			}
			if len(states) == 0 {
				return apierror.NewNotFoundError("No item found with the provided ID.").WithParam("item_id")
			}
			if !states[0].IsAssignable {
				return domain.RequireAssignable("item_id", "item", constants.ArchiveStatusArchived)
			}
			material, apiErr := txSvc.repos.NewMaterialRepo().GetByItemID(txCtx, params.OwnerAccountID, params.ItemID)
			if apiErr != nil {
				if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
					return apierror.NewFieldError("item_id", apierror.CodeInvalidFormat, "Only an item with a material section can be supplied.")
				}
				return apiErr
			}

			repo := txSvc.repos.NewSupplierMaterialRepo()
			exists, apiErr := repo.ExistsByMaterialAndSupplier(txCtx, params.OwnerAccountID, material.ID, params.SupplierAccountID)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("item_id", "This supplier already has a supplier item for this item.")
			}

			if _, apiErr := repo.Create(txCtx, supplierItemID, domain.CreateSupplierMaterialParams{
				OwnerAccountID:      params.OwnerAccountID,
				MaterialID:          material.ID,
				SupplierAccountID:   params.SupplierAccountID,
				SupplierPartNumber:  params.SupplierSKU,
				SupplierDescription: params.SupplierDescription,
				IsActive:            params.ArchiveStatus.IsActive(),
			}); apiErr != nil {
				return apiErr
			}
			created, apiErr := repo.GetItem(txCtx, params.OwnerAccountID, params.SupplierAccountID, supplierItemID)
			if apiErr != nil {
				return apiErr
			}
			result = created

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), supplierItemAuditEvent(constants.AuditActionCreate, created, audit.ComputeChanges(nil, created))); apiErr != nil {
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

func (s *supplierMaterialSvcImpl) UpdateSupplierItem(ctx context.Context, params domain.UpdateSupplierItemParams) (*domain.SupplierItem, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_item.update")
	defer span.End()

	identity, apiErr := internalIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.ArchiveStatus != nil && !params.ArchiveStatus.IsValid() {
		return nil, tracing.Trace(span, apierror.NewFieldError("status", apierror.CodeInvalidFormat, "status must be one of active or archived."))
	}
	if apiErr := checkSupplierDescription(params.SupplierDescription.ValuePtr()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.OwnerAccountID = identity.Target.AccountID

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.SupplierItem](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SupplierItem
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierMaterialSvcImpl) *apierror.APIError {
			repo := txSvc.repos.NewSupplierMaterialRepo()
			old, apiErr := repo.GetItem(txCtx, params.OwnerAccountID, params.SupplierAccountID, params.SupplierItemID)
			if apiErr != nil {
				return apiErr
			}
			next := *old
			if params.SupplierSKU != nil {
				next.SupplierSKU = *params.SupplierSKU
			}
			next.SupplierDescription = clearableOr(params.SupplierDescription, old.SupplierDescription)
			if params.ArchiveStatus != nil {
				next.Status = *params.ArchiveStatus
			}
			if apiErr := repo.UpdateItem(txCtx, params.OwnerAccountID, &next); apiErr != nil {
				return apiErr
			}
			updated, apiErr := repo.GetItem(txCtx, params.OwnerAccountID, params.SupplierAccountID, params.SupplierItemID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), supplierItemAuditEvent(constants.AuditActionUpdate, updated, audit.ComputeChanges(old, updated))); apiErr != nil {
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

// DeleteSupplierItem hard-deletes a supplier item. Nothing references one, so it is never in use; a repeated delete
// is 410.
func (s *supplierMaterialSvcImpl) DeleteSupplierItem(ctx context.Context, supplierAccountID, supplierItemID string) (*domain.SupplierItem, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_item.delete")
	defer span.End()

	identity, apiErr := internalIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ownerAccountID := identity.Target.AccountID

	item, apiErr := s.repos.NewSupplierMaterialRepo().GetItem(ctx, ownerAccountID, supplierAccountID, supplierItemID)
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			wasDeleted, checkErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeSupplierMaterial, supplierItemID, ownerAccountID)
			if checkErr != nil {
				return nil, tracing.Trace(span, checkErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewGoneError("This supplier item has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierMaterialSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeSupplierMaterial, item.ID, ownerAccountID, item); apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewSupplierMaterialRepo().DeleteItem(txCtx, ownerAccountID, item.ID); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), supplierItemAuditEvent(constants.AuditActionDelete, item, audit.ComputeChanges(item, (*domain.SupplierItem)(nil))))
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return item, nil
}

func supplierItemAuditEvent(action constants.AuditAction, item *domain.SupplierItem, changes []audit.FieldChange) audit.EventData {
	return audit.EventData{
		ServiceName:      domain.ServiceName,
		Action:           action,
		ResourceType:     constants.ObjectTypeSupplierItem,
		ResourceID:       item.ID,
		RootResourceType: constants.ObjectTypeSupplier,
		RootResourceID:   item.SupplierAccountID,
		Changes:          changes,
	}
}
