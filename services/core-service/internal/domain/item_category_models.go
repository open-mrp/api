package domain

import (
	"time"

	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

// Default category IDs that cannot be mutated.
const (
	DefaultCategoryShipping = "shipping"
	DefaultCategoryService  = "service"
	DefaultCategoryCredit   = "credit"
	DefaultCategoryTax      = "tax"
)

// IsDefaultCategory returns true if the given category ID is a system default.
func IsDefaultCategory(id string) bool {
	switch id {
	case DefaultCategoryShipping, DefaultCategoryService, DefaultCategoryCredit, DefaultCategoryTax:
		return true
	default:
		return false
	}
}

// CategoryRef is the lightweight category lookup used by item create paths and bulk
// upsert validation: the category's base unit and its type code (material_category /
// product_category), which constrains which item types may use it.
type CategoryRef struct {
	BaseUnitID           string
	ItemCategoryTypeCode string
}

// ItemCategoryFull represents a full item category with optional joined data.
// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportItemCategoriesParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type ItemCategoryFull struct {
	ID                   string
	Name                 string  `audit:"name"`
	Notes                *string `audit:"notes"`
	ItemCategoryTypeCode string  `audit:"item_category_type_code"`
	UnitGroupID          string  `audit:"unit_group_id"`
	// The lot the category's items are made in, as a value and its unit; both nil when the category has none.
	DefaultLotSizeValue  *string `audit:"default_lot_size_value"`
	DefaultLotSizeUnitID *string `audit:"default_lot_size_unit_id"`
	AccountID            *string
	CreatedAt            time.Time
	UpdatedAt            time.Time

	// Expandable sub-resources (populated via includes)
	Properties []*ItemCategoryProperty `audit:"properties"`
	UnitGroup  *ItemCategoryUnitGroup
}

// ItemCategoryProperty represents a property associated with an item category.
type ItemCategoryProperty struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ItemCategoryUnitGroup represents the unit group associated with an item category.
type ItemCategoryUnitGroup struct {
	ID         string
	Name       string
	BaseUnitID string
	Type       string
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Populated when unit_group.base_unit is included.
	BaseUnit *LightUnit
	// Populated when unit_group.associated_units is included.
	AssociatedUnits []*UnitGroupUnit
}

type ListItemCategoriesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Type      *string
	Includes  []string
}

type ListItemCategoriesResult struct {
	ItemCategories []*ItemCategoryFull
	PageInfo       pagination.PageInfo
}

type GetItemCategoryParams struct {
	AccountID      string
	ItemCategoryID string
	Includes       []string
}

type CreateItemCategoryParams struct {
	AccountID            string
	Name                 string
	Notes                *string
	ItemCategoryTypeCode string
	// UnitGroupID empty uses DefaultItemCategoryUnitGroupID.
	UnitGroupID string
	// PropertyIDs are attached to the new category.
	PropertyIDs []string
	// DefaultLotSize is the lot the category's items are made in; nil for none.
	DefaultLotSize *LotQuantityInput
	Includes       []string
}

// DefaultItemCategoryUnitGroupID is the system group a category gets when none is named: items in it are
// counted in each.
const DefaultItemCategoryUnitGroupID = "each_group"

type UpdateItemCategoryParams struct {
	AccountID      string
	ItemCategoryID string
	Name           *string
	Notes          *string
	// PropertyIDs, when non-nil, replaces the category's properties.
	PropertyIDs *[]string
	// DefaultLotSize sets or clears the lot the category's items are made in.
	DefaultLotSize field.Clearable[LotQuantityInput]
	Includes       []string
}

type UpdateItemCategoryWithUnitGroupParams struct {
	AccountID      string
	ItemCategoryID string
	Name           *string
	Notes          *string
	UnitGroupID    string
}

type DeleteItemCategoryParams struct {
	AccountID      string
	ItemCategoryID string
}

type AddItemCategoryPropertyParams struct {
	AccountID      string
	ItemCategoryID string
	PropertyID     string
}

// CreateItemCategoryPropertyParams names a new property to create in the account and attach to an item category.
type CreateItemCategoryPropertyParams struct {
	AccountID      string
	ItemCategoryID string
	Name           string
}

type RemoveItemCategoryPropertyParams struct {
	AccountID      string
	ItemCategoryID string
	PropertyID     string
}

type ChangeItemCategoryUnitGroupParams struct {
	AccountID      string
	ItemCategoryID string
	UnitGroupID    string
}

// UpsertItemCategoryParams holds the fields for a single location in a bulk upsert.
type UpsertItemCategoryParams struct {
	Name                 string
	Notes                *string
	ItemCategoryTypeCode string
	UnitGroup            ObjectIdentifier
	// PropertyNames is an optional list of property names to attach to this category.
	// Properties are matched by name (case-insensitive) within the account; names not
	// found are created automatically. Relations are additive — existing relations are
	// not removed.
	PropertyNames []string
}

// mirrors an upsert row with its unit group reference resolved to an id. Property names
// stay as written: they are found-or-created when the job runs, so nothing to resolve.
type ResolvedUpsertItemCategoryRow struct {
	Name                 string
	Notes                *string
	ItemCategoryTypeCode string
	UnitGroupID          string
	PropertyNames        []string
}

// carries the rows of a bulk item category upsert
type BulkUpsertItemCategoriesParams struct {
	ItemCategories []UpsertItemCategoryParams
}

// ItemCategoryReferences blocks deleting a category that items or price-list rules still use. The legacy account
// prices and volume discounts mirror into rules, so they read as rules too.
var ItemCategoryReferences = RegisterReferences(References{
	Noun: "item category",
	Probes: []ReferenceProbe{
		{Label: "items", Table: "item", Column: "item_category_id"},
		{Label: "price-list rules", Table: "price_list_rule_item_category", Column: "item_category_id"},
		{Label: "price-list rules", Table: "account_price_item_category", Column: "item_category_id"},
		{Label: "price-list rules", Table: "_item_categories_quantity_discounts", Column: "A"},
	},
})
