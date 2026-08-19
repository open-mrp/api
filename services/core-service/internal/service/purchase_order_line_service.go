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

var purchaseOrderLineSvcTracer = tracing.GetTracer("core-service.purchase_order_line_service")

type purchaseOrderLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type PurchaseOrderLineSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PurchaseOrderLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("purchase order line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("purchase order line service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("purchase order line service: tx manager is required")
	}
	return nil
}

func NewPurchaseOrderLineSvc(config *PurchaseOrderLineSvcConfig) domain.PurchaseOrderLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &purchaseOrderLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *purchaseOrderLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *purchaseOrderLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *purchaseOrderLineSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &purchaseOrderLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *purchaseOrderLineSvcImpl) CreatePurchaseOrderLine(ctx context.Context, params domain.CreatePurchaseOrderLineParams) (*domain.PurchaseOrderLine, *apierror.APIError) {
	ctx, span := purchaseOrderLineSvcTracer.Start(ctx, "service.purchase_order_line.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPurchaseOrderLineWritePermission(identity); apiErr != nil {
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

	// The same check the create path runs, so a line added afterwards cannot record a quantity in a unit the product is not measured in.
	if apiErr := validatePurchaseOrderLineUnits(ctx, s.repos, params.AccountID, []domain.CreatePurchaseOrderLineInput{{
		ProductID:      params.ProductID,
		QuantityUnitID: params.QuantityUnitID,
	}}, "quantity_unit_id"); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PurchaseOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.PurchaseOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderLineSvcImpl) *apierror.APIError {
			txOrderRepo := txSvc.repos.NewPurchaseOrderRepo()
			txLineRepo := txSvc.repos.NewPurchaseOrderLineRepo()

			// Validate order exists
			_, apiErr := txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Create line (repo handles quantity, rate, and line item number generation)
			created, apiErr := txLineRepo.Create(txCtx, lineID, params)
			if apiErr != nil {
				return apiErr
			}

			// If a receiving order exists for this PO, create a receiving order line for the new line
			txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()
			receivingOrderID, apiErr := txReceivingRepo.GetByOrderID(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			if receivingOrderID != nil {
				receivingLineID, apiErr := id.GenID(id.ReceivingOrderLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				qtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				if apiErr := txLineRepo.CreateQuantity(txCtx, qtyID, params.QuantityValue, params.QuantityUnitID); apiErr != nil {
					return apiErr
				}

				if apiErr := txReceivingRepo.CreateLine(txCtx, receivingLineID, *receivingOrderID, qtyID, lineID); apiErr != nil {
					return apiErr
				}
			}

			// Ensure supplier material link if item_id is set
			if params.ItemID != nil {
				if apiErr := ensureSupplierMaterialLink(txCtx, txSvc.repos, params.AccountID, params.SalesOrderID, *params.ItemID, params.ProductSKU); apiErr != nil {
					return apiErr
				}
			}

			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypePurchaseOrderLine,
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

func (s *purchaseOrderLineSvcImpl) UpdatePurchaseOrderLine(ctx context.Context, params domain.UpdatePurchaseOrderLineParams) (*domain.PurchaseOrderLine, *apierror.APIError) {
	ctx, span := purchaseOrderLineSvcTracer.Start(ctx, "service.purchase_order_line.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPurchaseOrderLineWritePermission(identity); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.PurchaseOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PurchaseOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderLineSvcImpl) *apierror.APIError {
			txOrderRepo := txSvc.repos.NewPurchaseOrderRepo()
			txLineRepo := txSvc.repos.NewPurchaseOrderLineRepo()

			// Validate order exists in account
			_, apiErr := txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Validate line belongs to order
			isInOrder, apiErr := txLineRepo.IsInOrder(txCtx, params.PurchaseOrderLineID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			if !isInOrder {
				return apierror.NewResourceNotFoundError("Purchase order line not found in this order.")
			}

			old, apiErr := txLineRepo.Get(txCtx, params.PurchaseOrderLineID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values. Since the SQL uses direct assignment (no COALESCE) for these fields, we must provide the existing value when the field was not sent.
			if params.ProductID == nil {
				params.ProductID = old.ProductID
			}
			if params.ItemID == nil {
				params.ItemID = old.ItemID
			}

			// Update line (repo handles quantity and rate updates internally)
			updated, apiErr := txLineRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			// If a receiving order exists for this PO, create a receiving order line for remaining quantity
			txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()
			receivingOrderID, apiErr := txReceivingRepo.GetByOrderID(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			if receivingOrderID != nil {
				if apiErr := txReceivingRepo.CreateLineForRemainingQuantity(txCtx, *receivingOrderID, params.PurchaseOrderLineID, params.AccountID); apiErr != nil {
					return apiErr
				}
			}

			// Ensure supplier material link if item changed
			if params.ItemID != nil {
				sku := ""
				if params.ProductSKU != nil {
					sku = *params.ProductSKU
				}
				if apiErr := ensureSupplierMaterialLink(txCtx, txSvc.repos, params.AccountID, params.SalesOrderID, *params.ItemID, sku); apiErr != nil {
					return apiErr
				}
			}

			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePurchaseOrderLine,
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

func (s *purchaseOrderLineSvcImpl) DeletePurchaseOrderLine(ctx context.Context, params domain.DeletePurchaseOrderLineParams) *apierror.APIError {
	ctx, span := purchaseOrderLineSvcTracer.Start(ctx, "service.purchase_order_line.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := checkPurchaseOrderLineWritePermission(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	// Validate purchase order exists and belongs to account
	orderRepo := s.repos.NewPurchaseOrderRepo()
	_, apiErr := orderRepo.Get(ctx, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	lineRepo := s.repos.NewPurchaseOrderLineRepo()

	// Validate line belongs to order
	isInOrder, apiErr := lineRepo.IsInOrder(ctx, params.PurchaseOrderLineID, params.SalesOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInOrder {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Purchase order line not found in this order."))
	}

	line, apiErr := lineRepo.Get(ctx, params.PurchaseOrderLineID, params.SalesOrderID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypePurchaseOrderLine, params.PurchaseOrderLineID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This purchase order line has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderLineSvcImpl) *apierror.APIError {
		txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()
		txLineRepo := txSvc.repos.NewPurchaseOrderLineRepo()

		// Delete receiving order lines associated with this PO line first
		if apiErr := txReceivingRepo.DeleteLinesByOrderLineID(txCtx, params.PurchaseOrderLineID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypePurchaseOrderLine, line.ID, line); apiErr != nil {
			return apiErr
		}

		if apiErr := txLineRepo.DeleteCascade(txCtx, params.PurchaseOrderLineID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(line, (*domain.PurchaseOrderLine)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypePurchaseOrderLine,
			ResourceID:   line.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

// checkPurchaseOrderLineWritePermission checks the appropriate write permission based on the target context. Internal actors need purchase_orders:update for their own account, or suppliers:update when targeting a supplier account.
func checkPurchaseOrderLineWritePermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionUpdate)
}
