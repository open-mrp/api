package constants

// PickSearchMatch is how a pick list's `q` is matched against the pick and order numbers it searches.
type PickSearchMatch string

const (
	// PickSearchMatchPrefix matches the start of the pick number and the customer PO number.
	PickSearchMatchPrefix PickSearchMatch = "prefix"
	// PickSearchMatchContains matches anywhere in the pick number and the customer PO number.
	PickSearchMatchContains PickSearchMatch = "contains"
)

func (s PickSearchMatch) IsValid() bool {
	switch s {
	case PickSearchMatchPrefix, PickSearchMatchContains:
		return true
	default:
		return false
	}
}

func (s PickSearchMatch) EnumValues() []string {
	return []string{string(PickSearchMatchPrefix), string(PickSearchMatchContains)}
}

func (s *PickSearchMatch) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
