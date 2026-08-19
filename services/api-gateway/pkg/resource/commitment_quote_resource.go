package apiresource

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// CommitmentQuoteStep is one rule's contribution to a previewed ship-by date.
//
// Returned as an ordered list so a caller can show why a date is what it is without reimplementing the arithmetic, and so the explanation cannot drift from the calculation that produced it.
type CommitmentQuoteStep struct {
	// Which rule applied.
	Code constants.CommitmentStep `json:"code" validate:"required"`
	// Where the running date stood after this rule.
	Date time.Time `json:"date" validate:"required"`
	// How far this rule pulled the date back. Zero means the rule applied and changed nothing, which is worth showing: it says the date was already on an open day.
	DaysMoved int32 `json:"days_moved"`
	// The rule's own parameter — where a transit estimate came from, or the cutoff time applied. Null for a rule that takes none, rather than an empty string: snapping onto an open day has no parameter to report.
	Detail *string `json:"detail"`
}

var (
	sampleCommitmentShipByDate     = time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	sampleCommitmentShipByCutoffAt = time.Date(2026, time.August, 20, 19, 0, 0, 0, time.UTC)
	sampleCommitmentLeadTimeDays   = int32(16)
	sampleCommitmentLeadTimeSource = constants.LeadTimeSourceManual
	sampleCommitmentTransitDays    = int32(3)
	sampleCommitmentTransitSource  = constants.TransitSourceCarrierLane
	sampleStepTransitDetail        = string(constants.TransitSourceCarrierLane)
	sampleStepCutoffDetail         = "15:00"
	sampleCommitmentArrivalDate    = time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
)

// The sample walks one lane end to end: a Saturday delivery promised to a customer who receives Monday to Friday, shipped by a plant that tenders Monday to Thursday with a 3pm pickup.
func SampleShipByDate() *time.Time         { return &sampleCommitmentShipByDate }
func SampleShipByCutoffAt() *time.Time     { return &sampleCommitmentShipByCutoffAt }
func SampleCommitmentLeadTimeDays() *int32 { return &sampleCommitmentLeadTimeDays }
func SampleCommitmentLeadTimeSource() *constants.LeadTimeSource {
	return &sampleCommitmentLeadTimeSource
}
func SampleCommitmentTransitDays() *int32 { return &sampleCommitmentTransitDays }
func SampleCommitmentEstimatedDeliveryDate() *time.Time {
	return &sampleCommitmentArrivalDate
}
func SampleCommitmentTransitSource() *constants.TransitSource {
	return &sampleCommitmentTransitSource
}

func SampleCommitmentQuoteSteps() []CommitmentQuoteStep {
	return []CommitmentQuoteStep{
		{Code: constants.CommitmentStepBasis, Date: time.Date(2026, time.August, 22, 0, 0, 0, 0, time.UTC)},
		{Code: constants.CommitmentStepReceiveCalendar, Date: time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC), DaysMoved: 1},
		{Code: constants.CommitmentStepCarrierTransit, Date: time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC), DaysMoved: 3, Detail: &sampleStepTransitDetail},
		{Code: constants.CommitmentStepShipCalendar, Date: sampleCommitmentShipByDate, DaysMoved: 0},
		{Code: constants.CommitmentStepPickupCutoff, Date: sampleCommitmentShipByDate, Detail: &sampleStepCutoffDetail},
	}
}
