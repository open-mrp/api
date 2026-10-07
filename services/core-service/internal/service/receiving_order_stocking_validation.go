package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pricing"
)

// stockingTolerance is how far the allocated and refused quantities may exceed a line's received quantity, in the line's unit, before the request is refused. It is the dashboard stocking dialog's own allowance, so a split it accepts is never rejected here for rounding.
var stockingTolerance = decimal.RequireFromString("0.001")

// validateStockingData refuses a stocking request that would book inventory the receiving order does not account for.
//
// Every line item must name a line of this order that is being stocked now (unstocked, with something received on it), at most once. Each allocation must be positive, at one of the account's storage locations, and each refusal non-negative, in a unit of the line's item. Together they may not exceed what was received on the line. The dashboard checks the same before it sends; without these checks a request could put stock away against a line that was already stocked, or against another order's line, and book inventory twice.
func validateStockingData(ctx context.Context, repos domain.RepoFactory, accountID string, data domain.StockingData, lines []*domain.ReceivingOrderLine, stockable map[string]struct{}) *apierror.APIError {
	if len(data.LineItems) == 0 {
		return nil
	}

	byID := make(map[string]*domain.ReceivingOrderLine, len(lines))
	for _, l := range lines {
		byID[l.ID] = l
	}

	seen := make(map[string]struct{}, len(data.LineItems))
	var itemIDs, productIDs, locationIDs []string
	seenItems, seenProducts, seenLocations := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	unitIDs := map[string]struct{}{}
	for _, li := range data.LineItems {
		if _, dup := seen[li.ReceivingOrderLineID]; dup {
			return apierror.NewValidationErrorWithParam("A receiving order line can appear only once in a stocking request.", "line_items.receiving_order_line_id")
		}
		seen[li.ReceivingOrderLineID] = struct{}{}

		line, ok := byID[li.ReceivingOrderLineID]
		if !ok {
			return apierror.NewValidationErrorWithParam("The receiving order line is not on this receiving order.", "line_items.receiving_order_line_id")
		}
		if _, ok := stockable[line.ID]; !ok {
			return apierror.NewValidationErrorWithParam("The receiving order line is already stocked or has nothing received on it.", "line_items.receiving_order_line_id")
		}

		for _, a := range li.Allocations {
			if !a.Quantity.Value.IsPositive() {
				return apierror.NewValidationErrorWithParam("An allocated quantity must be greater than zero.", "line_items.allocations.quantity.value")
			}
			unitIDs[a.Quantity.UnitID] = struct{}{}
			if a.LocationID != nil {
				locationIDs = appendUnseen(locationIDs, seenLocations, *a.LocationID)
			}
		}
		if li.RejectedQuantity != nil {
			if li.RejectedQuantity.Value.IsNegative() {
				return apierror.NewValidationErrorWithParam("A rejected quantity must not be negative.", "line_items.rejected_quantity.value")
			}
			unitIDs[li.RejectedQuantity.UnitID] = struct{}{}
		}
		unitIDs[line.QuantityUnitID] = struct{}{}

		switch {
		case line.OrderLineItemID != nil && *line.OrderLineItemID != "":
			itemIDs = appendUnseen(itemIDs, seenItems, *line.OrderLineItemID)
		case line.OrderLineProductID != nil && *line.OrderLineProductID != "":
			productIDs = appendUnseen(productIDs, seenProducts, *line.OrderLineProductID)
		}
	}

	if apiErr := validateStockingLocations(ctx, repos, accountID, locationIDs); apiErr != nil {
		return apiErr
	}

	pricingRepo := repos.NewPricingRepo()
	unitsByItem, apiErr := pricingRepo.ItemQuantityUnits(ctx, accountID, itemIDs)
	if apiErr != nil {
		return apiErr
	}
	unitsByProduct, apiErr := pricingRepo.ProductQuantityUnits(ctx, accountID, productIDs)
	if apiErr != nil {
		return apiErr
	}

	ids := make([]string, 0, len(unitIDs))
	for id := range unitIDs {
		ids = append(ids, id)
	}
	ratios, apiErr := repos.NewReceivingOrderRepo().GetUnitRatios(ctx, ids)
	if apiErr != nil {
		return apiErr
	}

	for _, li := range data.LineItems {
		line := byID[li.ReceivingOrderLineID]

		var allowed map[string]struct{}
		switch {
		case line.OrderLineItemID != nil && *line.OrderLineItemID != "":
			allowed = unitsByItem[*line.OrderLineItemID]
		case line.OrderLineProductID != nil && *line.OrderLineProductID != "":
			allowed = unitsByProduct[*line.OrderLineProductID]
		}

		lineValue, err := decimal.NewFromString(line.QuantityValue)
		if err != nil {
			return apierror.NewInternalError(err, "Failed to read the receiving order line's quantity.")
		}
		lineRatio, ok := ratios[line.QuantityUnitID]
		if !ok {
			return apierror.NewInvariantViolationError("The receiving order line's unit was not found.")
		}

		accounted := decimal.Zero
		add := func(q domain.ReceivedQuantity, param string) *apierror.APIError {
			if allowed != nil {
				if _, ok := allowed[q.UnitID]; !ok {
					return apierror.NewValidationErrorWithParam("The unit is not valid for this item.", param)
				}
			}
			if q.UnitID == line.QuantityUnitID {
				accounted = accounted.Add(q.Value)
				return nil
			}
			ratio, ok := ratios[q.UnitID]
			if !ok {
				return apierror.NewValidationErrorWithParam("The unit is not valid for this item.", param)
			}
			accounted = accounted.Add(pricing.ConvertQuantity(q.Value, ratio, lineRatio))
			return nil
		}

		for _, a := range li.Allocations {
			if apiErr := add(a.Quantity, "line_items.allocations.quantity.unit_id"); apiErr != nil {
				return apiErr
			}
		}
		if li.RejectedQuantity != nil {
			if apiErr := add(*li.RejectedQuantity, "line_items.rejected_quantity.unit_id"); apiErr != nil {
				return apiErr
			}
		}

		if accounted.GreaterThan(lineValue.Add(stockingTolerance)) {
			return apierror.NewValidationErrorWithParam("The allocated and rejected quantities add up to more than was received on the line.", "line_items.allocations")
		}
	}

	return nil
}

// validateStockingLocations refuses a put-away at a location that is not one of the account's, whose receipt nobody could find on the floor.
func validateStockingLocations(ctx context.Context, repos domain.RepoFactory, accountID string, locationIDs []string) *apierror.APIError {
	if len(locationIDs) == 0 {
		return nil
	}
	locations, apiErr := repos.NewLocationRepo().GetByIDs(ctx, accountID, locationIDs)
	if apiErr != nil {
		return apiErr
	}
	found := make(map[string]struct{}, len(locations))
	for _, l := range locations {
		found[l.ID] = struct{}{}
	}
	for _, id := range locationIDs {
		if _, ok := found[id]; !ok {
			return apierror.NewValidationErrorWithParam("The storage location was not found.", "line_items.allocations.location_id")
		}
	}
	return nil
}

// validateReceivedQuantityUnit refuses a quantity counted in a unit the line's item (or product) is not measured in.
func validateReceivedQuantityUnit(ctx context.Context, repos domain.RepoFactory, accountID string, line *domain.ReceivingOrderLine, q domain.ReceivedQuantity, param string) *apierror.APIError {
	var allowed map[string]struct{}
	pricingRepo := repos.NewPricingRepo()
	switch {
	case line.OrderLineItemID != nil && *line.OrderLineItemID != "":
		units, apiErr := pricingRepo.ItemQuantityUnits(ctx, accountID, []string{*line.OrderLineItemID})
		if apiErr != nil {
			return apiErr
		}
		allowed = units[*line.OrderLineItemID]
	case line.OrderLineProductID != nil && *line.OrderLineProductID != "":
		units, apiErr := pricingRepo.ProductQuantityUnits(ctx, accountID, []string{*line.OrderLineProductID})
		if apiErr != nil {
			return apiErr
		}
		allowed = units[*line.OrderLineProductID]
	default:
		return nil
	}
	if _, ok := allowed[q.UnitID]; !ok {
		return apierror.NewValidationErrorWithParam("The unit is not valid for this item.", param)
	}
	return nil
}
