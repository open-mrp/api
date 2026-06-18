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

var supplierSvcTracer = tracing.GetTracer("core-service.supplier_service")

type supplierSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SupplierSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *SupplierSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("supplier service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("supplier service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("supplier service: tx manager is required")
	}
	return nil
}

func NewSupplierSvc(config *SupplierSvcConfig) domain.SupplierSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &supplierSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *supplierSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *supplierSvcImpl) withTx(ctx context.Context, fn func(context.Context, *supplierSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &supplierSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *supplierSvcImpl) ListSuppliers(ctx context.Context, params domain.ListSuppliersParams) (*domain.ListSuppliersResult, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
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

	return s.repos.NewSupplierRepo().List(ctx, params)
}

func (s *supplierSvcImpl) GetSupplier(ctx context.Context, params domain.GetSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Internal actors need suppliers:read permission.
	// Supplier actors can view their own record.
	if identity.IsSupplierUser() {
		if identity.Actor.AccountID == nil || *identity.Actor.AccountID != params.SupplierID {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this supplier."))
		}
	} else {
		if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.OwnerAccountID = identity.Target.AccountID
	return s.repos.NewSupplierRepo().Get(ctx, params)
}

func (s *supplierSvcImpl) CreateSupplier(ctx context.Context, params domain.CreateSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID, apiErr := id.GenID(id.AccountIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	relationID, apiErr := id.GenID(id.AccountRelationIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Supplier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Supplier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
			txSupplierRepo := txSvc.repos.NewSupplierRepo()

			// Check for duplicate supplier number.
			exists, apiErr := txSupplierRepo.ExistsByNumber(txCtx, params.OwnerAccountID, params.Number, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A supplier with this number already exists.", "number")
			}

			// Create bill-to address if provided.
			var billToAddressID *string
			if params.BillToAddress != nil {
				addrID, geoID, acctAddrID, err := generateAddressIDs()
				if err != nil {
					return err
				}
				params.BillToAddress.AccountID = accountID
				_, apiErr := txSvc.repos.NewAddressRepo().Create(txCtx, addrID, geoID, acctAddrID, *params.BillToAddress)
				if apiErr != nil {
					return apiErr
				}
				billToAddressID = &addrID
			}

			// Create ship-to address if provided.
			var shipToAddressID *string
			if params.ShipToAddress != nil {
				addrID, geoID, acctAddrID, err := generateAddressIDs()
				if err != nil {
					return err
				}
				params.ShipToAddress.AccountID = accountID
				_, apiErr := txSvc.repos.NewAddressRepo().Create(txCtx, addrID, geoID, acctAddrID, *params.ShipToAddress)
				if apiErr != nil {
					return apiErr
				}
				shipToAddressID = &addrID
			}

			// Dashboard defaults ship-to to bill-to when only bill-to is provided.
			if shipToAddressID == nil && billToAddressID != nil {
				shipToAddressID = billToAddressID
			}

			created, apiErr := txSupplierRepo.Create(txCtx, accountID, relationID, params, billToAddressID, shipToAddressID)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSupplier,
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

func (s *supplierSvcImpl) UpdateSupplier(ctx context.Context, params domain.UpdateSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
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

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Supplier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Supplier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
			txSupplierRepo := txSvc.repos.NewSupplierRepo()

			old, apiErr := txSupplierRepo.Get(txCtx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierID, Includes: []string{"bill_to_address", "ship_to_address"}})
			if apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values. Since the SQL uses direct assignment (no COALESCE) for these fields, we must provide the existing value when the field was not sent.
			if params.BillToAddressID == nil && old.BillToAddress != nil {
				params.BillToAddressID = &old.BillToAddress.ID
			}
			if params.ShipToAddressID == nil && old.ShipToAddress != nil {
				params.ShipToAddressID = &old.ShipToAddress.ID
			}

			// Check for duplicate supplier number if it's being changed.
			if params.Number != nil {
				exists, apiErr := txSupplierRepo.ExistsByNumber(txCtx, params.OwnerAccountID, *params.Number, &params.SupplierID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A supplier with this number already exists.", "number")
				}
			}

			updated, apiErr := txSupplierRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSupplier,
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

func (s *supplierSvcImpl) DeleteSupplier(ctx context.Context, params domain.DeleteSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Dashboard uses suppliers:update permission for single delete.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	supplier, apiErr := s.repos.NewSupplierRepo().Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierID, Includes: []string{"bill_to_address", "ship_to_address"}})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeSupplier, params.SupplierID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This supplier has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.Supplier
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSupplier, supplier.ID, supplier); apiErr != nil {
			return apiErr
		}

		var txErr *apierror.APIError
		result, txErr = txSvc.repos.NewSupplierRepo().Delete(txCtx, params.OwnerAccountID, params.SupplierID)
		if txErr != nil {
			return txErr
		}

		changes := audit.ComputeChanges(result, (*domain.Supplier)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeSupplier,
			ResourceID:   result.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *supplierSvcImpl) BulkDeleteSuppliers(ctx context.Context, params domain.BulkDeleteSuppliersParams) *apierror.APIError {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.bulk_delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Dashboard uses suppliers:delete permission for bulk delete.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	// Fetch all suppliers before deletion for audit trail.
	supplierRepo := s.repos.NewSupplierRepo()
	suppliers := make([]*domain.Supplier, 0, len(params.SupplierIDs))
	for _, supplierID := range params.SupplierIDs {
		supplier, apiErr := supplierRepo.Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: supplierID, Includes: []string{"bill_to_address", "ship_to_address"}})
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		suppliers = append(suppliers, supplier)
	}

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
		deletedRecordRepo := txSvc.repos.NewDeletedRecordRepo()
		for _, supplier := range suppliers {
			if apiErr := deletedRecordRepo.Create(txCtx, constants.DeletedRecordResourceTypeSupplier, supplier.ID, supplier); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.repos.NewSupplierRepo().BulkDelete(txCtx, params.OwnerAccountID, params.SupplierIDs); apiErr != nil {
			return apiErr
		}

		publisher := audit.NewPublisher()
		outboxRepo := txSvc.repos.NewOutboxRepo()

		for _, supplier := range suppliers {
			changes := audit.ComputeChanges(supplier, (*domain.Supplier)(nil))

			if apiErr := publisher.Publish(txCtx, outboxRepo, audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionDelete,
				ResourceType: constants.ObjectTypeSupplier,
				ResourceID:   supplier.ID,
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

// generateAddressIDs creates IDs for address, geolocation, and account_address records.
func generateAddressIDs() (addressID, geolocationID, accountAddressID string, apiErr *apierror.APIError) {
	addressID, apiErr = id.GenID(id.AddressIDPrefix, nil)
	if apiErr != nil {
		return "", "", "", apiErr
	}
	geolocationID, apiErr = id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return "", "", "", apiErr
	}
	accountAddressID, apiErr = id.GenID(id.AccountAddressIDPrefix, nil)
	if apiErr != nil {
		return "", "", "", apiErr
	}
	return addressID, geolocationID, accountAddressID, nil
}
