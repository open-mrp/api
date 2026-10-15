package scheduling

import (
	"github.com/open-mrp/api/shared/constants"
)

// LotDefault is the lot an item is made in: how many, counted in what.
//
// The unit is not decoration. A doff of sock greige is 60 pairs and a doff of armsleeve greige is 60 eaches; the number is the same and the quantities are not, so a lot size without a unit cannot be reconciled with anything downstream of it.
type LotDefault struct {
	Quantity float64
	UnitID   string
	// Source explains which rule produced this lot, so a plan can say why it is knitting in sixties. Values: item_override, item_category, account_default.
	Source string
	// ItemCategoryID is the category the lot came from, empty for the other sources.
	ItemCategoryID string
}

// Lot sources, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	LotSourceItemOverride   = string(constants.ItemLotSourceItemOverride)
	LotSourceItemCategory   = string(constants.ItemLotSourceItemCategory)
	LotSourceAccountDefault = string(constants.ItemLotSourceAccountDefault)
)

// CategoryLot is the lot of the category one item is in.
type CategoryLot struct {
	ItemCategoryID string
	Quantity       float64
	UnitID         string
}

// LotResolutionInput is everything the chain needs, gathered once.
type LotResolutionInput struct {
	// ItemOverrides are per-item lot sizes set by hand. They carry no unit — an override changes how big a lot is, not what it is counted in.
	ItemOverrides map[string]float64
	// CategoryLotByItem is the lot of each item's category, for items whose category has one.
	CategoryLotByItem map[string]CategoryLot
	// AccountLotUnits is the account-wide fallback size.
	AccountLotUnits float64
	// UnitByItem is each item's own counting unit, used where the lot does not carry one of its own.
	UnitByItem map[string]string
}

// ResolveLotDefault decides what lot one item is made in.
//
// The chain, most specific first:
//
//  1. A per-item override, which changes the size but keeps the item's own unit.
//  2. The item's category. Intermediate items such as greige sit in categories of their own, so they carry their own lot rather than borrowing one from what they become.
//  3. The account default, counted in the item's own unit.
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

	if lot, ok := in.CategoryLotByItem[itemID]; ok && lot.Quantity > 0 {
		return LotDefault{
			Quantity:       lot.Quantity,
			UnitID:         lot.UnitID,
			Source:         LotSourceItemCategory,
			ItemCategoryID: lot.ItemCategoryID,
		}, true
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
