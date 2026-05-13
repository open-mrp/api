package constants

// RemovedResourceScope controls whether removed resources are included in a list.
type RemovedResourceScope string

const (
	// RemovedResourceScopeExcluded omits removed resources.
	RemovedResourceScopeExcluded RemovedResourceScope = "excluded"
	// RemovedResourceScopeIncluded includes removed resources.
	RemovedResourceScopeIncluded RemovedResourceScope = "included"
)

func (m RemovedResourceScope) IsValid() bool {
	switch m {
	case RemovedResourceScopeExcluded, RemovedResourceScopeIncluded:
		return true
	default:
		return false
	}
}

func (m RemovedResourceScope) EnumValues() []string {
	return []string{
		string(RemovedResourceScopeExcluded),
		string(RemovedResourceScopeIncluded),
	}
}
