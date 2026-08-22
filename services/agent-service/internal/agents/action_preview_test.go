package agents

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

// pathGateway answers GETs from a canned body per path, so a test can supply (or withhold) the target
// record's current state.
type pathGateway struct {
	bodies map[string]string
	calls  []domain.GatewayRequest
}

func (g *pathGateway) Do(_ context.Context, req domain.GatewayRequest) (string, error) {
	g.calls = append(g.calls, req)
	body, ok := g.bodies[req.Path]
	if !ok {
		return "", errors.New("not found")
	}
	return body, nil
}

const priceSchema = `{"type":"object","properties":{
	"id":{"type":"string","description":"Price ID."},
	"unit_price":{"type":"object","properties":{"amount":{"type":"string","format":"decimal","description":"Price per unit."}}},
	"minimum_quantity":{"type":"integer","description":"Smallest order this price applies to."},
	"effective_from":{"type":"string","format":"date"},
	"include":{"type":"array","items":{"type":"string"}}
}}`

func updatePriceDescriptor() EndpointToolDescriptor {
	return EndpointToolDescriptor{
		Slug:          "update_account_price",
		DisplayName:   "Update Customer Price",
		Method:        "PUT",
		RouteTemplate: "/v1/core/account-prices/{id}",
		InputSchema:   priceSchema,
		Params: []EndpointToolParam{
			{Name: "id", In: EndpointToolParamPath},
			{Name: "unit_price", In: EndpointToolParamBody},
			{Name: "minimum_quantity", In: EndpointToolParamBody},
			{Name: "effective_from", In: EndpointToolParamBody},
			{Name: "include", In: EndpointToolParamQuery, Array: true},
		},
	}
}

// TestBuildActionPreview_UpdateShowsBeforeAfter is the case the preview exists for: a reviewer must see
// which record changes, what each field holds today, and which fields the call leaves alone.
func TestBuildActionPreview_UpdateShowsBeforeAfter(t *testing.T) {
	gw := &pathGateway{bodies: map[string]string{
		"/v1/core/account-prices/acpr_1": `{"object":"account_price","id":"acpr_1","name":"ACME · Widget A","unit_price":{"amount":"12.50"},"minimum_quantity":100,"effective_from":"2026-01-01"}`,
	}}
	input := json.RawMessage(`{"id":"acpr_1","unit_price":{"amount":"11.00"},"minimum_quantity":100,"include":["customer"]}`)

	p := buildActionPreview(context.Background(), updatePriceDescriptor(), input, &domain.HandlerRunContext{GatewayClient: gw})
	if p == nil {
		t.Fatal("nil preview")
	}

	if p.Operation != "update" {
		t.Errorf("operation = %q, want update", p.Operation)
	}
	if p.Title != "Update Customer Price" {
		t.Errorf("title = %q", p.Title)
	}
	if !p.BeforeState {
		t.Error("expected current state to be read")
	}
	if p.Resource == nil || p.Resource.Object != "account_price" || p.Resource.ID != "acpr_1" {
		t.Fatalf("resource = %+v", p.Resource)
	}
	if p.Resource.Label != "ACME · Widget A" {
		t.Errorf("label = %q, want the record's own name", p.Resource.Label)
	}
	if !p.Resource.Linkable {
		t.Error("account_price has a detail page, so the preview header should be linkable")
	}

	// The addressed id belongs in the header, not among the changes.
	if len(p.Identifiers) != 1 || p.Identifiers[0].Key != "id" {
		t.Errorf("identifiers = %+v", p.Identifiers)
	}

	byKey := map[string]PreviewField{}
	for _, f := range p.Fields {
		byKey[f.Key] = f
	}
	if _, ok := byKey["include"]; ok {
		t.Error("`include` shapes the response and must not read as a field being set")
	}
	if _, ok := byKey["id"]; ok {
		t.Error("path parameters must not appear twice")
	}

	price, ok := byKey["unit_price.amount"]
	if !ok {
		t.Fatalf("nested field missing; got %v", keysOf(byKey))
	}
	if price.Label != "Unit price › Amount" {
		t.Errorf("label = %q, want the grouped label", price.Label)
	}
	if price.Before != "12.50" || price.After != "11.00" {
		t.Errorf("unit price %v → %v, want 12.50 → 11.00", price.Before, price.After)
	}
	if !price.Changed {
		t.Error("unit price changed")
	}
	if price.Format != "decimal" {
		t.Errorf("format = %q, want decimal", price.Format)
	}
	if price.Description != "Price per unit." {
		t.Errorf("description = %q, want the schema's", price.Description)
	}

	// A value re-sent unchanged is the noise the old raw-input card could not distinguish from a change.
	minQty, ok := byKey["minimum_quantity"]
	if !ok {
		t.Fatal("minimum_quantity missing")
	}
	if minQty.Changed {
		t.Errorf("minimum quantity resent as %v over %v must not read as a change", minQty.After, minQty.Before)
	}
}

// TestBuildActionPreview_NoBeforeStateMarksEveryFieldChanged proves an unreadable target never presents
// a missing current value as an empty one.
func TestBuildActionPreview_NoBeforeStateMarksEveryFieldChanged(t *testing.T) {
	gw := &pathGateway{bodies: map[string]string{}} // read fails
	input := json.RawMessage(`{"id":"acpr_1","minimum_quantity":50}`)

	p := buildActionPreview(context.Background(), updatePriceDescriptor(), input, &domain.HandlerRunContext{GatewayClient: gw})
	if p.BeforeState {
		t.Error("before_state must be false when the record could not be read")
	}
	for _, f := range p.Fields {
		if f.Before != nil {
			t.Errorf("field %q carries a before value with no state read", f.Key)
		}
		if !f.Changed {
			t.Errorf("field %q must count as a change when the current value is unknown", f.Key)
		}
	}
	if p.Resource == nil || p.Resource.ID != "acpr_1" {
		t.Errorf("resource should still name the addressed id, got %+v", p.Resource)
	}
}

// TestBuildActionPreview_CreateSkipsRead: a create has no target, so the preview must not spend a
// gateway round trip looking for one.
func TestBuildActionPreview_CreateSkipsRead(t *testing.T) {
	desc := EndpointToolDescriptor{
		Slug:          "create_account_price",
		DisplayName:   "Create Customer Price",
		Method:        "POST",
		RouteTemplate: "/v1/core/account-prices",
		InputSchema:   priceSchema,
		Params:        []EndpointToolParam{{Name: "unit_price", In: EndpointToolParamBody}},
	}
	gw := &pathGateway{bodies: map[string]string{}}

	p := buildActionPreview(context.Background(), desc, json.RawMessage(`{"unit_price":{"amount":"9.00"}}`), &domain.HandlerRunContext{GatewayClient: gw})
	if p.Operation != "create" {
		t.Errorf("operation = %q, want create", p.Operation)
	}
	if len(gw.calls) != 0 {
		t.Errorf("create issued %d reads, want none", len(gw.calls))
	}
	if p.Resource != nil {
		t.Errorf("create has no target record, got %+v", p.Resource)
	}
	if len(p.Fields) != 1 || p.Fields[0].Key != "unit_price.amount" {
		t.Errorf("fields = %+v", p.Fields)
	}
}

// TestBuildActionPreview_ActionRoute checks that a named operation is labelled as one and that its
// current state is read from the record, not the action sub-route.
func TestBuildActionPreview_ActionRoute(t *testing.T) {
	desc := EndpointToolDescriptor{
		Slug:          "issue_sales_order",
		DisplayName:   "Issue Sales Order",
		Method:        "PUT",
		RouteTemplate: "/v1/core/sales-orders/{id}/actions/issue",
		InputSchema:   `{"type":"object","properties":{"id":{"type":"string"}}}`,
		Params:        []EndpointToolParam{{Name: "id", In: EndpointToolParamPath}},
	}
	gw := &pathGateway{bodies: map[string]string{
		"/v1/core/sales-orders/so_1": `{"object":"sales_order","id":"so_1","number":"1042"}`,
	}}

	p := buildActionPreview(context.Background(), desc, json.RawMessage(`{"id":"so_1"}`), &domain.HandlerRunContext{GatewayClient: gw})
	if p.Operation != "action" {
		t.Errorf("operation = %q, want action", p.Operation)
	}
	if len(gw.calls) != 1 || gw.calls[0].Path != "/v1/core/sales-orders/so_1" {
		t.Fatalf("read path = %+v, want the record's own route", gw.calls)
	}
	if gw.calls[0].Method != "GET" {
		t.Errorf("preview read used %s; it must only ever GET", gw.calls[0].Method)
	}
	if p.Resource == nil || p.Resource.Label != "1042" {
		t.Errorf("resource = %+v", p.Resource)
	}
}

// TestBuildActionPreview_ArrayOfObjects covers a call that sets line items: each line's fields must be
// numbered so the reviewer can tell them apart.
func TestBuildActionPreview_ArrayOfObjects(t *testing.T) {
	desc := EndpointToolDescriptor{
		Slug:          "create_sales_order",
		DisplayName:   "Create Sales Order",
		Method:        "POST",
		RouteTemplate: "/v1/core/sales-orders",
		InputSchema:   `{"type":"object","properties":{"lines":{"type":"array","items":{"type":"object","properties":{"quantity":{"type":"integer"},"product_id":{"type":"string"}}}},"tags":{"type":"array","items":{"type":"string"}}}}`,
		Params:        []EndpointToolParam{{Name: "lines", In: EndpointToolParamBody}, {Name: "tags", In: EndpointToolParamBody}},
	}

	input := json.RawMessage(`{"lines":[{"quantity":10,"product_id":"prd_a"},{"quantity":4,"product_id":"prd_b"}],"tags":["rush","west"]}`)
	p := buildActionPreview(context.Background(), desc, input, &domain.HandlerRunContext{})

	byKey := map[string]PreviewField{}
	for _, f := range p.Fields {
		byKey[f.Key] = f
	}
	first, ok := byKey["lines.0.quantity"]
	if !ok {
		t.Fatalf("line fields missing: %v", keysOf(byKey))
	}
	if first.Label != "Line 1 › Quantity" {
		t.Errorf("label = %q, want a numbered line label", first.Label)
	}
	if _, ok := byKey["lines.1.product_id"]; !ok {
		t.Error("second line missing")
	}
	// A list of plain strings is one value, not one row per element.
	tags, ok := byKey["tags"]
	if !ok {
		t.Fatal("tags missing")
	}
	if tags.After != "rush, west" {
		t.Errorf("tags = %v, want a single joined row", tags.After)
	}
}

func TestBuildActionPreview_UnknownToolAndBadInput(t *testing.T) {
	if p := BuildActionPreview(context.Background(), "not_a_catalog_tool", json.RawMessage(`{}`), &domain.HandlerRunContext{}); p != nil {
		t.Error("a built-in tool has no endpoint to describe; want nil")
	}
	if p := buildActionPreview(context.Background(), updatePriceDescriptor(), json.RawMessage(`"not an object"`), &domain.HandlerRunContext{}); p != nil {
		t.Error("want nil for input that isn't a JSON object")
	}
}

func TestHumanizeKey(t *testing.T) {
	cases := map[string]string{
		"unit_price":       "Unit price",
		"id":               "ID",
		"customer_id":      "Customer ID",
		"minimum_quantity": "Minimum quantity",
		"weeks_of_sales":   "Weeks of sales",
		"image_url":        "Image URL",
	}
	for in, want := range cases {
		if got := humanizeKey(in); got != want {
			t.Errorf("humanizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func keysOf(m map[string]PreviewField) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestBuildActionPreview_RealCatalogTool runs the builder against the generated catalog rather than a
// hand-written descriptor, so the real schema shape (nested `rate`, `include` enum, PATCH semantics) is
// what gets exercised. The synthetic cases above can't catch a mismatch with what the API actually emits.
func TestBuildActionPreview_RealCatalogTool(t *testing.T) {
	gw := &pathGateway{bodies: map[string]string{
		"/v1/sales/account-prices/acpr_1": `{"object":"account_price","id":"acpr_1","name":"ACME · Widget A","rate":{"value":"12.50","numerator_unit_id":"unt_usd","denominator_unit_id":"unt_pair"},"product_line_id":"prdl_1"}`,
	}}
	input := json.RawMessage(`{"id":"acpr_1","rate":{"value":"11.00","numerator_unit_id":"unt_usd","denominator_unit_id":"unt_pair"},"include":["recipient_account"]}`)

	p := BuildActionPreview(context.Background(), "update_account_price", input, &domain.HandlerRunContext{GatewayClient: gw})
	if p == nil {
		t.Fatal("nil preview for a real catalog tool")
	}
	if p.Operation != "update" || p.Title != "Update Account Price" {
		t.Errorf("operation/title = %q/%q", p.Operation, p.Title)
	}
	if !p.BeforeState {
		t.Fatalf("current state not read; gateway saw %+v", gw.calls)
	}
	if p.Resource == nil || p.Resource.Label != "ACME · Widget A" || !p.Resource.Linkable {
		t.Fatalf("resource = %+v", p.Resource)
	}

	byKey := map[string]PreviewField{}
	for _, f := range p.Fields {
		byKey[f.Key] = f
	}
	value, ok := byKey["rate.value"]
	if !ok {
		t.Fatalf("rate.value missing; got %v", keysOf(byKey))
	}
	if value.Before != "12.50" || value.After != "11.00" || !value.Changed {
		t.Errorf("rate.value %v → %v (changed=%v)", value.Before, value.After, value.Changed)
	}
	if value.Format != "decimal" {
		t.Errorf("rate.value format = %q, want decimal from the real schema", value.Format)
	}
	// The units are re-sent because the rate is replaced whole, not merged — they must read as unchanged.
	for _, key := range []string{"rate.numerator_unit_id", "rate.denominator_unit_id"} {
		if f, ok := byKey[key]; !ok || f.Changed {
			t.Errorf("%s should be present and unchanged, got %+v", key, f)
		}
	}
	if _, ok := byKey["include"]; ok {
		t.Error("`include` must not read as a field being set")
	}
	// PATCH means absent fields are untouched; the preview must not invent rows for them.
	if _, ok := byKey["product_line_id"]; ok {
		t.Error("a field the call never sent must not appear as a change")
	}
}
