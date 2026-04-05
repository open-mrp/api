package service

import (
	"context"
	"fmt"
	"math"
	"strconv"

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

var salesOrderLineSvcTracer = tracing.GetTracer("core-service.sales_order_line_service")

type salesOrderLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SalesOrderLineSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *SalesOrderLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sales order line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("sales order line service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("sales order line service: tx manager is required")
	}
	return nil
}

func NewSalesOrderLineSvc(config *SalesOrderLineSvcConfig) domain.SalesOrderLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &salesOrderLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *salesOrderLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *salesOrderLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *salesOrderLineSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &salesOrderLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *salesOrderLineSvcImpl) CreateSalesOrderLine(ctx context.Context, params domain.CreateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineSvcTracer.Start(ctx, "service.sales_order_line.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSalesOrderLineWritePermission(identity, types.ActionUpdate); apiErr != nil {
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

	// Round prices to nearest cent to match legacy behavior
	params.UnitPriceValue = roundToNearestCent(params.UnitPriceValue)
	if params.UnitCostValue != nil {
		rounded := roundToNearestCent(*params.UnitCostValue)
		params.UnitCostValue = &rounded
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.SalesOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.SalesOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderLineSvcImpl) *apierror.APIError {
			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txLineRepo := txSvc.repos.NewSalesOrderLineRepo()
			txPickLineRepo := txSvc.repos.NewPickLineRepo()

			// Validate order exists
			_, apiErr := txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Create line
			created, apiErr := txLineRepo.Create(txCtx, lineID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSalesOrderLine,
				ResourceID:   created.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Create pick line for remaining quantity if the order has a pick
			pickID, apiErr := txOrderRepo.GetPickID(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			if pickID != nil {
				if apiErr := createPickLineForRemainingQuantity(txCtx, txLineRepo, txPickLineRepo, lineID, *pickID); apiErr != nil {
					return apiErr
				}
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

// createPickLineForRemainingQuantity creates a pick line for the remaining quantity
// of an order line that has not yet been picked, matching the legacy Dashboard behavior.
func createPickLineForRemainingQuantity(ctx context.Context, lineRepo domain.SalesOrderLineRepo, pickLineRepo domain.PickLineRepo, orderLineID, pickID string) *apierror.APIError {
	// Calculate remaining quantity to be picked
	remainingValue, unitID, apiErr := pickLineRepo.CalculateRemainingForOrderLine(ctx, orderLineID)
	if apiErr != nil {
		return apiErr
	}

	// Skip if nothing left to pick
	remainingFloat, err := strconv.ParseFloat(remainingValue, 64)
	if err != nil || remainingFloat <= 0 {
		return nil
	}

	// Check if an unpacked pick line already exists
	hasUnpacked, apiErr := pickLineRepo.HasUnpackedPickLineForOrderLine(ctx, orderLineID)
	if apiErr != nil {
		return apiErr
	}
	if hasUnpacked {
		return nil
	}

	// Generate IDs for the new pick line and its quantity
	pickLineID, apiErr := id.GenID(id.PickLineIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	// Create the quantity record
	if apiErr := lineRepo.CreateQuantity(ctx, quantityID, remainingValue, unitID); apiErr != nil {
		return apiErr
	}

	// Create the pick line
	return pickLineRepo.CreateForRemaining(ctx, pickLineID, quantityID, pickID, orderLineID)
}

// roundToNearestCent rounds a decimal string value to 2 decimal places.
func roundToNearestCent(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	rounded := math.Round(f*100) / 100
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

func (s *salesOrderLineSvcImpl) UpdateSalesOrderLine(ctx context.Context, params domain.UpdateSalesOrderLineParams) (*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineSvcTracer.Start(ctx, "service.sales_order_line.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSalesOrderLineWritePermission(identity, types.ActionUpdate); apiErr != nil {
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

	// Round prices to nearest cent to match legacy behavior
	if params.UnitPriceValue != nil {
		rounded := roundToNearestCent(*params.UnitPriceValue)
		params.UnitPriceValue = &rounded
	}
	if params.UnitCostValue != nil {
		rounded := roundToNearestCent(*params.UnitCostValue)
		params.UnitCostValue = &rounded
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.SalesOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SalesOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderLineSvcImpl) *apierror.APIError {
			txLineRepo := txSvc.repos.NewSalesOrderLineRepo()
			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txPickLineRepo := txSvc.repos.NewPickLineRepo()

			// Validate line belongs to order and account owns the order
			isInOrder, apiErr := txLineRepo.IsInOrder(txCtx, params.SalesOrderLineID, params.SalesOrderID, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			if !isInOrder {
				return apierror.NewResourceNotFoundError("Sales order line not found in this order.")
			}

			old, apiErr := txLineRepo.Get(txCtx, params.SalesOrderLineID)
			if apiErr != nil {
				return apiErr
			}

			updated, apiErr := txLineRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrderLine,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Create pick line for remaining quantity if the order has a pick
			pickID, apiErr := txOrderRepo.GetPickID(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			if pickID != nil {
				if apiErr := createPickLineForRemainingQuantity(txCtx, txLineRepo, txPickLineRepo, params.SalesOrderLineID, *pickID); apiErr != nil {
					return apiErr
				}
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

func (s *salesOrderLineSvcImpl) DeleteSalesOrderLine(ctx context.Context, params domain.DeleteSalesOrderLineParams) *apierror.APIError {
	ctx, span := salesOrderLineSvcTracer.Start(ctx, "service.sales_order_line.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := checkSalesOrderLineWritePermission(identity, types.ActionUpdate); apiErr != nil {
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

	lineRepo := s.repos.NewSalesOrderLineRepo()

	// Validate line belongs to order and account owns the order
	isInOrder, apiErr := lineRepo.IsInOrder(ctx, params.SalesOrderLineID, params.SalesOrderID, params.AccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInOrder {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Sales order line not found in this order."))
	}

	salesOrderLine, apiErr := lineRepo.Get(ctx, params.SalesOrderLineID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeSalesOrderLine, params.SalesOrderLineID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This sales order line has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderLineSvcImpl) *apierror.APIError {
		txLineRepo := txSvc.repos.NewSalesOrderLineRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSalesOrderLine, salesOrderLine.ID, salesOrderLine); apiErr != nil {
			return apiErr
		}

		if apiErr := txLineRepo.DeleteCascade(txCtx, params.SalesOrderLineID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(salesOrderLine, (*domain.SalesOrderLine)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeSalesOrderLine,
			ResourceID:   salesOrderLine.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

// checkSalesOrderLineWritePermission checks the appropriate write permission based on the identity context.
// Internal actors need sales_orders:{action} for their own account, or customers:update / suppliers:update for external accounts.
func checkSalesOrderLineWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainSalesOrders, action)
}
