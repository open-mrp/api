package constants

// SupplierMaterialStatus represents whether a supplier material link is active.
type SupplierMaterialStatus string

const (
	// SupplierMaterialStatusActive indicates the supplier material is active.
	SupplierMaterialStatusActive SupplierMaterialStatus = "active"
	// SupplierMaterialStatusInactive indicates the supplier material is inactive.
	SupplierMaterialStatusInactive SupplierMaterialStatus = "inactive"
)

func (s SupplierMaterialStatus) IsValid() bool {
	switch s {
	case SupplierMaterialStatusActive, SupplierMaterialStatusInactive:
		return true
	default:
		return false
	}
}

func (s SupplierMaterialStatus) EnumValues() []string {
	return []string{string(SupplierMaterialStatusActive), string(SupplierMaterialStatusInactive)}
}
