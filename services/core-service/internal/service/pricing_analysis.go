package service

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// pricingPeerGroup is the comparison bucket for outlier detection: one product line narrowed by one exact attribute set, priced on the same basis. Two customers are only comparable when their prices buy the same thing, and product line plus attributes is exactly what the pricing engine matches an account price on (see selectAccountPrice in sales_order_pricing.go) — so it is also the finest grain that means anything.
//
// The denominator unit belongs in the key because the values are compared as plain decimals: $8 per pair and $8 per dozen are not evidence about each other, and pooling them would produce a median that describes nothing.
type pricingPeerGroup struct {
	ProductLineID   string
	AttributeKey    string
	DenominatorUnit string
}

// pricingCandidate is one customer's contracted price for a peer group.
type pricingCandidate struct {
	CustomerID   string
	CustomerName string
	CustomerNo   string
	// Inherited marks a price the customer gets through its parent account rather than one recorded against it — easy to miss when auditing by customer.
	Inherited bool

	AccountPriceID    string
	ProductLineID     string
	AttributeKey      string
	AttributeIDs      []string
	NumeratorUnitAbbr string

	Value            decimal.Decimal
	NumeratorUnitID  string
	DenominatorUnit  string
	DenominatorLabel string
	CustomerGroupID  string

	// UnitCost is the representative cost of the products this price applies to, in the same denominator unit. Zero when no comparable cost could be established.
	UnitCost    decimal.Decimal
	HasUnitCost bool
}

// pricingFinding is one flagged price.
type pricingFinding struct {
	pricingCandidate

	// PeerMedian is the median contracted price across every customer in the peer group. Zero when the customer is the only one with a contracted price here.
	PeerMedian    decimal.Decimal
	HasPeerMedian bool
	// BelowPeerFraction is how far under the peer median this price sits, 0..1.
	BelowPeerFraction decimal.Decimal
	// GrossMargin is (price - cost) / price, only meaningful when HasUnitCost.
	GrossMargin decimal.Decimal

	BelowPeerMedian   bool
	BelowTargetMargin bool
}

func peerGroupOf(candidate pricingCandidate) pricingPeerGroup {
	return pricingPeerGroup{
		ProductLineID:   candidate.ProductLineID,
		AttributeKey:    candidate.AttributeKey,
		DenominatorUnit: candidate.DenominatorUnit,
	}
}

// attributeKeyFor builds the peer-group key from an account price's attribute ids. Sorted so that the same set always produces the same key regardless of load order.
func attributeKeyFor(attributeIDs []string) string {
	if len(attributeIDs) == 0 {
		return ""
	}
	ids := make([]string, len(attributeIDs))
	copy(ids, attributeIDs)
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// analyzePricing flags contracted prices that sit well below what comparable customers pay, or that fail to clear a target gross margin.
//
// The two checks answer different questions and deliberately do not gate each other: a price can be the best in its peer group and still be profitable, and a price every customer gets can still lose money. A finding carries whichever flags apply.
func analyzePricing(candidates []pricingCandidate, targetMargin, outlierTolerance decimal.Decimal) []pricingFinding {
	medians := pricingPeerMedians(candidates)

	findings := make([]pricingFinding, 0)
	for _, candidate := range candidates {
		finding := pricingFinding{pricingCandidate: candidate}

		group := peerGroupOf(candidate)
		if median, ok := medians[group]; ok && median.IsPositive() {
			finding.PeerMedian = median
			finding.HasPeerMedian = true
			if candidate.Value.LessThan(median) {
				finding.BelowPeerFraction = median.Sub(candidate.Value).Div(median)
				finding.BelowPeerMedian = finding.BelowPeerFraction.GreaterThanOrEqual(outlierTolerance)
			}
		}

		if candidate.HasUnitCost && candidate.Value.IsPositive() {
			finding.GrossMargin = candidate.Value.Sub(candidate.UnitCost).Div(candidate.Value)
			finding.BelowTargetMargin = finding.GrossMargin.LessThan(targetMargin)
		}

		if finding.BelowPeerMedian || finding.BelowTargetMargin {
			findings = append(findings, finding)
		}
	}

	// Worst first: the deepest margin shortfall, then the deepest discount against peers.
	sort.SliceStable(findings, func(i, j int) bool {
		li := pricingSeverity(findings[i])
		lj := pricingSeverity(findings[j])
		if !li.Equal(lj) {
			return li.GreaterThan(lj)
		}
		return findings[i].CustomerName < findings[j].CustomerName
	})
	return findings
}

// pricingSeverity ranks a finding so the worst offenders surface first. A margin shortfall outranks a peer discount of the same size: losing money is worse than being generous.
func pricingSeverity(finding pricingFinding) decimal.Decimal {
	severity := decimal.Zero
	if finding.BelowTargetMargin {
		severity = severity.Add(decimal.NewFromInt(1)).Sub(finding.GrossMargin)
	}
	if finding.BelowPeerMedian {
		severity = severity.Add(finding.BelowPeerFraction)
	}
	return severity
}

// pricingPeerMedians computes the median contracted price per peer group.
//
// Median rather than mean: one deeply discounted customer should not drag the benchmark down and thereby hide itself.
func pricingPeerMedians(candidates []pricingCandidate) map[pricingPeerGroup]decimal.Decimal {
	byGroup := make(map[pricingPeerGroup][]decimal.Decimal)
	for _, candidate := range candidates {
		byGroup[peerGroupOf(candidate)] = append(byGroup[peerGroupOf(candidate)], candidate.Value)
	}

	medians := make(map[pricingPeerGroup]decimal.Decimal, len(byGroup))
	for group, values := range byGroup {
		// A single contracted price is its own median, which would compare the customer against itself and never flag. Leave the group without a median.
		if len(values) < 2 {
			continue
		}
		medians[group] = medianOf(values)
	}
	return medians
}

// productMatchesPrice mirrors the engine's account-price matching: the price applies when the product is on its line and carries every attribute the price names. Kept in step with selectAccountPrice in sales_order_pricing.go.
func productMatchesPrice(productLineID string, productAttributeIDs []string, priceLineID string, priceAttributeIDs []string) bool {
	if productLineID == "" || productLineID != priceLineID {
		return false
	}
	if len(priceAttributeIDs) == 0 {
		return true
	}
	present := make(map[string]struct{}, len(productAttributeIDs))
	for _, id := range productAttributeIDs {
		present[id] = struct{}{}
	}
	for _, id := range priceAttributeIDs {
		if _, ok := present[id]; !ok {
			return false
		}
	}
	return true
}
