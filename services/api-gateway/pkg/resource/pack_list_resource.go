package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A pack-list document assembled for a shipment: the shipment's packed line items and shipping cases, the parent order's header, parties, and terms, and any order lines still back-ordered. It is generated on demand for printing and is a point-in-time snapshot; it is not persisted.
type PackList struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list"`
	// Selling account's display name.
	AccountName string `json:"account_name" validate:"required"`
	// Presigned download URL for the selling account's logo. Expires one hour after it is generated, so render it promptly rather than caching it.
	AccountLogoURL *string `json:"account_logo_url"`
	// Parent sales order number.
	SalesOrderNumber string `json:"sales_order_number" validate:"required"`
	// Customer's purchase order number.
	CustomerPO *string `json:"customer_po"`
	// Shipment number.
	ShipmentNumber string `json:"shipment_number" validate:"required"`
	// When the shipment was dispatched.
	ShippedAt *time.Time `json:"shipped_at"`
	// Billing party.
	BillTo *PackListParty `json:"bill_to"`
	// Shipping party.
	ShipTo *PackListParty `json:"ship_to"`
	// Additional contact lines shown under the billing party: the order's email recipients followed by the billing contact phone.
	ContactInformation []string `json:"contact_information"`
	// Carrier name.
	Carrier *string `json:"carrier"`
	// Service level name.
	CarrierOption *string `json:"carrier_option"`
	// Order priority name.
	Priority *string `json:"priority"`
	// Payment term name.
	PaymentTerm *string `json:"payment_term"`
	// Sales representative name.
	SalesRep *string `json:"sales_rep"`
	// Shipping cases on the shipment.
	ShippingCases *List[PackListCase] `json:"shipping_cases" validate:"required"`
	// Packed line items on the shipment.
	LineItems *List[PackListLineItem] `json:"line_items" validate:"required"`
	// Order lines with quantity still back-ordered.
	BackOrders *List[PackListBackOrder] `json:"back_orders" validate:"required"`
}

// A bill-to or ship-to party shown on a pack list.
type PackListParty struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list_party"`
	// Party name.
	Name string `json:"name"`
	// First line of the street address.
	StreetLine1 *string `json:"street_line_1"`
	// Second line of the street address.
	StreetLine2 *string `json:"street_line_2"`
	// City or locality.
	Locality *string `json:"locality"`
	// State or administrative area.
	State *string `json:"state"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code"`
	// Two-letter country code.
	Country *string `json:"country"`
	// Phone number.
	Phone *string `json:"phone"`
	// Email address.
	Email *string `json:"email"`
}

// A packed line item on a pack list.
type PackListLineItem struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list_line_item"`
	// The order line's line number.
	LineItemNumber *int32 `json:"line_item_number"`
	// Product SKU.
	SKU string `json:"sku"`
	// Product description.
	Description string `json:"description"`
	// Quantity packed into this shipment.
	Quantity string `json:"quantity" format:"decimal"`
	// Unit name.
	Unit string `json:"unit"`
}

// An order line with quantity still back-ordered after this shipment.
type PackListBackOrder struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list_back_order"`
	// The order line's line number.
	LineItemNumber *int32 `json:"line_item_number"`
	// Product SKU.
	SKU string `json:"sku"`
	// Product description.
	Description string `json:"description"`
	// Quantity ordered.
	QuantityOrdered string `json:"quantity_ordered" format:"decimal"`
	// Quantity shipped so far.
	QuantityShipped string `json:"quantity_shipped" format:"decimal"`
	// Quantity still back-ordered.
	QuantityBackOrdered string `json:"quantity_back_ordered" format:"decimal"`
	// Unit name.
	Unit string `json:"unit"`
}

// A shipping case on a pack list.
type PackListCase struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list_case"`
	// Case number.
	Number string `json:"number"`
	// Case weight.
	Weight string `json:"weight" format:"decimal"`
	// Abbreviation of the weight unit.
	WeightUnit string `json:"weight_unit"`
	// Carrier tracking number for the case.
	TrackingNumber *string `json:"tracking_number"`
	// Carrier name for the case.
	Carrier *string `json:"carrier"`
}

// --- Sample Data ---

var (
	samplePackListLogoURL     = "https://assets.augno.example.com/acme/logo.png?signature=abc123"
	samplePackListCustomerPO  = "PO-99887"
	samplePackListCarrier     = "UPS"
	samplePackListOption      = "Ground"
	samplePackListPriority    = "Standard"
	samplePackListTerm        = "Net 30"
	samplePackListSalesRep    = "Jordan Vega"
	samplePackListTracking    = "1Z999AA10123456784"
	samplePackListLineNumber  = int32(1)
	samplePackListBillStreet1 = "4200 Industrial Pkwy"
	samplePackListBillCity    = "Columbus"
	samplePackListBillState   = "OH"
	samplePackListBillZip     = "43204"
	samplePackListBillCountry = "US"
	samplePackListBillPhone   = "+1-614-555-0142"
)

var SamplePackListParty = &PackListParty{
	Object:      constants.ObjectTypePackListParty,
	Name:        "Acme Manufacturing",
	StreetLine1: &samplePackListBillStreet1,
	Locality:    &samplePackListBillCity,
	State:       &samplePackListBillState,
	PostalCode:  &samplePackListBillZip,
	Country:     &samplePackListBillCountry,
	Phone:       &samplePackListBillPhone,
}

var SamplePackListLineItem = &PackListLineItem{
	Object:         constants.ObjectTypePackListLineItem,
	LineItemNumber: &samplePackListLineNumber,
	SKU:            "WIDGET-BLUE",
	Description:    "Blue Widget, 10mm",
	Quantity:       "24",
	Unit:           "each",
}

var SamplePackListBackOrder = &PackListBackOrder{
	Object:              constants.ObjectTypePackListBackOrder,
	LineItemNumber:      &samplePackListLineNumber,
	SKU:                 "WIDGET-RED",
	Description:         "Red Widget, 10mm",
	QuantityOrdered:     "50",
	QuantityShipped:     "20",
	QuantityBackOrdered: "30",
	Unit:                "each",
}

var SamplePackListCase = &PackListCase{
	Object:         constants.ObjectTypePackListCase,
	Number:         "CASE-001",
	Weight:         "12.5",
	WeightUnit:     "lb",
	TrackingNumber: &samplePackListTracking,
	Carrier:        &samplePackListCarrier,
}

var samplePackListShippedAt = timeutil.TimestampToTime(sampleUpdatedAtTimestamp)

var SamplePackList = &PackList{
	Object:             constants.ObjectTypePackList,
	AccountName:        "Acme Manufacturing",
	AccountLogoURL:     &samplePackListLogoURL,
	SalesOrderNumber:   "000123",
	CustomerPO:         &samplePackListCustomerPO,
	ShipmentNumber:     SampleShipmentNumber,
	ShippedAt:          &samplePackListShippedAt,
	BillTo:             SamplePackListParty,
	ShipTo:             SamplePackListParty,
	ContactInformation: []string{"receiving@acme.example.com", "+1-614-555-0142"},
	Carrier:            &samplePackListCarrier,
	CarrierOption:      &samplePackListOption,
	Priority:           &samplePackListPriority,
	PaymentTerm:        &samplePackListTerm,
	SalesRep:           &samplePackListSalesRep,
	ShippingCases:      NewList([]PackListCase{*SamplePackListCase}, PageInfo{}),
	LineItems:          NewList([]PackListLineItem{*SamplePackListLineItem}, PageInfo{}),
	BackOrders:         NewList([]PackListBackOrder{*SamplePackListBackOrder}, PageInfo{}),
}

func (*PackList) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePackList)
}
