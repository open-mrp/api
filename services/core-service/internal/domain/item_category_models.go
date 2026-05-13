package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
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

// ItemCategoryFull represents a full item category with optional joined data.
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
