package constants

import "slices"

// RecordType is the kind of record a Record stub points at. Each value is that resource's object type, so a stub's `type` and the full resource's `object` always match.
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
	// RecordTypeBill indicates that the record is a bill.
	RecordTypeBill RecordType = "bill"
	// RecordTypeCreditNote indicates that the record is a credit note.
	RecordTypeCreditNote RecordType = "credit_note"
	// RecordTypeSupplierCredit indicates that the record is a supplier credit.
	RecordTypeSupplierCredit RecordType = "supplier_credit"
	// RecordTypePayment indicates that the record is a payment or refund.
	RecordTypePayment RecordType = "payment"
	// RecordTypeApplication indicates that the record is an application of a credit to a debit.
	RecordTypeApplication RecordType = "application"
	// RecordTypeJob indicates that the record is a manufacturing job.
	RecordTypeJob RecordType = "job"
	// RecordTypeWorkOrder indicates that the record is a work order.
	RecordTypeWorkOrder RecordType = "work_order"
	// RecordTypeBatch indicates that the record is a batch.
	RecordTypeBatch RecordType = "batch"
	// RecordTypeBatchScan indicates that the record is a batch scan.
	RecordTypeBatchScan RecordType = "batch_scan"
	// RecordTypeInventoryMovement indicates that the record is an inventory movement.
	RecordTypeInventoryMovement RecordType = "inventory_movement"
	// RecordTypeLot indicates that the record is a lot.
	RecordTypeLot RecordType = "lot"
	// RecordTypeCustomer indicates that the record is a customer.
	RecordTypeCustomer RecordType = "customer"
	// RecordTypeSupplier indicates that the record is a supplier.
	RecordTypeSupplier RecordType = "supplier"
	// RecordTypeItem indicates that the record is an item.
	RecordTypeItem RecordType = "item"
	// RecordTypeManufacturingMethod indicates that the record is a manufacturing method.
	RecordTypeManufacturingMethod RecordType = "manufacturing_method"
	// RecordTypePriceList indicates that the record is a price list.
	RecordTypePriceList RecordType = "price_list"
	// RecordTypeAccountUser indicates that the record is an account user.
	RecordTypeAccountUser RecordType = "account_user"
	// RecordTypeAPIKey indicates that the record is an API key.
	RecordTypeAPIKey RecordType = "api_key"
	// RecordTypeAgent indicates that the record is an agent.
	RecordTypeAgent RecordType = "agent"
	// RecordTypeDevice indicates that the record is a device.
	RecordTypeDevice RecordType = "device"
	// RecordTypeConversation indicates that the record is a conversation.
	RecordTypeConversation RecordType = "conversation"
	// RecordTypeWorkCenter indicates that the record is a work center.
	RecordTypeWorkCenter RecordType = "work_center"
	// RecordTypeMachine indicates that the record is a machine.
	RecordTypeMachine RecordType = "machine"
	// RecordTypeScanningStation indicates that the record is a scanning station.
	RecordTypeScanningStation RecordType = "scanning_station"
	// RecordTypeItemCategory indicates that the record is an item category.
	RecordTypeItemCategory RecordType = "item_category"
	// RecordTypeProductLine indicates that the record is a product line.
	RecordTypeProductLine RecordType = "product_line"
)

var recordTypes = []RecordType{
	RecordTypeSalesOrder, RecordTypePurchaseOrder, RecordTypeReceivingOrder, RecordTypePick, RecordTypeShipment,
	RecordTypeDelivery, RecordTypeProductionRun, RecordTypeInvoice, RecordTypeTransaction, RecordTypeSettlement,
	RecordTypeBill, RecordTypeCreditNote, RecordTypeSupplierCredit, RecordTypePayment, RecordTypeApplication,
	RecordTypeJob, RecordTypeWorkOrder, RecordTypeBatch, RecordTypeBatchScan, RecordTypeInventoryMovement, RecordTypeLot,
	RecordTypeCustomer, RecordTypeSupplier, RecordTypeItem, RecordTypeManufacturingMethod, RecordTypePriceList,
	RecordTypeAccountUser, RecordTypeAPIKey, RecordTypeAgent, RecordTypeDevice, RecordTypeConversation,
	RecordTypeWorkCenter, RecordTypeMachine, RecordTypeScanningStation, RecordTypeItemCategory, RecordTypeProductLine,
}

func (m RecordType) IsValid() bool {
	return slices.Contains(recordTypes, m)
}

func (m RecordType) EnumValues() []string {
	values := make([]string, len(recordTypes))
	for i, t := range recordTypes {
		values[i] = string(t)
	}
	return values
}

func (m *RecordType) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

// ObjectType is the object type of the resource the record type points at.
func (m RecordType) ObjectType() ObjectType {
	return ObjectType(m)
}
