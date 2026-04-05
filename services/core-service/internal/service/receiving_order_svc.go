package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

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

var receivingOrderSvcTracer = tracing.GetTracer("core-service.receiving_order_service")

type receivingOrderSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ReceivingOrderSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ReceivingOrderSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("receiving order service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("receiving order service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("receiving order service: tx manager is required")
	}
	return nil
}

func NewReceivingOrderSvc(config *ReceivingOrderSvcConfig) domain.ReceivingOrderSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &receivingOrderSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *receivingOrderSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *receivingOrderSvcImpl) withTx(ctx context.Context, fn func(context.Context, *receivingOrderSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &receivingOrderSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *receivingOrderSvcImpl) ListReceivingOrders(ctx context.Context, params domain.ListReceivingOrdersParams) (*domain.ListReceivingOrdersResult, *apierror.APIError) {
	ctx, span := receivingOrderSvcTracer.Start(ctx, "service.receiving_order.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	// Default to "open" status when not specified, matching Dashboard behavior.
	if params.Status == nil {
		defaultStatus := "open"
		params.Status = &defaultStatus
	}

	return s.repos.NewReceivingOrderRepo().List(ctx, params)
}

func (s *receivingOrderSvcImpl) GetReceivingOrder(ctx context.Context, params domain.GetReceivingOrderParams) (*domain.ReceivingOrder, *apierror.APIError) {
	ctx, span := receivingOrderSvcTracer.Start(ctx, "service.receiving_order.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewReceivingOrderRepo()

	order, apiErr := repo.Get(ctx, params.AccountID, params.ReceivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines, apiErr := repo.ListLines(ctx, params.ReceivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	order.Lines = lines

	return order, nil
}

func (s *receivingOrderSvcImpl) StockReceivingOrder(ctx context.Context, params domain.StockReceivingOrderParams) (*domain.ReceivingOrder, *apierror.APIError) {
	ctx, span := receivingOrderSvcTracer.Start(ctx, "service.receiving_order.stock")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReceivingOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ReceivingOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewReceivingOrderRepo()

			// Fetch old state for audit diff
			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.ReceivingOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Find unstocked lines (enforceNonZero: true)
			unstockedLines, apiErr := txRepo.FindUnstockedLineIDs(txCtx, params.ReceivingOrderID, params.AccountID, true)
			if apiErr != nil {
				return apiErr
			}

			if len(unstockedLines) > 0 {
				// Extract line IDs for stocking
				lineIDs := make([]string, len(unstockedLines))
				for i, line := range unstockedLines {
					lineIDs[i] = line.ID
				}

				// Stock those lines (set stocked_at)
				if apiErr := txRepo.StockLines(txCtx, lineIDs, params.AccountID); apiErr != nil {
					return apiErr
				}

				// Extract order line IDs for bulk create
				orderLineIDs := make([]string, len(unstockedLines))
				for i, line := range unstockedLines {
					orderLineIDs[i] = line.OrderLineID
				}

				// Bulk create new lines for remaining quantities
				if apiErr := txRepo.BulkCreateForRemainingQuantities(txCtx, params.ReceivingOrderID, orderLineIDs, params.AccountID); apiErr != nil {
					return apiErr
				}

				// Check if all stocked -> mark complete
				isComplete, apiErr := txRepo.MarkCompleteIfAllStocked(txCtx, params.ReceivingOrderID, params.AccountID)
				if apiErr != nil {
					return apiErr
				}

				// If complete, mark the purchase order as fulfilled
				if isComplete {
					purchaseOrderID, apiErr := txRepo.GetPurchaseOrderID(txCtx, params.ReceivingOrderID, params.AccountID)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txRepo.MarkPurchaseOrderFulfilled(txCtx, purchaseOrderID, params.AccountID); apiErr != nil {
						return apiErr
					}
				}

				// Create delivery records
				if apiErr := s.createDeliveryRecords(txCtx, txSvc, params); apiErr != nil {
					return apiErr
				}

				// Create inventory change logs for accepted allocations
				if apiErr := s.createInventoryChangeLogs(txCtx, txSvc, params, identity); apiErr != nil {
					return apiErr
				}

				// Allocate open inventory issues for items that received inventory
				if apiErr := s.allocateOpenIssues(txCtx, txSvc, params); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch order with lines
			order, apiErr := txRepo.Get(txCtx, params.AccountID, params.ReceivingOrderID)
			if apiErr != nil {
				return apiErr
			}

			lines, apiErr := txRepo.ListLines(txCtx, params.ReceivingOrderID)
			if apiErr != nil {
				return apiErr
			}
			order.Lines = lines

			result = order

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeReceivingOrder,
				ResourceID:   result.ID,
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

func (s *receivingOrderSvcImpl) ReceiveReceivingOrder(ctx context.Context, receivingOrderID string) (*domain.ReceivingOrder, *apierror.APIError) {
	ctx, span := receivingOrderSvcTracer.Start(ctx, "service.receiving_order.receive")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReceivingOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		repo := s.repos.NewReceivingOrderRepo()

		// Find unstocked lines (enforceNonZero: false)
		unstockedLines, apiErr := repo.FindUnstockedLineIDs(ctx, receivingOrderID, accountID, false)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Fetch old state for audit diff
		old, apiErr := repo.Get(ctx, accountID, receivingOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.ReceivingOrder

		// Only process if there are unstocked lines to receive
		if len(unstockedLines) > 0 {
			orderLineIDs := make([]string, len(unstockedLines))
			for i, line := range unstockedLines {
				orderLineIDs[i] = line.OrderLineID
			}

			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderSvcImpl) *apierror.APIError {
				txRepo := txSvc.repos.NewReceivingOrderRepo()

				if apiErr := txRepo.BulkReceiveRemainingQuantities(txCtx, receivingOrderID, orderLineIDs, accountID); apiErr != nil {
					return apiErr
				}

				updated, apiErr := txRepo.Get(txCtx, accountID, receivingOrderID)
				if apiErr != nil {
					return apiErr
				}

				lines, apiErr := txRepo.ListLines(txCtx, receivingOrderID)
				if apiErr != nil {
					return apiErr
				}
				updated.Lines = lines

				changes := audit.ComputeChanges(old, updated)

				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeReceivingOrder,
					ResourceID:   updated.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}

				result = updated

				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})

			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}

			return result, nil
		}

		order, apiErr := repo.Get(ctx, accountID, receivingOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		lines, apiErr := repo.ListLines(ctx, receivingOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		order.Lines = lines
		result = order

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderSvcImpl) *apierror.APIError {
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

func (s *receivingOrderSvcImpl) VoidReceivingOrder(ctx context.Context, receivingOrderID string) (*domain.ReceivingOrder, *apierror.APIError) {
	ctx, span := receivingOrderSvcTracer.Start(ctx, "service.receiving_order.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReceivingOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		repo := s.repos.NewReceivingOrderRepo()

		// Fetch old state for audit diff
		old, apiErr := repo.Get(ctx, accountID, receivingOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.ReceivingOrder

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewReceivingOrderRepo()

			if apiErr := txRepo.VoidAllLines(txCtx, receivingOrderID, accountID); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.DeleteDuplicateLines(txCtx, receivingOrderID, accountID); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.MarkIncompleteByID(txCtx, receivingOrderID, accountID); apiErr != nil {
				return apiErr
			}

			// Re-fetch updated order for audit diff
			updated, apiErr := txRepo.Get(txCtx, accountID, receivingOrderID)
			if apiErr != nil {
				return apiErr
			}

			lines, apiErr := txRepo.ListLines(txCtx, receivingOrderID)
			if apiErr != nil {
				return apiErr
			}
			updated.Lines = lines

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeReceivingOrder,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
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

// createDeliveryRecords creates delivery and delivery line records, plus inventory receipts
// for each accepted allocation. This matches the Dashboard's createByReceivingOrder behavior.
func (s *receivingOrderSvcImpl) createDeliveryRecords(ctx context.Context, txSvc *receivingOrderSvcImpl, params domain.StockReceivingOrderParams) *apierror.APIError {
	txRepo := txSvc.repos.NewReceivingOrderRepo()
	deliveryRepo := txSvc.repos.NewDeliveryRepo()
	mutationRepo := txSvc.repos.NewInventoryMutationRepo()

	// Get purchase order ID for delivery record
	purchaseOrderID, apiErr := txRepo.GetPurchaseOrderID(ctx, params.ReceivingOrderID, params.AccountID)
	if apiErr != nil {
		return apiErr
	}

	// Get the receiving order to build delivery number
	order, apiErr := txRepo.Get(ctx, params.AccountID, params.ReceivingOrderID)
	if apiErr != nil {
		return apiErr
	}

	// Build a map of receiving order line ID -> unit price info
	unitPrices, apiErr := txRepo.GetLineUnitPrices(ctx, params.ReceivingOrderID)
	if apiErr != nil {
		return apiErr
	}
	unitPriceMap := make(map[string]domain.ReceivingOrderLineUnitPrice)
	for _, up := range unitPrices {
		unitPriceMap[up.ReceivingOrderLineID] = up
	}

	// Count existing deliveries for number generation
	deliveryCount, apiErr := deliveryRepo.CountByPurchaseOrder(ctx, purchaseOrderID)
	if apiErr != nil {
		return apiErr
	}

	// Generate delivery ID and number
	deliveryID, apiErr := id.GenID(id.DeliveryIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	var deliveryNumber string
	if deliveryCount >= 1 {
		deliveryNumber = fmt.Sprintf("%s-%d", order.PurchaseOrderNumber, deliveryCount+1)
	} else {
		deliveryNumber = order.PurchaseOrderNumber
	}

	// Track acceptance/rejection
	hasAccepted := false
	hasRejected := false
	now := time.Now()

	type deliveryLineToCreate struct {
		lineID               string
		receivingOrderLineID string
		quantityID           string
		unitCostID           string
		storageLocationID    *string
		lotID                *string
		acceptedAt           *time.Time
		rejectedAt           *time.Time
	}
	var deliveryLines []deliveryLineToCreate

	for _, lineItem := range params.Data.LineItems {
		up, ok := unitPriceMap[lineItem.ReceivingOrderLineID]
		if !ok {
			continue
		}

		// Upsert lot if lot number provided
		var lotID *string
		if lineItem.LotNumber != nil && strings.TrimSpace(*lineItem.LotNumber) != "" {
			trimmedLot := strings.TrimSpace(*lineItem.LotNumber)
			genLotID, genErr := id.GenID(id.LotIDPrefix, nil)
			if genErr != nil {
				return genErr
			}
			actualLotID, apiErr := txRepo.UpsertLot(ctx, genLotID, params.AccountID, up.ItemID, trimmedLot)
			if apiErr != nil {
				return apiErr
			}
			lotID = &actualLotID
		}

		// Process accepted allocations
		for _, allocation := range lineItem.Allocations {
			hasAccepted = true

			// Create quantity for delivery line
			dlQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := mutationRepo.CreateQuantityForInventory(ctx, dlQuantityID, allocation.Quantity.String(), up.QuantityUnitID); apiErr != nil {
				return apiErr
			}

			// Create unit cost rate for delivery line (clone from order line)
			dlRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := mutationRepo.CreateRateForInventory(ctx, dlRateID, up.UnitPriceValue, up.UnitPriceNumeratorUnitID, up.UnitPriceDenominatorUnitID); apiErr != nil {
				return apiErr
			}

			lineIDVal, apiErr := id.GenID(id.DeliveryLineIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			deliveryLines = append(deliveryLines, deliveryLineToCreate{
				lineID:               lineIDVal,
				receivingOrderLineID: lineItem.ReceivingOrderLineID,
				quantityID:           dlQuantityID,
				unitCostID:           dlRateID,
				storageLocationID:    allocation.LocationID,
				lotID:                lotID,
				acceptedAt:           &now,
			})

			// Create inventory receipt
			receiptID, apiErr := id.GenID(id.InventoryReceiptIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			rcptQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := mutationRepo.CreateQuantityForInventory(ctx, rcptQuantityID, allocation.Quantity.String(), up.QuantityUnitID); apiErr != nil {
				return apiErr
			}

			rcptRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := mutationRepo.CreateRateForInventory(ctx, rcptRateID, up.UnitPriceValue, up.UnitPriceNumeratorUnitID, up.UnitPriceDenominatorUnitID); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.InsertInventoryReceiptForDelivery(ctx, receiptID, params.AccountID, up.ItemID, rcptQuantityID, rcptRateID, allocation.LocationID, lotID, &purchaseOrderID); apiErr != nil {
				return apiErr
			}
		}

		// Process rejected quantity
		if lineItem.RejectedQuantity != nil && lineItem.RejectedQuantity.GreaterThan(decimal.Zero) {
			hasRejected = true

			rejQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := mutationRepo.CreateQuantityForInventory(ctx, rejQuantityID, lineItem.RejectedQuantity.String(), up.QuantityUnitID); apiErr != nil {
				return apiErr
			}

			rejRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := mutationRepo.CreateRateForInventory(ctx, rejRateID, up.UnitPriceValue, up.UnitPriceNumeratorUnitID, up.UnitPriceDenominatorUnitID); apiErr != nil {
				return apiErr
			}

			rejLineID, apiErr := id.GenID(id.DeliveryLineIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			deliveryLines = append(deliveryLines, deliveryLineToCreate{
				lineID:               rejLineID,
				receivingOrderLineID: lineItem.ReceivingOrderLineID,
				quantityID:           rejQuantityID,
				unitCostID:           rejRateID,
				lotID:                lotID,
				rejectedAt:           &now,
			})
		}
	}

	// Determine delivery status
	statusCode := "rejected"
	if hasAccepted {
		statusCode = "accepted"
	}

	var acceptedAt *time.Time
	if hasAccepted {
		acceptedAt = &now
	}
	var rejectedAt *time.Time
	if hasRejected {
		rejectedAt = &now
	}

	// Create the delivery record
	if apiErr := deliveryRepo.CreateDelivery(ctx, deliveryID, deliveryNumber, purchaseOrderID, params.AccountID, statusCode, acceptedAt, rejectedAt); apiErr != nil {
		return apiErr
	}

	// Create all delivery lines
	for _, dl := range deliveryLines {
		if apiErr := deliveryRepo.CreateDeliveryLine(ctx, dl.lineID, deliveryID, dl.receivingOrderLineID, dl.quantityID, dl.unitCostID, dl.storageLocationID, dl.lotID, dl.acceptedAt, dl.rejectedAt); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// createInventoryChangeLogs creates inventory change log entries for each accepted allocation.
func (s *receivingOrderSvcImpl) createInventoryChangeLogs(ctx context.Context, txSvc *receivingOrderSvcImpl, params domain.StockReceivingOrderParams, identity *types.Identity) *apierror.APIError {
	txRepo := txSvc.repos.NewReceivingOrderRepo()
	mutationRepo := txSvc.repos.NewInventoryMutationRepo()

	unitPrices, apiErr := txRepo.GetLineUnitPrices(ctx, params.ReceivingOrderID)
	if apiErr != nil {
		return apiErr
	}
	unitPriceMap := make(map[string]domain.ReceivingOrderLineUnitPrice)
	for _, up := range unitPrices {
		unitPriceMap[up.ReceivingOrderLineID] = up
	}

	for _, lineItem := range params.Data.LineItems {
		if len(lineItem.Allocations) == 0 {
			continue
		}
		up, ok := unitPriceMap[lineItem.ReceivingOrderLineID]
		if !ok {
			continue
		}

		for _, allocation := range lineItem.Allocations {
			var responsibleUserID *string
			if identity.Actor != nil {
				responsibleUserID = &identity.Actor.ID
			}
			if apiErr := mutationRepo.CreateInventoryChangeLog(ctx, domain.CreateInventoryChangeLogParams{
				AccountID:         params.AccountID,
				ItemID:            up.ItemID,
				Measure:           allocation.Quantity,
				UnitID:            up.QuantityUnitID,
				ActionType:        "system_action",
				ResponsibleUserID: responsibleUserID,
			}); apiErr != nil {
				return apiErr
			}
		}
	}

	return nil
}

// allocateOpenIssues performs FIFO allocation of open inventory issues
// for each unique item that had accepted allocations during stocking.
func (s *receivingOrderSvcImpl) allocateOpenIssues(ctx context.Context, txSvc *receivingOrderSvcImpl, params domain.StockReceivingOrderParams) *apierror.APIError {
	txRepo := txSvc.repos.NewReceivingOrderRepo()
	reservationRepo := txSvc.repos.NewInventoryReservationRepo()

	unitPrices, apiErr := txRepo.GetLineUnitPrices(ctx, params.ReceivingOrderID)
	if apiErr != nil {
		return apiErr
	}
	unitPriceMap := make(map[string]domain.ReceivingOrderLineUnitPrice)
	for _, up := range unitPrices {
		unitPriceMap[up.ReceivingOrderLineID] = up
	}

	// Collect unique item IDs that received inventory
	uniqueItemIDs := make(map[string]bool)
	for _, lineItem := range params.Data.LineItems {
		if len(lineItem.Allocations) == 0 {
			continue
		}
		up, ok := unitPriceMap[lineItem.ReceivingOrderLineID]
		if !ok {
			continue
		}
		uniqueItemIDs[up.ItemID] = true
	}

	// For each unique item, allocate open issues against available receipts (FIFO)
	for itemID := range uniqueItemIDs {
		if apiErr := reservationRepo.AllocateOpenIssuesForItem(ctx, params.AccountID, itemID); apiErr != nil {
			return apiErr
		}
	}

	return nil
}
