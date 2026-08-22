package hubspotsync

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/shopspring/decimal"
)

// unitConvertFunc converts a measure from one unit to another. It matches the signature of domain.UnitConversionRepo.ConvertValue, letting orderTotal stay pure and unit-testable.
type unitConvertFunc func(ctx context.Context, measure decimal.Decimal, fromUnitID, toUnitID string) (decimal.Decimal, *apierror.APIError)

// orderTotal sums each line's extended price, returning a 2-decimal string for HubSpot's `amount` property.
//
// A line's quantity is expressed in its own unit (QuantityUnitID) while the unit price is a rate per the price's denominator unit (UnitPriceDenominatorUnitID); these can differ (e.g. quantity in cases, price per each), so the quantity is converted into the price's denominator unit before multiplying. Lines whose quantity/price fail to parse are skipped; a conversion failure aborts the whole total (returns the error) rather than silently undercounting revenue.
func orderTotal(ctx context.Context, lines []*domain.SalesOrderLine, convert unitConvertFunc) (string, *apierror.APIError) {
	total := decimal.Zero
	for _, line := range lines {
		qty, err := decimal.NewFromString(line.QuantityValue)
		if err != nil {
			continue
		}
		price, err := decimal.NewFromString(line.UnitPriceValue)
		if err != nil {
			continue
		}
		if line.QuantityUnitID != "" && line.UnitPriceDenominatorUnitID != "" && line.QuantityUnitID != line.UnitPriceDenominatorUnitID {
			converted, apiErr := convert(ctx, qty, line.QuantityUnitID, line.UnitPriceDenominatorUnitID)
			if apiErr != nil {
				return "", apiErr
			}
			qty = converted
		}
		total = total.Add(qty.Mul(price))
	}
	return total.Round(2).StringFixed(2), nil
}

// closeDate is the deal close date: the order's issue date when set, else its creation date.
func closeDate(order *domain.SalesOrder) time.Time {
	if order.IssuedAt != nil {
		return *order.IssuedAt
	}
	return order.CreatedAt
}

// deriveDomain extracts a bare hostname (e.g. "acme.com") from a customer URL, or "" if absent.
func deriveDomain(rawURL *string) string {
	if rawURL == nil || *rawURL == "" {
		return ""
	}
	s := strings.TrimSpace(*rawURL)
	if !strings.Contains(s, "://") {
		s = "//" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	return strings.TrimPrefix(host, "www.")
}

// splitName splits a full name into first and last; a single token becomes the first name.
func splitName(full string) (first, last string) {
	parts := strings.Fields(full)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], strings.Join(parts[1:], " ")
	}
}

// firstNonEmpty returns the first non-nil, non-empty pointed-to string.
func firstNonEmpty(values ...*string) string {
	for _, v := range values {
		if v != nil && *v != "" {
			return *v
		}
	}
	return ""
}

// firstNonEmptyStr returns the first non-empty string.
func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
