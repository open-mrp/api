package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func realizedTestLine(customer, item, qty, revenue, cost string) realizedLine {
	return realizedLine{
		CustomerID:   customer,
		CustomerName: customer,
		ItemID:       item,
		SKU:          item,
		Quantity:     dec(qty),
		Revenue:      dec(revenue),
		Cost:         dec(cost),
	}
}

// Several invoiced lines for the same customer and SKU roll into one achieved price, weighted by quantity rather than averaged per line.
func TestAggregateRealizedLines_WeightsByQuantity(t *testing.T) {
	lines := []realizedLine{
		realizedTestLine("acme", "it_1", "10", "100.00", "50.00"),
		realizedTestLine("acme", "it_1", "90", "450.00", "270.00"),
	}

	aggregates := aggregateRealizedLines(lines)
	if len(aggregates) != 1 {
		t.Fatalf("expected 1 aggregate, got %d", len(aggregates))
	}
	// 550 / 100 = 5.50, not the 7.50 a naive per-line mean would give.
	if got := aggregates[0].AveragePrice.StringFixed(2); got != "5.50" {
		t.Errorf("average price = %s, want 5.50", got)
	}
	if aggregates[0].LineCount != 2 {
		t.Errorf("line count = %d, want 2", aggregates[0].LineCount)
	}
}

// A customer achieving far less per unit than others on the same SKU is flagged.
func TestAnalyzeRealizedMargins_FlagsPeerOutlier(t *testing.T) {
	aggregates := aggregateRealizedLines([]realizedLine{
		realizedTestLine("a", "it_1", "10", "100.00", "10.00"),
		realizedTestLine("b", "it_1", "10", "100.00", "10.00"),
		realizedTestLine("c", "it_1", "10", "100.00", "10.00"),
		realizedTestLine("cheap", "it_1", "10", "50.00", "10.00"),
	})

	findings := analyzeRealizedMargins(aggregates, dec("0"), dec("0.15"))
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].CustomerName != "cheap" {
		t.Errorf("flagged %q, want cheap", findings[0].CustomerName)
	}
	if got := findings[0].BelowPeerFraction.StringFixed(2); got != "0.50" {
		t.Errorf("below-peer fraction = %s, want 0.50", got)
	}
}

// Different SKUs are separate benchmarks.
func TestAnalyzeRealizedMargins_PeerGroupsAreScopedBySKU(t *testing.T) {
	aggregates := aggregateRealizedLines([]realizedLine{
		realizedTestLine("a", "it_expensive", "1", "100.00", "10.00"),
		realizedTestLine("b", "it_expensive", "1", "100.00", "10.00"),
		realizedTestLine("a", "it_cheap", "1", "5.00", "1.00"),
		realizedTestLine("b", "it_cheap", "1", "5.00", "1.00"),
	})

	findings := analyzeRealizedMargins(aggregates, dec("0"), dec("0.15"))
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestAnalyzeRealizedMargins_FlagsBelowTargetMargin(t *testing.T) {
	aggregates := aggregateRealizedLines([]realizedLine{
		realizedTestLine("a", "it_1", "10", "100.00", "85.00"),
		realizedTestLine("b", "it_1", "10", "100.00", "85.00"),
	})

	findings := analyzeRealizedMargins(aggregates, dec("0.30"), dec("0.15"))
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if got := findings[0].GrossMargin.StringFixed(2); got != "0.15" {
		t.Errorf("margin = %s, want 0.15", got)
	}
}

// A line with no captured cost must not read as a 100% margin.
func TestAnalyzeRealizedMargins_ZeroCostIsNotFullMargin(t *testing.T) {
	aggregates := aggregateRealizedLines([]realizedLine{
		realizedTestLine("a", "it_1", "10", "100.00", "0"),
		realizedTestLine("b", "it_1", "10", "100.00", "0"),
	})

	findings := analyzeRealizedMargins(aggregates, dec("0.30"), dec("0.15"))
	for _, finding := range findings {
		if finding.HasGrossMargin {
			t.Errorf("margin should not be assessed without a cost, got %s", finding.GrossMargin)
		}
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings when margin cannot be assessed, got %+v", findings)
	}
}

// Ranking is by money at stake, so a thin margin on a big account outranks a worse percentage on a tiny one.
func TestAnalyzeRealizedMargins_RanksByMoneyAtStake(t *testing.T) {
	aggregates := aggregateRealizedLines([]realizedLine{
		realizedTestLine("small-but-awful", "it_1", "1", "10.00", "9.90"),
		realizedTestLine("big-and-thin", "it_2", "10000", "100000.00", "80000.00"),
	})

	findings := analyzeRealizedMargins(aggregates, dec("0.30"), dec("0.15"))
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].CustomerName != "big-and-thin" {
		t.Errorf("first finding = %q, want big-and-thin", findings[0].CustomerName)
	}
}

func TestRealizedPeerMedians_IgnoresSingleBuyer(t *testing.T) {
	aggregates := aggregateRealizedLines([]realizedLine{realizedTestLine("solo", "it_1", "1", "1.00", "0.10")})
	if _, ok := realizedPeerMedians(aggregates)["it_1"]; ok {
		t.Error("a single buyer should not produce a peer median")
	}
}

var _ = decimal.Zero
