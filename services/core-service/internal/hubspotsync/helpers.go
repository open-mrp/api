package hubspotsync

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pricing"

	"github.com/shopspring/decimal"
)

// orderTotal sums each line's extended price, returning a 2-decimal string for HubSpot's `amount` property.
//
// Each line is priced in its price's unit and rounded to the cent (shared/pricing), as the dashboard's order total is. Lines whose quantity/price fail to parse are skipped; a line whose units cannot be converted aborts the whole total rather than silently undercounting revenue.
func orderTotal(lines []*domain.SalesOrderLine) (string, *apierror.APIError) {
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
		conv, err := line.PriceUnitConversion()
		if err != nil {
			return "", apierror.NewInternalError(fmt.Errorf("line %s: %w", line.ID, err), "Failed to price a line in its price's unit.")
		}
		total = total.Add(pricing.LineTotal(qty, price, conv))
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
