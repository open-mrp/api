package service

import (
	"context"

	"github.com/open-mrp/apikit/apierror"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// legacyCurrencyPrice is a price as the rate row older code still reads it: the currency's unit per one of its unit.
// Writers fill it alongside the price's own columns until CLEAN drops the rate ids.
func legacyCurrencyPrice(ctx context.Context, units domain.UnitRepo, price domain.UnitPrice) (domain.CreateRateParams, *apierror.APIError) {
	currencyUnitID, apiErr := units.GetCurrencyBaseUnitID(ctx)
	if apiErr != nil {
		return domain.CreateRateParams{}, apiErr
	}
	return domain.CreateRateParams{Value: price.Amount, NumeratorUnitID: currencyUnitID, DenominatorUnitID: price.PerUnitID}, nil
}

// validatedLegacyCurrencyPrice is legacyCurrencyPrice after checking the price is per a unit that exists and is not a
// currency, reporting under fieldName.
// A price that was never sent (no currency) is reported as missing its currency.
func validatedLegacyCurrencyPrice(ctx context.Context, units domain.UnitRepo, price domain.UnitPrice, fieldName string) (domain.CreateRateParams, *apierror.APIError) {
	if price.Currency == "" {
		return domain.CreateRateParams{}, ValidateCostRateUnits(ctx, units, "", price.PerUnitID, fieldName)
	}
	rate, apiErr := legacyCurrencyPrice(ctx, units, price)
	if apiErr != nil {
		return rate, apiErr
	}
	return rate, ValidateCostRateUnits(ctx, units, rate.NumeratorUnitID, rate.DenominatorUnitID, fieldName)
}
