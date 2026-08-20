package constants

// Names the order a list of picks comes back in.
type PickSort string

const (
	// Orders by the sales order's delivery commitment, soonest first, so the floor sees the most urgent work at the top. Picks whose order has no ship-by date sort last.
	PickSortShipByDate PickSort = "ship_by_date"
	// Orders by when the pick was created, newest first.
	PickSortCreatedAt PickSort = "created_at"
)

func (s PickSort) IsValid() bool {
	switch s {
	case PickSortShipByDate, PickSortCreatedAt:
		return true
	default:
		return false
	}
}

func (s PickSort) EnumValues() []string {
	return []string{
		string(PickSortShipByDate),
		string(PickSortCreatedAt),
	}
}

func (s *PickSort) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
