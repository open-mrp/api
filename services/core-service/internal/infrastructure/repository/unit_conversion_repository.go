package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
)

type unitConversionRepo struct {
	queries *sqlc.Queries
}

func NewUnitConversionRepo(queries *sqlc.Queries) domain.UnitConversionRepo {
	return &unitConversionRepo{queries: queries}
}

// ConvertValue converts a measure from one unit to another.
// Formula: toBase = (value * ratioNum / ratioDen) + (offsetNum / offsetDen)
// fromBase = (baseValue - offsetNum / offsetDen) * ratioDen / ratioNum
func (r *unitConversionRepo) ConvertValue(ctx context.Context, measure decimal.Decimal, fromUnitID, toUnitID string) (decimal.Decimal, *apierror.APIError) {
	if fromUnitID == toUnitID {
		return measure, nil
	}

	rows, err := r.queries.GetUnitConversionFactorsPair(ctx, sqlc.GetUnitConversionFactorsPairParams{
		FromUnitID: fromUnitID,
		ToUnitID:   toUnitID,
	})
	if err != nil {
		return decimal.Zero, apierror.NewInternalError(err, "Failed to fetch unit conversion factors.")
	}

	if len(rows) < 2 {
		return decimal.Zero, apierror.NewInternalError(
			fmt.Errorf("expected 2 units, got %d for %s and %s", len(rows), fromUnitID, toUnitID),
			"Failed to find both units for conversion.",
		)
	}

	var fromRow, toRow sqlc.GetUnitConversionFactorsPairRow
	for _, row := range rows {
		if row.ID == fromUnitID {
			fromRow = row
		}
		if row.ID == toUnitID {
			toRow = row
		}
	}

	// Parse decimal values from strings.
	fromRatioNum, _ := decimal.NewFromString(fromRow.RatioNumerator)
	fromRatioDen, _ := decimal.NewFromString(fromRow.RatioDenominator)
	fromOffsetNum, _ := decimal.NewFromString(fromRow.OffsetNumerator)
	fromOffsetDen, _ := decimal.NewFromString(fromRow.OffsetDenominator)

	toRatioNum, _ := decimal.NewFromString(toRow.RatioNumerator)
	toRatioDen, _ := decimal.NewFromString(toRow.RatioDenominator)
	toOffsetNum, _ := decimal.NewFromString(toRow.OffsetNumerator)
	toOffsetDen, _ := decimal.NewFromString(toRow.OffsetDenominator)

	// Convert from source unit to base unit:
	// baseValue = (measure * ratioNum / ratioDen) + (offsetNum / offsetDen)
	fromRatio := decimal.NewFromFloat(1)
	if !fromRatioDen.IsZero() {
		fromRatio = fromRatioNum.Div(fromRatioDen)
	}
	fromOffset := decimal.Zero
	if !fromOffsetDen.IsZero() {
		fromOffset = fromOffsetNum.Div(fromOffsetDen)
	}
	baseValue := measure.Mul(fromRatio).Add(fromOffset)

	// Convert from base unit to target unit:
	// targetValue = (baseValue - offsetNum / offsetDen) * ratioDen / ratioNum
	toRatio := decimal.NewFromFloat(1)
	if !toRatioDen.IsZero() {
		toRatio = toRatioNum.Div(toRatioDen)
	}
	toOffset := decimal.Zero
	if !toOffsetDen.IsZero() {
		toOffset = toOffsetNum.Div(toOffsetDen)
	}

	if toRatio.IsZero() {
		return decimal.Zero, apierror.NewInternalError(
			fmt.Errorf("target unit %s has zero ratio", toUnitID),
			"Invalid unit conversion ratio.",
		)
	}

	targetValue := baseValue.Sub(toOffset).Div(toRatio)
	return targetValue, nil
}

// GetUnitFactors returns each requested unit's base-conversion factors. Units not found (or unscoped) are omitted from the map.
func (r *unitConversionRepo) GetUnitFactors(ctx context.Context, accountID string, unitIDs []string) (map[string]domain.UnitFactors, *apierror.APIError) {
	seen := make(map[string]struct{}, len(unitIDs))
	ids := make([]string, 0, len(unitIDs))
	for _, id := range unitIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	out := make(map[string]domain.UnitFactors, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := r.queries.GetUnitsByIDsScoped(ctx, sqlc.GetUnitsByIDsScopedParams{
		Ids:       ids,
		AccountID: sql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		out[row.ID] = domain.UnitFactors{
			RatioNum:      parseDecimalOrZero(row.RatioNumerator),
			RatioDen:      parseDecimalOrZero(row.RatioDenominator),
			OffsetNum:     parseDecimalOrZero(row.OffsetNumerator),
			OffsetDen:     parseDecimalOrZero(row.OffsetDenominator),
			IsBaseUnit:    row.IsBaseUnit,
			DimensionCode: row.UnitDimensionCode,
		}
	}
	return out, nil
}
