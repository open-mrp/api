package constants

import "strings"

// ItemCategoryType is what kind of items an item category holds.
type ItemCategoryType string

const (
	// ItemCategoryTypeMaterial holds materials.
	ItemCategoryTypeMaterial ItemCategoryType = "material"
	// ItemCategoryTypeProduct holds products and parts.
	ItemCategoryTypeProduct ItemCategoryType = "product"
)

// itemCategoryTypeCodeSuffix is what the stored type code adds to the API value: material_category.
const itemCategoryTypeCodeSuffix = "_category"

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

// Code is the type as item_category.item_category_type_code stores it.
func (m ItemCategoryType) Code() string {
	return string(m) + itemCategoryTypeCodeSuffix
}

// ItemCategoryTypeFromCode is the type a stored item_category_type_code holds.
func ItemCategoryTypeFromCode(code string) ItemCategoryType {
	return ItemCategoryType(strings.TrimSuffix(code, itemCategoryTypeCodeSuffix))
}
