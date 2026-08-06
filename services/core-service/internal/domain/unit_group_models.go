package domain

import (
	"time"

	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/pagination"
)

// UnitGroupFull represents a full unit group with its base unit and conversions.
// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportUnitGroupsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type UnitGroupFull struct {
	ID              string
	Name            string           `audit:"name"`
	Notes           *string          `audit:"notes"`
	Type            string           `audit:"type"`
	BaseUnit        LightUnit        `audit:"base_unit"`
	UnitConversions []*UnitGroupUnit `audit:"associated_units"`
	AccountID       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UnitGroupUnit represents a unit conversion within a unit group.
type UnitGroupUnit struct {
	ID                 string
	UnitID             string `audit:"unit_id"`
	UnitGroupID        string
	DiscountPercentage string `audit:"discount_percentage"`
	DiscountFixed      string `audit:"discount_fixed"`
	IsVisible          bool   `audit:"is_visible"`
	Unit               LightUnit
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ListUnitGroupsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Type      *string
	Includes  []string
}

type ListUnitGroupsResult struct {
	UnitGroups []*UnitGroupFull
	PageInfo   pagination.PageInfo
}

type GetUnitGroupParams struct {
	AccountID   string
	UnitGroupID string
	Includes    []string
}

type UnitGroupExistsParams struct {
	AccountID   string
	UnitGroupID string
}

// CreateUnitGroupParams is the service-level params for creating a unit group with its initial conversions.
type CreateUnitGroupParams struct {
	AccountID       string
	Name            string
	Notes           *string
	Type            string
	BaseUnitID      string
	UnitConversions []CreateUnitGroupUnitParams
	Includes        []string
}

// CreateUnitGroupUnitParams describes a single unit conversion to attach to a group.
type CreateUnitGroupUnitParams struct {
	UnitID             string
	DiscountPercentage string
	DiscountFixed      string
	IsVisible          bool
}

// UpdateUnitGroupParams is the service-level params for updating a unit group and optionally upserting conversions.
type UpdateUnitGroupParams struct {
	AccountID       string
	UnitGroupID     string
	Name            *string
	Notes           field.Clearable[string]
	BaseUnitID      *string
	UnitConversions *[]CreateUnitGroupUnitParams
	Includes        []string
}

type DeleteUnitGroupParams struct {
	AccountID   string
	UnitGroupID string
}

type UpsertUnitGroupUnitParams struct {
	AccountID          string
	UnitGroupID        string
	UnitGroupUnitID    string
	UnitID             string
	DiscountPercentage string
	DiscountFixed      string
	IsVisible          bool
	// IsVisibleProvided reports whether the caller supplied is_visible. When false, the upsert preserves the stored value on update (or defaults to true on create) rather than clobbering it to false.
	IsVisibleProvided bool
	Includes          []string
}

type UpsertUnitConversionParams struct {
	Unit               UnitIdentifier
	DiscountPercentage string
}

type UpsertUnitGroupParams struct {
	Name            string
	Notes           *string
	Type            string
	BaseUnit        UnitIdentifier
	UnitConversions []UpsertUnitConversionParams
}

// ResolvedUnitGroupConversion is one conversion after resolution.
type ResolvedUnitGroupConversion struct {
	UnitID             string
	DimensionCode      string
	DiscountPercentage string
}

// ResolvedUpsertUnitGroupRow is a unit-group upsert row after resolution.
type ResolvedUpsertUnitGroupRow struct {
	Name       string
	Notes      *string
	Type       string
	BaseUnitID string
	// BaseUnitDimensionCode is the resolved base unit's dimension, carried so Write can
	// check it against the group's (stored, immutable) type without a further lookup —
	// exactly as the conversions do.
	BaseUnitDimensionCode string
	Conversions           []ResolvedUnitGroupConversion
}

type BulkUpsertUnitGroupsParams struct {
	UnitGroups []UpsertUnitGroupParams
}

type DeleteUnitGroupUnitParams struct {
	AccountID       string
	UnitGroupID     string
	UnitGroupUnitID string
}

type GetUnitGroupUnitParams struct {
	AccountID       string
	UnitGroupID     string
	UnitGroupUnitID string
	Includes        []string
}
