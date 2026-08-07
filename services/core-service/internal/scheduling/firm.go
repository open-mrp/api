package scheduling

import (
	"math"
	"sort"
	"time"
)

// FirmRequirement is one open order line's outstanding quantity, expressed against the constraint item that has to produce it.
//
// This is demand the plan owes, as opposed to the demand it forecasts. The distinction matters because a forecast is an average the buffer absorbs, while an order is a date somebody was promised.
type FirmRequirement struct {
	// ItemID is the constraint item, not the finished good: the requirement has already been pooled onto whatever the plan actually builds.
	ItemID string
	// FinishedItemID is the finished good the order was placed for, kept so an at-risk order can name the SKU rather than the greige behind it.
	FinishedItemID string

	SalesOrderID     string
	SalesOrderNumber string
	SalesOrderLineID string

	// Units is the outstanding quantity in the constraint item's own unit.
	Units float64

	// ShipByWeek is the horizon week the finished good must exist in. Negative means the commitment is already behind the plan.
	ShipByWeek int
	// DueWeek is when the constraint stage must finish for the finishing lead time to still make ShipByWeek. Clamped at zero.
	DueWeek int

	// IsPastDue is set when the constraint stage would have had to start before the plan does. The clamp to week 0 hides that, so it is recorded rather than inferred from DueWeek.
	IsPastDue bool
	// IsUndated marks an order with no ship-by commitment, dated at the start of the horizon because it is issued and unshipped. Reported separately so a planner can tell a real promise from a guess.
	IsUndated bool
}

// FirmSchedule is the order book resolved into per-item, per-week quantities the sweep can draw down.
type FirmSchedule struct {
	// ByItemWeek[itemID][weekIndex] is the firm quantity due from the constraint stage in that week.
	ByItemWeek map[string][]float64
	// Requirements is the flat list, sorted, for diagnostics and at-risk reporting.
	Requirements []FirmRequirement
	// TotalUnits is the whole order book expressed at the constraint.
	TotalUnits float64
	// UndatedCount is how many requirements had no ship-by commitment.
	UndatedCount int
	// PastDueCount is how many needed the constraint stage to start before the horizon.
	PastDueCount int
}

// OpenOrderLine is one outstanding order line as loaded, before it is dated or pooled.
type OpenOrderLine struct {
	SalesOrderID     string
	SalesOrderNumber string
	// SalesOrderLineID is the line the quantity is owed against, which is what a shipment is measured against later.
	SalesOrderLineID string
	// FinishedItemID is the item the order was placed for.
	FinishedItemID string
	// ConstraintItemID is the item in the plan that produces it. Empty means nothing planned produces this order, and the line is dropped.
	ConstraintItemID string
	// Units is already scaled into the constraint item's unit by the caller, the same way pooled historical demand is.
	Units float64
	// ShipByDate is nil for an order issued before commitments were tracked.
	ShipByDate *time.Time
}

// BuildFirmSchedule dates the open order book against the horizon and pools it onto the constraint items.
//
// Dating walks backwards from the promise: the finished good has to exist by its ship-by week, so the constraint stage has to finish a finishing lead time earlier. A requirement whose constraint week lands before the horizon is not dropped — it is clamped to week 0 and flagged, because an order that needed to start last month is the single most useful thing the plan can tell a planner, and silently moving it forward would make the plan look achievable when it is not.
//
// Deterministic: requirements are sorted before they are returned, so two solves over the same order book produce the same diagnostics.
func BuildFirmSchedule(lines []OpenOrderLine, horizonStart time.Time, s Settings) FirmSchedule {
	out := FirmSchedule{ByItemWeek: map[string][]float64{}}
	if s.HorizonWeeks <= 0 {
		return out
	}

	// The constraint has to finish this many weeks before the ship date for finishing to still make it.
	finishOffset := max(int(math.Ceil(s.FinishLeadTimeWeeks)), 0)

	weekStart := dateOnlyUTC(horizonStart)

	for _, line := range lines {
		if line.ConstraintItemID == "" || line.Units <= 0 {
			continue
		}

		req := FirmRequirement{
			ItemID:           line.ConstraintItemID,
			FinishedItemID:   line.FinishedItemID,
			SalesOrderID:     line.SalesOrderID,
			SalesOrderNumber: line.SalesOrderNumber,
			SalesOrderLineID: line.SalesOrderLineID,
			Units:            line.Units,
		}

		if line.ShipByDate == nil {
			// Issued, unshipped, and never given a commitment. It is owed now, so it sits at the front of the horizon and says why.
			req.IsUndated = true
			req.ShipByWeek = 0
			req.DueWeek = 0
			req.IsPastDue = true
			out.UndatedCount++
		} else {
			req.ShipByWeek = weeksBetween(weekStart, dateOnlyUTC(*line.ShipByDate))
			due := req.ShipByWeek - finishOffset
			if due < 0 {
				req.IsPastDue = true
				due = 0
			}
			req.DueWeek = due
		}

		// A commitment beyond the horizon is a future plan's problem: counting it now would build stock this plan has no reason to hold.
		if req.DueWeek >= s.HorizonWeeks {
			continue
		}

		if req.IsPastDue {
			out.PastDueCount++
		}

		if out.ByItemWeek[req.ItemID] == nil {
			out.ByItemWeek[req.ItemID] = make([]float64, s.HorizonWeeks)
		}
		out.ByItemWeek[req.ItemID][req.DueWeek] += req.Units
		out.TotalUnits += req.Units
		out.Requirements = append(out.Requirements, req)
	}

	sort.SliceStable(out.Requirements, func(i, j int) bool {
		a, b := out.Requirements[i], out.Requirements[j]
		if a.DueWeek != b.DueWeek {
			return a.DueWeek < b.DueWeek
		}
		if a.ItemID != b.ItemID {
			return a.ItemID < b.ItemID
		}
		if a.SalesOrderNumber != b.SalesOrderNumber {
			return a.SalesOrderNumber < b.SalesOrderNumber
		}
		return a.SalesOrderID < b.SalesOrderID
	})

	return out
}

// CeilWeeks rounds a fractional week count up to whole weeks, which is how a schedule counts: a plan is written in weeks, and half a week of finishing still occupies one.
func CeilWeeks(weeks float64) int {
	return max(int(math.Ceil(weeks)), 0)
}

// weeksBetween is how many whole weeks after start the given date falls, floored so a date inside the current week is week 0 and a date before start is negative.
func weeksBetween(start, date time.Time) int {
	days := int(date.Sub(start).Hours() / 24)
	if days < 0 {
		// Go truncates toward zero, so a negative partial week has to floor explicitly or a date three days before the horizon would land in week 0 alongside one three days after it.
		return -((-days + 6) / 7)
	}
	return days / 7
}

// RequirementForWeek is the firm quantity due for one item in one week, safe against items with no order book at all.
func (f FirmSchedule) RequirementForWeek(itemID string, week int) float64 {
	weeks, ok := f.ByItemWeek[itemID]
	if !ok || week < 0 || week >= len(weeks) {
		return 0
	}
	return weeks[week]
}
