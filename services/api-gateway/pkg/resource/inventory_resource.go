package apiresource

import "github.com/augno/api/shared/constants"

// Item with on-hand inventory quantity.
type InventoryItem struct {
	// Item details.
	Item Item `json:"item" validate:"required"`
	// On-hand quantity.
	Quantity BaseQuantity `json:"quantity" validate:"required"`
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
	return nil
}
