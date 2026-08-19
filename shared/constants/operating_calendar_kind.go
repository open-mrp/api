package constants

// OperatingCalendarKind names which side of a shipment a calendar describes. The two are kept in one table because they are the same shape — open weekdays less dated closures — and an account often wants the same holiday set on both.
type OperatingCalendarKind string

const (
	// OperatingCalendarKindShip is the days a plant tenders freight to a carrier, and the only kind that carries a pickup cutoff.
	OperatingCalendarKindShip OperatingCalendarKind = "ship"
	// OperatingCalendarKindReceive is the days a customer's dock accepts freight.
	OperatingCalendarKindReceive OperatingCalendarKind = "receive"
)

func (m OperatingCalendarKind) IsValid() bool {
	switch m {
	case OperatingCalendarKindShip, OperatingCalendarKindReceive:
		return true
	default:
		return false
	}
}

func (m OperatingCalendarKind) EnumValues() []string {
	return []string{
		string(OperatingCalendarKindShip),
		string(OperatingCalendarKindReceive),
	}
}

func (m *OperatingCalendarKind) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
