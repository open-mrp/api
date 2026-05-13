package constants

// ItemMatchMode controls how the list-items search query is matched.
type ItemMatchMode string

const (
	// ItemMatchModeExact requires the search query to match exactly.
	ItemMatchModeExact ItemMatchMode = "exact"
	// ItemMatchModePartial allows partial (substring) matching.
	ItemMatchModePartial ItemMatchMode = "partial"
)

func (m ItemMatchMode) IsValid() bool {
	switch m {
	case ItemMatchModeExact, ItemMatchModePartial:
		return true
	default:
		return false
	}
}

func (m ItemMatchMode) EnumValues() []string {
	return []string{string(ItemMatchModeExact), string(ItemMatchModePartial)}
}
