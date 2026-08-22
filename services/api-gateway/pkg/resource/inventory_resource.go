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
	Item Item `json:"item" validate:"required"`
	// Current on-hand quantity of the item.
	//
	// On-hand is the stock from available receipts less anything already allocated, measured in the base unit of the item's category. Items with no recorded inventory report zero.
	Quantity *Quantity `json:"quantity" validate:"required"`
}

var SampleInventoryItem = &InventoryItem{
	Object:   constants.ObjectTypeInventoryItem,
	Item:     *SampleItem,
	Quantity: SampleQuantity,
}

func (*InventoryItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleInventoryItem)
}
