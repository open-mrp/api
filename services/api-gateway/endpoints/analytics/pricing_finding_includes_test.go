package analyticsep

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	// Registers the finding definitions whose sub-fields this file exercises.
	_ "github.com/augno/api/services/api-gateway/internal/resourceregistry"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

// These tests close the loop between the two halves of the include system that are written in different packages and can silently drift apart: the presenter stashes foreign keys under string keys, and the registry's ExtractIDs reads them back under string keys. A typo on either side produces no error — the relation just serializes as null forever — so the round trip is asserted rather than assumed.

func subFieldNamed(t *testing.T, ot constants.ObjectType, key string) resourcekit.SubField {
	t.Helper()
	def := resourcekit.Lookup(ot)
	if def == nil {
		t.Fatalf("no resource definition registered for %s", ot)
	}
	for _, sub := range def.Subs {
		if sub.Key == key {
			return sub
		}
	}
	t.Fatalf("%s has no registered sub-field %q", ot, key)
	return resourcekit.SubField{}
}

func configuredFindingForIncludes(t *testing.T) (context.Context, *apiresource.CustomerPricingFinding) {
	t.Helper()
	ctx := resourcekit.WithLoadMeta(context.Background())

	candidates := []pricingCandidate{{
		CustomerID:        "acc_buyer",
		CustomerName:      "Acme",
		AccountPriceID:    "acpr_1",
		ProductLineID:     "pl_1",
		AttributeKey:      "at_1,at_2",
		AttributeIDs:      []string{"at_1", "at_2"},
		NumeratorUnitID:   "un_usd",
		NumeratorUnitAbbr: "$",
		DenominatorUnit:   "un_pair",
		DenominatorLabel:  "pr",
		Value:             dec("8.00"),
		UnitCost:          dec("7.60"),
		HasUnitCost:       true,
	}}
	findings := analyzePricing(candidates, dec("0.30"), dec("0.15"))
	if len(findings) != 1 {
		t.Fatalf("expected the fixture to produce one finding, got %d", len(findings))
	}

	resp := presentPricingAnalysis(ctx, candidates, findings, nil)
	if resp.Findings == nil || len(resp.Findings.Data) != 1 {
		t.Fatal("presenter produced no finding")
	}
	return ctx, &resp.Findings.Data[0]
}

// The presenter must leave expandable fields nil; anything it assigned would be serialized verbatim, since nothing strips them back out.
func TestPresentPricingAnalysis_LeavesExpandablesUnset(t *testing.T) {
	_, finding := configuredFindingForIncludes(t)

	if finding.Customer != nil || finding.ProductLine != nil || finding.Attributes != nil {
		t.Error("expandable relations must be nil until the include resolver populates them")
	}
	if finding.UnitPrice == nil {
		t.Fatal("unit price should always be present")
	}
	if finding.UnitPrice.DisplayValue != "$8.00 / pr" {
		t.Errorf("display value = %q, want %q", finding.UnitPrice.DisplayValue, "$8.00 / pr")
	}
	if finding.Reason != constants.PricingFindingReasonBelowTargetMargin {
		t.Errorf("reason = %q, want below_target_margin", finding.Reason)
	}
	if finding.Origin != constants.AccountPriceOriginDirect {
		t.Errorf("origin = %q, want direct", finding.Origin)
	}
}

func TestPricingFindingIncludes_CustomerRoundTrip(t *testing.T) {
	ctx, finding := configuredFindingForIncludes(t)
	sub := subFieldNamed(t, constants.ObjectTypeCustomerPricingFinding, "customer")

	ids := sub.ExtractIDs(ctx, finding)
	if len(ids) != 1 || ids[0] != "acc_buyer" {
		t.Fatalf("extracted %v, want [acc_buyer] — the presenter and the registry disagree on the meta key", ids)
	}

	sub.Populate(ctx, finding, map[string]any{"acc_buyer": &apiresource.Customer{ID: "acc_buyer", Name: "Acme"}})
	if finding.Customer == nil || finding.Customer.ID != "acc_buyer" {
		t.Fatal("customer was not populated onto the finding")
	}
}

func TestPricingFindingIncludes_ProductLineRoundTrip(t *testing.T) {
	ctx, finding := configuredFindingForIncludes(t)
	sub := subFieldNamed(t, constants.ObjectTypeCustomerPricingFinding, "product_line")

	ids := sub.ExtractIDs(ctx, finding)
	if len(ids) != 1 || ids[0] != "pl_1" {
		t.Fatalf("extracted %v, want [pl_1]", ids)
	}

	sub.Populate(ctx, finding, map[string]any{"pl_1": &apiresource.ProductLine{ID: "pl_1"}})
	if finding.ProductLine == nil || finding.ProductLine.ID != "pl_1" {
		t.Fatal("product line was not populated onto the finding")
	}
}

// The attribute list is the one many-valued relation, so it exercises the slice side of the meta table.
func TestPricingFindingIncludes_AttributesRoundTrip(t *testing.T) {
	ctx, finding := configuredFindingForIncludes(t)
	sub := subFieldNamed(t, constants.ObjectTypeCustomerPricingFinding, "attributes")

	ids := sub.ExtractIDs(ctx, finding)
	if len(ids) != 2 || ids[0] != "at_1" || ids[1] != "at_2" {
		t.Fatalf("extracted %v, want [at_1 at_2]", ids)
	}

	sub.Populate(ctx, finding, map[string]any{
		"at_1": &apiresource.Attribute{ID: "at_1", Value: "Navy"},
		"at_2": &apiresource.Attribute{ID: "at_2", Value: "Large"},
	})
	if finding.Attributes == nil || len(finding.Attributes.Data) != 2 {
		t.Fatalf("attributes were not populated: %+v", finding.Attributes)
	}
}

// A unit loaded for the price attaches to every rate on the finding, so the peer median is labelled on the same basis as the price it is compared against.
func TestPricingFindingIncludes_UnitAttachesToEveryRate(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	candidates := []pricingCandidate{
		{CustomerID: "a", CustomerName: "A", AccountPriceID: "acpr_a", ProductLineID: "pl_1", DenominatorUnit: "un_pair", DenominatorLabel: "pr", NumeratorUnitID: "un_usd", NumeratorUnitAbbr: "$", Value: dec("6.00")},
		{CustomerID: "b", CustomerName: "B", AccountPriceID: "acpr_b", ProductLineID: "pl_1", DenominatorUnit: "un_pair", DenominatorLabel: "pr", NumeratorUnitID: "un_usd", NumeratorUnitAbbr: "$", Value: dec("10.00")},
		{CustomerID: "c", CustomerName: "C", AccountPriceID: "acpr_c", ProductLineID: "pl_1", DenominatorUnit: "un_pair", DenominatorLabel: "pr", NumeratorUnitID: "un_usd", NumeratorUnitAbbr: "$", Value: dec("10.00")},
	}
	findings := analyzePricing(candidates, decimal.Zero, dec("0.15"))
	resp := presentPricingAnalysis(ctx, candidates, findings, nil)
	finding := &resp.Findings.Data[0]
	if finding.PeerMedianPrice == nil {
		t.Fatal("expected the flagged finding to carry a peer median")
	}

	sub := subFieldNamed(t, constants.ObjectTypeCustomerPricingFinding, "unit_price.denominator_unit")
	ids := sub.ExtractIDs(ctx, finding)
	if len(ids) != 1 || ids[0] != "un_pair" {
		t.Fatalf("extracted %v, want [un_pair]", ids)
	}

	sub.Populate(ctx, finding, map[string]any{"un_pair": &apiresource.Unit{ID: "un_pair", Abbreviation: "pr"}})
	if finding.UnitPrice.DenominatorUnit == nil {
		t.Error("unit price denominator unit was not populated")
	}
	if finding.PeerMedianPrice.DenominatorUnit == nil {
		t.Error("peer median denominator unit was not populated")
	}
}

func realizedFindingForIncludes(t *testing.T) (context.Context, *apiresource.RealizedMarginFinding) {
	t.Helper()
	ctx := resourcekit.WithLoadMeta(context.Background())

	lines := []realizedLine{{
		CustomerID:      "acc_buyer",
		CustomerName:    "Acme",
		CustomerGroupID: "acgp_1",
		ItemID:          "it_1",
		SKU:             "SKU-1",
		ProductLineID:   "pl_1",
		UnitAbbr:        "pr",
		Quantity:        dec("100"),
		Revenue:         dec("1000.00"),
		Cost:            dec("900.00"),
	}}
	aggregates := aggregateRealizedLines(lines)
	findings := analyzeRealizedMargins(aggregates, dec("0.30"), dec("0.15"))
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}

	resp := presentRealizedMargins(ctx, len(lines), aggregates, findings, nil)
	if resp.Findings == nil || len(resp.Findings.Data) != 1 {
		t.Fatal("presenter produced no finding")
	}
	return ctx, &resp.Findings.Data[0]
}

func TestRealizedMarginFindingIncludes_RoundTrip(t *testing.T) {
	cases := []struct {
		key    string
		wantID string
		check  func(*apiresource.RealizedMarginFinding) bool
		loaded any
	}{
		{"customer", "acc_buyer", func(f *apiresource.RealizedMarginFinding) bool { return f.Customer != nil }, &apiresource.Customer{ID: "acc_buyer"}},
		{"customer_group", "acgp_1", func(f *apiresource.RealizedMarginFinding) bool { return f.CustomerGroup != nil }, &apiresource.AccountGroup{ID: "acgp_1"}},
		{"item", "it_1", func(f *apiresource.RealizedMarginFinding) bool { return f.Item != nil }, &apiresource.Item{ID: "it_1"}},
		{"product_line", "pl_1", func(f *apiresource.RealizedMarginFinding) bool { return f.ProductLine != nil }, &apiresource.ProductLine{ID: "pl_1"}},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			ctx, finding := realizedFindingForIncludes(t)
			sub := subFieldNamed(t, constants.ObjectTypeRealizedMarginFinding, tc.key)

			ids := sub.ExtractIDs(ctx, finding)
			if len(ids) != 1 || ids[0] != tc.wantID {
				t.Fatalf("extracted %v, want [%s]", ids, tc.wantID)
			}

			sub.Populate(ctx, finding, map[string]any{tc.wantID: tc.loaded})
			if !tc.check(finding) {
				t.Errorf("%s was not populated onto the finding", tc.key)
			}
		})
	}
}

// Money has no currency unit in the invoiced-sales payload, so it must render as a bare amount rather than inventing a symbol; the invoiced quantity does know its unit and keeps it.
func TestPresentRealizedMargins_ComputedAmounts(t *testing.T) {
	_, finding := realizedFindingForIncludes(t)

	if finding.QuantityInvoiced.DisplayValue != "100.00 pr" {
		t.Errorf("quantity display = %q, want %q", finding.QuantityInvoiced.DisplayValue, "100.00 pr")
	}
	if finding.Revenue.DisplayValue != "1000.00" {
		t.Errorf("revenue display = %q, want an unadorned amount", finding.Revenue.DisplayValue)
	}
	if finding.AverageUnitPrice.DisplayValue != "10.00 / pr" {
		t.Errorf("average price display = %q, want %q", finding.AverageUnitPrice.DisplayValue, "10.00 / pr")
	}
	if finding.Customer != nil || finding.Item != nil {
		t.Error("expandable relations must be nil until the include resolver populates them")
	}
}
