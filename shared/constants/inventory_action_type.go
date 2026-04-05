package constants

// InventoryActionType represents the type of action that caused an inventory change.
type InventoryActionType string

const (
	// InventoryActionTypeScan indicates a scan-based inventory change (e.g. production step).
	InventoryActionTypeScan InventoryActionType = "scan"
	// InventoryActionTypeUserAction indicates a manual user-initiated inventory change.
	InventoryActionTypeUserAction InventoryActionType = "user_action"
	// InventoryActionTypeSystemAction indicates a system-initiated inventory change.
	InventoryActionTypeSystemAction InventoryActionType = "system_action"
	// InventoryActionTypeUserCorrection indicates a user-initiated inventory correction.
	InventoryActionTypeUserCorrection InventoryActionType = "user_correction"
)

func (m InventoryActionType) IsValid() bool {
	switch m {
	case InventoryActionTypeScan, InventoryActionTypeUserAction,
		InventoryActionTypeSystemAction, InventoryActionTypeUserCorrection:
		return true
	default:
		return false
	}
}

func (m InventoryActionType) EnumValues() []string {
	return []string{
		string(InventoryActionTypeScan),
		string(InventoryActionTypeUserAction),
		string(InventoryActionTypeSystemAction),
		string(InventoryActionTypeUserCorrection),
	}
}
