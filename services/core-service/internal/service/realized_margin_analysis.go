package service

import (
	"sort"

	"github.com/shopspring/decimal"
)

// realizedKey is one customer's trade in one SKU. SKU rather than product line is the grain that matters: what a customer paid for a size 6 is no evidence about a size 14.
type realizedKey struct {
	CustomerID string
	ItemID     string
}

// realizedLine is one invoiced line, already converted out of the analytics payload.
type realizedLine struct {
	CustomerID      string
	CustomerName    string
	CustomerNo      string
	CustomerGroup   string
	CustomerGroupID string
	ItemID          string
	SKU             string
	ProductLineID   string
	ProductLine     string
	UnitAbbr        string

	Quantity decimal.Decimal
	Revenue  decimal.Decimal
	Cost     decimal.Decimal
}

// realizedAggregate is what one customer actually paid for one SKU over the window.
type realizedAggregate struct {
	realizedKey
	CustomerName    string
	CustomerNo      string
	CustomerGroup   string
	CustomerGroupID string
	SKU             string
	ProductLineID   string
	ProductLine     string
	UnitAbbr        string

	Quantity decimal.Decimal
	Revenue  decimal.Decimal
	Cost     decimal.Decimal
	// AveragePrice is revenue over quantity: the price actually achieved across the window, which is the only figure that survives mixed order sizes and overrides.
	AveragePrice decimal.Decimal
	LineCount    int
}

// realizedFinding is one flagged trading relationship.
type realizedFinding struct {
	realizedAggregate

	PeerMedianPrice   decimal.Decimal
	HasPeerMedian     bool
	BelowPeerFraction decimal.Decimal
	GrossMargin       decimal.Decimal
	HasGrossMargin    bool
	BelowPeerMedian   bool
	BelowTargetMargin bool
}

// aggregateRealizedLines rolls invoiced lines up to one row per customer and SKU.
func aggregateRealizedLines(lines []realizedLine) []realizedAggregate {
	byKey := make(map[realizedKey]*realizedAggregate)
	order := make([]realizedKey, 0)

	for _, line := range lines {
		if line.CustomerID == "" || line.ItemID == "" {
			continue
		}
		key := realizedKey{CustomerID: line.CustomerID, ItemID: line.ItemID}
		aggregate, ok := byKey[key]
		if !ok {
			aggregate = &realizedAggregate{
				realizedKey:     key,
				CustomerName:    line.CustomerName,
				CustomerNo:      line.CustomerNo,
				CustomerGroup:   line.CustomerGroup,
				CustomerGroupID: line.CustomerGroupID,
				SKU:             line.SKU,
				ProductLineID:   line.ProductLineID,
				ProductLine:     line.ProductLine,
				UnitAbbr:        line.UnitAbbr,
			}
			byKey[key] = aggregate
			order = append(order, key)
		}
		aggregate.Quantity = aggregate.Quantity.Add(line.Quantity)
		aggregate.Revenue = aggregate.Revenue.Add(line.Revenue)
		aggregate.Cost = aggregate.Cost.Add(line.Cost)
		aggregate.LineCount++
	}

	out := make([]realizedAggregate, 0, len(order))
	for _, key := range order {
		aggregate := byKey[key]
		if aggregate.Quantity.IsPositive() {
			aggregate.AveragePrice = aggregate.Revenue.Div(aggregate.Quantity)
		}
		out = append(out, *aggregate)
	}
	return out
}

// analyzeRealizedMargins flags what customers were actually charged, as opposed to what they are contracted to be charged.
//
// This is the only view that sees a price a rep typed on an order: a manual line override bypasses contracted pricing and discounts entirely, so it never appears in an audit of configured prices, only here.
func analyzeRealizedMargins(aggregates []realizedAggregate, targetMargin, outlierTolerance decimal.Decimal) []realizedFinding {
	medians := realizedPeerMedians(aggregates)

	findings := make([]realizedFinding, 0)
	for _, aggregate := range aggregates {
		if !aggregate.Revenue.IsPositive() {
			continue
		}
		finding := realizedFinding{realizedAggregate: aggregate}

		if median, ok := medians[aggregate.ItemID]; ok && median.IsPositive() {
			finding.PeerMedianPrice = median
			finding.HasPeerMedian = true
			if aggregate.AveragePrice.LessThan(median) {
				finding.BelowPeerFraction = median.Sub(aggregate.AveragePrice).Div(median)
				finding.BelowPeerMedian = finding.BelowPeerFraction.GreaterThanOrEqual(outlierTolerance)
			}
		}

		// A zero cost means the cost was never captured on the line, not that the item was free — treating it as a 100% margin would hide real problems.
		if aggregate.Cost.IsPositive() {
			finding.GrossMargin = aggregate.Revenue.Sub(aggregate.Cost).Div(aggregate.Revenue)
			finding.HasGrossMargin = true
			finding.BelowTargetMargin = finding.GrossMargin.LessThan(targetMargin)
		}

		if finding.BelowPeerMedian || finding.BelowTargetMargin {
			findings = append(findings, finding)
		}
	}

	// Ranked by money at stake rather than by percentage: a thin margin on a large account matters more than a terrible margin on one sample order.
	sort.SliceStable(findings, func(i, j int) bool {
		li := realizedSeverity(findings[i], targetMargin)
		lj := realizedSeverity(findings[j], targetMargin)
		if !li.Equal(lj) {
			return li.GreaterThan(lj)
		}
		return findings[i].CustomerName < findings[j].CustomerName
	})
	return findings
}

// realizedSeverity is the money the problem cost, so the ranking is by exposure rather than by percentage: the margin that should have been earned and was not, plus what the discount against peers gave away over the window.
func realizedSeverity(finding realizedFinding, targetMargin decimal.Decimal) decimal.Decimal {
	severity := decimal.Zero
	if finding.BelowTargetMargin {
		severity = severity.Add(finding.Revenue.Mul(targetMargin.Sub(finding.GrossMargin)))
	}
	if finding.BelowPeerMedian {
		severity = severity.Add(finding.PeerMedianPrice.Sub(finding.AveragePrice).Mul(finding.Quantity))
	}
	return severity
}

// medianOf is the middle value of a set. Median rather than mean throughout this analysis: one deeply discounted customer should not drag a benchmark down and thereby hide itself.
func medianOf(values []decimal.Decimal) decimal.Decimal {
	sorted := make([]decimal.Decimal, len(values))
	copy(sorted, values)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].LessThan(sorted[j]) })

	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return sorted[mid-1].Add(sorted[mid]).Div(decimal.NewFromInt(2))
}

// realizedPeerMedians is the median achieved price per SKU across customers.
func realizedPeerMedians(aggregates []realizedAggregate) map[string]decimal.Decimal {
	byItem := make(map[string][]decimal.Decimal)
	for _, aggregate := range aggregates {
		if aggregate.AveragePrice.IsPositive() {
			byItem[aggregate.ItemID] = append(byItem[aggregate.ItemID], aggregate.AveragePrice)
		}
	}

	medians := make(map[string]decimal.Decimal, len(byItem))
	for itemID, prices := range byItem {
		// One buyer is no benchmark; comparing a customer against itself never flags.
		if len(prices) < 2 {
			continue
		}
		medians[itemID] = medianOf(prices)
	}
	return medians
}
