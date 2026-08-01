package scheduling

import "math"

// MaxLotsPerCampaign bounds how many batches one campaign can be split into.
//
// A misconfigured lot size — one unit per doff, say — would otherwise turn a single week into tens of thousands of batch rows and a production run nobody can work. The release refuses rather than writing them, so the bad setting surfaces as an error instead of as an unusable run.
const MaxLotsPerCampaign = 500

// LotRoundingTolerance is the remainder below which a final short lot is folded into the previous one instead of becoming a batch of its own.
//
// Planned quantities are decimals, so a campaign that is conceptually six whole doffs can arrive as 359.9999999. Without this a release would emit a seventh batch of a millionth of a unit.
const LotRoundingTolerance = 1e-6

// SplitIntoLots breaks a planned quantity into the batch sizes the floor actually runs.
//
// Full lots come first and the remainder, if any, trails as one short lot. That ordering matters on the floor: the short doff is the one that gets cut when a week runs late, so it belongs at the end of the run rather than buried in the middle.
//
// A non-positive lot size means the item is not lotted, so the campaign is one batch.
func SplitIntoLots(quantity, lotUnits float64) []float64 {
	if quantity <= 0 {
		return nil
	}
	if lotUnits <= 0 || lotUnits >= quantity {
		return []float64{quantity}
	}

	full := int(math.Floor(quantity/lotUnits + LotRoundingTolerance))
	remainder := quantity - float64(full)*lotUnits

	lots := make([]float64, 0, full+1)
	for i := 0; i < full; i++ {
		lots = append(lots, lotUnits)
	}
	if remainder > LotRoundingTolerance {
		lots = append(lots, remainder)
	}

	return lots
}

// CountLots reports how many batches SplitIntoLots would produce, without building them. Used to validate a release before any row is written.
func CountLots(quantity, lotUnits float64) int {
	return len(SplitIntoLots(quantity, lotUnits))
}
