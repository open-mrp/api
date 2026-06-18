package constants

// InvoicePaymentStatus represents the payment state of an invoice. Modeled as an enum (rather than separate is_paid_in_full / is_over_paid booleans) so new states can be added later without a breaking change.
type InvoicePaymentStatus string

const (
	// InvoicePaymentStatusUnpaid indicates no payment has been received.
	InvoicePaymentStatusUnpaid InvoicePaymentStatus = "unpaid"
	// InvoicePaymentStatusPartiallyPaid indicates the invoice is partially paid.
	InvoicePaymentStatusPartiallyPaid InvoicePaymentStatus = "partially_paid"
	// InvoicePaymentStatusPaid indicates the invoice is paid in full.
	InvoicePaymentStatusPaid InvoicePaymentStatus = "paid"
	// InvoicePaymentStatusOverpaid indicates the invoice has been overpaid.
	InvoicePaymentStatusOverpaid InvoicePaymentStatus = "overpaid"
)

func (m InvoicePaymentStatus) IsValid() bool {
	switch m {
	case InvoicePaymentStatusUnpaid, InvoicePaymentStatusPartiallyPaid, InvoicePaymentStatusPaid, InvoicePaymentStatusOverpaid:
		return true
	default:
		return false
	}
}

func (m InvoicePaymentStatus) EnumValues() []string {
	return []string{
		string(InvoicePaymentStatusUnpaid),
		string(InvoicePaymentStatusPartiallyPaid),
		string(InvoicePaymentStatusPaid),
		string(InvoicePaymentStatusOverpaid),
	}
}
