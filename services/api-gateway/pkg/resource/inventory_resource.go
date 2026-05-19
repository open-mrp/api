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

// Paginated list of inventory items.
type ListInventoriesResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Pagination metadata.
	PageInfo PageInfo `json:"page_info"`
	// Inventory items.
	Data []InventoryItem `json:"data" validate:"required"`
	// Total count.
	Count int64 `json:"count" validate:"required"`
}

// SchemaExample returns an example of ListInventoriesResponse for documentation.
func (*ListInventoriesResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ListInventoriesResponse{
		Object:   constants.ObjectTypeList,
		PageInfo: PageInfo{},
		Data: []InventoryItem{
			{
				Object:   constants.ObjectTypeInventoryItem,
				Item:     *SampleItem,
				Quantity: SampleQuantity,
			},
		},
		Count: 1,
	})
}
