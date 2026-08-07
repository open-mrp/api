package scheduling

import (
	"math"
	"testing"
	"time"
)

var deliveryAsOf = time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)

func day(d int) time.Time { return time.Date(2026, time.August, d, 0, 0, 0, 0, time.UTC) }

func dayPtr(d int) *time.Time { t := day(d); return &t }

// An order shipped on its promised date is on time; the day after is not.
func TestDeliveryOutcome_OnTimeIsInclusiveOfThePromisedDay(t *testing.T) {
	t.Parallel()

	onTheDay := DeliveryOutcome{ShipByDate: day(10), FirstShipAt: dayPtr(10)}
	if !onTheDay.IsOnTime() {
		t.Fatal("shipping on the promised day is on time")
	}

	dayAfter := DeliveryOutcome{ShipByDate: day(10), FirstShipAt: dayPtr(11)}
	if dayAfter.IsOnTime() {
		t.Fatal("shipping the day after the promise is late")
	}

	unshipped := DeliveryOutcome{ShipByDate: day(10)}
	if unshipped.IsOnTime() {
		t.Fatal("an order that has not shipped cannot be on time")
	}
}

// In-full tolerates decimal noise but not a real short shipment.
func TestDeliveryOutcome_InFull(t *testing.T) {
	t.Parallel()

	exact := DeliveryOutcome{QuantityOrdered: 100, QuantityPacked: 100}
	if !exact.IsInFull() {
		t.Fatal("the whole quantity packed is in full")
	}

	noise := DeliveryOutcome{QuantityOrdered: 100, QuantityPacked: 99.9999999}
	if !noise.IsInFull() {
		t.Fatal("a rounding difference of a ten-millionth is not a short shipment")
	}

	short := DeliveryOutcome{QuantityOrdered: 100, QuantityPacked: 99}
	if short.IsInFull() {
		t.Fatal("one unit short is short")
	}
}

// A promise not yet met is not a promise kept: an unshipped order due in the window counts against on-time rather than being held back until it ships.
func TestAnalyzeDeliveryPerformance_UnshippedCountsAgainstOnTime(t *testing.T) {
	t.Parallel()

	_, overall := AnalyzeDeliveryPerformance([]DeliveryOutcome{
		{SalesOrderID: "so_1", ShipByDate: day(5), FirstShipAt: dayPtr(4), QuantityOrdered: 10, QuantityPacked: 10},
		{SalesOrderID: "so_2", ShipByDate: day(6)}, // due, never shipped, and today is the 20th
	}, nil, deliveryAsOf)

	if overall.CommittedOrderCount != 2 {
		t.Fatalf("committed = %d, want 2", overall.CommittedOrderCount)
	}
	if overall.OnTimeOrderCount != 1 {
		t.Fatalf("on time = %d, want 1", overall.OnTimeOrderCount)
	}
	if overall.NotYetShippedCount != 1 {
		t.Fatalf("not yet shipped = %d, want 1", overall.NotYetShippedCount)
	}
	if overall.OnTimePct == nil || math.Abs(*overall.OnTimePct-50) > 0.001 {
		t.Fatalf("on-time = %v, want 50 — the denominator is orders due, not orders shipped", overall.OnTimePct)
	}
	// The unshipped one is fourteen days past its date and is late now, not late later.
	if overall.LateOrderCount != 1 {
		t.Fatalf("late = %d, want 1", overall.LateOrderCount)
	}
}

// On-time-in-full is stricter than on-time: shipping on the day but short still misses.
func TestAnalyzeDeliveryPerformance_InFullIsStricterThanOnTime(t *testing.T) {
	t.Parallel()

	_, overall := AnalyzeDeliveryPerformance([]DeliveryOutcome{
		{SalesOrderID: "so_1", ShipByDate: day(5), FirstShipAt: dayPtr(5), QuantityOrdered: 100, QuantityPacked: 60},
	}, nil, deliveryAsOf)

	if overall.OnTimeOrderCount != 1 {
		t.Fatalf("on time = %d, want 1 — it did ship on the day", overall.OnTimeOrderCount)
	}
	if overall.OnTimeInFullCount != 0 {
		t.Fatalf("on time in full = %d, want 0 — sixty of a hundred is not in full", overall.OnTimeInFullCount)
	}
}

// A period with nothing due has no delivery rate. Rendering it as zero would read as total failure.
func TestAnalyzeDeliveryPerformance_NothingDueHasNoRate(t *testing.T) {
	t.Parallel()

	_, overall := AnalyzeDeliveryPerformance(nil, nil, deliveryAsOf)

	if overall.OnTimePct != nil || overall.OnTimeInFullPct != nil {
		t.Fatalf("expected nil ratios with nothing due, got %v / %v", overall.OnTimePct, overall.OnTimeInFullPct)
	}
	if overall.AverageDaysLate != nil {
		t.Fatal("no late orders means no average lateness, not zero days late")
	}
}

// Average lateness is measured over late orders only; averaging over everything would dilute a real problem into a number that looks fine.
func TestAnalyzeDeliveryPerformance_AverageLatenessCoversLateOrdersOnly(t *testing.T) {
	t.Parallel()

	_, overall := AnalyzeDeliveryPerformance([]DeliveryOutcome{
		{SalesOrderID: "so_1", ShipByDate: day(5), FirstShipAt: dayPtr(1), QuantityOrdered: 10, QuantityPacked: 10},
		{SalesOrderID: "so_2", ShipByDate: day(5), FirstShipAt: dayPtr(1), QuantityOrdered: 10, QuantityPacked: 10},
		{SalesOrderID: "so_3", ShipByDate: day(5), FirstShipAt: dayPtr(15), QuantityOrdered: 10, QuantityPacked: 10},
	}, nil, deliveryAsOf)

	if overall.AverageDaysLate == nil || math.Abs(*overall.AverageDaysLate-10) > 0.001 {
		t.Fatalf("average days late = %v, want 10 — averaged over the one late order, not all three", overall.AverageDaysLate)
	}
}

// The gap between promised and actual lead time is what a merchant renegotiates on, so both are reported.
func TestAnalyzeDeliveryPerformance_ReportsPromisedAndActualLeadTime(t *testing.T) {
	t.Parallel()

	_, overall := AnalyzeDeliveryPerformance([]DeliveryOutcome{
		{
			SalesOrderID: "so_1", ShipByDate: day(31), IssuedAt: dayPtr(1), FirstShipAt: dayPtr(21),
			QuantityOrdered: 10, QuantityPacked: 10, CommittedLeadTimeDays: 30,
		},
	}, nil, deliveryAsOf)

	if overall.AverageLeadTimeDays == nil || math.Abs(*overall.AverageLeadTimeDays-20) > 0.001 {
		t.Fatalf("actual lead time = %v, want 20", overall.AverageLeadTimeDays)
	}
	if overall.AverageCommittedLeadTimeDays == nil || math.Abs(*overall.AverageCommittedLeadTimeDays-30) > 0.001 {
		t.Fatalf("committed lead time = %v, want 30", overall.AverageCommittedLeadTimeDays)
	}
}

func TestAnalyzeDeliveryPerformance_BucketsByPeriod(t *testing.T) {
	t.Parallel()

	monthOf := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	buckets, overall := AnalyzeDeliveryPerformance([]DeliveryOutcome{
		{SalesOrderID: "so_1", ShipByDate: time.Date(2026, time.July, 5, 0, 0, 0, 0, time.UTC), FirstShipAt: dayPtr(1), QuantityOrdered: 10, QuantityPacked: 10},
		{SalesOrderID: "so_2", ShipByDate: day(5), FirstShipAt: dayPtr(4), QuantityOrdered: 10, QuantityPacked: 10},
		{SalesOrderID: "so_3", ShipByDate: day(6), FirstShipAt: dayPtr(19), QuantityOrdered: 10, QuantityPacked: 10},
	}, monthOf, deliveryAsOf)

	if len(buckets) != 2 {
		t.Fatalf("got %d buckets, want 2 (July and August)", len(buckets))
	}
	if !buckets[0].PeriodStart.Before(buckets[1].PeriodStart) {
		t.Fatal("buckets must come back chronologically")
	}
	if buckets[1].CommittedOrderCount != 2 || buckets[1].LateOrderCount != 1 {
		t.Fatalf("August: committed=%d late=%d, want 2/1", buckets[1].CommittedOrderCount, buckets[1].LateOrderCount)
	}
	if overall.CommittedOrderCount != 3 {
		t.Fatalf("overall committed = %d, want 3", overall.CommittedOrderCount)
	}
}

// The backlog is work still owed, so a shipped order leaves it however late it was.
func TestAnalyzeBacklogAging(t *testing.T) {
	t.Parallel()

	buckets := AnalyzeBacklogAging([]DeliveryOutcome{
		{SalesOrderID: "so_fresh", ShipByDate: day(18), QuantityOrdered: 10},                                            // 2 days late
		{SalesOrderID: "so_mid", ShipByDate: time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC), QuantityOrdered: 20}, // 26 days
		{SalesOrderID: "so_old", ShipByDate: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), QuantityOrdered: 30},   // >60 days
		{SalesOrderID: "so_shipped", ShipByDate: day(1), FirstShipAt: dayPtr(19), QuantityOrdered: 40},                  // late but gone
		{SalesOrderID: "so_future", ShipByDate: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), QuantityOrdered: 50},
	}, deliveryAsOf)

	byLabel := map[string]BacklogBucket{}
	for _, b := range buckets {
		byLabel[b.Label] = b
	}

	if byLabel["1_7_days"].OrderCount != 1 || byLabel["1_7_days"].Units != 10 {
		t.Fatalf("1-7 days: %+v", byLabel["1_7_days"])
	}
	if byLabel["8_30_days"].OrderCount != 1 || byLabel["8_30_days"].Units != 20 {
		t.Fatalf("8-30 days: %+v", byLabel["8_30_days"])
	}
	if byLabel["over_60_days"].OrderCount != 1 || byLabel["over_60_days"].Units != 30 {
		t.Fatalf("over 60 days: %+v", byLabel["over_60_days"])
	}
	if byLabel["31_60_days"].OrderCount != 0 {
		t.Fatalf("31-60 days should be empty: %+v", byLabel["31_60_days"])
	}

	total := 0
	for _, b := range buckets {
		total += b.OrderCount
	}
	if total != 3 {
		t.Fatalf("backlog holds %d orders, want 3 — a shipped order and a future one are not backlog", total)
	}
}

// A part-shipped order owes the remainder, not the whole quantity.
func TestAnalyzeBacklogAging_OwesTheRemainder(t *testing.T) {
	t.Parallel()

	buckets := AnalyzeBacklogAging([]DeliveryOutcome{
		{SalesOrderID: "so_1", ShipByDate: day(18), QuantityOrdered: 100, QuantityPacked: 70},
	}, deliveryAsOf)

	for _, b := range buckets {
		if b.Label == "1_7_days" {
			if b.Units != 30 {
				t.Fatalf("backlog units = %v, want 30 — the seventy already packed are not still owed", b.Units)
			}
			return
		}
	}
	t.Fatal("expected a 1-7 day bucket")
}
