package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// Commitment describes when a record is due to ship: what was asked for, what that resolved to, and which rule decided.
//
// It is a generic, reusable sub-resource shared by anything carrying a ship-by commitment — a sales order, the pick that fulfills it, or a preview of an order that does not exist yet.
//
// The three inputs are alternative answers to the same question and at most one is ever set; `lead_time_source` reports which of them, or which level of the customer chain, produced the date. They are written flat on the create and update bodies, the way a carrier is written as `carrier_id` and read back under `freight`.
type Commitment struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=commitment"`
	// Date delivery was promised to the customer, if one was committed.
	PromisedAt *time.Time `json:"promised_at"`
	// Days between issue and the ship-by date, set on this record alone in place of the customer's standing lead time.
	LeadTimeOverrideDays *int32 `json:"lead_time_override_days"`
	// The ship date pinned by hand, bypassing transit and the customer's receiving days.
	ShipByOverrideDate *time.Time `json:"ship_by_override_date"`
	// When the record is contractually due to ship.
	//
	// Stamped at issue. With a promised delivery date, this is that date less the carrier's transit for the order's lane and less any day the customer cannot receive on — when the order has to leave to arrive when promised. Otherwise it comes from a lead time, whether the order's own or the one on the customer, its parent account, its account group, or the account.
	//
	// Always a day the plant actually ships on, whichever rule produced it, and carries the plant's pickup cutoff as its time of day when the shipping calendar sets one — the moment freight has to be tendered by, not just the day. Midnight UTC means no cutoff is configured rather than a deadline at midnight.
	//
	// Recomputed while the order is still open whenever something it was derived from moves — the basis above, or the carrier, service level, or ship-to address the transit was quoted on. Renegotiating a customer's standing lead time or adding a holiday to a calendar does not reach back into commitments already made. Cleared if the order is unissued.
	ShipByDate *time.Time `json:"ship_by_date"`
	// Calendar days between issue and the ship-by date.
	LeadTimeDays *int32 `json:"lead_time_days"`
	// Which rule produced the ship-by date.
	LeadTimeSource *constants.LeadTimeSource `json:"lead_time_source"`
	// Business days the carrier needs to cover this lane, subtracted from the promised delivery date to reach the ship-by date.
	//
	// Only set when a delivery date was promised and the lane could be priced. Without it the ship-by date falls back to the promised date itself.
	TransitDays *int32 `json:"transit_days"`
	// Where the transit estimate came from.
	TransitSource *constants.TransitSource `json:"transit_source"`
	// Days the customer's receiving calendar and the plant's shipping calendar pulled the ship-by date back, beyond what carrier transit accounted for.
	//
	// Zero means every date along the way already fell on an open day. This is what explains a ship-by date that is earlier than transit alone would suggest.
	CalendarAdjustmentDays *int32 `json:"calendar_adjustment_days"`
	// When freight leaving on the ship-by date would reach the customer: transit walked forward from it and landed on a day their dock receives.
	//
	// Reported by the commitment preview, which is asked what a set of inputs would produce and so computes the arrival too. A record carries the commitment it was stamped with, not a projection, and leaves this null.
	EstimatedDeliveryDate *time.Time `json:"estimated_delivery_date"`
}

// The sample walks one lane end to end: a Saturday delivery promised to a customer who receives Monday to Friday, shipped by a plant that tenders Monday to Thursday with a 3pm pickup — which is why the ship-by date carries a time.
var (
	sampleCommitmentPromisedAt         = time.Date(2026, time.August, 22, 0, 0, 0, 0, time.UTC)
	sampleCommitmentShipByDate         = time.Date(2026, time.August, 20, 19, 0, 0, 0, time.UTC)
	sampleCommitmentLeadTimeDays       = int32(16)
	sampleCommitmentLeadTimeSource     = constants.LeadTimeSourceManual
	sampleCommitmentTransitDays        = int32(3)
	sampleCommitmentTransitSource      = constants.TransitSourceCarrierLane
	sampleCommitmentCalendarAdjustment = int32(2)
	sampleCommitmentArrivalDate        = time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
)

var SampleCommitment = &Commitment{
	Object:                 constants.ObjectTypeCommitment,
	PromisedAt:             &sampleCommitmentPromisedAt,
	ShipByDate:             &sampleCommitmentShipByDate,
	LeadTimeDays:           &sampleCommitmentLeadTimeDays,
	LeadTimeSource:         &sampleCommitmentLeadTimeSource,
	TransitDays:            &sampleCommitmentTransitDays,
	TransitSource:          &sampleCommitmentTransitSource,
	CalendarAdjustmentDays: &sampleCommitmentCalendarAdjustment,
}

// SampleQuotedCommitment is the same commitment as a preview reports it, with the arrival a record does not carry.
var SampleQuotedCommitment = func() *Commitment {
	c := *SampleCommitment
	c.EstimatedDeliveryDate = &sampleCommitmentArrivalDate
	return &c
}()

// ShipBy folds a stored ship-by date and pickup cutoff into the single instant the API reports.
//
// The two are stored apart because the date is what every range filter and index reads, while the cutoff is a wall-clock time in the plant's own zone. A caller has no use for that split: the deadline is one moment, and the date alone is that moment at midnight.
func ShipBy(shipByDate, cutoffAt *time.Time) *time.Time {
	if cutoffAt != nil {
		return cutoffAt
	}
	return shipByDate
}

func (*Commitment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCommitment)
}
