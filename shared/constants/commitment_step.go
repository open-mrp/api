package constants

// CommitmentStep names one rule in the derivation of a ship-by date. Returned as an ordered list so a caller can render why a date is what it is without reimplementing the arithmetic, and so the explanation cannot drift from the calculation.
type CommitmentStep string

const (
	// CommitmentStepBasis is the starting date, from whichever basis the order pinned or the lead-time chain resolved.
	CommitmentStepBasis CommitmentStep = "basis"
	// CommitmentStepReceiveCalendar is the move back onto a day the customer's dock accepts freight.
	CommitmentStepReceiveCalendar CommitmentStep = "receive_calendar"
	// CommitmentStepCarrierTransit is the walk back through the days the carrier moves freight.
	CommitmentStepCarrierTransit CommitmentStep = "carrier_transit"
	// CommitmentStepShipCalendar is the move back onto a day the plant tenders freight.
	CommitmentStepShipCalendar CommitmentStep = "ship_calendar"
	// CommitmentStepPickupCutoff is the time of day the ship-by date resolves to, from the plant's cutoff.
	CommitmentStepPickupCutoff CommitmentStep = "pickup_cutoff"
)

func (m CommitmentStep) IsValid() bool {
	switch m {
	case CommitmentStepBasis, CommitmentStepReceiveCalendar, CommitmentStepCarrierTransit, CommitmentStepShipCalendar, CommitmentStepPickupCutoff:
		return true
	default:
		return false
	}
}

func (m CommitmentStep) EnumValues() []string {
	return []string{
		string(CommitmentStepBasis),
		string(CommitmentStepReceiveCalendar),
		string(CommitmentStepCarrierTransit),
		string(CommitmentStepShipCalendar),
		string(CommitmentStepPickupCutoff),
	}
}

func (m *CommitmentStep) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
