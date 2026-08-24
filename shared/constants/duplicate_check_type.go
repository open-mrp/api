package constants

// DuplicateCheckType names which kind of record number a duplicate check looks at.
type DuplicateCheckType string

const (
	// DuplicateCheckTypeInvoiceNumber checks invoice numbers.
	DuplicateCheckTypeInvoiceNumber DuplicateCheckType = "invoice_number"
	// DuplicateCheckTypeOrderNumber checks sales order numbers.
	DuplicateCheckTypeOrderNumber DuplicateCheckType = "order_number"
	// DuplicateCheckTypeCustomerPONumber checks customer PO numbers on sales orders, scoped to a customer.
	DuplicateCheckTypeCustomerPONumber DuplicateCheckType = "customer_po_number"
)

func (t DuplicateCheckType) IsValid() bool {
	switch t {
	case DuplicateCheckTypeInvoiceNumber, DuplicateCheckTypeOrderNumber, DuplicateCheckTypeCustomerPONumber:
		return true
	default:
		return false
	}
}

func (t DuplicateCheckType) EnumValues() []string {
	return []string{
		string(DuplicateCheckTypeInvoiceNumber),
		string(DuplicateCheckTypeOrderNumber),
		string(DuplicateCheckTypeCustomerPONumber),
	}
}
