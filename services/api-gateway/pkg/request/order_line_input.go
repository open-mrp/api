package apirequest

import "github.com/augno/api/shared/field"

// Details of a single line item ordered from a supplier, used when creating a purchase order and when adding a line to an existing one.
type OrderLineInput struct {
	// ID of the product being ordered.
	ProductID string `json:"product_id" validate:"required"`
	// ID of the inventory item this line is linked to.
	//
	// Stock received against the line is booked into this item, so lines for goods you hold in inventory should reference one. Supplying an item also records the item's material as sourced from this order's supplier, with `product_sku` as the supplier part number, when that link does not exist yet.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// The product SKU recorded on the line.
	//
	// Stored on the line itself, so it stays stable even if the product's SKU changes later.
	ProductSKU string `json:"product_sku" validate:"required,max=255"`
	// The product description recorded on the line.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity ordered from the supplier.
	Quantity QuantityInput `json:"quantity" validate:"required"`
	// Agreed purchase price per unit for this line.
	//
	// This is also the cost carried into inventory: stock received against the line is costed at this rate.
	UnitPrice RateInput `json:"unit_price" validate:"required"`
	// Cost per unit recorded on the line, if you capture it separately from the agreed purchase price.
	//
	// Kept for reference only; it does not affect how stock received against the line is costed.
	UnitCost field.Optional[RateInput] `json:"unit_cost,omitzero"`
}
