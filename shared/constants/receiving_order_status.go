package constants

// ReceivingOrderStatus filters a list of receiving orders by completion.
type ReceivingOrderStatus string

const (
	// ReceivingOrderStatusOpen returns orders that have not been completed.
	ReceivingOrderStatusOpen ReceivingOrderStatus = "open"
	// ReceivingOrderStatusCompleted returns orders that have been completed.
	ReceivingOrderStatusCompleted ReceivingOrderStatus = "completed"
	// ReceivingOrderStatusAll returns orders in either state.
	ReceivingOrderStatusAll ReceivingOrderStatus = "all"
)

func (s ReceivingOrderStatus) IsValid() bool {
	switch s {
	case ReceivingOrderStatusOpen, ReceivingOrderStatusCompleted, ReceivingOrderStatusAll:
		return true
	default:
		return false
	}
}

func (s ReceivingOrderStatus) EnumValues() []string {
	return []string{
		string(ReceivingOrderStatusOpen),
		string(ReceivingOrderStatusCompleted),
		string(ReceivingOrderStatusAll),
	}
}
