package service

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Number formatting for the customer-facing documents (invoice, order acknowledgement, purchase
// order), ported from the dashboard renderers these replaced.
//
// The dashboard formats through numeral.js, whose `0,0.00` patterns round half-away-from-zero and
// always emit exactly the requested number of decimals. Go's decimal does neither by default, so the
// rules are spelled out here rather than left to String(): a quantity that printed as "1,200" in the
// old PDF must not start printing as "1,199.5" in the new one.

// formatMeasure renders a quantity the way BaseQuantityUtils.abbreviate does: rounded to `digits`
// decimals, thousands-separated, with exactly `digits` decimals shown, then the unit label.
//
// The label is whatever the caller passes. These documents pass the unit's full name ("1,200 pair")
// rather than its abbreviation, so an email and its PDF attachment read the same.
//
// digits is 0 for every quantity these documents show — the dashboard's default — so 1199.5 pairs
// print as "1,200 pr", not "1,199.5 pr".
func formatMeasure(d decimal.Decimal, unitAbbr string, digits int) string {
	out := addThousandsSep2(d.Round(int32(digits)).StringFixed(int32(digits)))
	if strings.TrimSpace(unitAbbr) != "" {
		out += " " + unitAbbr
	}
	return out
}

// formatRateAmount renders a unit price the way RateUtils.abbreviate does for a currency numerator:
// "$8.50 / pr" at `digits` decimals.
//
// The denominator is the rate's own pricing unit, which is not necessarily the line's quantity unit
// — an item can be stocked in pairs and priced by the dozen — so callers must pass the rate's
// abbreviation rather than the quantity's.
func formatRateAmount(price decimal.Decimal, denomAbbr string, digits int) string {
	rounded := price.Round(int32(digits))
	// The sign sits outside the currency symbol, as numeral's "$0,0.00" pattern places it: a credit
	// reads "-$8.50", never "$-8.50".
	s := "$" + addThousandsSep2(rounded.Abs().StringFixed(int32(digits)))
	if rounded.IsNegative() {
		s = "-" + s
	}
	if strings.TrimSpace(denomAbbr) != "" {
		s += " / " + denomAbbr
	}
	return s
}

// formatRawMeasure prints a measure verbatim with its unit, no rounding and no separators. The ship
// case table is the one place the dashboard interpolates the raw measure instead of abbreviating it,
// so a 1200 lb case reads "1200 lb" there and "1,200 lb" nowhere else.
func formatRawMeasure(d decimal.Decimal, unitAbbr string) string {
	out := d.String()
	if strings.TrimSpace(unitAbbr) != "" {
		out += " " + unitAbbr
	}
	return out
}

// addThousandsSep2 groups the integer part of an already-formatted decimal string, preserving sign
// and any fractional part.
func addThousandsSep2(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac, hasFrac := strings.Cut(s, ".")
	out := addThousandsSep(intPart)
	if hasFrac {
		out += "." + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}
