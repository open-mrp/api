package constants

// ItemCategoryType represents the kind of item category.
type ItemCategoryType string

const (
	// ItemCategoryTypeMaterial represents a category for raw materials or components.
	ItemCategoryTypeMaterial ItemCategoryType = "material_category"
	// ItemCategoryTypeProduct represents a category for finished products.
	ItemCategoryTypeProduct ItemCategoryType = "product_category"
)

func (m ItemCategoryType) IsValid() bool {
	switch m {
	case ItemCategoryTypeMaterial, ItemCategoryTypeProduct:
		return true
	default:
		return false
	}
}

func (m ItemCategoryType) EnumValues() []string {
	return []string{
		string(ItemCategoryTypeMaterial),
		string(ItemCategoryTypeProduct),
	}
}
