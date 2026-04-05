package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// UnitGroupFull represents a full unit group with its base unit and conversions.
type UnitGroupFull struct {
	ID              string
	Name            string  `audit:"name"`
	Notes           *string `audit:"notes"`
	Type            string  `audit:"type"`
	BaseUnit        LightUnit
	UnitConversions []*UnitGroupUnit
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
}

type ListUnitGroupsResult struct {
	UnitGroups []*UnitGroupFull
	PageInfo   pagination.PageInfo
}

type GetUnitGroupParams struct {
	AccountID   string
	UnitGroupID string
}

type CreateUnitGroupParams struct {
	AccountID       string
	Name            string
	Notes           *string
	Type            string
	BaseUnitID      string
	UnitConversions []CreateUnitGroupUnitParams
}

type CreateUnitGroupUnitParams struct {
	UnitID             string
	DiscountPercentage string
	DiscountFixed      string
	IsVisible          bool
}

type UpdateUnitGroupParams struct {
	AccountID       string
	UnitGroupID     string
	Name            *string
	Notes           **string
	BaseUnitID      *string
	UnitConversions *[]CreateUnitGroupUnitParams
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
}
