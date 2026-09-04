package scheduling

// Changeover models setup time as a function of how many inputs the next product introduces. Threading one extra yarn is quick; threading eight is not, and a flat average hides exactly the sequencing decision the planner is trying to make.
//
//	minutes(added) = clamp(min + slope * added, min, max)
//
// The slope is calibrated from history rather than configured, so the modeled average lands on the number the floor actually reports.
type Changeover struct {
	minMinutes float64
	maxMinutes float64
	slope      float64
}

// CalibrateChangeover solves for the slope that makes the model reproduce avgMinutes over the observed transitions.
//
// avgInputsAdded is the mean number of new inputs across historical product transitions. With no history (or no variation) the slope is zero and every changeover costs the minimum — deliberately optimistic, but the alternative is inventing a slope from nothing.
//
// Script: CO_SLOPE = (CO_AVG_MIN - CO_MIN_MIN) / avgYarnsAdded.
func CalibrateChangeover(minMinutes, avgMinutes, maxMinutes, avgInputsAdded float64) Changeover {
	c := Changeover{minMinutes: minMinutes, maxMinutes: maxMinutes}
	if avgInputsAdded > 0 {
		c.slope = (avgMinutes - minMinutes) / avgInputsAdded
	}
	if c.maxMinutes < c.minMinutes {
		c.maxMinutes = c.minMinutes
	}
	return c
}

// Minutes returns the modeled changeover time for a transition that introduces inputsAdded new inputs.
func (c Changeover) Minutes(inputsAdded int) float64 {
	if inputsAdded < 0 {
		inputsAdded = 0
	}
	minutes := c.minMinutes + c.slope*float64(inputsAdded)
	if minutes < c.minMinutes {
		return c.minMinutes
	}
	if minutes > c.maxMinutes {
		return c.maxMinutes
	}
	return minutes
}

// Slope exposes the calibrated minutes-per-added-input for diagnostics.
func (c Changeover) Slope() float64 { return c.slope }

// SetupCost is what one changeover costs, and therefore the "S" in the EOQ formula.
//
// It uses a dedicated technician rate rather than the per-machine-hour production labor allocation: a tech works the single machine through the changeover, so the thin allocated rate understates it and EOQ would come out too small — meaning too many short campaigns and more changeovers than the floor can absorb.
func SetupCost(avgMinutes, laborRate, overheadRate float64) float64 {
	rate := laborRate + overheadRate
	if rate <= 0 {
		// Script falls back to 8/hr rather than producing a zero setup cost, which would collapse EOQ toward a lot size of one.
		rate = 8
	}
	return (avgMinutes / 60) * rate
}
