package service

import (
	"context"
	"fmt"

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

var supplierMaterialSvcTracer = tracing.GetTracer("core-service.supplier_material_service")

// supplierDescriptionMaxBytes is what supplier_material.supplier_description, a TEXT column, holds.
const supplierDescriptionMaxBytes = 65535

type supplierMaterialSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SupplierMaterialSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *SupplierMaterialSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("supplier material service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("supplier material service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("supplier material service: tx manager is required")
	}
	return nil
}

func NewSupplierMaterialSvc(config *SupplierMaterialSvcConfig) domain.SupplierMaterialSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &supplierMaterialSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *supplierMaterialSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *supplierMaterialSvcImpl) withTx(ctx context.Context, fn func(context.Context, *supplierMaterialSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &supplierMaterialSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ListSupplierMaterials returns a paginated list of supplier materials for the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and suppliers:read permission.
// 2. Require the OpenMRP-Account header to scope the query.
// 3. Query the supplier material repository with the account ID and pagination params.
func (s *supplierMaterialSvcImpl) ListSupplierMaterials(ctx context.Context, params domain.ListSupplierMaterialsParams) (*domain.ListSupplierMaterialsResult, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_material.list")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	return s.repos.NewSupplierMaterialRepo().List(ctx, params)
}

// GetSupplierMaterial retrieves a single supplier material by supplier account ID and material ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and suppliers:read permission.
// 2. Require the OpenMRP-Account header.
// 3. Fetch the supplier material from the repository by owner account ID, supplier account ID, and material ID.
func (s *supplierMaterialSvcImpl) GetSupplierMaterial(ctx context.Context, supplierAccountID, materialID string) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_material.get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewSupplierMaterialRepo().GetBySupplierAndMaterialID(ctx, identity.Target.AccountID, supplierAccountID, materialID)
}

// CreateSupplierMaterial creates a new supplier material association, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and suppliers:create permission.
// 2. Generate a unique supplier material ID.
// 3. Upsert an idempotency key; if already finished, return the cached response.
// 4. Within a transaction, require the supplier and the material to be the caller's, then refuse a duplicate link.
// 5. Insert the supplier material record and cache the success response.
// 6. On error, cache the error response for idempotent replay.
func (s *supplierMaterialSvcImpl) CreateSupplierMaterial(ctx context.Context, params domain.CreateSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_material.create")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSupplierDescription(params.SupplierDescription); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	supplierMaterialID, apiErr := id.GenID(id.SupplierMaterialIDPrefix, nil)
	if apiErr != nil {
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.SupplierMaterial](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SupplierMaterial
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierMaterialSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSupplierMaterialRepo()

			// Both ends of the link must be the owner's: the supplier one of its suppliers, the material one of its items.
			if _, apiErr := txSvc.repos.NewSupplierRepo().Get(txCtx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierAccountID}); apiErr != nil {
				if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
					return apierror.NewNotFoundError("No supplier found with the provided ID.").WithParam("supplier_id")
				}
				return apiErr
			}
			material, apiErr := txSvc.repos.NewMaterialRepo().GetByID(txCtx, domain.GetMaterialParams{AccountID: params.OwnerAccountID, MaterialID: params.MaterialID})
			if apiErr != nil {
				if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
					return apierror.NewNotFoundError("No material found with the provided ID.").WithParam("material_id")
				}
				return apiErr
			}
			if apiErr := requireItemsAssignable(txCtx, txSvc.repos, params.OwnerAccountID, []string{material.ItemID}, "material_id"); apiErr != nil {
				return apiErr
			}

			exists, apiErr := txRepo.ExistsByMaterialAndSupplier(txCtx, params.OwnerAccountID, params.MaterialID, params.SupplierAccountID)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("material_id", "A supplier material with this material and supplier already exists.")
			}

			created, apiErr := txRepo.Create(txCtx, supplierMaterialID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), supplierMaterialAuditEvent(constants.AuditActionCreate, created, nil, created)); apiErr != nil {
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

// UpdateSupplierMaterial partially updates a supplier material, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and suppliers:update permission.
// 2. Upsert an idempotency key; if already finished, return the cached response.
// 3. Within a transaction, apply the updates and cache the success response.
// 4. On error, cache the error response for idempotent replay.
func (s *supplierMaterialSvcImpl) UpdateSupplierMaterial(ctx context.Context, params domain.UpdateSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_material.update")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSupplierDescription(params.SupplierDescription); apiErr != nil {
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
		cached, err := idemresponse.UnmarshalCachedResponse[domain.SupplierMaterial](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SupplierMaterial
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierMaterialSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSupplierMaterialRepo()

			old, apiErr := txRepo.GetBySupplierAndMaterialID(txCtx, params.OwnerAccountID, params.SupplierAccountID, params.MaterialID)
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), supplierMaterialAuditEvent(constants.AuditActionUpdate, updated, old, updated)); apiErr != nil {
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

// DeleteSupplierMaterial deletes a supplier material association by supplier and material ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and suppliers:update permission.
// 2. Require the OpenMRP-Account header.
// 3. Delete the supplier material from the repository and return the deleted record.
func (s *supplierMaterialSvcImpl) DeleteSupplierMaterial(ctx context.Context, params domain.DeleteSupplierMaterialParams) (*domain.SupplierMaterial, *apierror.APIError) {
	ctx, span := supplierMaterialSvcTracer.Start(ctx, "service.supplier_material.delete")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	entity, apiErr := s.repos.NewSupplierMaterialRepo().GetBySupplierAndMaterialID(ctx, params.OwnerAccountID, params.SupplierAccountID, params.MaterialID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.SupplierMaterial
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierMaterialSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeSupplierMaterial, entity.ID, params.OwnerAccountID, entity); apiErr != nil {
			return apiErr
		}

		deleted, apiErr := txSvc.repos.NewSupplierMaterialRepo().Delete(txCtx, params)
		if apiErr != nil {
			return apiErr
		}
		result = deleted

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), supplierMaterialAuditEvent(constants.AuditActionDelete, entity, entity, nil)); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// checkSupplierDescription refuses a description the column cannot hold. The gateway counts characters, and a
// character can take up to four bytes.
func checkSupplierDescription(description *string) *apierror.APIError {
	if description != nil && len(*description) > supplierDescriptionMaxBytes {
		return apierror.NewFieldError("supplier_description", apierror.CodeInvalidFormat, "The supplier description must be at most 65,535 bytes.")
	}
	return nil
}

// supplierMaterialAuditEvent records a supplier material write as the supplier item it is, so a row's history is one
// supplier_item resource whichever route wrote it.
func supplierMaterialAuditEvent(action constants.AuditAction, sm, before, after *domain.SupplierMaterial) audit.EventData {
	return supplierItemAuditEvent(action, supplierMaterialAsItem(sm), audit.ComputeChanges(supplierMaterialAsItem(before), supplierMaterialAsItem(after)))
}

// supplierMaterialAsItem is a supplier material in its supplier item fields; nil stays nil.
func supplierMaterialAsItem(sm *domain.SupplierMaterial) *domain.SupplierItem {
	if sm == nil {
		return nil
	}
	item := &domain.SupplierItem{
		ID:                  sm.ID,
		SupplierAccountID:   sm.SupplierAccountID,
		MaterialID:          sm.MaterialID,
		SupplierSKU:         sm.SupplierPartNumber,
		SupplierDescription: sm.SupplierDescription,
		Status:              constants.ArchiveStatusFromIsActive(sm.IsActive),
		CreatedAt:           sm.CreatedAt,
		UpdatedAt:           sm.UpdatedAt,
	}
	if sm.Material != nil {
		item.ItemID = sm.Material.ItemID
	}
	return item
}
