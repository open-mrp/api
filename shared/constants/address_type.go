package constants

// AddressType represents the public category of an address.
type AddressType string

const (
	// AddressTypeStandard indicates a normal address.
	AddressTypeStandard AddressType = "standard"
	// AddressTypeDropShip indicates a drop ship address.
	AddressTypeDropShip AddressType = "drop_ship"
)

func (m AddressType) IsValid() bool {
	switch m {
	case AddressTypeStandard, AddressTypeDropShip:
		return true
	default:
		return false
	}
}

func (m AddressType) EnumValues() []string {
	return []string{
		string(AddressTypeStandard),
		string(AddressTypeDropShip),
	}
}
