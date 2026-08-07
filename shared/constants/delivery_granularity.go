package constants

// DeliveryGranularity is the period delivery performance is bucketed into.
type DeliveryGranularity string

const (
	// DeliveryGranularityDay buckets by the day a commitment came due.
	DeliveryGranularityDay DeliveryGranularity = "day"
	// DeliveryGranularityWeek buckets by the week a commitment came due, starting Monday to match the production schedule.
	DeliveryGranularityWeek DeliveryGranularity = "week"
	// DeliveryGranularityMonth buckets by the month a commitment came due.
	DeliveryGranularityMonth DeliveryGranularity = "month"
)

func (m DeliveryGranularity) IsValid() bool {
	switch m {
	case DeliveryGranularityDay, DeliveryGranularityWeek, DeliveryGranularityMonth:
		return true
	default:
		return false
	}
}

func (m DeliveryGranularity) EnumValues() []string {
	return []string{
		string(DeliveryGranularityDay),
		string(DeliveryGranularityWeek),
		string(DeliveryGranularityMonth),
	}
}

func (m *DeliveryGranularity) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
