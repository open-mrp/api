package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// planItemUnits validates the units requested for an item and returns what to store. current is the item's units
// before the write (nil on create); a field the input leaves nil keeps its current value. A default order unit left
// out defaults to the stocking unit, or for a service to its first order unit; one that is no longer allowed after
// the order units are replaced falls back the same way. prefix is the request path the params sit under ("" or
// "items[3].").
func planItemUnits(ctx context.Context, repos domain.RepoFactory, accountID string, itemType constants.ItemTypeCode,
	current *domain.ItemUnits, in domain.ItemUnitsInput, prefix string) (*domain.ItemUnits, *apierror.APIError) {
	service := itemType == constants.ItemTypeCodeService
	plan := &domain.ItemUnits{OrderUnits: []domain.ItemOrderUnit{}}
	if current != nil {
		plan.StockingUnitID = current.StockingUnitID
		plan.DefaultOrderUnitID = current.DefaultOrderUnitID
		plan.OrderUnits = append(plan.OrderUnits, current.OrderUnits...)
	}

	if in.StockingUnitID != nil {
		if service {
			return nil, apierror.NewFieldError(prefix+"stocking_unit_id", apierror.CodeInvalidFormat, "A service is never stocked, so it has no stocking unit.")
		}
		plan.StockingUnitID = *in.StockingUnitID
	}
	if !service && plan.StockingUnitID == "" {
		return nil, apierror.NewFieldError(prefix+"stocking_unit_id", apierror.CodeMissingField, "stocking_unit_id is required for a material, part or product.")
	}
	if in.OrderUnits != nil {
		plan.OrderUnits = append([]domain.ItemOrderUnit{}, (*in.OrderUnits)...)
	}
	defaultGiven := in.DefaultOrderUnitID != nil
	if defaultGiven {
		plan.DefaultOrderUnitID = *in.DefaultOrderUnitID
	}

	ids := []string{plan.StockingUnitID, plan.DefaultOrderUnitID}
	for _, ou := range plan.OrderUnits {
		ids = append(ids, ou.UnitID)
	}
	factors, apiErr := repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, ids)
	if apiErr != nil {
		return nil, apiErr
	}

	dimension := ""
	if plan.StockingUnitID != "" {
		f, ok := factors[plan.StockingUnitID]
		if !ok {
			return nil, apierror.NewFieldError(prefix+"stocking_unit_id", apierror.CodeInvalidFormat, "The stocking unit does not exist.")
		}
		if f.DimensionCode == string(constants.UnitTypeCurrency) {
			return nil, apierror.NewFieldError(prefix+"stocking_unit_id", apierror.CodeInvalidFormat, "An item can't be stocked in a currency.")
		}
		dimension = f.DimensionCode
	}

	if service && !defaultGiven && len(plan.OrderUnits) > 0 && plan.OrderUnit(plan.DefaultOrderUnitID) == nil {
		plan.DefaultOrderUnitID = plan.OrderUnits[0].UnitID
	}
	if service && dimension == "" && plan.DefaultOrderUnitID != "" {
		if f, ok := factors[plan.DefaultOrderUnitID]; ok {
			dimension = f.DimensionCode
		}
	}

	seen := make(map[string]bool, len(plan.OrderUnits))
	for i := range plan.OrderUnits {
		ou := &plan.OrderUnits[i]
		param := fmt.Sprintf("%sorder_units[%d]", prefix, i)
		if ou.UnitID == "" {
			return nil, apierror.NewFieldError(param+".unit_id", apierror.CodeMissingField, "unit_id is required.")
		}
		if ou.UnitID == plan.StockingUnitID {
			return nil, apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "The stocking unit is always orderable; list only the other units.")
		}
		if seen[ou.UnitID] {
			return nil, apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "Each unit can be listed once.")
		}
		seen[ou.UnitID] = true
		f, ok := factors[ou.UnitID]
		if !ok {
			return nil, apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "The unit does not exist.")
		}
		if dimension != "" && f.DimensionCode != dimension {
			return nil, apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat,
				fmt.Sprintf("Every order unit must measure %s, like the item's stocking unit (a service's default order unit).", dimension))
		}
		if ou.PortalVisibility == "" {
			ou.PortalVisibility = constants.CustomerPortalVisibilityVisible
		}
		if !ou.PortalVisibility.IsValid() {
			return nil, apierror.NewFieldError(param+".portal_visibility", apierror.CodeInvalidFormat, "portal_visibility must be visible or hidden.")
		}
		if ou.StockingQuantity != nil {
			if service {
				return nil, apierror.NewFieldError(param+".quantity", apierror.CodeInvalidFormat, "A service has no stocking unit to count in.")
			}
			q, err := decimal.NewFromString(*ou.StockingQuantity)
			if err != nil || !q.IsPositive() {
				return nil, apierror.NewFieldError(param+".quantity", apierror.CodeInvalidFormat, "quantity must be a decimal greater than zero.")
			}
			v := q.String()
			ou.StockingQuantity = &v
		}
	}

	if plan.DefaultOrderUnitID == "" || !plan.Allows(plan.DefaultOrderUnitID) {
		if defaultGiven {
			return nil, apierror.NewFieldError(prefix+"default_order_unit_id", apierror.CodeInvalidFormat,
				"The default order unit must be the stocking unit or one of order_units.")
		}
		plan.DefaultOrderUnitID = plan.StockingUnitID
		if service && len(plan.OrderUnits) > 0 {
			plan.DefaultOrderUnitID = plan.OrderUnits[0].UnitID
		}
	}
	if plan.DefaultOrderUnitID == "" {
		return nil, apierror.NewFieldError(prefix+"order_units", apierror.CodeMissingField, "A service needs at least one order unit.")
	}
	plan.Stored = true
	return plan, nil
}

// storeItemUnits writes planned units onto an item.
func storeItemUnits(ctx context.Context, repos domain.RepoFactory, accountID, itemID string, plan *domain.ItemUnits) *apierror.APIError {
	units := repos.NewItemUnitsRepo()
	if apiErr := units.SetStockingUnit(ctx, itemID, plan.StockingUnitID); apiErr != nil {
		return apiErr
	}
	return units.Replace(ctx, accountID, itemID, plan.DefaultOrderUnitID, plan.OrderUnits)
}

// itemUnitsInputGiven reports whether a write names any unit field.
func itemUnitsInputGiven(in *domain.ItemUnitsInput) bool {
	return in != nil && (in.StockingUnitID != nil || in.DefaultOrderUnitID != nil || in.OrderUnits != nil)
}

// requireAllowedItemUnit refuses a unit that is neither the item's stocking unit nor one of its order units.
func requireAllowedItemUnit(units *domain.ItemUnits, unitID, param string) *apierror.APIError {
	if units.Allows(unitID) {
		return nil
	}
	return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "The unit must be the item's stocking unit or one of its order units.")
}
