package constants

// ServiceLevelCode identifies a shipping service level for a carrier. Values are carrier-specific and user-defined (e.g. "fedex_ground", "ups_next_day_air").
type ServiceLevelCode string

// IsValid returns true if the service level code is non-empty. ServiceLevelCode values are carrier-specific so any non-empty string is valid.
func (c ServiceLevelCode) IsValid() bool {
	return c != ""
}

// EnumValues returns an empty slice because ServiceLevelCode values are carrier-specific and user-defined.
func (c ServiceLevelCode) EnumValues() []string {
	return []string{}
}
