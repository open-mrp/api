package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Item with on-hand inventory quantity.
type InventoryItem struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=inventory_item"`
	// Item details.
	Item Item `json:"item" validate:"required"`
	// On-hand quantity.
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
