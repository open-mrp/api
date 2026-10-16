package constants

// ChangeStockingUnitMode says how a stocking unit change treats the numbers an item recorded in its old unit.
type ChangeStockingUnitMode string

const (
	// ChangeStockingUnitModeConvert restates quantities, costs and prices into the new unit of the same dimension.
	ChangeStockingUnitModeConvert ChangeStockingUnitMode = "convert"
	// ChangeStockingUnitModeRelabel keeps every number and only changes the unit, to fix one entered in error.
	ChangeStockingUnitModeRelabel ChangeStockingUnitMode = "relabel"
)

func (m ChangeStockingUnitMode) IsValid() bool {
	switch m {
	case ChangeStockingUnitModeConvert, ChangeStockingUnitModeRelabel:
		return true
	default:
		return false
	}
}

func (m ChangeStockingUnitMode) EnumValues() []string {
	return []string{string(ChangeStockingUnitModeConvert), string(ChangeStockingUnitModeRelabel)}
}

func (m *ChangeStockingUnitMode) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
