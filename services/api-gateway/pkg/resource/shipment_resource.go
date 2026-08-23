package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleShipmentID = "sh_pfygp2gl45y4"
const SampleShipmentNumber = "SH-001"
const SampleShipmentLineID = "shln_ysbxu08n6bbj"

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
	// Cleared if the shipment is voided.
	ShippedAt *time.Time `json:"shipped_at"`
	// Fulfillment priority, inherited from the sales order.
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// Number of shipping cases packed into this shipment.
	CaseCount int32 `json:"case_count"`
	// TODO: change from bool to a status type constant; check if its sued
	IsReadyToShip bool `json:"is_ready_to_ship"`
	// The customer receiving this shipment.
	Customer *Customer `json:"customer" expandable:"true"`
	// Carrier selection and freight billing for this shipment.
	Freight *Freight `json:"freight" expandable:"true"`
	// Destination shipping address.
	ShippingAddress *Address `json:"shipping_address" expandable:"true"`
	// User who shipped this shipment.
	ShippedBy *CreatedBy `json:"shipped_by" expandable:"true"`
	// TODO: Lines are a sub-object of cases, so move them
	// Lines recording the quantity shipped for each sales order line.
	Lines *List[ShipmentLine] `json:"lines" expandable:"true"`
	// Physical cases (packages) in this shipment, each with its own tracking and label details.
	ShippingCases *List[ShippingCaseDetail] `json:"shipping_cases" expandable:"true"`
	// Records this shipment sits between — its order, pick and invoice.
	Related *ShipmentRelated `json:"related"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Groups the records a shipment sits between: the order it fulfills, the pick it was packed from,
// and the invoice it raised. Returned only when at least one member has been expanded.
type ShipmentRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_related"`
	// The sales order this shipment fulfills.
	SalesOrder *Record `json:"sales_order" expandable:"true"`
	// The pick this shipment was packed from.
	Pick *Record `json:"pick" expandable:"true"`
	// The invoice raised when this shipment shipped.
	Invoice *Record `json:"invoice" expandable:"true"`
}

// A shipment line recording the quantity of a sales order line included in a shipment.
type ShipmentLine struct {
	// Shipment line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipment_line"`
	// The sales order line this shipment line fulfills.
	SalesOrderLine *SalesOrderLine `json:"sales_order_line" expandable:"true"`
	// What this line ships, as recorded on the originating sales order line.
	Item *Item `json:"item" expandable:"true"`
	// Quantity shipped on this line.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// TODO: get this setup
	Totals ShipmentLineTotals ``
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

type ShipmentLineTotals struct {
	Ordered     ShipmentLineStageTotal
	BackOrdered ShipmentLineStageTotal
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_totals"`
}

// The monetary amount that has reached one fulfillment stage, together with how far that stage has progressed.
type ShipmentLineStageTotal struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_stage_total"`
	// Progress through this stage, as a fraction between 0 and 1.
	//
	// Calculated as the quantity that has reached this stage divided by the quantity ordered, so `1` means the whole order has cleared the stage and `0` means nothing has reached it yet.
	Completion float64 `json:"completion"`
	// Amount that has reached this stage, as a decimal string (unit price times the quantity at this stage).
	Amount string `json:"amount" validate:"required" format:"decimal"`
}

// TODO: collaps the detail with the shipping case object whichever is more accurate
// A physical case (package) within a shipment, with its own tracking number, label and freight charge.
type ShippingCaseDetail struct {
	// Shipping case ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_case"`
	// Human-readable case number.
	Number string `json:"number" validate:"required"`
	// Serial Shipping Container Code (SSCC) identifying this case.
	//
	// Assigned automatically when the shipment is shipped if the case does not already have one, and kept if the shipment is later voided.
	SSCC *string `json:"sscc"`
	// Carrier tracking number for this case.
	//
	// Cleared when the shipment is voided.
	TrackingNumber *string `json:"tracking_number"`
	// ID of the Shippo transaction for this case's shipping label, when the label was purchased through the Shippo integration.
	ShippoTransactionID *string `json:"shippo_transaction_id"`
	// URL of the printable shipping label for this case.
	ShippingLabelURL *string `json:"shipping_label_url"`
	// Timestamp when this case was shipped.
	ShippedAt *time.Time `json:"shipped_at"`
	// Freight charge for this case.
	//
	// Reset to zero when the shipment is voided.
	FreightAmount *Quantity `json:"freight_amount"`
	// Shipping weight of this case.
	FreightWeight *Quantity `json:"freight_weight"`
	// The carrier handling this case.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// The carrier and service level options returned by rate shopping, along with the freight rule that shaped their rates.
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
	// - `minimum_order_met`: the customer's shipping term sets a free-shipping minimum order value and the order total exceeded it, so options are rated at zero. If the shipping term restricts free shipping to specific service levels, only those options are zeroed and the rest keep their carrier or flat rate.
	// - `flat_rate`: the customer's shipping term applies a flat shipping rate, which replaced every option's carrier rate.
	// - `none`: standard carrier rates apply with no exemption.
	ExemptionType *constants.FreightExemptionType `json:"exemption_type"`
	// Flat shipping amount applied to the options.
	//
	// Set when the customer's shipping term applies a flat rate, including when a met free-shipping minimum has already rated some options at zero.
	FlatRate *float64 `json:"flat_rate"`
}

// A single carrier and service level option returned by rate shopping.
type RateShopOption struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=rate_shop_option"`
	// The carrier that would handle the shipment.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// The carrier's service level, such as ground or overnight.
	ServiceLevel *ServiceLevel `json:"service_level" expandable:"true"`
	// Quoted shipping rate for this carrier and service level.
	//
	// `0` when the carrier is not linked to a live-rating account, or when the shipping term's free-shipping minimum order value has been met and this option qualifies for free shipping. When the customer's shipping term applies a flat rate, that amount replaces the rate on every option that is not already free.
	Rate float64 `json:"rate" validate:"required"`
	// Estimated number of days until delivery, when the carrier provides an estimate.
	EstimatedDays *int32 `json:"estimated_days"`
}

// The shipping rate estimated for a single carrier and service level.
type EstimateRateResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=estimate_rate_result"`
	// Estimated shipping rate.
	//
	// `0` when freight is exempt (a freight-exempt product line, a customer exempted by its own policy or by one of its groups, or a free-freight shipping term), when the free-shipping minimum order value is met for a service level the shipping term allows, or when the account has no live-rating integration or the carrier is not linked to one. When the customer's shipping term has a flat rate, the flat rate is returned instead.
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
	Object:        constants.ObjectTypeRateShopResult,
	Options:       NewList([]RateShopOption{*SampleRateShopOption}, PageInfo{}),
	ExemptionType: new(constants.FreightExemptionTypeNone),
}

func (*RateShopResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRateShopResult)
}

// --- Sample Data ---

var sampleShipmentNote = "Handle with care"
var sampleBillOfLading = "BOL-12345"
var sampleMasterTrackingNumber = "1Z999AA10123456784"

var sampleShippingCaseDetail = ShippingCaseDetail{
	ID:             SampleShippingCaseID,
	Object:         constants.ObjectTypeShippingCase,
	Number:         "SC-0001",
	SSCC:           new("003456789000000018"),
	TrackingNumber: new("1Z999AA10123456784"),
	ShippedAt:      timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	FreightAmount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "12.500000000000000000000000000000",
		DisplayValue: "$12.50",
		Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
	},
	FreightWeight: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "5.000000000000000000000000000000",
		DisplayValue: "5 lb",
		Unit:         newSampleUnit("Pound", "lb", constants.UnitTypeMass),
	},
	Carrier:   SampleCarrier,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleShipment = &Shipment{
	ID:                   SampleShipmentID,
	Object:               constants.ObjectTypeShipment,
	Number:               SampleShipmentNumber,
	Note:                 &sampleShipmentNote,
	BillOfLading:         &sampleBillOfLading,
	MasterTrackingNumber: &sampleMasterTrackingNumber,
	Status:               constants.ShipmentStatusShipped,
	ShippedAt:            timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	Priority:             SamplePriorityCode,
	CaseCount:            1,
	Customer:             SampleCustomer,
	Freight:              SampleFreight,
	ShippingAddress:      SampleAddress,
	ShippedBy:            SampleCreatedBy,
	Lines:                NewList([]ShipmentLine{*SampleShipmentLine}, PageInfo{}),
	ShippingCases:        NewList([]ShippingCaseDetail{sampleShippingCaseDetail}, PageInfo{}),
	Related:              &ShipmentRelated{Object: constants.ObjectTypeShipmentRelated},
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Shipment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShipment)
}

// TODO: Totals sample
var SampleShipmentLine = &ShipmentLine{
	ID:             SampleShipmentLineID,
	Object:         constants.ObjectTypeShipmentLine,
	SalesOrderLine: SampleSalesOrderLine,
	Item:           SampleItem,
	Quantity:       SampleQuantity,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShipmentLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShipmentLine)
}
