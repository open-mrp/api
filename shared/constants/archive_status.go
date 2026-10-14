package constants

// ArchiveStatus is the `status` of a resource that is retired rather than deleted while records still use it.
// Archived resources stay on those records but can't be newly assigned. Tables store it as `is_active`.
type ArchiveStatus string

const (
	// ArchiveStatusActive can be assigned to new and updated records.
	ArchiveStatusActive ArchiveStatus = "active"
	// ArchiveStatusArchived stays on the records already using it and can't be newly assigned.
	ArchiveStatusArchived ArchiveStatus = "archived"
)

func (s ArchiveStatus) IsValid() bool {
	switch s {
	case ArchiveStatusActive, ArchiveStatusArchived:
		return true
	default:
		return false
	}
}

func (s ArchiveStatus) EnumValues() []string {
	return []string{string(ArchiveStatusActive), string(ArchiveStatusArchived)}
}

// IsActive is the status as the `is_active` column stores it.
func (s ArchiveStatus) IsActive() bool {
	return s == ArchiveStatusActive
}

// ArchiveStatusFromIsActive is the status an `is_active` column holds.
func ArchiveStatusFromIsActive(isActive bool) ArchiveStatus {
	if isActive {
		return ArchiveStatusActive
	}
	return ArchiveStatusArchived
}

func (s *ArchiveStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
