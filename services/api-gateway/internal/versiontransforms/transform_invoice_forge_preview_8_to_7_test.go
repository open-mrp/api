package versiontransforms

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func invoiceQueryFromPreview7(route string, query url.Values) url.Values {
	return version.TransformQuery(version.V1_0_Forge_Preview7, version.V1_0_Forge_Preview8, constants.ObjectTypeInvoice, route, query)
}

// preview.7 matched q anywhere in every field, so its searches keep doing that on preview.8.
func TestInvoicePreview8To7_SearchMatchesAnywhere(t *testing.T) {
	query := invoiceQueryFromPreview7(invoiceListRoute, url.Values{"q": {"2410"}, "status": {"unpaid"}})

	if got := query.Get("q_match"); got != string(constants.InvoiceSearchMatchContains) {
		t.Errorf("q_match = %q, want %q", got, constants.InvoiceSearchMatchContains)
	}
	if got := query.Get("q"); got != "2410" {
		t.Errorf("q must pass through, got %q", got)
	}
	if got := query.Get("status"); got != "unpaid" {
		t.Errorf("other parameters must pass through, status = %q", got)
	}
}

func TestInvoicePreview8To7_ExplicitMatchIsKept(t *testing.T) {
	query := invoiceQueryFromPreview7(invoiceListRoute, url.Values{"q": {"2410"}, "q_match": {"prefix"}})

	if got := query.Get("q_match"); got != "prefix" {
		t.Errorf("q_match = %q, want the caller's own value", got)
	}
}

func TestInvoicePreview8To7_NoSearchIsUntouched(t *testing.T) {
	for name, query := range map[string]url.Values{
		"omitted": {"status": {"paid"}},
		"empty":   {"q": {""}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := invoiceQueryFromPreview7(invoiceListRoute, query); got.Has("q_match") {
				t.Errorf("a list without a search gained a q_match: %v", got)
			}
		})
	}
}

// The retrieve and update reject parameters they do not know, so they must never gain one.
func TestInvoicePreview8To7_OtherRoutesAreUntouched(t *testing.T) {
	query := invoiceQueryFromPreview7("/v1/finance/invoices/{id}", url.Values{"q": {"2410"}})

	if query.Has("q_match") {
		t.Errorf("the retrieve gained a q_match: %v", query)
	}
}

func TestInvoicePreview8To7_LatestIsUntouched(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview8, version.V1_0_Forge_Preview8,
		constants.ObjectTypeInvoice, invoiceListRoute, url.Values{"q": {"2410"}})

	if query.Has("q_match") {
		t.Errorf("a latest-version search must get the prefix default, not an explicit match: %v", query)
	}
}

// Older versions reach the transform through every adjacent bridge in turn.
func TestInvoicePreview8To7_ChainsToOlderVersions(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview1, version.V1_0_Forge_Preview8,
		constants.ObjectTypeInvoice, invoiceListRoute, url.Values{"q": {"2410"}})

	if got := query.Get("q_match"); got != string(constants.InvoiceSearchMatchContains) {
		t.Errorf("q_match = %q, want %q for a preview.1 search", got, constants.InvoiceSearchMatchContains)
	}
}

func TestInvoicePreview8To7_ResponsesPassThrough(t *testing.T) {
	single := map[string]any{"object": "invoice", "id": "inv_1", "number": "2410"}
	got := version.Transform(version.V1_0_Forge_Preview8, version.V1_0_Forge_Preview7, constants.ObjectTypeInvoice, single)
	if got["number"] != "2410" || got["id"] != "inv_1" {
		t.Errorf("the invoice shape did not change between versions, got %v", got)
	}

	list := map[string]any{"object": "list", "data": []any{map[string]any{"object": "invoice", "id": "inv_1"}}}
	got = version.Transform(version.V1_0_Forge_Preview8, version.V1_0_Forge_Preview7, constants.ObjectTypeInvoice, list)
	if data, ok := got["data"].([]any); !ok || len(data) != 1 {
		t.Errorf("a list of invoices must pass through, got %v", got)
	}

	if got := version.Transform(version.V1_0_Forge_Preview8, version.V1_0_Forge_Preview7, constants.ObjectTypeInvoice, nil); got != nil {
		t.Errorf("missing data must stay missing, got %v", got)
	}
}
