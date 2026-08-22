package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SamplePickID = "pk_6eilj488bq8d"
const SamplePickNumber = "PK-001"

// A warehouse picking task for a sales order, tracking the quantities to pull from inventory and pack for shipment.
//
// A pick is created automatically when a sales order is issued, with one line for each order line whose product is of type `sale`  service, shipping, tax, credit and return lines are skipped — and nothing picked yet.
type Pick struct {
	// Pick ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick"`
	// Human-readable number that identifies the pick, distinct from the `id`.
	Number string `json:"number" validate:"required"`
	// The customer associated with the sales order.
	Customer *Customer `json:"customer" expandable:"true"`
	// How urgently the pick should be worked.
	Priority constants.PriorityCode `json:"priority" validate:"required"`
	// Address the associated sales order ships to.
	ShipTo *Address `json:"ship_to"`
	// Number of lines on this pick.
	LineCount int32 `json:"line_count"`
	// Progress through picking and packing, aggregated over the pick's sale lines so a list row can render progress bars without expanding `lines`.
	Totals *PickTotals `json:"totals"`
	// Timestamp of the most recent shipment sent (null until shipped).
	LastShippedAt *time.Time `json:"last_shipped_at"`
	// The pick's lines, each tracking the quantity picked against one sales order line.
	Lines *List[PickLine] `json:"lines" expandable:"true"`
	// Records this pick sits between — the order it fulfills and the shipments packed from it.
	Related *PickRelated `json:"related"`
	// Timestamp when the pick was finished.
	FinishedAt *time.Time `json:"finished_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
	// When the associated sales order promised delivery.
	PromisedAt *time.Time `json:"promised_at"`
	// Date the order must ship by to meet its commitment.
	ShipByDate *time.Time `json:"ship_by_date"`
	// Days allowed to prepare the order before it ships.
	LeadTimeDays *int32 `json:"lead_time_days"`
	// Which rule in the customer/group/account chain produced `lead_time_days`.
	LeadTimeSource *constants.LeadTimeSource `json:"lead_time_source"`
	// Days the carrier is expected to take in transit.
	TransitDays *int32 `json:"transit_days"`
	// Whether `transit_days` came from a cached lane estimate or the service level's default.
	TransitSource *constants.TransitSource `json:"transit_source"`
}

// Groups the records a pick sits between — the order it fulfills and the shipments packed from it — and is returned only once at least one member has been expanded.
type PickRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_related"`
	// The sales order this pick fulfills.
	SalesOrder *Record `json:"sales_order" expandable:"true"`
	// Lists the shipments packed from this pick.
	Shipments *List[Record] `json:"shipments" expandable:"true"`
}

// Progress through each fulfillment stage of a pick.
type PickTotals struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_totals"`
	// How far picking has progressed.
	Picked PickStageTotal `json:"picked"`
	// How far packing has progressed.
	Packed PickStageTotal `json:"packed"`
}

// How far one fulfillment stage of a pick has progressed.
type PickStageTotal struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_stage_total"`
	// Progress as a fraction between 0 and 1.
	Completion float64 `json:"completion"`
}

var samplePickLeadTimeDays = int32(3)
var samplePickLeadTimeSource = constants.LeadTimeSourceAccountGroup
var samplePickTransitDays = int32(2)
var samplePickTransitSource = constants.TransitSourceServiceLevel

var SamplePick = &Pick{
	ID:        SamplePickID,
	Object:    constants.ObjectTypePick,
	Number:    SamplePickNumber,
	Customer:  SampleCustomer,
	Priority:  SamplePriorityCode,
	ShipTo:    SampleAddress,
	LineCount: 1,
	Totals: &PickTotals{
		Object: constants.ObjectTypePickTotals,
		Picked: PickStageTotal{Object: constants.ObjectTypePickStageTotal, Completion: 1},
		Packed: PickStageTotal{Object: constants.ObjectTypePickStageTotal, Completion: 0.5},
	},
	Lines:          NewList([]PickLine{*SamplePickLine}, PageInfo{}),
	Related:        &PickRelated{Object: constants.ObjectTypePickRelated},
	PromisedAt:     timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	ShipByDate:     timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	LeadTimeDays:   &samplePickLeadTimeDays,
	LeadTimeSource: &samplePickLeadTimeSource,
	TransitDays:    &samplePickTransitDays,
	TransitSource:  &samplePickTransitSource,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Pick) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePick)
}

// The shipment numbers for the sales order a pick belongs to.
type PickShipmentsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_shipments_response"`
	// Shipment numbers associated with the pick, oldest first.
	ShipmentNumbers []string `json:"shipment_numbers" validate:"required"`
	// Total number of matching shipments, ignoring `limit` and `offset`.
	Count int32 `json:"count" validate:"required"`
}

var SamplePickShipmentsResponse = &PickShipmentsResponse{
	Object:          constants.ObjectTypePickShipmentsResponse,
	ShipmentNumbers: []string{"SH-001", "SH-002"},
	Count:           2,
}

func (*PickShipmentsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickShipmentsResponse)
}
