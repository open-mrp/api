package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShipmentID = "sh_018b3a946651bfb6572b06b2b2"
const SampleShipmentNumber = "SH-001"
const SampleShipmentLineID = "shln_0133b6c3c807bf9c73581424c7"

// Full shipment resource.
type Shipment struct {
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
	// Shipment status code.
	Status constants.ShipmentStatus `json:"status" validate:"required"`
	// Timestamp when shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Associated sales order.
	SalesOrder *SalesOrder `json:"sales_order" expandable:"true"`
	// Associated customer.
	Customer *Customer `json:"customer" expandable:"true"`
	// Carrier selection and freight billing for this shipment.
	Freight *Freight `json:"freight" expandable:"true"`
	// Shipping address.
	ShippingAddress *Address `json:"shipping_address" expandable:"true"`
	// User who shipped this shipment.
	ShippedBy *AccountUser `json:"shipped_by" expandable:"true"`
	// Associated invoice.
	Invoice *Invoice `json:"invoice" expandable:"true"`
	// Pick associated with this shipment's order.
	Pick *Pick `json:"pick" expandable:"true"`
	// Shipment lines.
	Lines *List[ShipmentLine] `json:"lines" expandable:"true"`
	// Shipping cases.
	ShippingCases *List[ShippingCaseDetail] `json:"shipping_cases" expandable:"true"`
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
	SalesOrderLine *SalesOrderLine `json:"sales_order_line" expandable:"true"`
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

var SampleShipment = &Shipment{
	ID:                   SampleShipmentID,
	Object:               constants.ObjectTypeShipment,
	Number:               SampleShipmentNumber,
	Note:                 &sampleShipmentNote,
	BillOfLading:         &sampleBillOfLading,
	MasterTrackingNumber: &sampleMasterTrackingNumber,
	Status:               constants.ShipmentStatusShipped,
	Freight:              SampleFreight,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Shipment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShipment)
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
