package constants

// PaymentTermStatus represents the status of a payment term.
type PaymentTermStatus string

const (
	// PaymentTermStatusActive indicates that the payment term is active and can be used.
	PaymentTermStatusActive PaymentTermStatus = "active"
	// PaymentTermStatusInactive indicates that the payment term is inactive and cannot be used.
	PaymentTermStatusInactive PaymentTermStatus = "inactive"
)

func (s PaymentTermStatus) IsValid() bool {
	switch s {
	case PaymentTermStatusActive, PaymentTermStatusInactive:
		return true
	default:
		return false
	}
}

func (s PaymentTermStatus) EnumValues() []string {
	return []string{string(PaymentTermStatusActive), string(PaymentTermStatusInactive)}
}
