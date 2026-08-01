package scheduling

import (
	"sort"

	"github.com/augno/api/shared/constants"
)

// LotDefault is the lot an item is made in: how many, counted in what.
//
// The unit is not decoration. A doff of sock greige is 60 pairs and a doff of armsleeve greige is 60 eaches; the number is the same and the quantities are not, so a lot size without a unit cannot be reconciled with anything downstream of it.
type LotDefault struct {
	Quantity float64
	UnitID   string
	// Source explains which rule produced this lot, so a plan can say why it is knitting in sixties. Values: item_override, product_line, downstream_product_line, account_default.
	Source string
	// ProductLineID is the line the convention came from, empty for the account default.
	ProductLineID string
}

// Lot sources, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	LotSourceItemOverride          = string(constants.ItemLotSourceItemOverride)
	LotSourceProductLine           = string(constants.ItemLotSourceProductLine)
	LotSourceDownstreamProductLine = string(constants.ItemLotSourceDownstreamProductLine)
	LotSourceAccountDefault        = string(constants.ItemLotSourceAccountDefault)
)

// ProductLineLot is one line's configured lot convention.
type ProductLineLot struct {
	ProductLineID string
	Quantity      float64
	UnitID        string
}

// LotResolutionInput is everything the chain needs, gathered once.
type LotResolutionInput struct {
	// ItemOverrides are per-item lot sizes set by hand. They carry no unit — an override changes how big a lot is, not what it is counted in.
	ItemOverrides map[string]float64
	// ProductLineByItem maps an item to the line it sells under. Intermediate items — greige — are absent, which is what makes downstream inheritance necessary.
	ProductLineByItem map[string]string
	// LotByProductLine is the configured convention per line.
	LotByProductLine map[string]ProductLineLot
	// DownstreamByItem is what each planned item becomes, used to inherit a lot for greige that has no product line of its own.
	DownstreamByItem map[string][]FinishedGood
	// AccountLotUnits is the account-wide fallback size.
	AccountLotUnits float64
	// UnitByItem is each item's own counting unit, used for the account fallback where there is no product line to take a unit from.
	UnitByItem map[string]string
}

// ResolveLotDefault decides what lot one item is made in.
//
// The chain, most specific first:
//
//  1. A per-item override, which changes the size but keeps the item's own unit.
//  2. The item's own product line, for anything that is itself sellable.
//  3. The product lines of what the item becomes. Greige has no product line of its own — it is not sold — so a doff of sock greige takes its lot from the sock line. Where a greige feeds several lines, the one carrying the most demand wins.
//  4. The account default, counted in the item's own unit.
//
// Returns false when nothing in the chain yields a usable lot, which means the item is planned unlotted rather than in a lot of some guessed size.
func ResolveLotDefault(itemID string, in LotResolutionInput) (LotDefault, bool) {
	if override, ok := in.ItemOverrides[itemID]; ok && override > 0 {
		return LotDefault{
			Quantity: override,
			UnitID:   in.UnitByItem[itemID],
			Source:   LotSourceItemOverride,
		}, true
	}

	if lineID, ok := in.ProductLineByItem[itemID]; ok {
		if lot, ok := in.LotByProductLine[lineID]; ok && lot.Quantity > 0 {
			return LotDefault{
				Quantity:      lot.Quantity,
				UnitID:        lot.UnitID,
				Source:        LotSourceProductLine,
				ProductLineID: lineID,
			}, true
		}
	}

	if lot, ok := inheritedLot(itemID, in); ok {
		return lot, true
	}

	if in.AccountLotUnits > 0 {
		return LotDefault{
			Quantity: in.AccountLotUnits,
			UnitID:   in.UnitByItem[itemID],
			Source:   LotSourceAccountDefault,
		}, true
	}

	return LotDefault{}, false
}

// inheritedLot takes a greige item's lot from the finished goods it becomes.
//
// Demand decides between competing lines rather than, say, count of SKUs: a greige that mostly becomes socks should be knitted in the sock line's doff even if it also feeds one low-volume armsleeve. Ties break on product line id so the same plan resolves the same way every time — the levelling is deterministic and this must not be the thing that makes it wobble.
func inheritedLot(itemID string, in LotResolutionInput) (LotDefault, bool) {
	demandByLine := map[string]float64{}
	for _, finished := range in.DownstreamByItem[itemID] {
		if finished.ProductLineID == "" {
			continue
		}
		if _, ok := in.LotByProductLine[finished.ProductLineID]; !ok {
			continue
		}
		// Seeded at zero first: a finished good with no demand history still votes, or a brand-new SKU would be invisible to the choice.
		if _, seen := demandByLine[finished.ProductLineID]; !seen {
			demandByLine[finished.ProductLineID] = 0
		}
		for _, month := range finished.Monthly {
			demandByLine[finished.ProductLineID] += month.Quantity
		}
	}
	if len(demandByLine) == 0 {
		return LotDefault{}, false
	}

	lineIDs := make([]string, 0, len(demandByLine))
	for lineID := range demandByLine {
		lineIDs = append(lineIDs, lineID)
	}
	sort.Strings(lineIDs)

	best, bestDemand := "", -1.0
	for _, lineID := range lineIDs {
		if demandByLine[lineID] > bestDemand {
			best, bestDemand = lineID, demandByLine[lineID]
		}
	}

	lot := in.LotByProductLine[best]
	return LotDefault{
		Quantity:      lot.Quantity,
		UnitID:        lot.UnitID,
		Source:        LotSourceDownstreamProductLine,
		ProductLineID: best,
	}, true
}
