package service

import "testing"

func candidate(name, line, attrs, price string) pricingCandidate {
	return pricingCandidate{
		CustomerID:    name,
		CustomerName:  name,
		ProductLineID: line,
		AttributeKey:  attrs,
		Value:         dec(price),
	}
}

func withUnit(c pricingCandidate, unit string) pricingCandidate {
	c.DenominatorUnit = unit
	return c
}

// Prices on different per-unit bases are not evidence about each other: $8 per pair and $8 per dozen buy different things, so pooling them would produce a median that describes nothing.
func TestAnalyzePricing_PeerGroupsAreScopedByDenominatorUnit(t *testing.T) {
	candidates := []pricingCandidate{
		withUnit(candidate("a", "pl_1", "", "10.00"), "unit_pair"),
		withUnit(candidate("b", "pl_1", "", "10.00"), "unit_pair"),
		// Priced per dozen, so cheaper per unit is expected and must not be flagged
		// against the per-pair prices above.
		withUnit(candidate("c", "pl_1", "", "100.00"), "unit_dozen"),
		withUnit(candidate("d", "pl_1", "", "100.00"), "unit_dozen"),
	}

	findings := analyzePricing(candidates, dec("0"), dec("0.15"))
	if len(findings) != 0 {
		t.Fatalf("expected no findings across different per-unit bases, got %+v", findings)
	}
}

func withCost(c pricingCandidate, cost string) pricingCandidate {
	c.UnitCost = dec(cost)
	c.HasUnitCost = true
	return c
}

// A customer paying well under what comparable customers pay is flagged; customers at or near the median are not.
func TestAnalyzePricing_FlagsPeerOutlier(t *testing.T) {
	candidates := []pricingCandidate{
		candidate("acme", "pl_1", "", "10.00"),
		candidate("beta", "pl_1", "", "10.00"),
		candidate("gamma", "pl_1", "", "10.00"),
		candidate("cheap", "pl_1", "", "6.00"),
	}

	findings := analyzePricing(candidates, dec("0"), dec("0.15"))

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].CustomerName != "cheap" {
		t.Errorf("flagged %q, want cheap", findings[0].CustomerName)
	}
	if !findings[0].BelowPeerMedian {
		t.Error("expected BelowPeerMedian")
	}
	if got := findings[0].BelowPeerFraction.StringFixed(2); got != "0.40" {
		t.Errorf("below-peer fraction = %s, want 0.40", got)
	}
}

// Prices for different attribute sets are not comparable, so they must not pool into one benchmark — a size-2 price is not evidence about a size-14 price.
func TestAnalyzePricing_PeerGroupsAreScopedByAttributes(t *testing.T) {
	candidates := []pricingCandidate{
		candidate("acme", "pl_1", "attr_small", "10.00"),
		candidate("beta", "pl_1", "attr_small", "10.00"),
		// Cheaper, but for a different attribute set: not an outlier.
		candidate("acme", "pl_1", "attr_large", "4.00"),
		candidate("beta", "pl_1", "attr_large", "4.00"),
	}

	findings := analyzePricing(candidates, dec("0"), dec("0.15"))
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d (%+v)", len(findings), findings)
	}
}

// One deeply discounted customer must not drag the benchmark down far enough to hide itself — which is exactly what a mean would do.
func TestAnalyzePricing_MedianResistsASingleOutlier(t *testing.T) {
	candidates := []pricingCandidate{
		candidate("a", "pl_1", "", "10.00"),
		candidate("b", "pl_1", "", "10.00"),
		candidate("c", "pl_1", "", "10.00"),
		candidate("d", "pl_1", "", "1.00"),
	}

	findings := analyzePricing(candidates, dec("0"), dec("0.15"))
	if len(findings) != 1 || findings[0].CustomerName != "d" {
		t.Fatalf("expected only d flagged, got %+v", findings)
	}
	if got := findings[0].PeerMedian.StringFixed(2); got != "10.00" {
		t.Errorf("median = %s, want 10.00", got)
	}
}

// A lone contracted price has no peer to compare against and must not be flagged as an outlier against itself.
func TestAnalyzePricing_SinglePriceHasNoPeerMedian(t *testing.T) {
	findings := analyzePricing([]pricingCandidate{candidate("solo", "pl_1", "", "1.00")}, dec("0"), dec("0.15"))
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

// The margin check is independent of the peer check: a price everyone gets can still fail to clear target margin.
func TestAnalyzePricing_FlagsBelowTargetMargin(t *testing.T) {
	candidates := []pricingCandidate{
		withCost(candidate("a", "pl_1", "", "10.00"), "8.50"),
		withCost(candidate("b", "pl_1", "", "10.00"), "8.50"),
	}

	findings := analyzePricing(candidates, dec("0.30"), dec("0.15"))
	if len(findings) != 2 {
		t.Fatalf("expected both flagged, got %d", len(findings))
	}
	for _, finding := range findings {
		if finding.BelowPeerMedian {
			t.Error("should not be a peer outlier: both pay the same")
		}
		if !finding.BelowTargetMargin {
			t.Error("expected BelowTargetMargin")
		}
		if got := finding.GrossMargin.StringFixed(2); got != "0.15" {
			t.Errorf("margin = %s, want 0.15", got)
		}
	}
}

// A price clearing the target with no peer discount produces no finding at all.
func TestAnalyzePricing_HealthyPriceIsNotFlagged(t *testing.T) {
	candidates := []pricingCandidate{
		withCost(candidate("a", "pl_1", "", "10.00"), "5.00"),
		withCost(candidate("b", "pl_1", "", "10.00"), "5.00"),
	}

	findings := analyzePricing(candidates, dec("0.30"), dec("0.15"))
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

// Margin shortfalls outrank peer discounts so the costly problems read first.
func TestAnalyzePricing_OrdersMarginShortfallsFirst(t *testing.T) {
	candidates := []pricingCandidate{
		candidate("peer-outlier", "pl_1", "", "5.00"),
		candidate("a", "pl_1", "", "10.00"),
		candidate("b", "pl_1", "", "10.00"),
		withCost(candidate("loss-maker", "pl_2", "", "10.00"), "9.90"),
		withCost(candidate("c", "pl_2", "", "10.00"), "9.90"),
	}

	findings := analyzePricing(candidates, dec("0.30"), dec("0.15"))
	if len(findings) < 2 {
		t.Fatalf("expected several findings, got %d", len(findings))
	}
	if !findings[0].BelowTargetMargin {
		t.Errorf("first finding should be a margin shortfall, got %+v", findings[0])
	}
}

func TestProductMatchesPrice(t *testing.T) {
	cases := []struct {
		name       string
		productAtt []string
		priceAtt   []string
		productLn  string
		priceLn    string
		want       bool
	}{
		{"no attributes matches whole line", []string{"a"}, nil, "pl_1", "pl_1", true},
		{"subset matches", []string{"a", "b", "c"}, []string{"a", "c"}, "pl_1", "pl_1", true},
		{"missing attribute does not match", []string{"a"}, []string{"a", "b"}, "pl_1", "pl_1", false},
		{"different line never matches", []string{"a"}, nil, "pl_1", "pl_2", false},
		{"product with no line never matches", []string{"a"}, nil, "", "pl_1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := productMatchesPrice(tc.productLn, tc.productAtt, tc.priceLn, tc.priceAtt)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAttributeKeyFor_IsOrderIndependent(t *testing.T) {
	if attributeKeyFor([]string{"b", "a"}) != attributeKeyFor([]string{"a", "b"}) {
		t.Error("attribute key depends on input order")
	}
	if attributeKeyFor(nil) != "" {
		t.Error("empty attribute set should produce an empty key")
	}
}
