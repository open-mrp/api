package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// An item together with its current on-hand inventory quantity.
type InventoryItem struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_item"`
	// The item this inventory entry reports on.
	Item *Item `json:"item" validate:"required"`
	// The item's quantity, in the base unit of its category.
	//
	// Derived rather than stored. Normally the current on-hand stock: available receipts less anything already allocated. When the list was asked for `as_of` a past instant, the last inventory level logged by then instead. Items with no recorded inventory report zero.
	Quantity *ComputedQuantity `json:"quantity" validate:"required"`
	// The product line the item sells under.
	//
	// Null for an item that is not a product or has no line, and unless requested with `include=product_line`.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
}

var SampleInventoryItem = &InventoryItem{
	Object:      constants.ObjectTypeInventoryItem,
	Item:        SampleItem,
	Quantity:    SampleComputedQuantity,
	ProductLine: SampleProductLine,
}

func (*InventoryItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInventoryItem)
}
