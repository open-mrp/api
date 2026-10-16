package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
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
	// UnitGroupID is empty for a line whose products are measured in their category's group.
	UnitGroupID string `audit:"unit_group_id"`
	// Supply is where the line's products come from when they don't say for themselves; nil falls through to the account default.
	Supply    *constants.Supply       `audit:"supply"`
	Status    constants.ArchiveStatus `audit:"status"`
	AccountID *string
	CreatedAt time.Time
	UpdatedAt time.Time

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
	// IsActive narrows to active (true) or archived (false) lines; nil returns both.
	IsActive *bool
	Includes []string
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
	Description      *string
	UnitGroupID      *string
	CommissionPolicy constants.CommissionPolicy
	FreightPolicy    constants.FreightPolicy
	Supply           *constants.Supply
	Status           constants.ArchiveStatus
	Includes         []string
}

type UpdateProductLineParams struct {
	AccountID        string
	ProductLineID    string
	Name             *string
	Description      field.Clearable[string]
	CommissionPolicy *constants.CommissionPolicy
	FreightPolicy    *constants.FreightPolicy
	UnitGroupID      *string
	Supply           field.Clearable[constants.Supply]
	Status           *constants.ArchiveStatus
	Includes         []string
}

type DeleteProductLineParams struct {
	AccountID     string
	ProductLineID string
}

// UpsertProductLineParams holds the fields for a single product line in a bulk upsert.
type UpsertProductLineParams struct {
	Name string
	// UnitGroup is optional; a zero identifier leaves the line without a unit group.
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
	// Source names the rule that produced this lot: item_override, item_category or account_default.
	Source string
	// ItemCategoryID is the category the lot came from, empty for the other sources.
	ItemCategoryID string
}

// LotQuantityInput is a lot on its way in: a decimal string and the unit it counts.
//
// Both are required together. A size with no unit cannot say whether 60 means pairs or eaches, which is the distinction the whole setting exists to draw.
type LotQuantityInput struct {
	Value  string
	UnitID string
}

// ProductLineReferences blocks deleting a line that products, prices, sales targets or territories still use.
var ProductLineReferences = RegisterReferences(References{
	Noun:       "product line",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "products", Table: "product", Column: "product_line_id",
			Where: "EXISTS (SELECT 1 FROM `item` i WHERE i.id = `product`.`item_id` AND i.deleted_at IS NULL)"},
		{Label: "price-list rules", Table: "price_list_rule_product_line", Column: "product_line_id"},
		{Label: "price-list rules", Table: "account_price", Column: "product_line_id"},
		{Label: "price-list rules", Table: "_product_lines_quantity_discounts", Column: "A"},
		{Label: "sales targets", Table: "product_line_target", Column: "product_line_id"},
		{Label: "territories", Table: "territory", Column: "product_line_id"},
	},
})
