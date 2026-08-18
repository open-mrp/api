package constants

// EmailRecordType represents the type of record to email to its configured recipients.
type EmailRecordType string

const (
	// EmailRecordTypeInvoice emails an invoice to the contacts on its sales order set to receive invoice emails.
	EmailRecordTypeInvoice EmailRecordType = "invoice"
	// EmailRecordTypeSalesOrder sends a sales order's acknowledgement to its acknowledgement recipients.
	EmailRecordTypeSalesOrder EmailRecordType = "sales_order"
	// EmailRecordTypePurchaseOrder sends a purchase order's submission to its submission recipients.
	EmailRecordTypePurchaseOrder EmailRecordType = "purchase_order"
)

func (t EmailRecordType) IsValid() bool {
	switch t {
	case EmailRecordTypeInvoice, EmailRecordTypeSalesOrder, EmailRecordTypePurchaseOrder:
		return true
	default:
		return false
	}
}

func (t EmailRecordType) EnumValues() []string {
	return []string{string(EmailRecordTypeInvoice), string(EmailRecordTypeSalesOrder), string(EmailRecordTypePurchaseOrder)}
}

func (t *EmailRecordType) StringPtr() *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}
