package constants

// RecipientDocument is a kind of document a customer's or supplier's contact can receive by email.
type RecipientDocument string

const (
	// RecipientDocumentAcknowledgment is the acknowledgment sent when an order is issued.
	RecipientDocumentAcknowledgment RecipientDocument = "acknowledgment"
	// RecipientDocumentInvoice is an invoice.
	RecipientDocumentInvoice RecipientDocument = "invoice"
	// RecipientDocumentShipment is a shipment notice, with tracking.
	RecipientDocumentShipment RecipientDocument = "shipment"
	// RecipientDocumentStatement is an account statement.
	RecipientDocumentStatement RecipientDocument = "statement"
)

func (d RecipientDocument) IsValid() bool {
	switch d {
	case RecipientDocumentAcknowledgment, RecipientDocumentInvoice, RecipientDocumentShipment, RecipientDocumentStatement:
		return true
	}
	return false
}

func (d RecipientDocument) EnumValues() []string {
	return []string{
		string(RecipientDocumentAcknowledgment), string(RecipientDocumentInvoice),
		string(RecipientDocumentShipment), string(RecipientDocumentStatement),
	}
}

func (d *RecipientDocument) StringPtr() *string {
	if d == nil {
		return nil
	}
	s := string(*d)
	return &s
}
