package constants

// ItemTypeCode represents the type of an item.
type ItemTypeCode string

const (
	// ItemTypeCodeProduct represents a finished product.
	ItemTypeCodeProduct ItemTypeCode = "product"
	// ItemTypeCodeMaterial represents a raw material or component.
	ItemTypeCodeMaterial ItemTypeCode = "material"
	// ItemTypeCodePart represents a part used in production.
	ItemTypeCodePart ItemTypeCode = "part"
)

func (m ItemTypeCode) IsValid() bool {
	switch m {
	case ItemTypeCodeProduct, ItemTypeCodeMaterial, ItemTypeCodePart:
		return true
	default:
		return false
	}
}

func (m ItemTypeCode) EnumValues() []string {
	return []string{
		string(ItemTypeCodeProduct),
		string(ItemTypeCodeMaterial),
		string(ItemTypeCodePart),
	}
}
