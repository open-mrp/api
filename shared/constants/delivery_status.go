package constants

// DeliveryStatus represents the status of a delivery.
type DeliveryStatus string

const (
	// DeliveryStatusAccepted indicates the delivery has been accepted.
	DeliveryStatusAccepted DeliveryStatus = "accepted"
	// DeliveryStatusRejected indicates the delivery has been rejected.
	DeliveryStatusRejected DeliveryStatus = "rejected"
)

func (m DeliveryStatus) IsValid() bool {
	switch m {
	case DeliveryStatusAccepted, DeliveryStatusRejected:
		return true
	default:
		return false
	}
}

func (m DeliveryStatus) EnumValues() []string {
	return []string{
		string(DeliveryStatusAccepted),
		string(DeliveryStatusRejected),
	}
}
