package constants

// AccountRelationRole is the role an account_relation plays from the owner account's perspective. It is the discriminator stored in account_relation.account_relation_role_code and the key a contact role is configured against (the customer-support contact, the supplier contact, etc.).
type AccountRelationRole string

const (
	// AccountRelationRoleCustomer marks a counterparty the owner sells to (portal customers).
	AccountRelationRoleCustomer AccountRelationRole = "customer"
	// AccountRelationRoleSupplier marks a counterparty the owner buys from.
	AccountRelationRoleSupplier AccountRelationRole = "supplier"
	// AccountRelationRolePartner marks a non-buy/sell counterparty relationship.
	AccountRelationRolePartner AccountRelationRole = "partner"
)

func (r AccountRelationRole) IsValid() bool {
	switch r {
	case AccountRelationRoleCustomer, AccountRelationRoleSupplier, AccountRelationRolePartner:
		return true
	default:
		return false
	}
}

func (r AccountRelationRole) EnumValues() []string {
	return []string{
		string(AccountRelationRoleCustomer),
		string(AccountRelationRoleSupplier),
		string(AccountRelationRolePartner),
	}
}

func (r *AccountRelationRole) StringPtr() *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}
