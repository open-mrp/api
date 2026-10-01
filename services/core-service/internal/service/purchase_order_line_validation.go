package service

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// validatePurchaseOrderLineUnits rejects a line whose quantity is expressed in a unit the product is not measured in.
//
// Without this a purchase order can record "1 dollar" of a product sold in pairs, and because issuing an order copies its lines onto the receiving order, the nonsense quantity is what stock gets booked against. The sales-order path has always checked this; purchase orders did not, and the two now answer the same way for the same mistake.
//
// A line for a material names only the item it restocks, not a product, so such a line is checked against the unit group of the item's category instead.
//
// A product or item the account does not own resolves to no units at all and is reported as unknown, which is the more accurate complaint: the caller's problem is the reference, not the unit.
func validatePurchaseOrderLineUnits(ctx context.Context, repos domain.RepoFactory, accountID string, lines []domain.CreatePurchaseOrderLineInput, param string) *apierror.APIError {
	if len(lines) == 0 {
		return nil
	}

	var productIDs, itemIDs []string
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		switch {
		case line.ProductID != nil && *line.ProductID != "":
			productIDs = appendUnseen(productIDs, seen, *line.ProductID)
		case line.ItemID != nil && *line.ItemID != "":
			itemIDs = appendUnseen(itemIDs, seen, *line.ItemID)
		default:
			return apierror.NewValidationErrorWithParam("A line must name a product or an item.", "product_id")
		}
	}

	pricingRepo := repos.NewPricingRepo()
	unitsByProduct, apiErr := pricingRepo.ProductQuantityUnits(ctx, accountID, productIDs)
	if apiErr != nil {
		return apiErr
	}
	unitsByItem, apiErr := pricingRepo.ItemQuantityUnits(ctx, accountID, itemIDs)
	if apiErr != nil {
		return apiErr
	}

	for _, line := range lines {
		var units map[string]struct{}
		var ok bool
		if line.ProductID != nil && *line.ProductID != "" {
			if units, ok = unitsByProduct[*line.ProductID]; !ok {
				return apierror.NewValidationErrorWithParam("Product not found.", "product_id")
			}
		} else if units, ok = unitsByItem[*line.ItemID]; !ok {
			return apierror.NewValidationErrorWithParam("Item not found.", "item_id")
		}
		// An empty set means the unit group has no units configured, which is a catalog problem rather than a bad request. Saying so beats blaming the unit the caller sent.
		if len(units) == 0 {
			return apierror.NewValidationError("The product's unit group is not configured.")
		}
		if _, ok := units[line.QuantityUnitID]; !ok {
			return apierror.NewValidationErrorWithParam("The unit is not valid for this product.", param)
		}
	}

	return nil
}

func appendUnseen(ids []string, seen map[string]struct{}, id string) []string {
	if _, ok := seen[id]; ok {
		return ids
	}
	seen[id] = struct{}{}
	return append(ids, id)
}
