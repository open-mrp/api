package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// cartonLine mirrors the gateway's totals_test carton: three cartons of twelve pairs priced per pair
// ($22.50/pair) is $810, not the $67.50 a raw quantity * rate would give. The amounts must match the
// gateway's salesOrderTotalsFromOrder byte for byte so moving the sum into core changes no response.
func cartonLine() *domain.SalesOrderLine {
	picked, packed, invoiced := "2", "1", "1"
	return &domain.SalesOrderLine{
		QuantityValue:                   "3",
		QuantityPickedValue:             &picked,
		QuantityPackedValue:             &packed,
		QuantityInvoicedValue:           &invoiced,
		UnitPriceValue:                  "22.5",
		PricingQuantityRatioNumerator:   "24.000000000000000000000000000000",
		PricingQuantityRatioDenominator: "1.000000000000000000000000000000",
		PricingPriceRatioNumerator:      "2.000000000000000000000000000000",
		PricingPriceRatioDenominator:    "1.000000000000000000000000000000",
	}
}

func TestComputeSalesOrderTotalsConvertsToThePriceUnit(t *testing.T) {
	t.Parallel()

	totals := computeSalesOrderTotals([]*domain.SalesOrderLine{cartonLine()})
	require.NotNil(t, totals)
	require.True(t, totals.Available)
	require.Equal(t, "810", totals.Ordered)
	require.Equal(t, "540", totals.Picked)
	require.Equal(t, "270", totals.Packed)
	require.Equal(t, "270", totals.Invoiced)
}

func TestComputeSalesOrderTotalsSumsAndRoundsEachLine(t *testing.T) {
	t.Parallel()

	same := &domain.SalesOrderLine{
		QuantityValue: "2", UnitPriceValue: "5",
		PricingQuantityRatioNumerator: "1", PricingQuantityRatioDenominator: "1",
		PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1",
	}
	totals := computeSalesOrderTotals([]*domain.SalesOrderLine{cartonLine(), same})
	require.NotNil(t, totals)
	require.Equal(t, "820", totals.Ordered)

	third := func() *domain.SalesOrderLine {
		return &domain.SalesOrderLine{
			QuantityValue: "1", UnitPriceValue: "0.334",
			PricingQuantityRatioNumerator: "1", PricingQuantityRatioDenominator: "1",
			PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1",
		}
	}
	rounded := computeSalesOrderTotals([]*domain.SalesOrderLine{third(), third(), third()})
	require.Equal(t, "0.99", rounded.Ordered, "three lines of 0.33, not one of 1.002")
}

func TestComputeSalesOrderTotalsUnavailable(t *testing.T) {
	t.Parallel()

	t.Run("an order with no lines is unavailable", func(t *testing.T) {
		totals := computeSalesOrderTotals(nil)
		require.NotNil(t, totals)
		require.False(t, totals.Available)
	})

	t.Run("a line that cannot be priced makes the whole order unavailable", func(t *testing.T) {
		line := cartonLine()
		line.PricingQuantityRatioNumerator, line.PricingQuantityRatioDenominator = "", ""
		line.PricingPriceRatioNumerator, line.PricingPriceRatioDenominator = "", ""
		totals := computeSalesOrderTotals([]*domain.SalesOrderLine{line})
		require.NotNil(t, totals)
		require.False(t, totals.Available)
		require.Empty(t, totals.Ordered)
	})

	t.Run("a zero ratio term is unavailable", func(t *testing.T) {
		line := cartonLine()
		line.PricingPriceRatioDenominator = "0"
		totals := computeSalesOrderTotals([]*domain.SalesOrderLine{line})
		require.False(t, totals.Available)
	})
}
