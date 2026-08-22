package hubspotsync

import (
	"context"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/shopspring/decimal"
)

func strptr(s string) *string { return &s }

func TestOrderTotal(t *testing.T) {
	// identityConvert returns the measure unchanged and records that conversion was attempted.
	// caseToEach converts a quantity in "case" to "ea" at 12:1.
	caseToEach := func(_ context.Context, m decimal.Decimal, from, to string) (decimal.Decimal, *apierror.APIError) {
		if from == "case" && to == "ea" {
			return m.Mul(decimal.NewFromInt(12)), nil
		}
		return m, nil
	}
	convertErr := func(_ context.Context, m decimal.Decimal, _, _ string) (decimal.Decimal, *apierror.APIError) {
		return decimal.Zero, apierror.NewInternalError(errors.New("no factors"), "conversion failed")
	}

	tests := []struct {
		name    string
		lines   []*domain.SalesOrderLine
		convert unitConvertFunc
		want    string
		wantErr bool
	}{
		{name: "empty", lines: nil, convert: caseToEach, want: "0.00"},
		{
			name: "matching units multiply directly",
			lines: []*domain.SalesOrderLine{
				{QuantityValue: "3", QuantityUnitID: "ea", UnitPriceValue: "10.00", UnitPriceDenominatorUnitID: "ea"},
			},
			convert: caseToEach,
			want:    "30.00",
		},
		{
			name: "mismatched units convert quantity into the price denominator unit",
			lines: []*domain.SalesOrderLine{
				// 3 cases at $10/ea → 36 ea × $10 = $360.00
				{QuantityValue: "3", QuantityUnitID: "case", UnitPriceValue: "10.00", UnitPriceDenominatorUnitID: "ea"},
			},
			convert: caseToEach,
			want:    "360.00",
		},
		{
			name: "unparseable line is skipped",
			lines: []*domain.SalesOrderLine{
				{QuantityValue: "abc", QuantityUnitID: "ea", UnitPriceValue: "10.00", UnitPriceDenominatorUnitID: "ea"},
				{QuantityValue: "5", QuantityUnitID: "ea", UnitPriceValue: "2.00", UnitPriceDenominatorUnitID: "ea"},
			},
			convert: caseToEach,
			want:    "10.00",
		},
		{
			name: "conversion failure aborts the total",
			lines: []*domain.SalesOrderLine{
				{QuantityValue: "3", QuantityUnitID: "case", UnitPriceValue: "10.00", UnitPriceDenominatorUnitID: "ea"},
			},
			convert: convertErr,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, apiErr := orderTotal(context.Background(), tt.lines, tt.convert)
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
