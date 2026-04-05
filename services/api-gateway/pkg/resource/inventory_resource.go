package apiresource

import "github.com/augno/api/shared/constants"

// InventoryItem represents an item with its on-hand inventory quantity.
type InventoryItem struct {
	// The item details.
	Item Item `json:"item" validate:"required"`
	// The on-hand quantity.
	Quantity BaseQuantity `json:"quantity" validate:"required"`
}

// ListInventoriesResponse represents the response from listing all item inventories.
type ListInventoriesResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=list"`
	// Pagination metadata.
	PageInfo PageInfo `json:"page_info"`
	// The inventory items.
	Data []InventoryItem `json:"data" validate:"required"`
	// The total count.
	Count int64 `json:"count" validate:"required"`
}

// SchemaExample returns an example of ListInventoriesResponse for documentation.
func (*ListInventoriesResponse) SchemaExample() any {
	return nil
}
