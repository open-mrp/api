package constants

// DocumentType identifies a kind of document the platform generates for an account, as a PDF, a printout or
// the email that carries it.
type DocumentType string

const (
	// DocumentTypeInvoice indicates an invoice.
	DocumentTypeInvoice DocumentType = "invoice"
	// DocumentTypeOrderAcknowledgement indicates a sales order acknowledgement.
	DocumentTypeOrderAcknowledgement DocumentType = "order_acknowledgement"
	// DocumentTypePurchaseOrder indicates a purchase order sent to a supplier.
	DocumentTypePurchaseOrder DocumentType = "purchase_order"
	// DocumentTypePackList indicates a shipment's pack list.
	DocumentTypePackList DocumentType = "pack_list"
	// DocumentTypePickTicket indicates a pick ticket.
	DocumentTypePickTicket DocumentType = "pick_ticket"
	// DocumentTypeBatchTraveler indicates the routing sheet that accompanies a batch through production.
	DocumentTypeBatchTraveler DocumentType = "batch_traveler"
	// DocumentTypePriceList indicates a customer price list export.
	DocumentTypePriceList DocumentType = "price_list"
	// DocumentTypeTransactionReceipt indicates a payment receipt.
	DocumentTypeTransactionReceipt DocumentType = "transaction_receipt"
)

func (t DocumentType) IsValid() bool {
	switch t {
	case DocumentTypeInvoice, DocumentTypeOrderAcknowledgement, DocumentTypePurchaseOrder, DocumentTypePackList,
		DocumentTypePickTicket, DocumentTypeBatchTraveler, DocumentTypePriceList, DocumentTypeTransactionReceipt:
		return true
	default:
		return false
	}
}

func (t DocumentType) EnumValues() []string {
	return []string{
		string(DocumentTypeInvoice), string(DocumentTypeOrderAcknowledgement), string(DocumentTypePurchaseOrder),
		string(DocumentTypePackList), string(DocumentTypePickTicket), string(DocumentTypeBatchTraveler),
		string(DocumentTypePriceList), string(DocumentTypeTransactionReceipt),
	}
}

func (t *DocumentType) StringPtr() *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

// DocumentTypes lists every document type in declaration order.
func DocumentTypes() []DocumentType {
	values := DocumentType("").EnumValues()
	types := make([]DocumentType, len(values))
	for i, v := range values {
		types[i] = DocumentType(v)
	}
	return types
}
