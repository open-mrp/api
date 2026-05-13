package constants

// InventoryUpdateOperation controls how quantity_change is applied when updating item inventory.
type InventoryUpdateOperation string

const (
	// InventoryUpdateOperationAdjust adds quantity_change to the current inventory.
	InventoryUpdateOperationAdjust InventoryUpdateOperation = "adjust"
	// InventoryUpdateOperationReconcile sets inventory to the exact value given by quantity_change.
	InventoryUpdateOperationReconcile InventoryUpdateOperation = "reconcile"
)

func (o InventoryUpdateOperation) IsValid() bool {
	switch o {
	case InventoryUpdateOperationAdjust, InventoryUpdateOperationReconcile:
		return true
	default:
		return false
	}
}

func (o InventoryUpdateOperation) EnumValues() []string {
	return []string{string(InventoryUpdateOperationAdjust), string(InventoryUpdateOperationReconcile)}
}
