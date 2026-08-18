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

// DeliveryListStatus filters a delivery list by status. It is deliberately not DeliveryStatus: it carries an `all` sentinel that is not a status a delivery can be in.
type DeliveryListStatus string

const (
	// DeliveryListStatusAll returns deliveries of every status.
	DeliveryListStatusAll DeliveryListStatus = "all"
	// DeliveryListStatusAccepted returns only deliveries that accepted goods into inventory.
	DeliveryListStatusAccepted DeliveryListStatus = "accepted"
	// DeliveryListStatusRejected returns only deliveries where nothing was accepted into inventory.
	DeliveryListStatusRejected DeliveryListStatus = "rejected"
)

func (s DeliveryListStatus) IsValid() bool {
	switch s {
	case DeliveryListStatusAll, DeliveryListStatusAccepted, DeliveryListStatusRejected:
		return true
	default:
		return false
	}
}

func (s DeliveryListStatus) EnumValues() []string {
	return []string{
		string(DeliveryListStatusAll),
		string(DeliveryListStatusAccepted),
		string(DeliveryListStatusRejected),
	}
}

func (s *DeliveryListStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
