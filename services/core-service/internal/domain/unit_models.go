package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type Unit struct {
	ID                string
	Name              string
	Abbreviation      string
	UnitDimensionCode string
	RatioNumerator    string
	RatioDenominator  string
	OffsetNumerator   string
	OffsetDenominator string
	IsBaseUnit        bool
	AccountID         *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ListUnitsParams struct {
	AccountID    string
	Cursor       *string
	Limit        int32
	Query        *string
	Type         *string
	UnitGroupIDs []string
}

type ListUnitsResult struct {
	Units    []*Unit
	PageInfo pagination.PageInfo
}

type GetUnitParams struct {
	AccountID string
	UnitID    string
}

type CreateUnitParams struct {
	AccountID         string
	Name              string
	Abbreviation      string
	UnitDimensionCode string
	RatioNumerator    string
	RatioDenominator  string
	OffsetNumerator   string
	OffsetDenominator string
	IsBaseUnit        bool
}

type UpdateUnitParams struct {
	AccountID         string
	UnitID            string
	Name              *string
	Abbreviation      *string
	RatioNumerator    *string
	RatioDenominator  *string
	OffsetNumerator   *string
	OffsetDenominator *string
}

type DeleteUnitParams struct {
	AccountID string
	UnitID    string
}
