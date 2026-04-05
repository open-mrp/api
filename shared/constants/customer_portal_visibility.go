package constants

// CustomerPortalVisibility represents whether a resource is visible in the customer portal.
type CustomerPortalVisibility string

const (
	// CustomerPortalVisibilityVisible indicates the resource is visible in the customer portal.
	CustomerPortalVisibilityVisible CustomerPortalVisibility = "visible"
	// CustomerPortalVisibilityHidden indicates the resource is hidden from the customer portal.
	CustomerPortalVisibilityHidden CustomerPortalVisibility = "hidden"
)

func (m CustomerPortalVisibility) IsValid() bool {
	switch m {
	case CustomerPortalVisibilityVisible, CustomerPortalVisibilityHidden:
		return true
	default:
		return false
	}
}

func (m CustomerPortalVisibility) EnumValues() []string {
	return []string{
		string(CustomerPortalVisibilityVisible),
		string(CustomerPortalVisibilityHidden),
	}
}
