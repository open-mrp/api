package constants

// OwnerType indicates the provenance of a resource.
type OwnerType string

const (
	// OwnerTypeSystem indicates the resource is a platform-provided system default.
	OwnerTypeSystem OwnerType = "system"
	// OwnerTypeAccount indicates the resource is owned by a specific account.
	OwnerTypeAccount OwnerType = "account"
)

func (m OwnerType) IsValid() bool {
	switch m {
	case OwnerTypeSystem, OwnerTypeAccount:
		return true
	default:
		return false
	}
}

func (m OwnerType) EnumValues() []string {
	return []string{
		string(OwnerTypeSystem),
		string(OwnerTypeAccount),
	}
}
