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
	// The customer's own purchase order number for the sales order this pick fulfills.
	CustomerPurchaseOrderNumber *string `json:"customer_purchase_order_number"`
	// Free-form note carried from the sales order this pick fulfills.
	Note *string `json:"note"`
	// The customer associated with the sales order.
	Customer *Customer `json:"customer" expandable:"true"`
	// Who created the sales order this pick fulfills, and their relation (internal/customer/system).
	CreatedBy *CreatedBy `json:"created_by" expandable:"true"`
	// Carrier selection and freight billing carried from the sales order this pick fulfills.
	Freight *Freight `json:"freight" expandable:"true"`
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
	// When the order this pick fulfills is due to ship, carried from the order so the pick can explain its own deadline without expanding it.
	//
	// The calendar adjustment and the overrides that produced the date stay on the order; a pick carries the date, not the authoring history behind it.
	Commitment *Commitment `json:"commitment"`
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

var samplePickCustomerPurchaseOrderNumber = "PO-12345"
var samplePickNote = "Rush order"

// A pick carries only what the order denormalizes onto it.
var samplePickCommitment = &Commitment{
	Object:         constants.ObjectTypeCommitment,
	PromisedAt:     SampleCommitment.PromisedAt,
	ShipByDate:     SampleCommitment.ShipByDate,
	LeadTimeDays:   SampleCommitment.LeadTimeDays,
	LeadTimeSource: SampleCommitment.LeadTimeSource,
	TransitDays:    SampleCommitment.TransitDays,
	TransitSource:  SampleCommitment.TransitSource,
}

var SamplePick = &Pick{
	ID:                          SamplePickID,
	Object:                      constants.ObjectTypePick,
	Number:                      SamplePickNumber,
	CustomerPurchaseOrderNumber: &samplePickCustomerPurchaseOrderNumber,
	Note:                        &samplePickNote,
	Customer:                    SampleCustomer,
	CreatedBy:                   SampleCreatedBy,
	Freight:                     SampleFreight,
	Priority:                    SamplePriorityCode,
	ShipTo:                      SampleAddress,
	LineCount:                   1,
	Totals: &PickTotals{
		Object: constants.ObjectTypePickTotals,
		Picked: PickStageTotal{Object: constants.ObjectTypePickStageTotal, Completion: 1},
		Packed: PickStageTotal{Object: constants.ObjectTypePickStageTotal, Completion: 0.5},
	},
	Lines:      NewList([]PickLine{*SamplePickLine}, PageInfo{}),
	Related:    &PickRelated{Object: constants.ObjectTypePickRelated},
	Commitment: samplePickCommitment,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Pick) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePick)
}
