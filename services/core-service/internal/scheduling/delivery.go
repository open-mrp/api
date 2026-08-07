package scheduling

import (
	"math"
	"sort"
	"time"
)

// DeliveryOutcome is one order's commitment and what actually happened to it.
type DeliveryOutcome struct {
	SalesOrderID     string
	SalesOrderNumber string
	BuyerAccountID   string

	ShipByDate time.Time
	IssuedAt   *time.Time
	// FirstShipAt is nil for an order that has not shipped at all.
	FirstShipAt *time.Time

	QuantityOrdered float64
	QuantityPacked  float64

	// CommittedLeadTimeDays is what was promised; zero when the order predates commitment tracking.
	CommittedLeadTimeDays int
}

// IsShipped reports whether anything has left for this order.
func (o DeliveryOutcome) IsShipped() bool { return o.FirstShipAt != nil }

// IsInFull reports whether the whole ordered quantity has been packed.
//
// A tolerance rather than an exact comparison: quantities are decimals and a rounding difference of a millionth of a unit is not a short shipment.
func (o DeliveryOutcome) IsInFull() bool {
	if o.QuantityOrdered <= 0 {
		return false
	}
	return o.QuantityPacked >= o.QuantityOrdered-1e-6
}

// IsOnTime reports whether the first shipment left on or before the promised date.
//
// Measured on the first shipment rather than the last: the promise is that the order starts moving by that date, and judging on the last shipment would fail an order the customer received on time in two boxes.
func (o DeliveryOutcome) IsOnTime() bool {
	if o.FirstShipAt == nil {
		return false
	}
	return !dateOnlyUTC(*o.FirstShipAt).After(dateOnlyUTC(o.ShipByDate))
}

// ActualLeadTimeDays is how long the order actually took, or nil when it has not shipped or was never issued.
func (o DeliveryOutcome) ActualLeadTimeDays() *int {
	if o.FirstShipAt == nil || o.IssuedAt == nil {
		return nil
	}
	days := int(dateOnlyUTC(*o.FirstShipAt).Sub(dateOnlyUTC(*o.IssuedAt)).Hours() / 24)
	return &days
}

// DaysLate is how far past the promise the first shipment went, or how far past today an unshipped order already is. Zero or negative means not late.
func (o DeliveryOutcome) DaysLate(asOf time.Time) int {
	reference := dateOnlyUTC(asOf)
	if o.FirstShipAt != nil {
		reference = dateOnlyUTC(*o.FirstShipAt)
	}
	return int(reference.Sub(dateOnlyUTC(o.ShipByDate)).Hours() / 24)
}

// DeliveryPerformance is the delivery picture for one period, or for the whole window.
type DeliveryPerformance struct {
	// PeriodStart is the first day of the bucket; zero for the overall summary.
	PeriodStart time.Time

	CommittedOrderCount int
	ShippedOrderCount   int
	OnTimeOrderCount    int
	OnTimeInFullCount   int
	LateOrderCount      int
	// NotYetShippedCount is orders due in this period that have not shipped at all. They count against on-time, because a promise not yet met is not a promise kept.
	NotYetShippedCount int

	// Ratios are nil rather than zero when nothing was due, so a quiet week does not render as total failure.
	OnTimePct       *float64
	OnTimeInFullPct *float64

	// AverageDaysLate is measured over late orders only; nil when none were late. Averaging over every order would dilute a real problem into a number that looks fine.
	AverageDaysLate *float64
	// AverageLeadTimeDays is how long shipped orders actually took, and AverageCommittedLeadTimeDays what they were promised. The gap between them is what a merchant renegotiates on.
	AverageLeadTimeDays          *float64
	AverageCommittedLeadTimeDays *float64
}

// BacklogBucket is one age band of orders already past their promise and still unshipped.
type BacklogBucket struct {
	// Label names the band; MinDaysLate and MaxDaysLate bound it, with MaxDaysLate zero meaning unbounded.
	Label       string
	MinDaysLate int
	MaxDaysLate int
	OrderCount  int
	Units       float64
}

// backlogBands are the age bands past-due orders are reported in. Coarse on purpose: the difference between eleven and twelve days late is not a decision, and the difference between a week and a month is.
var backlogBands = []struct {
	label   string
	minDays int
	maxDays int
}{
	{"1_7_days", 1, 7},
	{"8_30_days", 8, 30},
	{"31_60_days", 31, 60},
	{"over_60_days", 61, 0},
}

// AnalyzeDeliveryPerformance turns a window of commitments into delivery performance, bucketed and overall.
//
// Only orders carrying a commitment participate. An order with no ship-by date cannot be late, and counting it as on time would inflate the rate with orders nobody promised anything about — the count of those is reported separately by the caller so the exclusion is visible.
//
// An order due in the window that has not shipped counts against on-time rather than being held back until it does. A promise not yet met is not a promise kept, and excluding open orders would let a plant with a growing late backlog report perfect delivery.
func AnalyzeDeliveryPerformance(outcomes []DeliveryOutcome, bucketOf func(time.Time) time.Time, asOf time.Time) (buckets []DeliveryPerformance, overall DeliveryPerformance) {
	byPeriod := map[time.Time]*deliveryAccumulator{}
	total := &deliveryAccumulator{}

	for _, o := range outcomes {
		period := time.Time{}
		if bucketOf != nil {
			period = bucketOf(o.ShipByDate)
		}
		acc := byPeriod[period]
		if acc == nil {
			acc = &deliveryAccumulator{}
			byPeriod[period] = acc
		}
		acc.add(o, asOf)
		total.add(o, asOf)
	}

	periods := make([]time.Time, 0, len(byPeriod))
	for period := range byPeriod {
		periods = append(periods, period)
	}
	sort.Slice(periods, func(i, j int) bool { return periods[i].Before(periods[j]) })

	buckets = make([]DeliveryPerformance, 0, len(periods))
	for _, period := range periods {
		result := byPeriod[period].result()
		result.PeriodStart = period
		buckets = append(buckets, result)
	}

	return buckets, total.result()
}

// AnalyzeBacklogAging groups orders that are past their promise and still unshipped into age bands.
//
// Shipped orders are excluded however late they were: this is a queue of work still owed, not a record of past misses. How late a shipped order was belongs to the on-time rate.
func AnalyzeBacklogAging(outcomes []DeliveryOutcome, asOf time.Time) []BacklogBucket {
	out := make([]BacklogBucket, 0, len(backlogBands))
	for _, band := range backlogBands {
		out = append(out, BacklogBucket{Label: band.label, MinDaysLate: band.minDays, MaxDaysLate: band.maxDays})
	}

	for _, o := range outcomes {
		if o.IsShipped() {
			continue
		}
		late := o.DaysLate(asOf)
		if late < 1 {
			continue
		}
		for i, band := range backlogBands {
			if late >= band.minDays && (band.maxDays == 0 || late <= band.maxDays) {
				out[i].OrderCount++
				// What is still owed, not what was ordered: a part-shipped order's backlog is the remainder.
				out[i].Units += math.Max(0, o.QuantityOrdered-o.QuantityPacked)
				break
			}
		}
	}

	return out
}

type deliveryAccumulator struct {
	committed      int
	shipped        int
	onTime         int
	onTimeInFull   int
	late           int
	notYetShipped  int
	daysLateSum    int
	daysLateCount  int
	leadTimeSum    int
	leadTimeCount  int
	committedSum   int
	committedCount int
}

func (a *deliveryAccumulator) add(o DeliveryOutcome, asOf time.Time) {
	a.committed++

	if o.CommittedLeadTimeDays > 0 {
		a.committedSum += o.CommittedLeadTimeDays
		a.committedCount++
	}

	if !o.IsShipped() {
		a.notYetShipped++
		// An unshipped order already past its date is late now, not late later.
		if o.DaysLate(asOf) > 0 {
			a.late++
			a.daysLateSum += o.DaysLate(asOf)
			a.daysLateCount++
		}
		return
	}

	a.shipped++
	if days := o.ActualLeadTimeDays(); days != nil {
		a.leadTimeSum += *days
		a.leadTimeCount++
	}

	if o.IsOnTime() {
		a.onTime++
		if o.IsInFull() {
			a.onTimeInFull++
		}
		return
	}

	a.late++
	a.daysLateSum += o.DaysLate(asOf)
	a.daysLateCount++
}

func (a *deliveryAccumulator) result() DeliveryPerformance {
	out := DeliveryPerformance{
		CommittedOrderCount: a.committed,
		ShippedOrderCount:   a.shipped,
		OnTimeOrderCount:    a.onTime,
		OnTimeInFullCount:   a.onTimeInFull,
		LateOrderCount:      a.late,
		NotYetShippedCount:  a.notYetShipped,
	}

	// The denominator is every order that was due, not every order that shipped. Measuring against shipments only would let unshipped late orders disappear from the score entirely.
	if a.committed > 0 {
		out.OnTimePct = pctPtr(float64(a.onTime) / float64(a.committed) * 100)
		out.OnTimeInFullPct = pctPtr(float64(a.onTimeInFull) / float64(a.committed) * 100)
	}
	if a.daysLateCount > 0 {
		out.AverageDaysLate = pctPtr(float64(a.daysLateSum) / float64(a.daysLateCount))
	}
	if a.leadTimeCount > 0 {
		out.AverageLeadTimeDays = pctPtr(float64(a.leadTimeSum) / float64(a.leadTimeCount))
	}
	if a.committedCount > 0 {
		out.AverageCommittedLeadTimeDays = pctPtr(float64(a.committedSum) / float64(a.committedCount))
	}
	return out
}

// pctPtr rounds to two decimals and returns a pointer, so an absent measurement stays distinguishable from a real zero.
func pctPtr(v float64) *float64 {
	rounded := math.Round(v*100) / 100
	return &rounded
}
