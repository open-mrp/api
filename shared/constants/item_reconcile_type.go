package constants

// ItemReconcileType is how a bulk reconcile applies each quantity to the item's current quantity.
type ItemReconcileType string

const (
	// ItemReconcileTypeAddition adds the quantity to the item's current quantity.
	ItemReconcileTypeAddition ItemReconcileType = "addition"
	// ItemReconcileTypeForce sets the item's current quantity to exactly the given quantity.
	ItemReconcileTypeForce ItemReconcileType = "force"
)

func (t ItemReconcileType) IsValid() bool {
	switch t {
	case ItemReconcileTypeAddition, ItemReconcileTypeForce:
		return true
	default:
		return false
	}
}

func (t ItemReconcileType) EnumValues() []string {
	return []string{
		string(ItemReconcileTypeAddition),
		string(ItemReconcileTypeForce),
	}
}
