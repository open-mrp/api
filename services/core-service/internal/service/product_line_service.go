package service

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var productLineSvcTracer = tracing.GetTracer("core-service.product_line_service")

type productLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type ProductLineSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service the async bulk upsert records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ProductLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("product line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("product line service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("product line service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("product line service: tx manager is required")
	}
	return nil
}

func NewProductLineSvc(config *ProductLineSvcConfig) domain.ProductLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *productLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productLineSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ListProductLines returns a paginated list of product lines visible to the caller's account.
func (s *productLineSvcImpl) ListProductLines(ctx context.Context, params domain.ListProductLinesParams) (*domain.ListProductLinesResult, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := checkProductLineReadPermission(identity); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductLineRepo()

	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "unit_group") {
		for _, pl := range result.ProductLines {
			unitGroup, apiErr := repo.GetUnitGroup(ctx, params.AccountID, pl.UnitGroupID, params.Includes)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			pl.UnitGroup = unitGroup
		}
	}

	return result, nil
}

// GetProductLine returns a single product line by ID.
func (s *productLineSvcImpl) GetProductLine(ctx context.Context, params domain.GetProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := checkProductLineReadPermission(identity); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductLineRepo()

	productLine, apiErr := repo.Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "unit_group") {
		unitGroup, apiErr := repo.GetUnitGroup(ctx, params.AccountID, productLine.UnitGroupID, params.Includes)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		productLine.UnitGroup = unitGroup
	}

	return productLine, nil
}

// validateLotDefault rejects a lot convention the line cannot express.
//
// Both halves move together — a size without a unit cannot say whether 60 means pairs or eaches — and the unit has to belong to the line's unit group, or the lot is counted in something no product in the line is measured in.
func validateLotDefault(
	ctx context.Context,
	repo domain.ProductLineRepo,
	unitGroupID string,
	lot *domain.LotQuantityInput,
) *apierror.APIError {
	if lot == nil {
		return nil
	}
	if lot.UnitID == "" {
		return apierror.NewValidationErrorWithParam(
			"A default lot needs a unit as well as a value.", "default_lot")
	}

	value, err := strconv.ParseFloat(lot.Value, 64)
	if err != nil {
		return apierror.NewValidationErrorWithParam(
			"The default lot value must be a number.", "default_lot")
	}
	if value <= 0 {
		return apierror.NewValidationErrorWithParam(
			"The default lot value must be greater than zero.", "default_lot")
	}

	inGroup, apiErr := repo.IsUnitInGroup(ctx, unitGroupID, lot.UnitID)
	if apiErr != nil {
		return apiErr
	}
	if !inGroup {
		return apierror.NewValidationErrorWithParam(
			"The default lot unit must belong to this product line's unit group.", "default_lot")
	}
	return nil
}

// CreateProductLine creates a new product line.
//
// Business logic:
// 1. Rejects a name already used by another line visible to the account.
// 2. Validates the referenced unit group exists (scoped to the account).
// 3. Validates the default lot, when supplied, carries both a value and a unit belonging to the line's unit group.
// 4. Creates the line (and its lot quantity row) idempotently inside a transaction.
// 5. Publishes an audit event for the creation in the same transaction.
func (s *productLineSvcImpl) CreateProductLine(ctx context.Context, params domain.CreateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productLineID, apiErr := id.GenID(id.ProductLineIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductLineFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductLineFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductLineRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A product line with this name already exists.", "name")
			}

			// Validate the referenced unit group exists before persisting so a bogus unit_group_id is rejected and the tx rolls back instead of committing a dangling reference.
			if _, apiErr := txRepo.GetUnitGroup(txCtx, params.AccountID, params.UnitGroupID, nil); apiErr != nil {
				return apiErr
			}

			if apiErr := validateLotDefault(txCtx, txRepo, params.UnitGroupID, params.DefaultLot); apiErr != nil {
				return apiErr
			}

			created, apiErr := txRepo.Create(txCtx, productLineID, params)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "unit_group") {
				unitGroup, apiErr := txRepo.GetUnitGroup(txCtx, params.AccountID, created.UnitGroupID, params.Includes)
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
				ResourceType: constants.ObjectTypeProductLine,
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

// UpdateProductLine partially updates a product line. Default product lines cannot be updated.
//
// Business logic:
// 1. Rejects edits to system-default product lines.
// 2. Rejects a name already used by another line visible to the account.
// 3. Validates the referenced unit group, when changed, exists (scoped to the account).
// 4. Validates the default lot against the unit group the line will have after the edit.
// 5. Applies the update (including lot set/edit/clear) idempotently inside a transaction.
// 6. Publishes an audit event with the computed field changes in the same transaction.
func (s *productLineSvcImpl) UpdateProductLine(ctx context.Context, params domain.UpdateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultProductLine(params.ProductLineID) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Default product lines cannot be updated."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductLineFull](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductLineFull
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductLineRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetProductLineParams{
				AccountID:     params.AccountID,
				ProductLineID: params.ProductLineID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.ProductLineID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A product line with this name already exists.", "name")
				}
			}

			// Validate the referenced unit group exists before persisting so a bogus unit_group_id is rejected and the tx rolls back instead of committing a dangling reference.
			if params.UnitGroupID != nil && *params.UnitGroupID != "" {
				if _, apiErr := txRepo.GetUnitGroup(txCtx, params.AccountID, *params.UnitGroupID, nil); apiErr != nil {
					return apiErr
				}
			}

			// Checked against the group the line will have after this edit, not the one it had before: moving a line to a new unit group and setting its lot in the same request must be judged as a whole.
			if !params.ClearDefaultLot {
				unitGroupID := old.UnitGroupID
				if params.UnitGroupID != nil && *params.UnitGroupID != "" {
					unitGroupID = *params.UnitGroupID
				}
				if apiErr := validateLotDefault(txCtx, txRepo, unitGroupID, params.DefaultLot); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "unit_group") {
				unitGroup, apiErr := txRepo.GetUnitGroup(txCtx, params.AccountID, updated.UnitGroupID, params.Includes)
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
				ResourceType: constants.ObjectTypeProductLine,
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

// DeleteProductLine deletes a product line. Default product lines cannot be deleted.
//
// Business logic:
// 1. Rejects deletion of system-default product lines.
// 2. Returns an already-deleted error when a tombstone exists for the line.
// 3. Records a deleted-record tombstone and deletes the line in one transaction.
// 4. Publishes an audit event for the deletion in the same transaction.
func (s *productLineSvcImpl) DeleteProductLine(ctx context.Context, productLineID string) *apierror.APIError {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if domain.IsDefaultProductLine(productLineID) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Default product lines cannot be deleted."))
	}

	accountID := identity.Target.AccountID

	productLine, apiErr := s.repos.NewProductLineRepo().Get(ctx, domain.GetProductLineParams{
		AccountID:     accountID,
		ProductLineID: productLineID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProductLine, productLineID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This product line has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productLineSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProductLine, productLine.ID, productLine); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewProductLineRepo().Delete(txCtx, domain.DeleteProductLineParams{
			AccountID:     accountID,
			ProductLineID: productLineID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(productLine, (*domain.ProductLineFull)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductLine,
			ResourceID:   productLine.ID,
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

// BatchGetProductLinesByIDs returns product lines by ID for the api-gateway include resolver.
func (s *productLineSvcImpl) BatchGetProductLinesByIDs(ctx context.Context, ids []string) ([]*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineSvcTracer.Start(ctx, "service.product_line.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if identity.IsInternalActor() {
		if apiErr := checkProductLineReadPermission(identity); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}
	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewProductLineRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

// checkProductLineReadPermission checks the appropriate read permission based on the identity context. Internal actors need product_lines:read for their own account, or customers:read / suppliers:read for external accounts.
func checkProductLineReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainProductLines, types.ActionRead)
}
