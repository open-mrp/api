package constants

// CustomerRelationshipType represents a customer's position in an account hierarchy.
type CustomerRelationshipType string

const (
	// CustomerRelationshipTypeStandalone indicates the customer has no parent or child accounts.
	CustomerRelationshipTypeStandalone CustomerRelationshipType = "standalone"
	// CustomerRelationshipTypeParent indicates the customer has child accounts.
	CustomerRelationshipTypeParent CustomerRelationshipType = "parent"
	// CustomerRelationshipTypeChild indicates the customer belongs to a parent account.
	CustomerRelationshipTypeChild CustomerRelationshipType = "child"
)

func (m CustomerRelationshipType) IsValid() bool {
	switch m {
	case CustomerRelationshipTypeStandalone, CustomerRelationshipTypeParent, CustomerRelationshipTypeChild:
		return true
	default:
		return false
	}
}

func (m CustomerRelationshipType) EnumValues() []string {
	return []string{
		string(CustomerRelationshipTypeStandalone),
		string(CustomerRelationshipTypeParent),
		string(CustomerRelationshipTypeChild),
	}
}

// CustomerParentAccountStatus filters whether customers have child accounts.
type CustomerParentAccountStatus string

const (
	// CustomerParentAccountStatusParent indicates customers with child accounts.
	CustomerParentAccountStatusParent CustomerParentAccountStatus = "parent"
	// CustomerParentAccountStatusNonParent indicates customers without child accounts.
	CustomerParentAccountStatusNonParent CustomerParentAccountStatus = "non_parent"
)

func (m CustomerParentAccountStatus) IsValid() bool {
	switch m {
	case CustomerParentAccountStatusParent, CustomerParentAccountStatusNonParent:
		return true
	default:
		return false
	}
}

func (m CustomerParentAccountStatus) EnumValues() []string {
	return []string{
		string(CustomerParentAccountStatusParent),
		string(CustomerParentAccountStatusNonParent),
	}
}
