package constants

// CreatedByRelation describes the relationship of an order's creator to the account that owns the order.
type CreatedByRelation string

const (
	// CreatedByRelationInternal indicates the creator was an internal user of the owning account.
	CreatedByRelationInternal CreatedByRelation = "internal"
	// CreatedByRelationCustomer indicates the creator was a customer of the owning account.
	CreatedByRelationCustomer CreatedByRelation = "customer"
	// CreatedByRelationSystem indicates the resource was created by the system, with no human actor (e.g. EDI import).
	CreatedByRelationSystem CreatedByRelation = "system"
)

func (r CreatedByRelation) IsValid() bool {
	switch r {
	case CreatedByRelationInternal, CreatedByRelationCustomer, CreatedByRelationSystem:
		return true
	default:
		return false
	}
}

func (r CreatedByRelation) EnumValues() []string {
	return []string{
		string(CreatedByRelationInternal),
		string(CreatedByRelationCustomer),
		string(CreatedByRelationSystem),
	}
}

func (r *CreatedByRelation) StringPtr() *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}
