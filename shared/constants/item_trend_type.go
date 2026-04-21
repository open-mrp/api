package constants

// ItemTrendType represents a time-series trend that can be fetched for an item.
type ItemTrendType string

const (
	// ItemTrendTypeInventory returns 30 days of inventory-log measurements.
	ItemTrendTypeInventory ItemTrendType = "inventory"
)

func (m ItemTrendType) IsValid() bool {
	switch m {
	case ItemTrendTypeInventory:
		return true
	default:
		return false
	}
}

func (m ItemTrendType) EnumValues() []string {
	return []string{
		string(ItemTrendTypeInventory),
	}
}
