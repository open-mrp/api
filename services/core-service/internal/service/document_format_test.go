package service

import (
	"testing"
)

// The formatting rules these documents inherited from the dashboard's numeral.js patterns. Each case
// records what the legacy renderer produced for the same input, so a change to the Go formatters
// that would alter a customer-visible figure fails here rather than in a supplier's inbox.

func TestFormatMeasureMatchesQuantityAbbreviate(t *testing.T) {
	t.Parallel()

	// BaseQuantityUtils.abbreviate(quantity) with its digits=0 default.
	tests := []struct {
		name   string
		value  string
		abbr   string
		digits int
		want   string
	}{
		{"whole units keep no decimals", "1200", "pr", 0, "1,200 pr"},
		{"thousands are separated", "1200000", "ea", 0, "1,200,000 ea"},
		{"fractions round half up", "1199.5", "pr", 0, "1,200 pr"},
		{"fractions round down below half", "1199.4", "pr", 0, "1,199 pr"},
		{"under a thousand is ungrouped", "999", "ea", 0, "999 ea"},
		{"zero renders as zero", "0", "ea", 0, "0 ea"},
		{"negatives keep their sign", "-1200", "ea", 0, "-1,200 ea"},
		{"a blank unit is omitted with no trailing space", "12", "", 0, "12"},
		{"explicit digits are always emitted", "12.5", "lb", 2, "12.50 lb"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatMeasure(dec(test.value), test.abbr, test.digits); got != test.want {
				t.Errorf("formatMeasure(%s, %q, %d) = %q, want %q", test.value, test.abbr, test.digits, got, test.want)
			}
		})
	}
}

func TestFormatRateAmountMatchesRateAbbreviate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  string
		abbr   string
		digits int
		want   string
	}{
		{"invoice prices show two decimals", "8.5", "pr", 2, "$8.50 / pr"},
		{"purchase orders show four", "8.5", "pr", 4, "$8.5000 / pr"},
		// The reason the purchase order asks for four: a sub-cent component price is the whole
		// figure the supplier invoices against, and two decimals erase it.
		{"a sub-cent price survives four decimals", "0.0125", "ea", 4, "$0.0125 / ea"},
		{"the same price collapses at two", "0.0125", "ea", 2, "$0.01 / ea"},
		{"thousands are separated", "1234.5", "cs", 2, "$1,234.50 / cs"},
		{"a blank pricing unit drops the divider", "8.5", "", 2, "$8.50"},
		{"negatives keep their sign inside the amount", "-8.5", "pr", 2, "-$8.50 / pr"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatRateAmount(dec(test.value), test.abbr, test.digits); got != test.want {
				t.Errorf("formatRateAmount(%s, %q, %d) = %q, want %q", test.value, test.abbr, test.digits, got, test.want)
			}
		})
	}
}

// The ship case table interpolates the stored measure directly rather than abbreviating it, so it is
// the one column that neither rounds nor groups.
func TestFormatRawMeasureIsVerbatim(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		abbr  string
		want  string
	}{
		{"1200", "lb", "1200 lb"},
		{"12.5", "lb", "12.5 lb"},
		{"0", "lb", "0 lb"},
		{"12", "", "12"},
	}
	for _, test := range tests {
		if got := formatRawMeasure(dec(test.value), test.abbr); got != test.want {
			t.Errorf("formatRawMeasure(%s, %q) = %q, want %q", test.value, test.abbr, got, test.want)
		}
	}
}

func TestFormatMoneyMatchesCurrencyAbbreviate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  string
	}{
		{"1234.5", "$1,234.50"},
		{"0", "$0.00"},
		{"-1234.5", "-$1,234.50"},
		{"1000000", "$1,000,000.00"},
		{"0.005", "$0.01"},
	}
	for _, test := range tests {
		if got := formatMoney(dec(test.value)); got != test.want {
			t.Errorf("formatMoney(%s) = %q, want %q", test.value, got, test.want)
		}
	}
}

// formatCount backs the invoice's Ordered / Invoiced columns, which the legacy PDF printed through
// numeral('0,0') — rounded to whole units, grouped, and with no unit appended.
func TestFormatCountMatchesNumeralInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  string
	}{
		{"1200", "1,200"},
		{"1199.5", "1,200"},
		{"1199.4", "1,199"},
		{"0", "0"},
	}
	for _, test := range tests {
		if got := formatCount(dec(test.value)); got != test.want {
			t.Errorf("formatCount(%s) = %q, want %q", test.value, got, test.want)
		}
	}
}
