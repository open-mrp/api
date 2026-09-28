package salesorderep

import (
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/open-mrp/api/shared/proto/core"
)

// Totals price each quantity in the unit its price is quoted per: three cartons of twelve pairs at
// $22.50 a pair are $810, not $67.50.
func cartonLine() *pb.SalesOrderLineInfo {
	picked, packed, invoiced := "2", "1", "1"
	return &pb.SalesOrderLineInfo{
		QuantityValue:         "3",
		QuantityPickedValue:   &picked,
		QuantityPackedValue:   &packed,
		QuantityInvoicedValue: &invoiced,
		UnitPriceValue:        "22.5",
		// A carton of twelve pairs, priced per pair (two eaches).
		PricingQuantityRatioNumerator:   "24.000000000000000000000000000000",
		PricingQuantityRatioDenominator: "1.000000000000000000000000000000",
		PricingPriceRatioNumerator:      "2.000000000000000000000000000000",
		PricingPriceRatioDenominator:    "1.000000000000000000000000000000",
	}
}

func TestBuildLineTotalsConvertsToThePriceUnit(t *testing.T) {
	t.Parallel()

	totals := buildLineTotals(cartonLine())
	require.NotNil(t, totals)
	require.Equal(t, "810", totals.Ordered)
	require.Equal(t, "540", totals.Picked.Amount)
	require.Equal(t, "270", totals.Packed.Amount)
	require.Equal(t, "270", totals.Invoiced.Amount)
}

func TestSalesOrderTotalsConvertToThePriceUnit(t *testing.T) {
	t.Parallel()

	same := &pb.SalesOrderLineInfo{QuantityValue: "2", UnitPriceValue: "5",
		PricingQuantityRatioNumerator: "1", PricingQuantityRatioDenominator: "1", PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1"}
	totals := salesOrderTotalsFromOrder(&pb.SalesOrderInfo{Lines: []*pb.SalesOrderLineInfo{cartonLine(), same}})
	require.NotNil(t, totals)
	require.Equal(t, "820", totals.Ordered)
}

func TestTotalsWithoutAConversion(t *testing.T) {
	t.Parallel()

	t.Run("a core that predates the field prices as before", func(t *testing.T) {
		line := cartonLine()
		line.PricingQuantityRatioNumerator, line.PricingQuantityRatioDenominator = "", ""
		line.PricingPriceRatioNumerator, line.PricingPriceRatioDenominator = "", ""
		require.Equal(t, "67.5", buildLineTotals(line).Ordered)
	})

	t.Run("a line core cannot price yields no totals rather than wrong ones", func(t *testing.T) {
		line := cartonLine()
		line.PricingQuantityRatioNumerator, line.PricingQuantityRatioDenominator = "", ""
		line.PricingPriceRatioNumerator, line.PricingPriceRatioDenominator = "", ""
		line.PricingUnavailable = true
		require.Nil(t, buildLineTotals(line))
		require.Nil(t, salesOrderTotalsFromOrder(&pb.SalesOrderInfo{Lines: []*pb.SalesOrderLineInfo{line}}))
	})

	t.Run("a zero ratio yields no totals", func(t *testing.T) {
		line := cartonLine()
		line.PricingPriceRatioDenominator = "0"
		require.Nil(t, buildLineTotals(line))
	})
}

// When core computes the totals it sends them on the proto; the gateway maps the amounts and pairs each
// stage with the order-level completion fraction rather than re-summing the lines.
func TestSalesOrderTotalsFromProto(t *testing.T) {
	t.Parallel()

	t.Run("maps computed amounts and completions", func(t *testing.T) {
		info := &pb.SalesOrderInfo{
			PickedCompletion:   0.5,
			PackedCompletion:   0.25,
			InvoicedCompletion: 0.25,
			Totals:             &pb.SalesOrderTotalsInfo{Available: true, Ordered: "810", Picked: "540", Packed: "270", Invoiced: "270"},
		}
		totals, handled := salesOrderTotalsFromProto(info)
		require.True(t, handled)
		require.NotNil(t, totals)
		require.Equal(t, "810", totals.Ordered)
		require.Equal(t, "540", totals.Picked.Amount)
		require.InDelta(t, 0.5, totals.Picked.Completion, 0)
		require.Equal(t, "270", totals.Packed.Amount)
		require.Equal(t, "270", totals.Invoiced.Amount)
	})

	t.Run("unavailable totals resolve to null, not a fallback", func(t *testing.T) {
		totals, handled := salesOrderTotalsFromProto(&pb.SalesOrderInfo{Totals: &pb.SalesOrderTotalsInfo{Available: false}})
		require.True(t, handled)
		require.Nil(t, totals)
	})

	t.Run("an older core that sends no totals is not handled", func(t *testing.T) {
		totals, handled := salesOrderTotalsFromProto(&pb.SalesOrderInfo{Lines: []*pb.SalesOrderLineInfo{cartonLine()}})
		require.False(t, handled)
		require.Nil(t, totals)
	})
}

// Each line is rounded to the cent before it is summed, as the dashboard's calculateTotalOrdered does.
func TestSalesOrderTotalsRoundEachLine(t *testing.T) {
	t.Parallel()

	third := func() *pb.SalesOrderLineInfo {
		return &pb.SalesOrderLineInfo{QuantityValue: "1", UnitPriceValue: "0.334",
			PricingQuantityRatioNumerator: "1", PricingQuantityRatioDenominator: "1", PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1"}
	}
	totals := salesOrderTotalsFromOrder(&pb.SalesOrderInfo{Lines: []*pb.SalesOrderLineInfo{third(), third(), third()}})
	require.Equal(t, "0.99", totals.Ordered, "three lines of 0.33, not one of 1.002")
}
