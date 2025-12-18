package types

type IdentityActorType string

const (
	IdentityActorTypeInternal   IdentityActorType = "internal"
	IdentityActorTypeCustomer   IdentityActorType = "customer"
	IdentityActorTypeSupplier   IdentityActorType = "supplier"
	IdentityActorTypeUnassigned IdentityActorType = "unassigned"
	// IdentityActorTypePartner               IdentityActorType = "partner"
)

func ParseIdentityActorType(code string) (IdentityActorType, bool) {
	switch IdentityActorType(code) {
	case IdentityActorTypeInternal,
		IdentityActorTypeCustomer,
		IdentityActorTypeSupplier,
		IdentityActorTypeUnassigned:
		return IdentityActorType(code), true
	default:
		return "", false
	}
}
