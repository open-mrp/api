package service

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// validatePurchaseOrderLineUnits rejects a line whose quantity is expressed in a unit the product is not measured in.
//
// Without this a purchase order can record "1 dollar" of a product sold in pairs, and because issuing an order copies its lines onto the receiving order, the nonsense quantity is what stock gets booked against. The sales-order path has always checked this; purchase orders did not, and the two now answer the same way for the same mistake.
//
// A product the account does not own resolves to no units at all and is reported as an unknown product, which is the more accurate complaint: the caller's problem is the product reference, not the unit.
func validatePurchaseOrderLineUnits(ctx context.Context, repos domain.RepoFactory, accountID string, lines []domain.CreatePurchaseOrderLineInput, param string) *apierror.APIError {
	if len(lines) == 0 {
		return nil
	}

	productIDs := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if line.ProductID == "" {
			continue
		}
		if _, ok := seen[line.ProductID]; ok {
			continue
		}
		seen[line.ProductID] = struct{}{}
		productIDs = append(productIDs, line.ProductID)
	}
	if len(productIDs) == 0 {
		return nil
	}

	unitsByProduct, apiErr := repos.NewPricingRepo().ProductQuantityUnits(ctx, accountID, productIDs)
	if apiErr != nil {
		return apiErr
	}

	for _, line := range lines {
		if line.ProductID == "" {
			continue
		}
		units, ok := unitsByProduct[line.ProductID]
		if !ok {
			return apierror.NewValidationErrorWithParam("Product not found.", "product_id")
		}
		// An empty set means the product's unit group has no units configured, which is a catalog problem rather than a bad request. Saying so beats blaming the unit the caller sent.
		if len(units) == 0 {
			return apierror.NewValidationError("The product's unit group is not configured.")
		}
		if _, ok := units[line.QuantityUnitID]; !ok {
			return apierror.NewValidationErrorWithParam("The unit is not valid for this product.", param)
		}
	}

	return nil
}
