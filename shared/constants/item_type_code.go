package constants

// ItemTypeCode is what kind of item an item is.
type ItemTypeCode string

const (
	// ItemTypeCodeProduct is a physical item you sell.
	ItemTypeCodeProduct ItemTypeCode = "product"
	// ItemTypeCodeMaterial is bought and consumed in production.
	ItemTypeCodeMaterial ItemTypeCode = "material"
	// ItemTypeCodePart is produced and used to make other items.
	ItemTypeCodePart ItemTypeCode = "part"
	// ItemTypeCodeService is something you sell that is never stocked, picked or shipped.
	ItemTypeCodeService ItemTypeCode = "service"
)

func (m ItemTypeCode) IsValid() bool {
	switch m {
	case ItemTypeCodeProduct, ItemTypeCodeMaterial, ItemTypeCodePart, ItemTypeCodeService:
		return true
	default:
		return false
	}
}

func (m ItemTypeCode) EnumValues() []string {
	return []string{
		string(ItemTypeCodeMaterial),
		string(ItemTypeCodePart),
		string(ItemTypeCodeProduct),
		string(ItemTypeCodeService),
	}
}

// IsPhysical reports whether items of this type are stocked, picked, packed, shipped, produced or consumed.
func (m ItemTypeCode) IsPhysical() bool {
	return m != ItemTypeCodeService
}
