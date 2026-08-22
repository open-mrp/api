package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
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
	UnitGroupID          string
	Includes             []string
}

type UpdateItemCategoryParams struct {
	AccountID      string
	ItemCategoryID string
	Name           *string
	Notes          *string
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
