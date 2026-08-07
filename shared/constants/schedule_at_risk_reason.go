package constants

// ScheduleAtRiskReason is why a plan does not meet an order's ship-by commitment.
type ScheduleAtRiskReason string

const (
	// ScheduleAtRiskReasonPastDue indicates production needed to start before the plan begins.
	ScheduleAtRiskReasonPastDue ScheduleAtRiskReason = "past_due"
	// ScheduleAtRiskReasonUndated indicates the order carries no ship-by commitment and is treated as owed now.
	ScheduleAtRiskReasonUndated ScheduleAtRiskReason = "undated"
	// ScheduleAtRiskReasonShort indicates the plan projects less stock than the order needs in the week it is needed.
	ScheduleAtRiskReasonShort ScheduleAtRiskReason = "short"
)

func (m ScheduleAtRiskReason) IsValid() bool {
	switch m {
	case ScheduleAtRiskReasonPastDue, ScheduleAtRiskReasonUndated, ScheduleAtRiskReasonShort:
		return true
	default:
		return false
	}
}

func (m ScheduleAtRiskReason) EnumValues() []string {
	return []string{
		string(ScheduleAtRiskReasonPastDue),
		string(ScheduleAtRiskReasonUndated),
		string(ScheduleAtRiskReasonShort),
	}
}

func (m *ScheduleAtRiskReason) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
