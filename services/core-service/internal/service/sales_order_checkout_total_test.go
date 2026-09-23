package service

import (
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sameUnitLine(id, qty, price string) *domain.SalesOrderLine {
	return &domain.SalesOrderLine{
		ID:                              id,
		QuantityValue:                   qty,
		UnitPriceValue:                  price,
		PricingQuantityRatioNumerator:   "1",
		PricingQuantityRatioDenominator: "1",
		PricingPriceRatioNumerator:      "1",
		PricingPriceRatioDenominator:    "1",
	}
}

func TestSalesOrderTotalCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []*domain.SalesOrderLine
		want  int64
	}{
		{
			name:  "same unit line",
			lines: []*domain.SalesOrderLine{sameUnitLine("sol_1", "2", "5.00")},
			want:  1000,
		},
		{
			name: "mixed unit line prices in the rate's unit",
			lines: []*domain.SalesOrderLine{{
				ID:                              "sol_2",
				QuantityValue:                   "3",
				UnitPriceValue:                  "22.50",
				PricingQuantityRatioNumerator:   "24",
				PricingQuantityRatioDenominator: "1",
				PricingPriceRatioNumerator:      "2",
				PricingPriceRatioDenominator:    "1",
			}},
			want: 81000,
		},
		{
			name: "sums all lines including a negative discount line",
			lines: []*domain.SalesOrderLine{
				sameUnitLine("sol_3", "1", "100.00"),
				sameUnitLine("sol_4", "1", "-10.00"),
			},
			want: 9000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, apiErr := salesOrderTotalCents(tt.lines)
			require.Nil(t, apiErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSalesOrderTotalCentsRejectsUnpriceableLine(t *testing.T) {
	t.Parallel()

	_, apiErr := salesOrderTotalCents([]*domain.SalesOrderLine{{
		ID:             "sol_bad",
		QuantityValue:  "1",
		UnitPriceValue: "5.00",
	}})
	require.NotNil(t, apiErr)
}
