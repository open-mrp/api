package constants

// ShipmentStatus represents the status of a shipment.
type ShipmentStatus string

const (
	// ShipmentStatusPacked indicates that the shipment has been packed.
	ShipmentStatusPacked ShipmentStatus = "packed"
	// ShipmentStatusShipped indicates that the shipment has been shipped.
	ShipmentStatusShipped ShipmentStatus = "shipped"
)

func (s ShipmentStatus) IsValid() bool {
	switch s {
	case ShipmentStatusPacked, ShipmentStatusShipped:
		return true
	default:
		return false
	}
}

func (s ShipmentStatus) EnumValues() []string {
	return []string{string(ShipmentStatusPacked), string(ShipmentStatusShipped)}
}
