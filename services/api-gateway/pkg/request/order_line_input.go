package apirequest

import "github.com/augno/api/shared/field"

// OrderLineInput represents the shared fields for creating an order line item.
// Used as an embedded struct in purchase order and sales order line inputs.
type OrderLineInput struct {
	// The product ID.
	ProductID string `json:"product_id" validate:"required"`
	// The item ID.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// The product SKU.
	ProductSKU string `json:"product_sku" validate:"required,max=255"`
	// The product description.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// The quantity value.
	QuantityValue string `json:"quantity_value" validate:"required" format:"decimal"`
	// The quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// The unit price value.
	UnitPriceValue string `json:"unit_price_value" validate:"required" format:"decimal"`
	// The unit price numerator unit ID.
	UnitPriceNumeratorUnitID string `json:"unit_price_numerator_unit_id" validate:"required"`
	// The unit price denominator unit ID.
	UnitPriceDenominatorUnitID string `json:"unit_price_denominator_unit_id" validate:"required"`
	// The unit cost value.
	UnitCostValue field.Optional[string] `json:"unit_cost_value,omitzero" format:"decimal"`
	// The unit cost numerator unit ID.
	UnitCostNumeratorUnitID field.Optional[string] `json:"unit_cost_numerator_unit_id,omitzero" validate:"omitempty"`
	// The unit cost denominator unit ID.
	UnitCostDenominatorUnitID field.Optional[string] `json:"unit_cost_denominator_unit_id,omitzero" validate:"omitempty"`
}
