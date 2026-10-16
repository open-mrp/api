package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const (
	SamplePriceListID     = "prls_x8k2m4q7v1np"
	SamplePriceListRuleID = "prlsrw_h3j7t2w9c5qd"
)

// A set of pricing rules you assign to customers or account groups.
//
// A customer's line price comes from the most specific matching rule across the lists that apply to it: its own, its parent customer's, and its group's.
type PriceList struct {
	// Price list ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=price_list"`
	// Whether the list's rules apply.
	//
	// `archived` lists stop applying to new orders immediately. Orders already priced keep their prices.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Name of the list, such as `Distributor pricing`.
	Name string `json:"name" validate:"required"`
	// What the list is for. `null` when not set.
	Description *string `json:"description"`
	// Three-letter ISO 4217 currency code of the list's prices, lowercase. Always `usd`.
	Currency constants.Currency `json:"currency" validate:"required"`
	// The list's rules, newest first.
	//
	// `null` unless included. Holds the first 10; when there are more, `page_info.next_page_url` continues at List Price List Rules.
	Rules *List[PriceListRule] `json:"rules" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A price for a scope of items.
//
// Empty scope lists match any item. An item must have every listed attribute.
type PriceListRule struct {
	// Price list rule ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=price_list_rule"`
	// The items the rule prices.
	Scope *PriceListRuleScope `json:"scope" validate:"required"`
	// The price this rule sets.
	//
	// `fixed`: exactly `unit_price`. `percent_of_base`: that percent of the item's `sales.unit_price`, so `"80"` is 20% off and `"120"` is a 20% markup. `tiered`: the price depends on the total quantity ordered across every line in the rule's scope, counted in `unit`; the highest tier reached applies.
	Price *PriceListPrice `json:"price" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// The items a price-list rule prices. Each list is empty to match any item; an item must match every list that isn't.
type PriceListRuleScope struct {
	// Items the rule prices. `null` unless included.
	Items *List[Item] `json:"items" expandable:"true"`
	// Product lines whose items the rule prices. `null` unless included.
	ProductLines *List[ProductLine] `json:"product_lines" expandable:"true"`
	// Item categories whose items the rule prices. `null` unless included.
	Categories *List[ItemCategory] `json:"categories" expandable:"true"`
	// Attributes an item must all have for the rule to price it. `null` unless included.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
}

// The price a price-list rule sets.
type PriceListPrice struct {
	// How the rule sets the price.
	//
	// - `fixed`: exactly `unit_price`.
	// - `percent_of_base`: `percent_of_base` percent of the item's `sales.unit_price`.
	// - `tiered`: the highest of `tiers` reached by the quantity ordered, counted in `unit`.
	Type constants.PriceListPriceType `json:"type" validate:"required"`
	// The price per unit. `null` unless `type` is `fixed`.
	UnitPrice *UnitPrice `json:"unit_price"`
	// Percent of the item's base sales price, greater than 0 (`"80"` is 20% off, `"235.3"` a markup). `null` unless `type` is `percent_of_base`.
	PercentOfBase *string `json:"percent_of_base" format:"decimal"`
	// The unit quantities are counted in across every order line in the rule's scope. `null` unless `type` is `tiered`.
	Unit *Unit `json:"unit"`
	// The quantity breaks, lowest `min_quantity` first, at most 20. Empty unless `type` is `tiered`.
	Tiers []PriceListTier `json:"tiers" validate:"required"`
}

// One quantity break of a tiered price.
type PriceListTier struct {
	// The quantity, counted in the rule's `unit`, from which this tier applies.
	MinQuantity string `json:"min_quantity" validate:"required" format:"decimal"`
	// Percent of the item's base sales price. `null` when the tier sets `unit_price`.
	PercentOfBase *string `json:"percent_of_base" format:"decimal"`
	// The price per unit. `null` when the tier sets `percent_of_base`.
	UnitPrice *UnitPrice `json:"unit_price"`
}

var SamplePriceListRule = &PriceListRule{
	ID:     SamplePriceListRuleID,
	Object: constants.ObjectTypePriceListRule,
	Scope:  &PriceListRuleScope{},
	Price: &PriceListPrice{
		Type:          constants.PriceListPriceTypePercentOfBase,
		PercentOfBase: new("80"),
		Tiers:         []PriceListTier{},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SamplePriceList = &PriceList{
	ID:          SamplePriceListID,
	Object:      constants.ObjectTypePriceList,
	Status:      constants.ArchiveStatusActive,
	Name:        "Distributor pricing",
	Description: new("Prices for distributors buying by the carton."),
	Currency:    constants.CurrencyUSD,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PriceList) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SamplePriceList) }

func (*PriceListRule) SchemaExample() any { return apiexample.ValidateAndMarshalToMap(SamplePriceListRule) }
