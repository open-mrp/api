package apiendpoint

import "github.com/open-mrp/api/services/auth-service/pkg/types"

// CounterpartyPermissions names the permission a request needs when it acts in a customer's account and when it acts in a supplier's account. A zero entry leaves RequiredPermissions in force for that kind of account.
type CounterpartyPermissions struct {
	Customer types.Permission
	Supplier types.Permission
}

// Counterparties requires customers:<action> in a customer's account and suppliers:<action> in a supplier's account.
func Counterparties(action types.Action) CounterpartyPermissions {
	return CounterpartyPermissions{
		Customer: types.Permission{Domain: types.PermissionDomainCustomers, Action: action},
		Supplier: types.Permission{Domain: types.PermissionDomainSuppliers, Action: action},
	}
}

// For returns the permission the identity's target account calls for, and false when the account is the caller's own or no entry covers its kind.
func (c CounterpartyPermissions) For(identity *types.Identity) (types.Permission, bool) {
	var p types.Permission
	switch {
	case identity.IsTargetCustomerAccount():
		p = c.Customer
	case identity.IsTargetSupplierAccount():
		p = c.Supplier
	}
	return p, p != types.Permission{}
}
