package domain

import "time"

// RateValue is a minimal currency-per-unit rate: a value with numerator and denominator unit ids. Used as the override input and as the engine output.
type RateValue struct {
	Value             string
	NumeratorUnitID   string
	DenominatorUnitID string
}

// SalesOrderPriceLineInput is one line to price. The volume-discount stage sums quantities across all lines that share a chosen discount, so the engine takes every line at once.
type SalesOrderPriceLineInput struct {
	ProductID         string
	QuantityValue     string
	QuantityUnitID    string
	OverrideUnitPrice *RateValue
}

// SalesOrderLinePrice is the computed unit price for a single line, expressed as currency (numerator) per ordered unit (denominator).
type SalesOrderLinePrice struct {
	Value             string
	NumeratorUnitID   string
	DenominatorUnitID string
}

// QuoteSalesOrderLinePricesParams holds the parameters for a price quote.
type QuoteSalesOrderLinePricesParams struct {
	AccountID      string
	BuyerAccountID string
	Lines          []SalesOrderPriceLineInput
}

// SalesOrderLineQuote is one priced line returned by a quote (no order is created).
type SalesOrderLineQuote struct {
	ProductID string
	UnitPrice RateValue
}

// QuoteSalesOrderFreightParams identifies the existing order whose freight is re-estimated.
type QuoteSalesOrderFreightParams struct {
	AccountID    string
	SalesOrderID string
}

// SalesOrderFreightQuote is the freshly estimated freight (shipping) charge for an order, expressed as currency (numerator) per shipping unit (denominator). No order is mutated.
type SalesOrderFreightQuote struct {
	UnitPrice RateValue
}

// ResolvedSalesOrderLine is a fully resolved create-line: the caller's product + quantity, with the SKU/description defaulted from the product, the item derived from the product, the unit cost pulled from the item, and the unit price computed server-side (or taken from an internal override). Produced by resolving the create line inputs against the pricing bundle in one pass.
type ResolvedSalesOrderLine struct {
	ProductID          string
	ItemID             string
	ProductSKU         string
	ProductDescription *string
	QuantityValue      string
	QuantityUnitID     string
	UnitPrice          RateValue
	UnitCost           RateValue
}

// --- Pricing data bundle loaded from the repository -------------------------

// PricingProduct is the list-price data the engine needs for one product.
type PricingProduct struct {
	ProductID string
	// ItemID, SKU, Description are item-derived fields used to tie the line to inventory and default the line's recorded SKU/description.
	ItemID      string
	SKU         string
	Description *string
	// UnitCost is the item's cost rate (pulled server-side, never a caller input).
	UnitCost                  string
	UnitCostNumeratorUnitID   string
	UnitCostDenominatorUnitID string
	// ProductLineID is nil when the product has no product line. Such products never match an account price and use the item category's unit group.
	ProductLineID *string
	// UnitValue is the item's list-price rate value (currency numerator per item-category base-unit denominator).
	UnitValue                  string
	UnitValueNumeratorUnitID   string
	UnitValueDenominatorUnitID string
	// ProductLineUnitGroupID is set when the product has a product line.
	ProductLineUnitGroupID *string
	// CategoryUnitGroupID is the item category's unit group (fallback group).
	CategoryUnitGroupID string
	// ItemCategoryID is the item's category id, used for volume-discount category scoping.
	ItemCategoryID string
	// AttributeIDs are the product's (item's) attribute ids.
	AttributeIDs []string
}

// PricingUnit carries the raw unit-conversion data for normalizeQuantity.
type PricingUnit struct {
	ID                string
	RatioNumerator    string
	RatioDenominator  string
	OffsetNumerator   string
	OffsetDenominator string
	IsBaseUnit        bool
}

// PricingUnitGroupUnit is a unit's conversion discount within a unit group.
type PricingUnitGroupUnit struct {
	UnitGroupID        string
	UnitID             string
	DiscountPercentage string
	DiscountFixed      string
}

// PricingAccountPrice is an absolute price override for a product line and recipient. The recipient is either the buyer or its parent account.
type PricingAccountPrice struct {
	ID            string
	ProductLineID string
	UnitValue     string
	// Numerator/Denominator unit ids of the override rate.
	NumeratorUnitID   string
	DenominatorUnitID string
	AttributeIDs      []string
	CreatedAt         time.Time
}

// PricingVolumeDiscount is a quantity discount applicable to the buyer, with its tiers and the set of units a line quantity may be normalized into.
type PricingVolumeDiscount struct {
	ID                   string
	MatchesCustomerGroup bool
	Tiers                []PricingVolumeDiscountTier
	AcceptableUnitIDs    []string
	// Scoping: a product matches the discount only if it satisfies every non-empty dimension (product line AND item category AND attributes). An empty dimension is a wildcard. Customer-group scoping is already applied when the discount is loaded.
	ProductLineIDs []string
	CategoryIDs    []string
	AttributeIDs   []string
}

// PricingVolumeDiscountTier is one threshold/percentage tier of a discount.
type PricingVolumeDiscountTier struct {
	Threshold          string
	DiscountPercentage string
}

// PricingBundle is everything the engine needs for a single computation, keyed for O(1) lookup. Loaded in one repository call.
type PricingBundle struct {
	// Products keyed by product id (only products that exist for the account).
	Products map[string]*PricingProduct
	// Units keyed by unit id.
	Units map[string]*PricingUnit
	// UnitGroupUnits keyed by unitGroupID -> unitID.
	UnitGroupUnits map[string]map[string]*PricingUnitGroupUnit
	// AccountPrices in created_at ASC order (last match wins, so callers iterate and keep the last applicable).
	AccountPrices []*PricingAccountPrice
	// VolumeDiscounts applicable to the buyer.
	VolumeDiscounts []*PricingVolumeDiscount
}

// LoadPricingBundleParams identifies what to load for a pricing computation.
type LoadPricingBundleParams struct {
	OwnerAccountID string
	BuyerAccountID string
	ProductIDs     []string
	// OrderedUnitIDs are the per-line ordered unit ids; combined with the products' base denominator units to fetch all needed conversion units.
	OrderedUnitIDs []string
}
