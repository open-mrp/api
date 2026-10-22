package constants

// PortalRegistrationLookupType is the kind of number a buyer gives to find the customer they are joining.
type PortalRegistrationLookupType string

const (
	// PortalRegistrationLookupTypeCustomerNumber finds the customer by its seller-assigned customer number.
	PortalRegistrationLookupTypeCustomerNumber PortalRegistrationLookupType = "customer_number"
	// PortalRegistrationLookupTypeSalesOrderNumber finds the customer that placed the sales order with this number.
	PortalRegistrationLookupTypeSalesOrderNumber PortalRegistrationLookupType = "sales_order_number"
	// PortalRegistrationLookupTypeInvoiceNumber finds the customer billed on the invoice with this number.
	PortalRegistrationLookupTypeInvoiceNumber PortalRegistrationLookupType = "invoice_number"
)

func (t PortalRegistrationLookupType) IsValid() bool {
	switch t {
	case PortalRegistrationLookupTypeCustomerNumber,
		PortalRegistrationLookupTypeSalesOrderNumber,
		PortalRegistrationLookupTypeInvoiceNumber:
		return true
	}
	return false
}

func (t PortalRegistrationLookupType) EnumValues() []string {
	return []string{
		string(PortalRegistrationLookupTypeCustomerNumber),
		string(PortalRegistrationLookupTypeSalesOrderNumber),
		string(PortalRegistrationLookupTypeInvoiceNumber),
	}
}

func (t *PortalRegistrationLookupType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}
