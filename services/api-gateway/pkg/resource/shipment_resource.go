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

// A shipment of packed goods fulfilling a sales order, from packing through dispatch.
type Shipment struct {
	// Shipment ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment"`
	// Human-readable shipment number.
	Number string `json:"number" validate:"required"`
	// Note attached to this shipment.
	Note *string `json:"note"`
	// Bill of lading number.
	BillOfLading *string `json:"bill_of_lading"`
	// Carrier master tracking number covering the shipment as a whole.
	//
	// Individual shipping cases carry their own per-case tracking numbers.
	MasterTrackingNumber *string `json:"master_tracking_number"`
	// Current status of the shipment.
	//
	// - `packed`: the shipment has been packed but not yet dispatched.
	// - `shipped`: the shipment has left the facility (`shipped_at` is set).
	Status constants.ShipmentStatus `json:"status" validate:"required"`
	// Timestamp when the shipment was shipped.
	//
	// Null until the shipment is shipped; cleared again if the shipment is voided.
	ShippedAt *time.Time `json:"shipped_at"`
	// The sales order this shipment fulfills.
	SalesOrder *SalesOrder `json:"sales_order" expandable:"true"`
	// The customer receiving this shipment.
	Customer *Customer `json:"customer" expandable:"true"`
	// Carrier selection and freight billing for this shipment.
	Freight *Freight `json:"freight" expandable:"true"`
	// Destination shipping address.
	ShippingAddress *Address `json:"shipping_address" expandable:"true"`
	// User who shipped this shipment.
	ShippedBy *AccountUser `json:"shipped_by" expandable:"true"`
	// Associated invoice.
	Invoice *Invoice `json:"invoice" expandable:"true"`
	// Pick associated with this shipment's order.
	Pick *Pick `json:"pick" expandable:"true"`
	// Lines recording the quantity shipped for each sales order line.
	Lines *List[ShipmentLine] `json:"lines" expandable:"true"`
	// Physical cases (packages) in this shipment, each with its own tracking and label details.
	ShippingCases *List[ShippingCaseDetail] `json:"shipping_cases" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A shipment line recording the quantity of a sales order line included in a shipment.
type ShipmentLine struct {
	// Shipment line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_line"`
	// The sales order line this shipment line fulfills.
	SalesOrderLine *SalesOrderLine `json:"sales_order_line" expandable:"true"`
	// The shipped item (the order line's item).
	//
	// Populated inline when lines are included, carried directly from the order line's item id.
	Item *Item `json:"item"`
	// Quantity shipped.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A physical case (package) in a shipment, as shown in shipment detail views.
type ShippingCaseDetail struct {
	// Shipping case ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// Human-readable case number.
	Number string `json:"number" validate:"required"`
	// Serial Shipping Container Code (SSCC) identifying this case.
	//
	// Assigned automatically when the shipment is shipped if the case does not already have one.
	SSCC *string `json:"sscc"`
	// Carrier tracking number for this case.
	TrackingNumber *string `json:"tracking_number"`
	// ID of the Shippo transaction for this case's shipping label, when the label was purchased through the Shippo integration.
	ShippoTransactionID *string `json:"shippo_transaction_id"`
	// URL of the printable shipping label for this case.
	ShippingLabelURL *string `json:"shipping_label_url"`
	// Timestamp when shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Freight charge for this case.
	FreightAmount *Quantity `json:"freight_amount"`
	// Shipping weight of this case.
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
	// Available rate options, sorted by rate ascending.
	//
	// Empty when freight is exempt for the order.
	Options *List[RateShopOption] `json:"options" validate:"required"`
	// Why a special freight outcome was applied to these options, if any.
	//
	// - `freight_exempt`: the order is exempt from freight; no options are returned.
	// - `minimum_order_met`: the free-shipping minimum order value was reached, so eligible options are rated at zero.
	// - `flat_rate`: a flat shipping rate was applied to the options (see `flat_rate`).
	// - `none`: standard carrier rates apply with no exemption.
	ExemptionType *string `json:"exemption_type"`
	// Flat shipping amount applied to the options.
	//
	// Set only when `exemption_type` is `flat_rate`.
	FlatRate *float64 `json:"flat_rate"`
}

// A single carrier and service level option returned by rate shopping.
type RateShopOption struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate_shop_option"`
	// Carrier.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Service level.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Quoted shipping rate for this carrier and service level.
	//
	// `0` when the carrier is not configured for live rating, or when a free-shipping rule (such as a met minimum order value) applies to this option.
	Rate float64 `json:"rate" validate:"required"`
	// Estimated number of days until delivery, when the carrier provides an estimate.
	EstimatedDays *int32 `json:"estimated_days"`
}

// Result of estimating a shipping rate.
type EstimateRateResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=estimate_rate_result"`
	// Estimated shipping rate.
	//
	// `0` when freight is exempt (a freight-exempt product line or customer, or a free-freight shipping term), when the free-shipping minimum order value is met, or when the carrier is not configured for live rating. When the customer's shipping term has a flat rate, the flat rate is returned.
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
