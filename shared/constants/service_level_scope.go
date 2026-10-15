package constants

// ServiceLevelScope is which service levels a shipping term's free shipping covers.
type ServiceLevelScope string

const (
	// ServiceLevelScopeAll covers every service level.
	ServiceLevelScopeAll ServiceLevelScope = "all"
	// ServiceLevelScopeSelected covers only the term's listed service levels.
	ServiceLevelScopeSelected ServiceLevelScope = "selected"
)

func (s ServiceLevelScope) IsValid() bool {
	switch s {
	case ServiceLevelScopeAll, ServiceLevelScopeSelected:
		return true
	default:
		return false
	}
}

func (s ServiceLevelScope) EnumValues() []string {
	return []string{string(ServiceLevelScopeAll), string(ServiceLevelScopeSelected)}
}
