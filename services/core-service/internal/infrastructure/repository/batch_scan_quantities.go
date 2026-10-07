package repository

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// sumOutputsInUnit totals what has been split or moved off a batch — firsts, seconds and waste — in
// one unit, converting each term into it.
func sumOutputsInUnit(ctx context.Context, conv domain.UnitConversionRepo, outputs []domain.BaseBatch, unit domain.LightUnit) (decimal.Decimal, *apierror.APIError) {
	total := decimal.Zero
	add := func(q *domain.BatchQuantity) *apierror.APIError {
		if q == nil {
			return nil
		}
		converted, apiErr := domain.ConvertScanQuantity(ctx, conv, q.Measure, q.Unit, unit)
		if apiErr != nil {
			return apiErr
		}
		total = total.Add(converted)
		return nil
	}
	for i := range outputs {
		if apiErr := add(&outputs[i].Quantity); apiErr != nil {
			return decimal.Zero, apiErr
		}
		if apiErr := add(outputs[i].Seconds); apiErr != nil {
			return decimal.Zero, apiErr
		}
		if apiErr := add(outputs[i].Waste); apiErr != nil {
			return decimal.Zero, apiErr
		}
	}
	return total, nil
}
