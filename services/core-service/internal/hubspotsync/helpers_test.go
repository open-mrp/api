package hubspotsync

import (
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func strptr(s string) *string { return &s }

func TestOrderTotal(t *testing.T) {
	// line builds an order line whose quantity unit is num/den of the unit its price is quoted per.
	line := func(qty, price, num, den string) *domain.SalesOrderLine {
		return &domain.SalesOrderLine{QuantityValue: qty, UnitPriceValue: price,
			PricingQuantityRatioNumerator: num, PricingQuantityRatioDenominator: den, PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1"}
	}

	tests := []struct {
		name    string
		lines   []*domain.SalesOrderLine
		want    string
		wantErr bool
	}{
		{name: "empty", lines: nil, want: "0.00"},
		{name: "matching units multiply directly", lines: []*domain.SalesOrderLine{line("3", "10.00", "1", "1")}, want: "30.00"},
		{
			name: "mismatched units convert quantity into the price denominator unit",
			// 3 cases of 12 at $10/ea → 36 ea × $10 = $360.00
			lines: []*domain.SalesOrderLine{line("3", "10.00", "12", "1")},
			want:  "360.00",
		},
		{
			name:  "unparseable line is skipped",
			lines: []*domain.SalesOrderLine{line("abc", "10.00", "1", "1"), line("5", "2.00", "1", "1")},
			want:  "10.00",
		},
		{name: "an unusable conversion aborts the total", lines: []*domain.SalesOrderLine{line("3", "10.00", "12", "0")}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, apiErr := orderTotal(tt.lines)
			if tt.wantErr {
				if apiErr == nil {
					t.Fatalf("orderTotal() expected error, got nil")
				}
				return
			}
			if apiErr != nil {
				t.Fatalf("orderTotal() unexpected error: %v", apiErr)
			}
			if got != tt.want {
				t.Errorf("orderTotal() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeriveDomain(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "empty", in: strptr(""), want: ""},
		{name: "bare host", in: strptr("acme.com"), want: "acme.com"},
		{name: "with scheme", in: strptr("https://acme.com"), want: "acme.com"},
		{name: "with www and path", in: strptr("https://www.acme.com/about"), want: "acme.com"},
		{name: "uppercase", in: strptr("HTTP://ACME.COM"), want: "acme.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveDomain(tt.in); got != tt.want {
				t.Errorf("deriveDomain() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitName(t *testing.T) {
	tests := []struct {
		in          string
		first, last string
	}{
		{in: "", first: "", last: ""},
		{in: "Cher", first: "Cher", last: ""},
		{in: "Ada Lovelace", first: "Ada", last: "Lovelace"},
		{in: "Jean Luc Picard", first: "Jean", last: "Luc Picard"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			first, last := splitName(tt.in)
			if first != tt.first || last != tt.last {
				t.Errorf("splitName(%q) = (%q, %q), want (%q, %q)", tt.in, first, last, tt.first, tt.last)
			}
		})
	}
}
