package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// Default product line IDs that cannot be mutated.
const (
	DefaultProductLineShipping = "shipping"
	DefaultProductLineService  = "service"
	DefaultProductLineCredit   = "credit"
	DefaultProductLineTax      = "tax"
)

// IsDefaultProductLine returns true if the given product line ID is a system default.
func IsDefaultProductLine(id string) bool {
	switch id {
	case DefaultProductLineShipping, DefaultProductLineService, DefaultProductLineCredit, DefaultProductLineTax:
		return true
	default:
		return false
	}
}

// ProductLineFull represents a full product line with optional joined data.
type ProductLineFull struct {
	ID               string
	Name             string                     `audit:"name"`
	Description      *string                    `audit:"description"`
	Notes            *string                    `audit:"notes"`
	CommissionPolicy constants.CommissionPolicy `audit:"commission_policy"`
	FreightPolicy    constants.FreightPolicy    `audit:"freight_policy"`
	UnitGroupID      string                     `audit:"unit_group_id"`
	// The lot products in this line are made in, carried as a quantity so the number and its unit stay one value: 60 pairs and 60 eaches are different lots, and a size on its own cannot say which. Flattened here the way credit_limit is on a customer.
	DefaultLotID     *string `audit:"default_lot_id"`
	DefaultLotValue  *string `audit:"default_lot_value"`
	DefaultLotUnitID *string `audit:"default_lot_unit_id"`
	AccountID        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Expandable sub-resources (populated via includes)
	UnitGroup *ProductLineUnitGroup
}

// ProductLineUnitGroup represents the unit group associated with a product line.
type ProductLineUnitGroup struct {
	ID         string
	Name       string
	BaseUnitID string
	Type       string
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Populated when product_line.unit_group.base_unit is included.
	BaseUnit *LightUnit
	// Populated when product_line.unit_group.associated_units is included.
	AssociatedUnits []*UnitGroupUnit
}

type ListProductLinesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Includes  []string
}

// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportProductLinesParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type ListProductLinesResult struct {
	ProductLines []*ProductLineFull
	PageInfo     pagination.PageInfo
}

type GetProductLineParams struct {
	AccountID     string
	ProductLineID string
	Includes      []string
}

type CreateProductLineParams struct {
	AccountID        string
	Name             string
	UnitGroupID      string
	CommissionPolicy constants.CommissionPolicy
	FreightPolicy    constants.FreightPolicy
	DefaultLot       *LotQuantityInput
	Includes         []string
}

type UpdateProductLineParams struct {
	AccountID        string
	ProductLineID    string
	Name             *string
	CommissionPolicy *constants.CommissionPolicy
	FreightPolicy    *constants.FreightPolicy
	UnitGroupID      *string
	DefaultLot       *LotQuantityInput
	// ClearDefaultLot removes the line's lot convention entirely.
	ClearDefaultLot bool
	Includes        []string
}

type DeleteProductLineParams struct {
	AccountID     string
	ProductLineID string
}

// UpsertProductLineParams holds the fields for a single product line in a bulk upsert.
type UpsertProductLineParams struct {
	Name             string
	UnitGroup        ObjectIdentifier
	CommissionPolicy constants.CommissionPolicy
	FreightPolicy    constants.FreightPolicy
}

// BulkUpsertProductLinesParams is the input for a bulk upsert of product lines.
type BulkUpsertProductLinesParams struct {
	ProductLines []UpsertProductLineParams
}

// ResolvedUpsertProductLineRow is a product line upsert row with its unit group reference
// resolved to an id. No JSON tags: the engine round-trips job_items against this type, an
// internal column.
type ResolvedUpsertProductLineRow struct {
	Name             string
	UnitGroupID      string
	CommissionPolicy constants.CommissionPolicy
	FreightPolicy    constants.FreightPolicy
}

// ItemLotDefault is the lot one item is made in, resolved through the whole chain.
//
// Quantity is zero when nothing anywhere supplies a lot: the caller gets a unit and no number, so a form defaults to empty rather than to a size nobody chose.
type ItemLotDefault struct {
	ItemID   string
	Quantity float64
	UnitID   string
	// Source names the rule that produced this lot: item_override, product_line, downstream_product_line or account_default.
	Source string
	// ProductLineID is the line the convention came from, empty for the account default.
	ProductLineID string
}

// ProductLineLotDefault is one line's configured lot convention.
type ProductLineLotDefault struct {
	ProductLineID string
	Quantity      float64
	UnitID        string
}

// LotQuantityInput is a lot on its way in: a decimal string and the unit it counts.
//
// Both are required together. A size with no unit cannot say whether 60 means pairs or eaches, which is the distinction the whole setting exists to draw.
type LotQuantityInput struct {
	Value  string
	UnitID string
}
