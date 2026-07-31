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
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
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

			// Validate order exists (and read its buyer for pricing).
			order, apiErr := txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Price the line server-side from the product when the caller did not supply a
			// unit price — a line added to an existing order is priced identically to a line
			// created with the order (customer price, discounts, account-price overrides). An
			// explicit price from an internal user is honored as an override. The unit cost is
			// always resolved from the product here, never taken as caller input.
			if apiErr := resolveCreateLinePricing(txCtx, txSvc.repos, identity.IsInternalUser(), order.BuyerAccountID, &params); apiErr != nil {
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
				ResourceType:     constants.ObjectTypeSalesOrderLine,
				ResourceID:       created.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   created.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			// Reconcile pick lines when the order has a pick and the new line is a sale
			// line. Freight/credit (system) lines are never picked, so adding one must not
			// seed a pick line.
			pickID, apiErr := txOrderRepo.GetPickID(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			isSaleLine := created.ProductTypeCode != nil && *created.ProductTypeCode == string(constants.ProductTypeCodeSale)
			if pickID != nil && isSaleLine {
				if apiErr := reconcilePickForOrderLine(txCtx, txLineRepo, txPickLineRepo, txSvc.repos.NewPickRepo(), params.AccountID, lineID, *pickID); apiErr != nil {
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

// resequenceOrderLines compacts an order's line_item_numbers to a contiguous 1..N in the
// current display order — product lines first (keeping their relative order), then
// credit/freight (system) lines. Run after removing a line so gaps close and freight/credit
// stay at the bottom; without it a deleted product line leaves a hole and later additions
// slot above the freight line.
func resequenceOrderLines(ctx context.Context, repos domain.RepoFactory, salesOrderID string) *apierror.APIError {
	lineRepo := repos.NewSalesOrderLineRepo()

	positions, apiErr := lineRepo.GetLineOrder(ctx, salesOrderID)
	if apiErr != nil {
		return apiErr
	}

	// Product lines first (in current order), then system lines (in current order).
	desired := make([]*domain.SalesOrderLinePosition, 0, len(positions))
	for _, p := range positions {
		if !p.IsSystem {
			desired = append(desired, p)
		}
	}
	for _, p := range positions {
		if p.IsSystem {
			desired = append(desired, p)
		}
	}

	for i, p := range desired {
		newNumber := int32(i + 1)
		if p.LineItemNumber == newNumber {
			continue
		}
		if apiErr := lineRepo.SetLineItemNumber(ctx, p.ID, newNumber); apiErr != nil {
			return apiErr
		}
		if apiErr := audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType:     constants.ObjectTypeSalesOrderLine,
			ResourceID:       p.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrderID,
			Changes:          []audit.FieldChange{audit.NewFieldChange("line_item_number", p.LineItemNumber, newNumber)},
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// resolveCreateLinePricing fills a create-line's unit price, unit cost, item, and
// SKU/description from the product's server-side pricing — the same engine order-create
// uses (resolveSalesOrderCreateLines). When the caller supplied a unit price it is passed
// through as an override (honored only for internal users); otherwise the line is priced
// from the product. The unit cost is always product-derived, never caller input.
func resolveCreateLinePricing(ctx context.Context, repos domain.RepoFactory, isInternalActor bool, buyerAccountID string, params *domain.CreateSalesOrderLineParams) *apierror.APIError {
	var override *domain.RateValue
	if params.UnitPriceValue != "" {
		override = &domain.RateValue{
			Value:             params.UnitPriceValue,
			NumeratorUnitID:   params.UnitPriceNumeratorUnitID,
			DenominatorUnitID: params.UnitPriceDenominatorUnitID,
		}
	}

	var sku *string
	if params.ProductSKU != "" {
		sku = &params.ProductSKU
	}

	resolved, apiErr := resolveSalesOrderCreateLines(ctx, repos, params.AccountID, buyerAccountID, isInternalActor, []domain.CreateSalesOrderLineInput{{
		ProductID:          params.ProductID,
		QuantityValue:      params.QuantityValue,
		QuantityUnitID:     params.QuantityUnitID,
		ProductSKU:         sku,
		ProductDescription: params.ProductDescription,
		UnitPrice:          override,
	}})
	if apiErr != nil {
		return apiErr
	}
	r := resolved[0]

	params.UnitPriceValue = r.UnitPrice.Value
	params.UnitPriceNumeratorUnitID = r.UnitPrice.NumeratorUnitID
	params.UnitPriceDenominatorUnitID = r.UnitPrice.DenominatorUnitID
	params.ProductSKU = r.ProductSKU
	params.ProductDescription = r.ProductDescription
	if r.ItemID != "" {
		params.ItemID = &r.ItemID
	}
	// Only attach a unit cost when the product actually carries one, so we never create a
	// zero/empty cost rate for a product with no cost.
	if r.UnitCost.Value != "" {
		params.UnitCostValue = &r.UnitCost.Value
		params.UnitCostNumeratorUnitID = &r.UnitCost.NumeratorUnitID
		params.UnitCostDenominatorUnitID = &r.UnitCost.DenominatorUnitID
	}
	return nil
}

// reconcilePickForOrderLine keeps an order line's pick lines consistent with its
// current ordered quantity after the line is created or its quantity changes. It is
// only called for sale-type lines on orders that already have a pick.
//
// outstanding = ordered - already-packed drives the reconciliation:
//   - outstanding > 0: quantity still needs picking. Ensure a single open (unpacked)
//     pick line exists to hold it — seeded at 0 picked, filled later by the pick
//     action (a pick line's quantity is the amount already picked, so seeding it with
//     the remaining amount would make a line added to a picked order read as already
//     picked) — and reopen the pick, since there is work left to do. This is the
//     increase / add-a-line path and matches the legacy Dashboard behavior.
//   - outstanding <= 0: the packed lines already cover the (possibly reduced) order,
//     so any open pick line is surplus. Delete the open lines and finish the pick when
//     everything that remains is packed. This is the decrease path — e.g. a line
//     picked+packed at 10, bumped to 15 (opening a remainder line), then dropped back
//     to 10 drops that now-unneeded remainder line.
func reconcilePickForOrderLine(ctx context.Context, lineRepo domain.SalesOrderLineRepo, pickLineRepo domain.PickLineRepo, pickRepo domain.PickRepo, accountID, orderLineID, pickID string) *apierror.APIError {
	orderedStr, packedStr, unitID, apiErr := pickLineRepo.GetOrderLinePackProgress(ctx, orderLineID)
	if apiErr != nil {
		return apiErr
	}

	ordered, err := strconv.ParseFloat(orderedStr, 64)
	if err != nil {
		return apierror.NewInternalError(err, "Could not parse ordered quantity for pick reconciliation.")
	}
	packed, err := strconv.ParseFloat(packedStr, 64)
	if err != nil {
		return apierror.NewInternalError(err, "Could not parse packed quantity for pick reconciliation.")
	}

	if ordered-packed > 0 {
		// Ensure an open pick line exists for the outstanding quantity.
		hasUnpacked, apiErr := pickLineRepo.HasUnpackedPickLineForOrderLine(ctx, orderLineID)
		if apiErr != nil {
			return apiErr
		}
		if !hasUnpacked {
			pickLineID, apiErr := id.GenID(id.PickLineIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := lineRepo.CreateQuantity(ctx, quantityID, "0", unitID); apiErr != nil {
				return apiErr
			}
			if apiErr := pickLineRepo.CreateForRemaining(ctx, pickLineID, quantityID, pickID, orderLineID); apiErr != nil {
				return apiErr
			}
		}
		// There is outstanding work, so the pick is not finished.
		return pickRepo.ClearFinishedAt(ctx, accountID, pickID)
	}

	// Packed already covers the order: drop any surplus open pick line, then finish the
	// pick if every line that remains is packed.
	if apiErr := pickLineRepo.DeleteUnpackedForOrderLine(ctx, orderLineID); apiErr != nil {
		return apiErr
	}
	return pickRepo.MarkFinishedIfAllPacked(ctx, pickID)
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
				ResourceType:     constants.ObjectTypeSalesOrderLine,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   updated.SalesOrderID,
				Changes:          changes,
			}); apiErr != nil {
				return apiErr
			}

			// Keep downstream fulfillment records in sync with a quantity edit. Pick,
			// shipment, and invoice lines each snapshot the order line's quantity into
			// their own quantity row when created and are never touched again, so an
			// edit here would leave them stale. The rules:
			//   - Unit always follows on all three (a unit edit is a data correction,
			//     and the pack/invoiced rollups sum these values without conversion).
			//   - Value follows for shipment and invoice lines only when it still
			//     mirrors the pre-update ordered quantity; partial snapshots keep the
			//     amount that actually moved (legacy billing semantics).
			//   - Pick line values are picking progress and are handled by the pick
			//     reconciliation below instead.
			// Unit price needs no sync — invoice reads resolve it live from the order
			// line's rate.
			if params.QuantityValue != nil || params.QuantityUnitID != nil {
				if updated.QuantityUnitID != old.QuantityUnitID {
					if apiErr := txLineRepo.SyncPickLineQuantityUnits(txCtx, params.SalesOrderLineID, updated.QuantityUnitID); apiErr != nil {
						return apiErr
					}
				}
				if apiErr := txLineRepo.SyncInvoiceLineQuantities(txCtx, params.SalesOrderLineID, old.QuantityValue, updated.QuantityValue, updated.QuantityUnitID); apiErr != nil {
					return apiErr
				}
				if apiErr := txLineRepo.SyncShipmentLineQuantities(txCtx, params.SalesOrderLineID, old.QuantityValue, updated.QuantityValue, updated.QuantityUnitID); apiErr != nil {
					return apiErr
				}
			}

			// Reconcile pick lines with the new quantity when the order has a pick — but
			// only for sale product lines. Freight/credit (system) lines are never picked,
			// so updating one must not touch pick lines (matches legacy order-line.repo.ts,
			// which gates this on product.productType === 'sale').
			pickID, apiErr := txOrderRepo.GetPickID(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}
			isSaleLine := old.ProductTypeCode != nil && *old.ProductTypeCode == string(constants.ProductTypeCodeSale)
			if pickID != nil && isSaleLine {
				if apiErr := reconcilePickForOrderLine(txCtx, txLineRepo, txPickLineRepo, txSvc.repos.NewPickRepo(), params.AccountID, params.SalesOrderLineID, *pickID); apiErr != nil {
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
	orderRepo := s.repos.NewSalesOrderRepo()

	// Validate line belongs to order and account owns the order
	isInOrder, apiErr := lineRepo.IsInOrder(ctx, params.SalesOrderLineID, params.SalesOrderID, params.AccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInOrder {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Sales order line not found in this order."))
	}

	// Block deletion of lines from a fulfilled order (hard block, no admin override).
	order, apiErr := orderRepo.Get(ctx, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if order.SalesOrderStatusCode == string(constants.SalesOrderStatusCodeFulfilled) {
		return tracing.Trace(span, apierror.NewResourceConflictError("Cannot delete lines from a fulfilled order."))
	}

	// Block deletion once the line is part of any shipment (packed or shipped). A packed
	// line has a shipment_line, so deleting it would orphan committed shipment/pick state.
	hasShipment, apiErr := lineRepo.HasShipmentAgainstOrderLine(ctx, params.SalesOrderLineID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if hasShipment {
		return tracing.Trace(span, apierror.NewResourceConflictError("Cannot delete a line item that has a shipment against it."))
	}

	// Matches Dashboard's OrderUtils.isEditable admin gate: a completed-but-not-fulfilled order, or an order with any shipped shipment, may still have lines deleted — but only by an admin.
	hasShippedShipment, apiErr := orderRepo.HasShippedShipment(ctx, params.SalesOrderID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if order.CompletedAt != nil || hasShippedShipment {
		if apiErr := identity.CheckIsAdmin(); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
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
		txOrderRepo := txSvc.repos.NewSalesOrderRepo()
		txPickRepo := txSvc.repos.NewPickRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSalesOrderLine, salesOrderLine.ID, salesOrderLine); apiErr != nil {
			return apiErr
		}

		// DeleteCascade removes this line's pick lines (and their quantities) along with
		// the line itself.
		if apiErr := txLineRepo.DeleteCascade(txCtx, params.SalesOrderLineID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(salesOrderLine, (*domain.SalesOrderLine)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType:     constants.ObjectTypeSalesOrderLine,
			ResourceID:       salesOrderLine.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   salesOrderLine.SalesOrderID,
			Changes:          changes,
		}); apiErr != nil {
			return apiErr
		}

		// Close the gap the removed line left: re-sequence the remaining lines to 1..N so
		// product lines stay contiguous and credit/freight lines stay at the bottom.
		// Otherwise a deleted product line leaves a hole that pushes later additions above
		// the freight line (e.g. delete both product lines and freight stays numbered 3).
		if apiErr := resequenceOrderLines(txCtx, txSvc.repos, params.SalesOrderID); apiErr != nil {
			return apiErr
		}

		// Reconcile the order's pick now that this line's pick lines are gone.
		pickID, apiErr := txOrderRepo.GetPickID(txCtx, params.SalesOrderID)
		if apiErr != nil {
			return apiErr
		}
		if pickID != nil {
			lineCount, apiErr := txPickRepo.CountLines(txCtx, *pickID)
			if apiErr != nil {
				return apiErr
			}
			if lineCount == 0 {
				// The pick has no lines left, so the order has nothing to fulfill: delete
				// the empty pick and revert the order to estimate, releasing reserved
				// inventory — the same teardown the unissue action performs.
				if apiErr := txOrderRepo.DeletePickBySalesOrder(txCtx, params.SalesOrderID); apiErr != nil {
					return apiErr
				}
				if apiErr := txOrderRepo.DeleteInventoryAllocationsByReservedIssues(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
					return apiErr
				}
				if apiErr := txOrderRepo.DeleteReservedInventoryIssues(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
					return apiErr
				}
				if apiErr := txOrderRepo.UpdateStatus(txCtx, params.AccountID, params.SalesOrderID, string(constants.SalesOrderStatusCodeEstimate), nil, nil); apiErr != nil {
					return apiErr
				}
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType:     constants.ObjectTypeSalesOrder,
					ResourceID:       order.ID,
					RootResourceType: constants.ObjectTypeSalesOrder,
					RootResourceID:   order.ID,
					Changes: audit.ComputeChanges(order, &domain.SalesOrder{
						ID:                   order.ID,
						Number:               order.Number,
						SalesOrderStatusCode: string(constants.SalesOrderStatusCodeEstimate),
					}),
				}); apiErr != nil {
					return apiErr
				}
			} else {
				// Lines remain: finish the pick if every one that is left is packed.
				if apiErr := txPickRepo.MarkFinishedIfAllPacked(txCtx, *pickID); apiErr != nil {
					return apiErr
				}
			}
		}

		return nil
	})
}

func (s *salesOrderLineSvcImpl) ReorderSalesOrderLines(ctx context.Context, params domain.ReorderSalesOrderLinesParams) ([]*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderLineSvcTracer.Start(ctx, "service.sales_order_line.reorder")
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

	var result []*domain.SalesOrderLine
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderLineSvcImpl) *apierror.APIError {
		txOrderRepo := txSvc.repos.NewSalesOrderRepo()
		txLineRepo := txSvc.repos.NewSalesOrderLineRepo()

		// Validate the order exists and the account owns it.
		if _, apiErr := txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
			return apiErr
		}

		positions, apiErr := txLineRepo.GetLineOrder(txCtx, params.SalesOrderID)
		if apiErr != nil {
			return apiErr
		}

		// Partition into product lines (reorderable) and credit/freight lines, which stay at the bottom in their current relative order.
		productPositions := make(map[string]*domain.SalesOrderLinePosition, len(positions))
		systemInOrder := make([]*domain.SalesOrderLinePosition, 0, len(positions))
		currentByID := make(map[string]int32, len(positions))
		for _, p := range positions {
			currentByID[p.ID] = p.LineItemNumber
			if p.IsSystem {
				systemInOrder = append(systemInOrder, p)
				continue
			}
			productPositions[p.ID] = p
		}

		// The submitted list must be exactly the order's product lines: no duplicates, unknown/foreign IDs, credit/freight lines, or omissions.
		if len(params.LineIDs) != len(productPositions) {
			return apierror.NewValidationError("The reordered list must contain every product line on the order exactly once.")
		}
		seen := make(map[string]bool, len(params.LineIDs))
		for _, lineID := range params.LineIDs {
			if seen[lineID] {
				return apierror.NewValidationError("Duplicate line in the reordered list: " + lineID)
			}
			if _, ok := productPositions[lineID]; !ok {
				return apierror.NewValidationError("Line does not belong to this order's product lines: " + lineID)
			}
			seen[lineID] = true
		}

		// Final order: submitted product lines first, then credit/freight lines. Assign contiguous numbers starting at 1.
		desired := make([]string, 0, len(positions))
		desired = append(desired, params.LineIDs...)
		for _, p := range systemInOrder {
			desired = append(desired, p.ID)
		}

		for i, lineID := range desired {
			newNumber := int32(i + 1)
			if currentByID[lineID] == newNumber {
				continue
			}
			if apiErr := txLineRepo.SetLineItemNumber(txCtx, lineID, newNumber); apiErr != nil {
				return apiErr
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeSalesOrderLine,
				ResourceID:       lineID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   params.SalesOrderID,
				Changes:          []audit.FieldChange{audit.NewFieldChange("line_item_number", currentByID[lineID], newNumber)},
			}); apiErr != nil {
				return apiErr
			}
		}

		lines, apiErr := txLineRepo.List(txCtx, params.SalesOrderID)
		if apiErr != nil {
			return apiErr
		}
		result = lines
		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// checkSalesOrderLineWritePermission checks the appropriate write permission based on the identity context. Internal actors need sales_orders:{action} for their own account, or customers:update / suppliers:update for external accounts.
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
