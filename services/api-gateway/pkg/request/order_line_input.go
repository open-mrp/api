package apirequest

import "github.com/augno/api/shared/field"

// OrderLineInput represents the shared fields for creating an order line item.
//
// Used as an embedded struct in purchase order and sales order line inputs.
type OrderLineInput struct {
	// ID of the product being ordered.
	ProductID string `json:"product_id" validate:"required"`
	// ID of the inventory item to tie the line to.
	//
	// Lines tied to an item have inventory reserved for them when the order is issued.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// The product SKU recorded on the line.
	//
	// Stored on the line itself, so it stays stable even if the product's SKU changes later.
	ProductSKU string `json:"product_sku" validate:"required,max=255"`
	// The product description recorded on the line.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity ordered, as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required" format:"decimal"`
	// ID of the unit of measure for the quantity.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// Price charged per unit, as a decimal string.
	UnitPriceValue string `json:"unit_price_value" validate:"required" format:"decimal"`
	// Unit ID for the unit price's numerator (the unit being charged, e.g. a currency unit).
	UnitPriceNumeratorUnitID string `json:"unit_price_numerator_unit_id" validate:"required"`
	// Unit ID for the unit price's denominator (the unit being sold, e.g. `each`).
	UnitPriceDenominatorUnitID string `json:"unit_price_denominator_unit_id" validate:"required"`
	// Internal cost per unit, as a decimal string.
	UnitCostValue field.Optional[string] `json:"unit_cost_value,omitzero" format:"decimal"`
	// Unit ID for the unit cost's numerator (the unit being charged, e.g. a currency unit).
	UnitCostNumeratorUnitID field.Optional[string] `json:"unit_cost_numerator_unit_id,omitzero" validate:"omitempty"`
	// Unit ID for the unit cost's denominator (the unit being costed, e.g. `each`).
	UnitCostDenominatorUnitID field.Optional[string] `json:"unit_cost_denominator_unit_id,omitzero" validate:"omitempty"`
}
