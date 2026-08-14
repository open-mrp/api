package analyticsep

import (
	"context"
	"testing"

	// Registers the finding definitions whose sub-fields this file exercises.
	_ "github.com/augno/api/services/api-gateway/internal/resourceregistry"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
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

	// The sweep runs in core-service now, so the presenter is exercised against the
	// flagged row it actually receives.
	out := presentCustomerPricing(ctx, configuredPricingResponse(nil, nil))
	if out.Findings == nil || len(out.Findings.Data) != 1 {
		t.Fatal("presenter produced no finding")
	}
	return ctx, &out.Findings.Data[0]
}

// configuredPricingResponse is one flagged price as core-service reports it.
func configuredPricingResponse(peerMedian, fraction *string) *pb.AnalyzeCustomerPricingResponse {
	return &pb.AnalyzeCustomerPricingResponse{
		PricesAnalyzed: 1,
		Findings: []*pb.CustomerPricingFindingProto{{
			AccountPriceId:          "acpr_1",
			CustomerId:              "acc_buyer",
			ProductLineId:           "pl_1",
			AttributeIds:            []string{"at_1", "at_2"},
			UnitPrice:               "8.00",
			NumeratorUnitId:         "un_usd",
			NumeratorUnitAbbr:       "$",
			DenominatorUnitId:       "un_pair",
			DenominatorAbbr:         "pr",
			PeerMedianPrice:         peerMedian,
			BelowPeerMedianFraction: fraction,
			Origin:                  string(constants.AccountPriceOriginDirect),
			Reason:                  string(constants.PricingFindingReasonBelowTargetMargin),
		}},
	}
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
	median := "10.0000"
	fraction := "0.2000"
	out := presentCustomerPricing(ctx, configuredPricingResponse(&median, &fraction))
	finding := &out.Findings.Data[0]
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

	// The roll-up happens in core-service now, so the presenter is exercised against
	// the aggregated row it actually receives.
	median := "12.0000"
	fraction := "0.1667"
	resp := &pb.AnalyzeRealizedMarginsResponse{
		LinesAnalyzed:         1,
		RelationshipsAnalyzed: 1,
		Findings: []*pb.RealizedMarginFindingProto{{
			CustomerId:              "acc_buyer",
			CustomerGroupId:         "acgp_1",
			ItemId:                  "it_1",
			ProductLineId:           "pl_1",
			UnitAbbreviation:        "pr",
			QuantityInvoiced:        "100",
			Revenue:                 "1000.00",
			Cost:                    "900.00",
			AverageUnitPrice:        "10.00",
			PeerMedianPrice:         &median,
			BelowPeerMedianFraction: &fraction,
			LineCount:               1,
			Reason:                  string(constants.PricingFindingReasonBelowTargetMargin),
		}},
	}

	out := presentRealizedMargins(ctx, resp)
	if out.Findings == nil || len(out.Findings.Data) != 1 {
		t.Fatal("presenter produced no finding")
	}
	return ctx, &out.Findings.Data[0]
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
