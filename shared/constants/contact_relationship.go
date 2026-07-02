package constants

// ContactRelationship describes how the caller relates to the account a matched contact belongs to.
type ContactRelationship string

const (
	// ContactRelationshipCustomer indicates the contact belongs to one of the caller's customers.
	ContactRelationshipCustomer ContactRelationship = "customer"
	// ContactRelationshipSupplier indicates the contact belongs to one of the caller's suppliers.
	ContactRelationshipSupplier ContactRelationship = "supplier"
	// ContactRelationshipSelf indicates the contact belongs to the caller's own account.
	ContactRelationshipSelf ContactRelationship = "self"
)

func (r ContactRelationship) IsValid() bool {
	switch r {
	case ContactRelationshipCustomer, ContactRelationshipSupplier, ContactRelationshipSelf:
		return true
	default:
		return false
	}
}

func (r ContactRelationship) EnumValues() []string {
	return []string{string(ContactRelationshipCustomer), string(ContactRelationshipSupplier), string(ContactRelationshipSelf)}
}

func (r *ContactRelationship) StringPtr() *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}
