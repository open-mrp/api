package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type Unit struct {
	ID                string
	Name              string `audit:"name"`
	Abbreviation      string `audit:"abbreviation"`
	UnitDimensionCode string `audit:"unit_dimension_code"`
	RatioNumerator    string `audit:"ratio_numerator"`
	RatioDenominator  string `audit:"ratio_denominator"`
	OffsetNumerator   string `audit:"offset_numerator"`
	OffsetDenominator string `audit:"offset_denominator"`
	IsBaseUnit        bool   `audit:"is_base_unit"`
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

// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportUnitsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type ListUnitsResult struct {
	Units    []*Unit
	PageInfo pagination.PageInfo
}

type GetUnitParams struct {
	AccountID string
	UnitID    string
}

type UpsertUnitTxParams struct {
	Unit    *UpsertUnitParams
	OldUnit *Unit
}

type UpsertUnitParams struct {
	Name              string
	Abbreviation      string
	UnitDimensionCode string
	RatioNumerator    string
	RatioDenominator  string
	OffsetNumerator   string
	OffsetDenominator string
	IsBaseUnit        bool
}

type BulkUpsertUnitsParams struct {
	Units []UpsertUnitParams
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

// ValidateUnitsParams holds parameters for validating units by abbreviation.
type ValidateUnitsParams struct {
	AccountID string
	UnitMap   map[string]string // key -> abbreviation
}

// ValidateUnitsResult contains matched units keyed by the original map key.
type ValidateUnitsResult struct {
	Units map[string]*Unit
}

type UpsertUnitResult struct {
	Unit *Unit
}
