package scheduling

import (
	"math"
	"sort"
	"time"
)

// BatchMeasurement is one historical batch, as loaded from the database.
type BatchMeasurement struct {
	BatchID          string
	ItemID           string
	SKU              string
	ScannedAt        time.Time
	Quantity         float64
	ProductionStepID string
	MachineID        string
	MachineName      string
	UnitCost         float64

	// LaborTimeValue is the production step's labor time, in the units LaborTime describes: a time per a quantity, not necessarily per one base unit.
	LaborTimeValue float64
	// LaborTime converts LaborTimeValue to seconds per base quantity unit; see SecondsPerUnitFromLaborTime.
	LaborTime    LaborTimeConversion
	LaborRate    float64
	OverheadRate float64

	// RunCreatedAt is when the production run was opened; paired with ScannedAt it gives the observed lead time for this batch.
	RunCreatedAt *time.Time
}

// ItemMeasurement is what history tells us about one constraint item.
type ItemMeasurement struct {
	ItemID string
	SKU    string

	// SecondsPerUnit is the run rate: how long one unit occupies the constraint.
	SecondsPerUnit float64
	UnitCost       float64
	// OverheadRate is the production step's machine burden. Production labor is deliberately NOT included: a changeover is worked by a technician on the single machine, so setup cost uses the dedicated changeover labor rate from settings instead of the thin per-machine-hour production allocation. Including both would double-count labor and inflate EOQ.
	OverheadRate float64

	// LotCount is how many batches were produced; the script treats one batch as one lot regardless of its quantity, and so does this.
	LotCount int
	Quantity float64

	// EligibleMachineID is the set of machines that have actually run this item. Empty means unconstrained.
	EligibleMachineID map[string]bool

	// MeasuredLeadTimeWeeks is the mean observed run-open to scan time. Zero when no usable samples existed, in which case the settings default applies.
	MeasuredLeadTimeWeeks float64
	LeadTimeSampleCount   int

	ProductionStepID string
}

// leadTimeSampleMaxDays discards implausible lead-time samples. A run left open for months is a data-entry artifact, not a real lead time, and a handful of them would dominate the mean.
const leadTimeSampleMaxDays = 120

// secondsPerHour is the time dimension's base (the hour) expressed in seconds; a labor-time unit's ratio gives its size in hours, so multiplying by this yields seconds.
const secondsPerHour = 3600

// LaborTimeConversion carries the unit ratios a production step's labor time is quoted in: the time unit of the rate's numerator, and the quantity unit of its denominator. Each ratio is that unit's size in its dimension's base unit, kept as the stored numerator/denominator pair.
//
// This is the labor-time twin of pricing.UnitConversion, which converts a price quoted per one unit against a quantity measured in another. A rate is meaningless without both halves: 554 sec/pr and 554 sec/ea are different speeds.
//
// The zero value converts nothing; SecondsPerUnitFromLaborTime reads a non-positive ratio as "no usable unit" and leaves that half alone.
type LaborTimeConversion struct {
	TimeRatioNumerator       float64
	TimeRatioDenominator     float64
	QuantityRatioNumerator   float64
	QuantityRatioDenominator float64
}

// SecondsPerUnitFromLaborTime converts a production step's labor time to seconds per one BASE quantity unit, using both of the rate's unit ratios.
//
// The time half: the numerator unit's ratio is its size in the time dimension's base, the hour, so seconds = value * (num/den) * 3600 — minute (1/60) gives 60, hour (1/1) gives 3600, day (24/1) gives 86400, and any account-defined time unit converts by the same rule.
//
// The quantity half: the denominator unit's ratio is its size in the quantity base, so a rate quoted per that unit is divided by it to reach seconds per base unit. A step rated 554 sec/pr is 277 sec per each, because a pair is two eaches. Callers measure quantity in base units — the scan quantity is normalized the same way — so skipping this division counts every pair-rated step twice, which is what put OEE Performance over 100%.
//
// A non-positive ratio (a scan that carried no usable unit on that half) leaves that half unconverted rather than zeroing a run rate.
func SecondsPerUnitFromLaborTime(value float64, conv LaborTimeConversion) float64 {
	seconds := value
	if conv.TimeRatioNumerator > 0 && conv.TimeRatioDenominator > 0 {
		seconds = seconds * (conv.TimeRatioNumerator / conv.TimeRatioDenominator) * secondsPerHour
	}
	if conv.QuantityRatioNumerator > 0 && conv.QuantityRatioDenominator > 0 {
		seconds = seconds / (conv.QuantityRatioNumerator / conv.QuantityRatioDenominator)
	}
	return seconds
}

// MeasureItems aggregates batch history into per-item measurements.
//
// Determinism: the result is sorted by SKU. Callers iterate it directly, so returning a map would reintroduce the nondeterminism the port exists to remove.
func MeasureItems(batches []BatchMeasurement) []ItemMeasurement {
	type accumulator struct {
		m             ItemMeasurement
		leadTimeDays  []float64
		sawMeasurable bool
	}

	byItem := make(map[string]*accumulator)

	for _, b := range batches {
		acc, ok := byItem[b.ItemID]
		if !ok {
			acc = &accumulator{m: ItemMeasurement{
				ItemID:            b.ItemID,
				SKU:               b.SKU,
				EligibleMachineID: make(map[string]bool),
				ProductionStepID:  b.ProductionStepID,
			}}
			byItem[b.ItemID] = acc
		}

		// One batch is one lot, whatever its quantity. This matches the script and is what makes the lot count a proxy for changeover frequency.
		acc.m.LotCount++
		acc.m.Quantity += b.Quantity

		if b.MachineID != "" {
			acc.m.EligibleMachineID[b.MachineID] = true
		}

		// Rates come off the production step and are identical across that step's batches; take the first non-zero rather than averaging noise.
		if !acc.sawMeasurable && b.LaborTimeValue > 0 {
			acc.m.SecondsPerUnit = SecondsPerUnitFromLaborTime(b.LaborTimeValue, b.LaborTime)
			acc.m.OverheadRate = b.OverheadRate
			acc.sawMeasurable = true
		}
		if acc.m.UnitCost == 0 && b.UnitCost > 0 {
			acc.m.UnitCost = b.UnitCost
		}
		if acc.m.ProductionStepID == "" {
			acc.m.ProductionStepID = b.ProductionStepID
		}

		if b.RunCreatedAt != nil && !b.ScannedAt.IsZero() {
			days := b.ScannedAt.Sub(*b.RunCreatedAt).Hours() / 24
			if days >= 0 && days < leadTimeSampleMaxDays {
				acc.leadTimeDays = append(acc.leadTimeDays, days)
			}
		}
	}

	out := make([]ItemMeasurement, 0, len(byItem))
	for _, acc := range byItem {
		if len(acc.leadTimeDays) > 0 {
			var total float64
			for _, d := range acc.leadTimeDays {
				total += d
			}
			acc.m.MeasuredLeadTimeWeeks = (total / float64(len(acc.leadTimeDays))) / 7
			acc.m.LeadTimeSampleCount = len(acc.leadTimeDays)
		}
		out = append(out, acc.m)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out
}

// AverageInputsAdded measures how many new inputs a typical product transition introduces, which is what calibrates the changeover slope.
//
// Campaigns are derived per machine by run-length-encoding the scan-ordered item sequence: consecutive batches of the same item are one campaign, so one changeover. stepInputs maps a production step to the set of input items it consumes.
func AverageInputsAdded(batches []BatchMeasurement, stepInputs map[string]map[string]bool) float64 {
	byMachine := make(map[string][]BatchMeasurement)
	for _, b := range batches {
		if b.MachineID == "" {
			continue
		}
		byMachine[b.MachineID] = append(byMachine[b.MachineID], b)
	}

	machineIDs := make([]string, 0, len(byMachine))
	for id := range byMachine {
		machineIDs = append(machineIDs, id)
	}
	sort.Strings(machineIDs)

	var transitions int
	var totalAdded int

	for _, machineID := range machineIDs {
		machineBatches := byMachine[machineID]
		sort.SliceStable(machineBatches, func(i, j int) bool {
			if !machineBatches[i].ScannedAt.Equal(machineBatches[j].ScannedAt) {
				return machineBatches[i].ScannedAt.Before(machineBatches[j].ScannedAt)
			}
			return machineBatches[i].BatchID < machineBatches[j].BatchID
		})

		var previousStepID string
		var havePrevious bool
		var previousItemID string

		for _, b := range machineBatches {
			if havePrevious && b.ItemID == previousItemID {
				continue // same campaign, no changeover
			}
			if havePrevious {
				transitions++
				totalAdded += countInputsAdded(stepInputs[previousStepID], stepInputs[b.ProductionStepID])
			}
			previousItemID = b.ItemID
			previousStepID = b.ProductionStepID
			havePrevious = true
		}
	}

	if transitions == 0 {
		return 0
	}
	return float64(totalAdded) / float64(transitions)
}

// countInputsAdded counts inputs present in next but not in previous. Only additions count: removing a yarn costs nothing, threading a new one does.
func countInputsAdded(previous, next map[string]bool) int {
	var added int
	for itemID := range next {
		if !previous[itemID] {
			added++
		}
	}
	return added
}

// MeanFloat is a small helper used by the measurement passes.
func MeanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}

// StdDevFloat returns the sample standard deviation, or zero for fewer than two observations.
func StdDevFloat(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := MeanFloat(values)
	var sumSq float64
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	variance := sumSq / float64(len(values)-1)
	if variance <= 0 {
		return 0
	}
	return math.Sqrt(variance)
}
