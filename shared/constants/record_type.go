package constants

// RecordType represents the kind of business record referenced by a Record.
type RecordType string

const (
	// RecordTypeSalesOrder indicates that the record is a sales order.
	RecordTypeSalesOrder RecordType = "sales_order"
	// RecordTypePurchaseOrder indicates that the record is a purchase order.
	RecordTypePurchaseOrder RecordType = "purchase_order"
	// RecordTypeReceivingOrder indicates that the record is a receiving order.
	RecordTypeReceivingOrder RecordType = "receiving_order"
	// RecordTypePick indicates that the record is a pick.
	RecordTypePick RecordType = "pick"
	// RecordTypeShipment indicates that the record is a shipment.
	RecordTypeShipment RecordType = "shipment"
	// RecordTypeDelivery indicates that the record is a delivery.
	RecordTypeDelivery RecordType = "delivery"
	// RecordTypeProductionRun indicates that the record is a production run.
	RecordTypeProductionRun RecordType = "production_run"
	// RecordTypeInvoice indicates that the record is an invoice.
	RecordTypeInvoice RecordType = "invoice"
	// RecordTypeTransaction indicates that the record is a transaction.
	RecordTypeTransaction RecordType = "transaction"
	// RecordTypeSettlement indicates that the record is a settlement.
	RecordTypeSettlement RecordType = "settlement"
)

func (m RecordType) IsValid() bool {
	switch m {
	case RecordTypeSalesOrder, RecordTypePurchaseOrder, RecordTypeReceivingOrder, RecordTypePick, RecordTypeShipment, RecordTypeDelivery, RecordTypeProductionRun, RecordTypeInvoice, RecordTypeTransaction, RecordTypeSettlement:
		return true
	default:
		return false
	}
}

func (m RecordType) EnumValues() []string {
	return []string{string(RecordTypeSalesOrder), string(RecordTypePurchaseOrder), string(RecordTypeReceivingOrder), string(RecordTypePick), string(RecordTypeShipment), string(RecordTypeDelivery), string(RecordTypeProductionRun), string(RecordTypeInvoice), string(RecordTypeTransaction), string(RecordTypeSettlement)}
}

func (m *RecordType) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
