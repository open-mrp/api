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
	Includes         []string
}

type UpdateProductLineParams struct {
	AccountID        string
	ProductLineID    string
	Name             *string
	CommissionPolicy *constants.CommissionPolicy
	FreightPolicy    *constants.FreightPolicy
	UnitGroupID      *string
	Includes         []string
}

type DeleteProductLineParams struct {
	AccountID     string
	ProductLineID string
}
