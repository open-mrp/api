package constants

// InventoryMovementType is the business event an inventory movement records. Who caused it is the movement's actor,
// never its type.
type InventoryMovementType string

const (
	// InventoryMovementTypeReceipt indicates stock received from a supplier.
	InventoryMovementTypeReceipt InventoryMovementType = "receipt"
	// InventoryMovementTypeShipment indicates stock shipped to a customer, or returned to stock when a shipment is undone.
	InventoryMovementTypeShipment InventoryMovementType = "shipment"
	// InventoryMovementTypeConsumption indicates stock consumed by production.
	InventoryMovementTypeConsumption InventoryMovementType = "consumption"
	// InventoryMovementTypeOutput indicates stock produced.
	InventoryMovementTypeOutput InventoryMovementType = "output"
	// InventoryMovementTypeTransfer indicates stock moved between locations.
	InventoryMovementTypeTransfer InventoryMovementType = "transfer"
	// InventoryMovementTypeAdjustment indicates a manual change by an amount.
	InventoryMovementTypeAdjustment InventoryMovementType = "adjustment"
	// InventoryMovementTypeReconciliation indicates a count that set stock to an exact quantity.
	InventoryMovementTypeReconciliation InventoryMovementType = "reconciliation"
	// InventoryMovementTypeReturn indicates stock returned by a customer.
	InventoryMovementTypeReturn InventoryMovementType = "return"
)

func (t InventoryMovementType) IsValid() bool {
	switch t {
	case InventoryMovementTypeReceipt, InventoryMovementTypeShipment, InventoryMovementTypeConsumption,
		InventoryMovementTypeOutput, InventoryMovementTypeTransfer, InventoryMovementTypeAdjustment,
		InventoryMovementTypeReconciliation, InventoryMovementTypeReturn:
		return true
	default:
		return false
	}
}

func (t InventoryMovementType) EnumValues() []string {
	return []string{
		string(InventoryMovementTypeReceipt),
		string(InventoryMovementTypeShipment),
		string(InventoryMovementTypeConsumption),
		string(InventoryMovementTypeOutput),
		string(InventoryMovementTypeTransfer),
		string(InventoryMovementTypeAdjustment),
		string(InventoryMovementTypeReconciliation),
		string(InventoryMovementTypeReturn),
	}
}

// InventoryMovementReason is why stock was corrected by an adjustment or reconciliation.
type InventoryMovementReason string

const (
	// InventoryMovementReasonDamaged indicates stock written off as damaged.
	InventoryMovementReasonDamaged InventoryMovementReason = "damaged"
	// InventoryMovementReasonCountCorrection indicates a correction after a count.
	InventoryMovementReasonCountCorrection InventoryMovementReason = "count_correction"
	// InventoryMovementReasonFound indicates stock found that wasn't on the books.
	InventoryMovementReasonFound InventoryMovementReason = "found"
	// InventoryMovementReasonScrap indicates stock scrapped.
	InventoryMovementReasonScrap InventoryMovementReason = "scrap"
	// InventoryMovementReasonOther indicates any other reason.
	InventoryMovementReasonOther InventoryMovementReason = "other"
)

func (r InventoryMovementReason) IsValid() bool {
	switch r {
	case InventoryMovementReasonDamaged, InventoryMovementReasonCountCorrection, InventoryMovementReasonFound,
		InventoryMovementReasonScrap, InventoryMovementReasonOther:
		return true
	default:
		return false
	}
}

func (r InventoryMovementReason) EnumValues() []string {
	return []string{
		string(InventoryMovementReasonDamaged),
		string(InventoryMovementReasonCountCorrection),
		string(InventoryMovementReasonFound),
		string(InventoryMovementReasonScrap),
		string(InventoryMovementReasonOther),
	}
}

func (t *InventoryMovementType) StringPtr() *string   { return stringPtrEnum(t) }
func (r *InventoryMovementReason) StringPtr() *string { return stringPtrEnum(r) }
