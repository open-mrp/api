package scheduling

import (
	"math"
	"sort"
	"time"

	"github.com/augno/api/services/core-service/internal/forecast"
	"github.com/augno/api/shared/constants"
)

// Demand basis codes, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	DemandBasisTrailing12  = string(constants.ScheduleDemandBasisTrailing12)
	DemandBasisSeasonalEMA = string(constants.ScheduleDemandBasisSeasonalEMA)
)

// Demand override types, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	OverrideTypeAbsolute     = string(constants.DemandOverrideAdjustmentAbsolute)
	OverrideTypeDeltaUnits   = string(constants.DemandOverrideAdjustmentDeltaUnits)
	OverrideTypeDeltaPercent = string(constants.DemandOverrideAdjustmentDeltaPercent)
)

// Override scopes, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	OverrideScopeItem        = string(constants.DemandOverrideScopeItem)
	OverrideScopeProductLine = string(constants.DemandOverrideScopeProductLine)
	OverrideScopeAccount     = string(constants.DemandOverrideScopeAccount)
)

// MonthlyDemand is one month of demand for one item.
type MonthlyDemand struct {
	MonthStart time.Time
	Quantity   float64
}

// DemandOverride is management's adjustment to the forecast baseline. This is the only mechanism for departing from history — there is no growth multiplier.
type DemandOverride struct {
	ID          string
	ScopeCode   string
	ScopeRefID  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	TypeCode    string
	Value       float64
	ReasonCode  string
	CreatedAt   time.Time
}

// AppliedOverride records an override that actually changed a number, so the plan can explain why it differs from history.
type AppliedOverride struct {
	OverrideID string    `json:"override_id"`
	ItemID     string    `json:"item_id"`
	MonthStart time.Time `json:"month_start"`
	Before     float64   `json:"before"`
	After      float64   `json:"after"`
	TypeCode   string    `json:"adjustment"`
	ReasonCode string    `json:"reason"`
}

// ItemDemand is the resolved demand picture for one constraint item.
type ItemDemand struct {
	ItemID string

	// AnnualDemand is the forward-looking annual run rate the policy uses.
	AnnualDemand float64
	// TrailingAnnual is the last twelve complete months, for comparison.
	TrailingAnnual float64
	// SigmaWeeklyPooled is the pooled weekly standard deviation at the constraint. Pooled as sqrt(sum of squares) because one constraint item feeds many finished SKUs whose variability partially cancels.
	SigmaWeeklyPooled float64
	// FinishedGoods is the per-SKU detail behind the pooled figures above.
	FinishedGoods []FinishedGoodDemand
	// SigmaDownstreamSum is the plain sum of downstream sigmas, used for the finished goods buffer where the risk does not pool.
	SigmaDownstreamSum float64
	DownstreamCount    int
}

// DemandInput is everything the demand pass needs, already loaded.
type DemandInput struct {
	AsOf         time.Time
	BasisCode    string
	ForecastZ    float64
	WeeksPerYear int
	// ForecastMonths is how far forward to project when using the seasonal-EMA basis.
	ForecastMonths int

	// MonthlyByItem is monthly demand per constraint item, already pooled from the finished goods that item becomes. Series need not be dense.
	MonthlyByItem map[string][]MonthlyDemand
	// DownstreamByItem is the per-finished-good series behind each constraint item, needed to compute the downstream sigma sum separately from the pooled one.
	DownstreamByItem map[string][]FinishedGood

	Overrides []DemandOverride
	// ItemsByProductLine resolves a product-line-scoped override onto items.
	ItemsByProductLine map[string][]string
}

// ResolveDemand computes the demand picture for every item, applying overrides.
//
// Returns items sorted by ID and the list of overrides that actually changed a number.
func ResolveDemand(in DemandInput) ([]ItemDemand, []AppliedOverride) {
	weeksPerYear := float64(in.WeeksPerYear)
	if weeksPerYear <= 0 {
		weeksPerYear = 52
	}

	itemIDs := make([]string, 0, len(in.MonthlyByItem))
	for id := range in.MonthlyByItem {
		itemIDs = append(itemIDs, id)
	}
	sort.Strings(itemIDs)

	overridesByItem := indexOverrides(in.Overrides, in.ItemsByProductLine, itemIDs)

	out := make([]ItemDemand, 0, len(itemIDs))
	var applied []AppliedOverride

	for _, itemID := range itemIDs {
		series := sortedSeries(in.MonthlyByItem[itemID])

		// Overrides land on the monthly series BEFORE any forecasting, so a forecast basis projects the adjusted history rather than ignoring the adjustment.
		adjusted, itemApplied := applyOverrides(itemID, series, overridesByItem[itemID])
		applied = append(applied, itemApplied...)

		demand := ItemDemand{ItemID: itemID}
		demand.TrailingAnnual = trailingTwelve(adjusted, in.AsOf)

		switch in.BasisCode {
		case DemandBasisSeasonalEMA:
			demand.AnnualDemand = forecastAnnual(adjusted, in.AsOf, in.ForecastMonths, in.ForecastZ)
		default:
			demand.AnnualDemand = demand.TrailingAnnual
		}

		// Weekly sigma from monthly sigma: a month is roughly 52/12 weeks, and variance scales with time, so the weekly figure divides by its square root.
		monthlyValues := make([]float64, 0, len(adjusted))
		for _, m := range adjusted {
			monthlyValues = append(monthlyValues, m.Quantity)
		}
		weeksPerMonth := weeksPerYear / 12
		demand.SigmaWeeklyPooled = StdDevFloat(monthlyValues) / math.Sqrt(weeksPerMonth)

		// Downstream sigmas pool as the square root of the sum of squares at the constraint, but the finished-goods buffers are per-SKU and so add plainly.
		var sumSquares, plainSum float64
		for _, downstream := range in.DownstreamByItem[itemID] {
			values := make([]float64, 0, len(downstream.Monthly))
			for _, m := range downstream.Monthly {
				values = append(values, m.Quantity)
			}
			sigma := StdDevFloat(values) / math.Sqrt(weeksPerMonth)
			sumSquares += sigma * sigma
			plainSum += sigma

			// Kept per finished good so the pooled buffer can be decomposed back into per-SKU targets. Annual demand is the trailing window's own total, not a share of the pooled figure — a finished SKU's reorder point has to answer for that SKU's demand.
			var annual float64
			for _, m := range downstream.Monthly {
				annual += m.Quantity
			}
			demand.FinishedGoods = append(demand.FinishedGoods, FinishedGoodDemand{
				ItemID:        downstream.ItemID,
				SKU:           downstream.SKU,
				ProductLineID: downstream.ProductLineID,
				AnnualDemand:  annual,
				SigmaWeekly:   sigma,
				OnHand:        downstream.OnHand,
			})
		}
		demand.DownstreamCount = len(in.DownstreamByItem[itemID])
		demand.SigmaDownstreamSum = plainSum
		if sumSquares > 0 {
			// Prefer the pooled figure derived from the downstream detail when we have it; the constraint-level series is an aggregate and understates variance.
			demand.SigmaWeeklyPooled = math.Sqrt(sumSquares)
		}

		out = append(out, demand)
	}

	return out, applied
}

// indexOverrides expands product-line overrides onto items and buckets by item.
func indexOverrides(overrides []DemandOverride, itemsByLine map[string][]string, knownItems []string) map[string][]DemandOverride {
	known := make(map[string]bool, len(knownItems))
	for _, id := range knownItems {
		known[id] = true
	}

	out := make(map[string][]DemandOverride)
	for _, o := range overrides {
		switch o.ScopeCode {
		case OverrideScopeItem:
			if known[o.ScopeRefID] {
				out[o.ScopeRefID] = append(out[o.ScopeRefID], o)
			}
		case OverrideScopeProductLine:
			for _, itemID := range itemsByLine[o.ScopeRefID] {
				if known[itemID] {
					out[itemID] = append(out[itemID], o)
				}
			}
		case OverrideScopeAccount:
			// Account scope reaches every planned item — the "scale everything for growth" knob. Absolute values make no sense fanned out to the whole plan, so the service refuses them at write time.
			for _, itemID := range knownItems {
				out[itemID] = append(out[itemID], o)
			}
		}
	}

	// Application order is absolute, then delta units, then percent — a percentage has to act on the already-adjusted number or the result depends on input ordering. Ties break on creation time so the newest absolute wins.
	rank := map[string]int{OverrideTypeAbsolute: 0, OverrideTypeDeltaUnits: 1, OverrideTypeDeltaPercent: 2}
	for itemID := range out {
		list := out[itemID]
		sort.SliceStable(list, func(i, j int) bool {
			if rank[list[i].TypeCode] != rank[list[j].TypeCode] {
				return rank[list[i].TypeCode] < rank[list[j].TypeCode]
			}
			if !list[i].CreatedAt.Equal(list[j].CreatedAt) {
				return list[i].CreatedAt.Before(list[j].CreatedAt)
			}
			return list[i].ID < list[j].ID
		})
		out[itemID] = list
	}
	return out
}

// applyOverrides layers overrides onto a monthly series, returning the adjusted series and a record of every change that actually moved a number.
//
// An override covering a month with no history inserts that month, so "a big customer is about to order" works for an item that has never sold.
func applyOverrides(itemID string, series []MonthlyDemand, overrides []DemandOverride) ([]MonthlyDemand, []AppliedOverride) {
	if len(overrides) == 0 {
		return series, nil
	}

	byMonth := make(map[time.Time]float64, len(series))
	for _, m := range series {
		byMonth[m.MonthStart] = m.Quantity
	}

	var applied []AppliedOverride

	for _, o := range overrides {
		for _, monthStart := range monthsBetween(o.PeriodStart, o.PeriodEnd) {
			before := byMonth[monthStart]

			var after float64
			switch o.TypeCode {
			case OverrideTypeAbsolute:
				after = o.Value
			case OverrideTypeDeltaUnits:
				after = before + o.Value
			case OverrideTypeDeltaPercent:
				after = before * (1 + o.Value/100)
			default:
				continue
			}
			if after < 0 {
				after = 0
			}
			if after == before {
				continue
			}

			byMonth[monthStart] = after
			applied = append(applied, AppliedOverride{
				OverrideID: o.ID,
				ItemID:     itemID,
				MonthStart: monthStart,
				Before:     before,
				After:      after,
				TypeCode:   o.TypeCode,
				ReasonCode: o.ReasonCode,
			})
		}
	}

	out := make([]MonthlyDemand, 0, len(byMonth))
	for monthStart, quantity := range byMonth {
		out = append(out, MonthlyDemand{MonthStart: monthStart, Quantity: quantity})
	}
	return sortedSeries(out), applied
}

// monthsBetween enumerates month starts from start to end inclusive.
func monthsBetween(start, end time.Time) []time.Time {
	start = monthStartOf(start)
	end = monthStartOf(end)
	if end.Before(start) {
		return nil
	}

	var out []time.Time
	for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 1, 0) {
		out = append(out, cursor)
	}
	return out
}

func monthStartOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func sortedSeries(series []MonthlyDemand) []MonthlyDemand {
	out := make([]MonthlyDemand, len(series))
	copy(out, series)
	sort.SliceStable(out, func(i, j int) bool { return out[i].MonthStart.Before(out[j].MonthStart) })
	return out
}

// trailingTwelve sums the twelve complete months ending before asOf's month. The current partial month is excluded: including it would read as a collapse in demand.
func trailingTwelve(series []MonthlyDemand, asOf time.Time) float64 {
	lastComplete := monthStartOf(asOf).AddDate(0, -1, 0)
	windowStart := lastComplete.AddDate(0, -11, 0)

	var total float64
	for _, m := range series {
		if m.MonthStart.Before(windowStart) || m.MonthStart.After(lastComplete) {
			continue
		}
		total += m.Quantity
	}
	return total
}

// forecastAnnual projects forward with the shared seasonal-EMA engine and sums the horizon into an annual run rate.
func forecastAnnual(series []MonthlyDemand, asOf time.Time, forecastMonths int, zScore float64) float64 {
	if forecastMonths <= 0 {
		forecastMonths = 12
	}

	lastComplete := monthStartOf(asOf).AddDate(0, -1, 0)

	observations := make([]forecast.Observation, 0, len(series))
	for _, m := range series {
		if m.MonthStart.After(lastComplete) {
			continue // never forecast from a partial month
		}
		observations = append(observations, forecast.Observation{MonthStart: m.MonthStart, Value: m.Quantity})
	}
	if len(observations) == 0 {
		return 0
	}

	var total float64
	for _, p := range forecast.SeasonalEMA(observations, lastComplete, forecastMonths, zScore) {
		total += p.Forecast
	}

	// Scale a shorter horizon up to a full year so the policy always reasons about an annual rate.
	if forecastMonths != 12 {
		total = total * 12 / float64(forecastMonths)
	}
	return total
}
