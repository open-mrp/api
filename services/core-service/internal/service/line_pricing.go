package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pricing"
)

// lineConversions collects each line's price-unit conversion, keyed by line ID, so a document
// builder can price its lines without an error path of its own. A line whose units cannot be
// converted fails the whole document: stating the wrong amount is worse than a late document.
func lineConversions[L any](lines []L, read func(L) (string, pricing.UnitConversion, error)) (map[string]pricing.UnitConversion, *apierror.APIError) {
	out := make(map[string]pricing.UnitConversion, len(lines))
	for _, line := range lines {
		id, conv, err := read(line)
		if err != nil {
			return nil, apierror.NewInternalError(fmt.Errorf("line %s: %w", id, err), "Failed to price a line in its price's unit.")
		}
		out[id] = conv
	}
	return out, nil
}

func invoiceLineConversions(lines []*domain.InvoiceLine) (map[string]pricing.UnitConversion, *apierror.APIError) {
	return lineConversions(lines, func(l *domain.InvoiceLine) (string, pricing.UnitConversion, error) {
		conv, err := l.PriceUnitConversion()
		return l.ID, conv, err
	})
}

func salesOrderLineConversions(lines []*domain.SalesOrderLine) (map[string]pricing.UnitConversion, *apierror.APIError) {
	return lineConversions(lines, func(l *domain.SalesOrderLine) (string, pricing.UnitConversion, error) {
		conv, err := l.PriceUnitConversion()
		return l.ID, conv, err
	})
}

func purchaseOrderLineConversions(lines []*domain.PurchaseOrderLine) (map[string]pricing.UnitConversion, *apierror.APIError) {
	return lineConversions(lines, func(l *domain.PurchaseOrderLine) (string, pricing.UnitConversion, error) {
		conv, err := l.PriceUnitConversion()
		return l.ID, conv, err
	})
}

// conversionFor is the line's conversion, or the identity for a line the map does not hold, which
// only a builder handed no conversions (a test, or a document with no lines to price) meets.
func conversionFor(convs map[string]pricing.UnitConversion, lineID string) pricing.UnitConversion {
	if conv, ok := convs[lineID]; ok {
		return conv
	}
	return pricing.Identity
}

// unitPair is a line's quantity unit and the units its price is quoted in and per.
type unitPair struct{ Quantity, PriceNumerator, Price string }

// unitPairConversions builds the conversion for each pair from the units' stored ratios. It serves
// lines that have not been saved yet, which carry no conversion of their own, and applies the same
// eligibility as the line queries: a pair the dashboard would price differently (units of different
// dimensions, a unit with an offset, a price in a non-base currency unit) is an error. The units are
// read only when some pair actually differs.
func unitPairConversions(ctx context.Context, repos domain.RepoFactory, accountID string, pairs []unitPair) (map[unitPair]pricing.UnitConversion, *apierror.APIError) {
	out := make(map[unitPair]pricing.UnitConversion, len(pairs))
	var ids []string
	for _, p := range pairs {
		if p.Quantity == p.Price || p.Quantity == "" || p.Price == "" {
			out[p] = pricing.Identity
			continue
		}
		ids = append(ids, p.Quantity, p.Price, p.PriceNumerator)
	}
	if len(ids) == 0 {
		return out, nil
	}

	factors, apiErr := repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	for _, p := range pairs {
		if _, done := out[p]; done {
			continue
		}
		qty, okQty := factors[p.Quantity]
		price, okPrice := factors[p.Price]
		num, okNum := factors[p.PriceNumerator]
		if !okQty || !okPrice || !okNum || !priceableUnits(qty, price, num) {
			return nil, apierror.NewInternalError(fmt.Errorf("units %s per %s/%s cannot be priced", p.Quantity, p.PriceNumerator, p.Price), "Failed to price a line in its price's unit.")
		}
		out[p] = pricing.Between(qty.RatioNum, qty.RatioDen, price.RatioNum, price.RatioDen)
	}
	return out, nil
}

// priceableUnits mirrors the line queries' pricing eligibility (see the line-pricing skill).
func priceableUnits(qty, price, num domain.UnitFactors) bool {
	isRatioOne := func(f domain.UnitFactors) bool { return f.RatioNum.Equal(f.RatioDen) }
	return qty.DimensionCode == price.DimensionCode && qty.DimensionCode != num.DimensionCode &&
		qty.OffsetNum.IsZero() && price.OffsetNum.IsZero() &&
		(!qty.IsBaseUnit || isRatioOne(qty)) && (!price.IsBaseUnit || isRatioOne(price)) &&
		(num.IsBaseUnit || (isRatioOne(num) && num.OffsetNum.IsZero())) &&
		!qty.RatioNum.IsZero() && !qty.RatioDen.IsZero() && !price.RatioNum.IsZero() && !price.RatioDen.IsZero()
}

// resolvedLineConversions is unitPairConversions over a create request's resolved lines.
func resolvedLineConversions(ctx context.Context, repos domain.RepoFactory, accountID string, lines []domain.ResolvedSalesOrderLine) (map[unitPair]pricing.UnitConversion, *apierror.APIError) {
	pairs := make([]unitPair, len(lines))
	for i, l := range lines {
		pairs[i] = resolvedLineUnits(l)
	}
	return unitPairConversions(ctx, repos, accountID, pairs)
}

func resolvedLineUnits(l domain.ResolvedSalesOrderLine) unitPair {
	return unitPair{Quantity: l.QuantityUnitID, PriceNumerator: l.UnitPrice.NumeratorUnitID, Price: l.UnitPrice.DenominatorUnitID}
}
