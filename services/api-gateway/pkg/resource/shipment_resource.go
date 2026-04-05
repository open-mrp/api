package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShipmentDetailID = "sh_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleShipmentNumber = "SH-001"
const SampleShipmentLineID = "shln_01jm4r6700f8nwq3v5hx2d9ktp"

// ShipmentStatus is a sub-resource for shipment status.
type ShipmentStatus struct {
	// The status code.
	Code string `json:"code" validate:"required"`
	// The display name of the status.
	Name string `json:"name" validate:"required"`
}

// ShipmentBilling represents carrier billing info on a shipment.
type ShipmentBilling struct {
	// The carrier billing type (e.g. "third_party").
	Type string `json:"type" validate:"required"`
	// The carrier billing account number.
	Account *string `json:"account"`
	// The billing address country.
	Country *string `json:"country"`
	// The billing address postal code.
	Zip *string `json:"zip"`
}

// ShipmentDetail represents a full shipment API resource.
type ShipmentDetail struct {
	// The unique identifier for the shipment.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment"`
	// The shipment number.
	Number string `json:"number" validate:"required"`
	// A note attached to this shipment.
	Note *string `json:"note"`
	// The bill of lading number.
	BillOfLading *string `json:"bill_of_lading"`
	// The master tracking number for this shipment.
	MasterTrackingNumber *string `json:"master_tracking_number"`
	// The shipment status.
	Status ShipmentStatus `json:"status" validate:"required"`
	// The timestamp when the shipment was shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// The sales order associated with this shipment.
	SalesOrder *SalesOrderDetail `json:"sales_order" expandable:"true"`
	// The customer associated with this shipment.
	Customer *Customer `json:"customer" expandable:"true"`
	// The carrier for this shipment.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The service level for this shipment.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// The shipping address.
	ShippingAddress *Address `json:"shipping_address" expandable:"true"`
	// The user who shipped this shipment.
	ShippedBy *AccountUser `json:"shipped_by" expandable:"true"`
	// The invoice associated with this shipment.
	Invoice *Invoice `json:"invoice" expandable:"true"`
	// The pick associated with this shipment's order.
	Pick *PickDetail `json:"pick" expandable:"true"`
	// The carrier billing information.
	Billing *ShipmentBilling `json:"billing"`
	// The shipment lines.
	Lines *List[ShipmentLine] `json:"lines" expandable:"true"`
	// The shipping cases.
	ShippingCases *List[ShippingCaseDetail] `json:"shipping_cases" expandable:"true"`
	// The timestamp when the shipment was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the shipment was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ShipmentSummary is the list view of a shipment.
type ShipmentSummary struct {
	// The unique identifier for the shipment.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_summary"`
	// The shipment number.
	Number string `json:"number" validate:"required"`
	// A note attached to this shipment.
	Note *string `json:"note"`
	// The bill of lading number.
	BillOfLading *string `json:"bill_of_lading"`
	// The master tracking number for this shipment.
	MasterTrackingNumber *string `json:"master_tracking_number"`
	// The shipment status.
	Status ShipmentStatus `json:"status" validate:"required"`
	// The timestamp when the shipment was shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// The sales order associated with this shipment.
	SalesOrder *SalesOrderDetail `json:"sales_order" expandable:"true"`
	// The customer associated with this shipment.
	Customer *Customer `json:"customer" expandable:"true"`
	// The carrier for this shipment.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The service level for this shipment.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// The timestamp when the shipment was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the shipment was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ShipmentLine represents a shipment line API resource.
type ShipmentLine struct {
	// The unique identifier for the shipment line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_line"`
	// The sales order line associated with this shipment line.
	SalesOrderLine *SalesOrderLineDetail `json:"sales_order_line" expandable:"true"`
	// The quantity shipped.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// The timestamp when the shipment line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the shipment line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// ShippingCaseDetail represents a shipping case in shipment detail views.
type ShippingCaseDetail struct {
	// The unique identifier for the shipping case.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// The human-readable case number.
	Number string `json:"number" validate:"required"`
	// The Serial Shipping Container Code.
	SSCC *string `json:"sscc"`
	// The carrier tracking number for this case.
	TrackingNumber *string `json:"tracking_number"`
	// The Shippo transaction ID for this case.
	ShippoTransactionID *string `json:"shippo_transaction_id"`
	// The URL for the shipping label.
	ShippingLabelURL *string `json:"shipping_label_url"`
	// The timestamp when the case was shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// The freight amount for this case.
	FreightAmount *Quantity `json:"freight_amount"`
	// The freight weight for this case.
	FreightWeight *Quantity `json:"freight_weight"`
	// The carrier for this case.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The timestamp when the shipping case was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the shipping case was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// RateShopResult represents the result of rate shopping.
type RateShopResult struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate_shop_result"`
	// The available rate options.
	Options *List[RateShopOption] `json:"options" validate:"required"`
	// The exemption type, if applicable.
	ExemptionType *string `json:"exemption_type"`
	// The flat rate amount, if applicable.
	FlatRate *float64 `json:"flat_rate"`
}

// RateShopOption represents a single rate shop option.
type RateShopOption struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate_shop_option"`
	// The carrier for this option.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The service level for this option.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// The rate amount.
	Rate float64 `json:"rate" validate:"required"`
	// The estimated delivery days.
	EstimatedDays *int32 `json:"estimated_days"`
}

// EstimateRateResult represents the result of estimating a rate.
type EstimateRateResult struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=estimate_rate_result"`
	// The estimated rate amount.
	Rate float64 `json:"rate" validate:"required"`
}

// --- Sample Data ---

var sampleShipmentNote = "Handle with care"
var sampleBillOfLading = "BOL-12345"
var sampleMasterTrackingNumber = "1Z999AA10123456784"
var sampleShipmentBillingType = "third_party"
var sampleShipmentBillingAccount = "123456"
var sampleShipmentBillingCountry = "US"
var sampleShipmentBillingZip = "90210"

var SampleShipmentDetail = &ShipmentDetail{
	ID:                   SampleShipmentDetailID,
	Object:               constants.ObjectTypeShipment,
	Number:               SampleShipmentNumber,
	Note:                 &sampleShipmentNote,
	BillOfLading:         &sampleBillOfLading,
	MasterTrackingNumber: &sampleMasterTrackingNumber,
	Status: ShipmentStatus{
		Code: "shipped",
		Name: "Shipped",
	},
	Billing: &ShipmentBilling{
		Type:    sampleShipmentBillingType,
		Account: &sampleShipmentBillingAccount,
		Country: &sampleShipmentBillingCountry,
		Zip:     &sampleShipmentBillingZip,
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShipmentDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShipmentDetail)
}

var SampleShipmentSummary = &ShipmentSummary{
	ID:                   SampleShipmentDetailID,
	Object:               constants.ObjectTypeShipmentSummary,
	Number:               SampleShipmentNumber,
	Note:                 &sampleShipmentNote,
	BillOfLading:         &sampleBillOfLading,
	MasterTrackingNumber: &sampleMasterTrackingNumber,
	Status: ShipmentStatus{
		Code: "shipped",
		Name: "Shipped",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShipmentSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShipmentSummary)
}

var SampleShipmentLine = &ShipmentLine{
	ID:     SampleShipmentLineID,
	Object: constants.ObjectTypeShipmentLine,
	Quantity: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "10.000000000000000000000000000000",
		DisplayValue: "10 kg",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShipmentLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShipmentLine)
}
