package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

// A pack-list document assembled for a shipment: the shipment's packed line items and shipping cases, the parent order's header, parties, and terms, and any order lines still back-ordered.
//
// The document is generated on demand for printing and is a point-in-time snapshot of the shipment and its order; it is not persisted and cannot be retrieved again by ID.
type PackList struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list"`
	// Selling account's display name.
	AccountName string `json:"account_name" validate:"required"`
	// Presigned download URL for the selling account's logo.
	//
	// The URL expires one hour after it is generated, so render it promptly rather than caching it. Logo lookup is best effort: if the account has no logo or it cannot be resolved, the rest of the document is still returned.
	AccountLogoURL *string `json:"account_logo_url"`
	// Parent sales order number.
	SalesOrderNumber string `json:"sales_order_number" validate:"required"`
	// Customer's purchase order number.
	CustomerPO *string `json:"customer_po"`
	// Shipment number.
	ShipmentNumber string `json:"shipment_number" validate:"required"`
	// When the shipment was dispatched.
	ShippedAt *time.Time `json:"shipped_at"`
	// The party the order is billed to, as recorded on the parent sales order.
	BillTo *PackListParty `json:"bill_to"`
	// The party the goods are shipped to, as recorded on the parent sales order.
	ShipTo *PackListParty `json:"ship_to"`
	// Additional contact lines shown under the billing party: the sales order contacts set to receive invoice emails, followed by the billing contact phone.
	ContactInformation []string `json:"contact_information"`
	// Name of the carrier moving the shipment.
	Carrier *string `json:"carrier"`
	// Name of the carrier service level used for the shipment, such as `Ground`.
	CarrierOption *string `json:"carrier_option"`
	// Name of the parent order's priority.
	Priority *string `json:"priority"`
	// Name of the parent order's payment term.
	PaymentTerm *string `json:"payment_term"`
	// Name of the sales representative on the parent order.
	SalesRep *string `json:"sales_rep"`
	// Shipping cases on the shipment, ordered by case number.
	ShippingCases *List[PackListCase] `json:"shipping_cases" validate:"required"`
	// Line items packed into this shipment, ordered by the order line's line number.
	LineItems *List[PackListLineItem] `json:"line_items" validate:"required"`
	// Order lines that still have quantity outstanding once everything packed so far is accounted for.
	//
	// Only physical sale lines appear here; charge and adjustment lines such as freight, tax, and credits are excluded. The quantities span the whole order, not just this shipment.
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
	// Line number of the sales order line this shipment line was packed from.
	LineItemNumber *int32 `json:"line_item_number"`
	// Product SKU.
	SKU string `json:"sku"`
	// Product description.
	Description string `json:"description"`
	// Quantity packed into this shipment.
	Quantity string `json:"quantity" format:"decimal"`
	// Name of the unit the quantity is measured in.
	Unit string `json:"unit"`
}

// An order line that still has quantity outstanding once everything packed so far is accounted for.
type PackListBackOrder struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pack_list_back_order"`
	// The sales order line's line number.
	LineItemNumber *int32 `json:"line_item_number"`
	// Product SKU.
	SKU string `json:"sku"`
	// Product description.
	Description string `json:"description"`
	// Quantity ordered on the line.
	QuantityOrdered string `json:"quantity_ordered" format:"decimal"`
	// Quantity packed for the line across every shipment on the order, not just this one.
	QuantityShipped string `json:"quantity_shipped" format:"decimal"`
	// Quantity still outstanding: the quantity ordered less the quantity already packed.
	QuantityBackOrdered string `json:"quantity_back_ordered" format:"decimal"`
	// Name of the unit the quantities are measured in.
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
	samplePackListLogoURL     = "https://assets.openmrp.example.com/acme/logo.png?signature=abc123"
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
