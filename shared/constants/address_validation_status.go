package constants

// AddressValidationStatus represents the result of validating an address.
type AddressValidationStatus string

const (
	// AddressValidationStatusValid indicates the address was validated successfully.
	AddressValidationStatusValid AddressValidationStatus = "valid"
	// AddressValidationStatusInvalid indicates the address could not be validated.
	AddressValidationStatusInvalid AddressValidationStatus = "invalid"
)

func (m AddressValidationStatus) IsValid() bool {
	switch m {
	case AddressValidationStatusValid, AddressValidationStatusInvalid:
		return true
	default:
		return false
	}
}

func (m AddressValidationStatus) EnumValues() []string {
	return []string{
		string(AddressValidationStatusValid),
		string(AddressValidationStatusInvalid),
	}
}
