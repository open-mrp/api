package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShipmentDetailID = "sh_018b3a946651bfb6572b06b2b2"
const SampleShipmentNumber = "SH-001"
const SampleShipmentLineID = "shln_0133b6c3c807bf9c73581424c7"

// Shipment status sub-resource.
type ShipmentStatus struct {
	// Status code.
	Code string `json:"code" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// Carrier billing info on a shipment.
type ShipmentBilling struct {
	// Carrier billing type (e.g. "third_party").
	Type string `json:"type" validate:"required"`
	// Carrier billing account number.
	Account *string `json:"account"`
	// Billing address country.
	Country *string `json:"country"`
	// Billing address postal code.
	Zip *string `json:"zip"`
}

// Full shipment resource.
type ShipmentDetail struct {
	// Shipment ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment"`
	// Shipment number.
	Number string `json:"number" validate:"required"`
	// Note attached to this shipment.
	Note *string `json:"note"`
	// Bill of lading number.
	BillOfLading *string `json:"bill_of_lading"`
	// Master tracking number.
	MasterTrackingNumber *string `json:"master_tracking_number"`
	// Shipment status.
	Status ShipmentStatus `json:"status" validate:"required"`
	// Timestamp when shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Associated sales order.
	SalesOrder *SalesOrderDetail `json:"sales_order" expandable:"true"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Shipping address.
	ShippingAddress *Address `json:"shipping_address" expandable:"true"`
	// User who shipped this shipment.
	ShippedBy *AccountUser `json:"shipped_by" expandable:"true"`
	// Associated invoice.
	Invoice *Invoice `json:"invoice" expandable:"true"`
	// Pick associated with this shipment's order.
	Pick *PickDetail `json:"pick" expandable:"true"`
	// Carrier billing information.
	Billing *ShipmentBilling `json:"billing"`
	// Shipment lines.
	Lines *List[ShipmentLine] `json:"lines" expandable:"true"`
	// Shipping cases.
	ShippingCases *List[ShippingCaseDetail] `json:"shipping_cases" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Shipment list view resource.
type ShipmentSummary struct {
	// Shipment ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_summary"`
	// Shipment number.
	Number string `json:"number" validate:"required"`
	// Note attached to this shipment.
	Note *string `json:"note"`
	// Bill of lading number.
	BillOfLading *string `json:"bill_of_lading"`
	// Master tracking number.
	MasterTrackingNumber *string `json:"master_tracking_number"`
	// Shipment status.
	Status ShipmentStatus `json:"status" validate:"required"`
	// Timestamp when shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Associated sales order.
	SalesOrder *SalesOrderDetail `json:"sales_order" expandable:"true"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Shipment line resource.
type ShipmentLine struct {
	// Shipment line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_line"`
	// Associated sales order line.
	SalesOrderLine *SalesOrderLineDetail `json:"sales_order_line" expandable:"true"`
	// Quantity shipped.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Shipping case resource in shipment detail views.
type ShippingCaseDetail struct {
	// Shipping case ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// Human-readable case number.
	Number string `json:"number" validate:"required"`
	// Serial Shipping Container Code.
	SSCC *string `json:"sscc"`
	// Carrier tracking number.
	TrackingNumber *string `json:"tracking_number"`
	// Shippo transaction ID.
	ShippoTransactionID *string `json:"shippo_transaction_id"`
	// Shipping label URL.
	ShippingLabelURL *string `json:"shipping_label_url"`
	// Timestamp when shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Freight amount.
	FreightAmount *Quantity `json:"freight_amount"`
	// Freight weight.
	FreightWeight *Quantity `json:"freight_weight"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Result of rate shopping.
type RateShopResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate_shop_result"`
	// Available rate options.
	Options *List[RateShopOption] `json:"options" validate:"required"`
	// Exemption type, if applicable.
	ExemptionType *string `json:"exemption_type"`
	// Flat rate amount, if applicable.
	FlatRate *float64 `json:"flat_rate"`
}

// Rate shop option.
type RateShopOption struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate_shop_option"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Rate amount.
	Rate float64 `json:"rate" validate:"required"`
	// Estimated delivery days.
	EstimatedDays *int32 `json:"estimated_days"`
}

// Result of estimating a shipping rate.
type EstimateRateResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=estimate_rate_result"`
	// Estimated rate amount.
	Rate float64 `json:"rate" validate:"required"`
}

var SampleEstimateRateResult = &EstimateRateResult{
	Object: constants.ObjectTypeEstimateRateResult,
	Rate:   42.5,
}

func (*EstimateRateResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEstimateRateResult)
}

var sampleRateShopEstimatedDays int32 = 3

var SampleRateShopOption = &RateShopOption{
	Object:        constants.ObjectTypeRateShopOption,
	Carrier:       SampleCarrier,
	ServiceLevel:  SampleServiceLevel,
	Rate:          12.34,
	EstimatedDays: &sampleRateShopEstimatedDays,
}

var SampleRateShopResult = &RateShopResult{
	Object:  constants.ObjectTypeRateShopResult,
	Options: NewList([]RateShopOption{*SampleRateShopOption}, PageInfo{}),
}

func (*RateShopResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRateShopResult)
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

func ExpandableShipmentStub(id, number string, ts time.Time) *ShipmentDetail {
	if id == "" {
		id = SampleShipmentDetailID
	}
	if number == "" {
		number = SampleShipmentNumber
	}
	if ts.IsZero() {
		ts = time.Unix(0, 0).UTC()
	}
	return &ShipmentDetail{
		ID:     id,
		Object: constants.ObjectTypeShipment,
		Number: number,
		Status: ShipmentStatus{
			Code: string(constants.ShipmentStatusPacked),
			Name: "Packed",
		},
		CreatedAt: ts,
		UpdatedAt: ts,
	}
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
