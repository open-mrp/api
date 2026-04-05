package types

type IdentityRelationType string

const (
	IdentityRelationTypeInternal   IdentityRelationType = "internal"
	IdentityRelationTypeCustomer   IdentityRelationType = "customer"
	IdentityRelationTypeSupplier   IdentityRelationType = "supplier"
	IdentityRelationTypeUnassigned IdentityRelationType = "unassigned"
	// IdentityRelationTypePartner               IdentityRelationType = "partner"
)

func (t IdentityRelationType) IsValid() bool {
	switch t {
	case IdentityRelationTypeInternal,
		IdentityRelationTypeCustomer,
		IdentityRelationTypeSupplier,
		IdentityRelationTypeUnassigned:
		return true
	default:
		return false
	}
}

func (t IdentityRelationType) EnumValues() []string {
	return []string{
		string(IdentityRelationTypeInternal),
		string(IdentityRelationTypeCustomer),
		string(IdentityRelationTypeSupplier),
		string(IdentityRelationTypeUnassigned),
	}
}

func ParseIdentityRelationType(code string) (IdentityRelationType, bool) {
	t := IdentityRelationType(code)
	if t.IsValid() {
		return t, true
	}
	return "", false
}
