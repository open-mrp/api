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

// InvoiceListStatus filters an invoice list by payment state. It is deliberately not InvoicePaymentStatus: it carries an `all` sentinel that is not a state an invoice can be in, and it buckets partially-paid invoices under `unpaid` rather than exposing them separately.
type InvoiceListStatus string

const (
	// InvoiceListStatusAll applies no payment-state filtering, the same as omitting the parameter.
	InvoiceListStatusAll InvoiceListStatus = "all"
	// InvoiceListStatusPaid returns only invoices marked paid in full.
	InvoiceListStatusPaid InvoiceListStatus = "paid"
	// InvoiceListStatusUnpaid returns only invoices that are neither paid in full nor overpaid, including invoices carrying partial payments.
	InvoiceListStatusUnpaid InvoiceListStatus = "unpaid"
	// InvoiceListStatusOverpaid returns only invoices whose applied payments exceed the invoiced amount.
	InvoiceListStatusOverpaid InvoiceListStatus = "overpaid"
)

func (s InvoiceListStatus) IsValid() bool {
	switch s {
	case InvoiceListStatusAll, InvoiceListStatusPaid, InvoiceListStatusUnpaid, InvoiceListStatusOverpaid:
		return true
	default:
		return false
	}
}

func (s InvoiceListStatus) EnumValues() []string {
	return []string{
		string(InvoiceListStatusAll),
		string(InvoiceListStatusPaid),
		string(InvoiceListStatusUnpaid),
		string(InvoiceListStatusOverpaid),
	}
}

func (s *InvoiceListStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
