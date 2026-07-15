package apirequest

import "github.com/augno/api/shared/field"

// Shared fields for a line item on a purchase order or sales order.
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
	// Quantity ordered.
	Quantity QuantityInput `json:"quantity" validate:"required"`
	// Price charged per unit.
	UnitPrice RateInput `json:"unit_price" validate:"required"`
	// Internal cost per unit.
	UnitCost field.Optional[RateInput] `json:"unit_cost,omitzero"`
}
