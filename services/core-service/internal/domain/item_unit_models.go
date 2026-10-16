package domain

import (
	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/constants"
)

// ItemUnits is the units an item is kept and ordered in.
type ItemUnits struct {
	// StockingUnitID is the unit the item's inventory, costs and usage are kept in. Empty for a service.
	StockingUnitID string `audit:"stocking_unit_id"`
	// DefaultOrderUnitID is the unit an order line selects by default: the stocking unit or one of OrderUnits.
	DefaultOrderUnitID string `audit:"default_order_unit_id"`
	// OrderUnits are the units the item can be ordered in besides its stocking unit, in their listed order.
	OrderUnits []ItemOrderUnit `audit:"order_units"`
	// Stored is false while the item's units are still derived from its unit groups (not yet backfilled).
	Stored bool
}

// ItemOrderUnit is one unit an item can be ordered in.
type ItemOrderUnit struct {
	UnitID           string                             `audit:"unit_id"`
	PortalVisibility constants.CustomerPortalVisibility `audit:"portal_visibility"`
	Discount         *Discount                          `audit:"discount"`
	// StockingQuantity is how many stocking units one of this unit holds for this item, overriding the unit's ratio.
	StockingQuantity *string `audit:"quantity"`
}

// AllowedUnitIDs is every unit the item can be ordered, received or adjusted in: the stocking unit and the order
// units.
func (u *ItemUnits) AllowedUnitIDs() []string {
	if u == nil {
		return nil
	}
	out := make([]string, 0, len(u.OrderUnits)+1)
	if u.StockingUnitID != "" {
		out = append(out, u.StockingUnitID)
	}
	for _, ou := range u.OrderUnits {
		out = append(out, ou.UnitID)
	}
	return out
}

// Allows reports whether unitID is the stocking unit or one of the order units.
func (u *ItemUnits) Allows(unitID string) bool {
	if u == nil || unitID == "" {
		return false
	}
	if unitID == u.StockingUnitID {
		return true
	}
	return u.OrderUnit(unitID) != nil
}

// OrderUnit returns the order unit for unitID, or nil when it is not one.
func (u *ItemUnits) OrderUnit(unitID string) *ItemOrderUnit {
	if u == nil {
		return nil
	}
	for i := range u.OrderUnits {
		if u.OrderUnits[i].UnitID == unitID {
			return &u.OrderUnits[i]
		}
	}
	return nil
}

// ApplyDiscount takes the order unit's discount off a price already converted to the unit. A discount larger than the
// price leaves zero.
func (ou *ItemOrderUnit) ApplyDiscount(price decimal.Decimal) decimal.Decimal {
	if ou == nil || ou.Discount == nil {
		return price
	}
	switch ou.Discount.Type {
	case constants.DiscountTypePercentOff:
		if ou.Discount.PercentOff == nil {
			return price
		}
		pct, err := decimal.NewFromString(*ou.Discount.PercentOff)
		if err != nil {
			return price
		}
		price = price.Mul(decimal.NewFromInt(1).Sub(pct.Div(decimal.NewFromInt(100))))
	case constants.DiscountTypeAmountOff:
		if ou.Discount.AmountOff == nil {
			return price
		}
		amount, err := decimal.NewFromString(ou.Discount.AmountOff.Amount)
		if err != nil {
			return price
		}
		price = price.Sub(amount)
	}
	if price.IsNegative() {
		return decimal.Zero
	}
	return price
}

// ItemTypeRef is an item's id and type.
type ItemTypeRef struct {
	ID   string
	Type constants.ItemTypeCode
}

// ItemUnitsInput is the units requested on an item create or update. Nil fields are unchanged (or defaulted on
// create).
type ItemUnitsInput struct {
	StockingUnitID     *string
	DefaultOrderUnitID *string
	// OrderUnits replaces the order units when non-nil; an empty slice removes them all.
	OrderUnits *[]ItemOrderUnit
}

// ChangeStockingUnitMode says how a stocking unit change treats the numbers recorded in the old unit.
type ChangeStockingUnitMode string

const (
	// ChangeStockingUnitModeConvert restates quantities, costs and prices into the new unit of the same dimension.
	ChangeStockingUnitModeConvert ChangeStockingUnitMode = "convert"
	// ChangeStockingUnitModeRelabel keeps every number and only changes the unit, to fix one entered in error.
	ChangeStockingUnitModeRelabel ChangeStockingUnitMode = "relabel"
)

// IsValid reports whether m is a known mode.
func (m ChangeStockingUnitMode) IsValid() bool {
	return m == ChangeStockingUnitModeConvert || m == ChangeStockingUnitModeRelabel
}

// ChangeStockingUnitParams asks to move an item to a new stocking unit.
type ChangeStockingUnitParams struct {
	ItemID         string
	StockingUnitID string
	Mode           ChangeStockingUnitMode
}

// SetOrderUnitsFilter picks the items Set Order Units applies to; every filter given must match.
type SetOrderUnitsFilter struct {
	ItemIDs        []string
	ProductLineIDs []string
	CategoryIDs    []string
	AttributeIDs   []string
}

// SetOrderUnitsParams asks to replace the order units and default order unit of every matching item.
type SetOrderUnitsParams struct {
	Filter             SetOrderUnitsFilter
	OrderUnits         []ItemOrderUnit
	DefaultOrderUnitID *string
}

// ResolvedSetOrderUnitsRow is one item a Set Order Units job writes.
type ResolvedSetOrderUnitsRow struct {
	ItemID             string
	ItemType           constants.ItemTypeCode
	OrderUnits         []ItemOrderUnit
	DefaultOrderUnitID *string
}

// ResolvedChangeStockingUnitRow is the item a Change Stocking Unit job restates.
type ResolvedChangeStockingUnitRow struct {
	ItemID         string
	StockingUnitID string
	Mode           ChangeStockingUnitMode
}
