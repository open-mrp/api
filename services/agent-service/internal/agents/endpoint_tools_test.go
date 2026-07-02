package agents

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/augno/api/services/agent-service/internal/domain"
)

type fakeGatewayClient struct {
	got    domain.GatewayRequest
	result string
}

func (f *fakeGatewayClient) Do(_ context.Context, req domain.GatewayRequest) (string, error) {
	f.got = req
	return f.result, nil
}

func TestEndpointToolHandler_RoutesParams(t *testing.T) {
	desc := EndpointToolDescriptor{
		Slug:          "retrieve_thing",
		Method:        "GET",
		RouteTemplate: "/v1/things/{id}",
		Params: []EndpointToolParam{
			{Name: "id", In: EndpointToolParamPath},
			{Name: "expand", In: EndpointToolParamQuery},
			{Name: "tags", In: EndpointToolParamQuery, Array: true},
		},
	}
	fake := &fakeGatewayClient{result: "ok"}
	handler := endpointToolHandler(desc)

	input := json.RawMessage(`{"id":"thing_1","expand":"owner","tags":["a","b"]}`)
	out, err := handler(context.Background(), input, &domain.HandlerRunContext{GatewayClient: fake})
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Errorf("result = %q, want ok", out)
	}
	if fake.got.Path != "/v1/things/thing_1" {
		t.Errorf("path = %q, want /v1/things/thing_1", fake.got.Path)
	}
	if got := fake.got.Query.Get("expand"); got != "owner" {
		t.Errorf("expand query = %q, want owner", got)
	}
	if got := fake.got.Query["tags"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("tags query = %v, want [a b]", got)
	}
	if len(fake.got.Body) != 0 {
		t.Errorf("GET request should have no body, got %q", string(fake.got.Body))
	}
}

func TestEndpointToolHandler_BuildsBody(t *testing.T) {
	desc := EndpointToolDescriptor{
		Slug:          "create_thing",
		Method:        "POST",
		RouteTemplate: "/v1/things",
		Params: []EndpointToolParam{
			{Name: "name", In: EndpointToolParamBody},
			{Name: "lines", In: EndpointToolParamBody},
		},
	}
	fake := &fakeGatewayClient{result: "created"}
	handler := endpointToolHandler(desc)

	input := json.RawMessage(`{"name":"widget","lines":[{"qty":2}]}`)
	if _, err := handler(context.Background(), input, &domain.HandlerRunContext{GatewayClient: fake}); err != nil {
		t.Fatal(err)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(fake.got.Body, &body); err != nil {
		t.Fatalf("body is not valid JSON: %v (%q)", err, string(fake.got.Body))
	}
	if string(body["name"]) != `"widget"` {
		t.Errorf("body.name = %s, want \"widget\"", body["name"])
	}
	if string(body["lines"]) != `[{"qty":2}]` {
		t.Errorf("body.lines = %s", body["lines"])
	}
}

func TestEndpointToolHandler_SetsIdempotencyKeyForMutatingCalls(t *testing.T) {
	runCtx := &domain.HandlerRunContext{
		GatewayClient: &fakeGatewayClient{result: "ok"},
		RunID:         "run_1",
		ToolUseID:     "toolu_1",
	}

	cases := []struct {
		method  string
		wantKey string
	}{
		{"POST", "run_1:toolu_1"},
		{"PATCH", "run_1:toolu_1"},
		{"GET", ""},    // reads are not deduped by the gateway
		{"DELETE", ""}, // gateway only dedupes POST/PATCH
	}
	for _, tc := range cases {
		fake := &fakeGatewayClient{result: "ok"}
		runCtx.GatewayClient = fake
		handler := endpointToolHandler(EndpointToolDescriptor{Slug: "x", Method: tc.method, RouteTemplate: "/v1/x"})
		if _, err := handler(context.Background(), nil, runCtx); err != nil {
			t.Fatalf("%s: %v", tc.method, err)
		}
		if fake.got.IdempotencyKey != tc.wantKey {
			t.Errorf("%s: idempotency key = %q, want %q", tc.method, fake.got.IdempotencyKey, tc.wantKey)
		}
	}
}

func TestEndpointToolHandler_NoIdempotencyKeyWithoutToolUseID(t *testing.T) {
	// A missing run or tool-use ID must not produce a partial, non-unique key (e.g. "run_1:").
	fake := &fakeGatewayClient{result: "ok"}
	handler := endpointToolHandler(EndpointToolDescriptor{Slug: "x", Method: "POST", RouteTemplate: "/v1/x"})
	if _, err := handler(context.Background(), nil, &domain.HandlerRunContext{GatewayClient: fake, RunID: "run_1"}); err != nil {
		t.Fatal(err)
	}
	if fake.got.IdempotencyKey != "" {
		t.Errorf("idempotency key = %q, want empty when ToolUseID is unset", fake.got.IdempotencyKey)
	}
}

func TestEndpointToolHandler_NoGatewayClient(t *testing.T) {
	handler := endpointToolHandler(EndpointToolDescriptor{Slug: "x", Method: "GET", RouteTemplate: "/v1/x"})
	if _, err := handler(context.Background(), nil, &domain.HandlerRunContext{}); err == nil {
		t.Error("expected error when gateway client is nil")
	}
}

func TestSearchEndpointToolsScopedToAllowed(t *testing.T) {
	// Only allow one customer tool; a search that would otherwise match many
	// customer endpoints must return only the granted one.
	allowed := map[string]bool{"list_customers": true}
	got := SearchEndpointTools("customer", allowed, 10)
	if len(got) != 1 || got[0].Slug != "list_customers" {
		slugs := make([]string, len(got))
		for i, d := range got {
			slugs[i] = d.Slug
		}
		t.Fatalf("search returned %v, want only [list_customers]", slugs)
	}

	// A tool that exists in the catalog but is NOT allowed must never surface.
	if hits := SearchEndpointTools("create customer", map[string]bool{"list_customers": true}, 10); len(hits) > 0 {
		for _, d := range hits {
			if d.Slug == "create_customer" {
				t.Error("create_customer surfaced despite not being granted")
			}
		}
	}

	// Empty allow-set yields nothing.
	if hits := SearchEndpointTools("customer", map[string]bool{}, 10); len(hits) != 0 {
		t.Errorf("empty grant should return no tools, got %d", len(hits))
	}
}

func TestSearchEndpointToolsRespectsLimit(t *testing.T) {
	allowed := map[string]bool{}
	for _, d := range EndpointTools {
		allowed[d.Slug] = true
	}
	got := SearchEndpointTools("list", allowed, 3)
	if len(got) > 3 {
		t.Errorf("search returned %d, want <= 3", len(got))
	}
}

func TestHandleSearchAPIToolsRecordsReveals(t *testing.T) {
	runCtx := &domain.HandlerRunContext{
		AllowedEndpointToolSlugs: map[string]bool{"list_customers": true, "retrieve_customer": true},
	}
	out, err := HandleSearchAPITools(context.Background(), json.RawMessage(`{"query":"customer"}`), runCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(runCtx.RevealedToolSlugs) == 0 {
		t.Fatal("expected reveals to be recorded")
	}
	for slug := range runCtx.RevealedToolSlugs {
		if !runCtx.AllowedEndpointToolSlugs[slug] {
			t.Errorf("revealed %q which is not granted", slug)
		}
	}
	if !strings.Contains(out, "list_customers") {
		t.Errorf("search result should mention matched tools, got: %s", out)
	}
}

func TestRevealedSlugsForQueries_ReplaysAgainstGrant(t *testing.T) {
	// Replaying an earlier "customer" search surfaces the granted customer tools, restoring
	// the reveal state a follow-up turn would otherwise have to re-search for.
	revealed := RevealedSlugsForQueries([]string{"list customers"}, grantTools("list_customers", "retrieve_customer"))
	if !revealed["list_customers"] {
		t.Errorf("expected list_customers to be re-revealed, got %v", revealed)
	}

	// Permission re-check: the same query replayed against a grant that no longer includes a tool
	// must NOT re-reveal it — an earlier discovery cannot outlive the permission behind it.
	revealed = RevealedSlugsForQueries([]string{"update a customer"}, grantTools("list_customers"))
	if revealed["update_customer"] {
		t.Error("update_customer re-revealed despite the agent no longer being granted it")
	}

	// No grant at all reveals nothing, regardless of recorded queries.
	if got := RevealedSlugsForQueries([]string{"list customers"}, map[string]bool{}); len(got) != 0 {
		t.Errorf("empty grant should reveal nothing, got %v", got)
	}

	// Multiple recorded queries union their matches.
	revealed = RevealedSlugsForQueries([]string{"list customers", "create a product"}, grantTools("list_customers", "create_product"))
	if !revealed["list_customers"] || !revealed["create_product"] {
		t.Errorf("expected union of both queries' matches, got %v", revealed)
	}
}

func TestRegisterEndpointTools(t *testing.T) {
	reg := NewToolHandlerRegistry()
	RegisterEndpointTools(reg)
	for _, d := range EndpointTools {
		if _, ok := reg.Get(d.Slug); !ok {
			t.Errorf("slug %q not registered", d.Slug)
		}
	}
}

// --- search quality ---

// grantAllTools grants every catalog tool, so the search runs against the full real catalog — the set
// an unrestricted agent sees.
func grantAllTools() map[string]bool {
	m := make(map[string]bool, len(EndpointTools))
	for _, d := range EndpointTools {
		m[d.Slug] = true
	}
	return m
}

func grantTools(slugs ...string) map[string]bool {
	m := make(map[string]bool, len(slugs))
	for _, s := range slugs {
		m[s] = true
	}
	return m
}

func resultSlugs(ds []EndpointToolDescriptor) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Slug
	}
	return out
}

func hasSlug(ds []EndpointToolDescriptor, slug string) bool {
	for _, d := range ds {
		if d.Slug == slug {
			return true
		}
	}
	return false
}

// TestSearchEndpointTools_FindsRelevantTool runs natural-language queries an agent would actually issue
// against the real generated catalog and asserts the obviously-correct tool comes back as the top hit.
// This is the core "does it find good tools" guard: if matching/ranking regresses, these break.
func TestSearchEndpointTools_FindsRelevantTool(t *testing.T) {
	all := grantAllTools()
	cases := []struct {
		query string
		want  string
	}{
		{"create a customer", "create_customer"},
		{"list customers", "list_customers"},
		{"update a customer", "update_customer"},
		{"delete a customer", "delete_customer"},
		{"merge customers", "merge_customers"},
		{"create a sales order", "create_sales_order"},
		{"list sales orders", "list_sales_orders"},
		{"create a product", "create_product"},
		{"list products", "list_products"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := SearchEndpointTools(tc.query, all, searchResultLimit)
			if len(got) == 0 {
				t.Fatalf("query %q returned nothing, want %q on top", tc.query, tc.want)
			}
			if got[0].Slug != tc.want {
				t.Errorf("query %q: top hit = %q, want %q (results: %v)", tc.query, got[0].Slug, tc.want, resultSlugs(got))
			}
		})
	}
}

// TestSearchEndpointTools_RanksMoreSpecificHigher: a tool whose slug matches more of the query's terms
// outranks a looser match on a shared term. Both are granted, so only ordering is under test.
func TestSearchEndpointTools_RanksMoreSpecificHigher(t *testing.T) {
	allowed := grantTools("create_customer", "list_customers", "update_customer")
	got := SearchEndpointTools("create customer", allowed, 10)
	if len(got) == 0 || got[0].Slug != "create_customer" {
		t.Fatalf(`"create customer" should rank create_customer first, got %v`, resultSlugs(got))
	}
	// The other customer tools still match on the shared "customer" term — just lower.
	if !hasSlug(got, "list_customers") {
		t.Errorf("list_customers should still match on the shared term, got %v", resultSlugs(got))
	}
}

// TestSearchEndpointTools_WordOrderAndPunctuationIgnored: matching is term-overlap, so reordering words
// or surrounding them with punctuation/casing finds the same tool.
func TestSearchEndpointTools_WordOrderAndPunctuationIgnored(t *testing.T) {
	allowed := grantTools("list_customers")
	for _, q := range []string{"list customers", "customers list", "  CUSTOMERS, list!  ", "list-customers"} {
		got := SearchEndpointTools(q, allowed, 10)
		if len(got) != 1 || got[0].Slug != "list_customers" {
			t.Errorf("query %q: got %v, want [list_customers]", q, resultSlugs(got))
		}
	}
}

// TestSearchEndpointTools_NoMatchReturnsEmpty: a query with no term overlap (and no near-miss) returns
// nothing rather than dumping the catalog — a real query string with zero matches must not reveal
// arbitrary tools.
func TestSearchEndpointTools_NoMatchReturnsEmpty(t *testing.T) {
	if got := SearchEndpointTools("xylophone wizardry photosynthesis", grantAllTools(), 10); len(got) != 0 {
		t.Errorf("nonsense query returned %v, want none", resultSlugs(got))
	}
}

// TestSearchEndpointTools_ToleratesTypos: misspelled queries (missing/transposed letters) still find the
// right tool via fuzzy matching, since an LLM (or human) issuing the search will occasionally typo.
func TestSearchEndpointTools_ToleratesTypos(t *testing.T) {
	all := grantAllTools()
	cases := []struct {
		query string
		want  string
	}{
		{"create custmer", "create_customer"},          // dropped letter
		{"list customrs", "list_customers"},            // dropped letter
		{"updaet customer", "update_customer"},         // transposed letters
		{"delete custommer", "delete_customer"},        // doubled letter
		{"craete a sales order", "create_sales_order"}, // transposed letters in the verb
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := SearchEndpointTools(tc.query, all, searchResultLimit)
			if len(got) == 0 {
				t.Fatalf("typo query %q returned nothing, want %q", tc.query, tc.want)
			}
			if got[0].Slug != tc.want {
				t.Errorf("typo query %q: top hit = %q, want %q (results: %v)", tc.query, got[0].Slug, tc.want, resultSlugs(got))
			}
		})
	}
}

// TestSearchEndpointTools_ShortTokensNotFuzzed: 1–3 character terms are too ambiguous to fuzzy-match (too
// many unrelated words are one edit apart), so they only match exactly.
func TestSearchEndpointTools_ShortTokensNotFuzzed(t *testing.T) {
	// "car" is one edit from "cart"/"card"/"care" etc.; granting only create_customer, a "car" query must
	// not fuzzy-surface it (create_customer has no "car" anywhere as an exact match).
	if got := SearchEndpointTools("car", grantTools("create_customer"), 10); len(got) != 0 {
		t.Errorf("short token fuzzy-matched unexpectedly: %v", resultSlugs(got))
	}
}

// TestSearchEndpointTools_EmptyQueryBrowsesAllowedAlphabetically: with no query terms the tool acts as a
// browse — every granted tool, ordered by slug, capped at the limit (not a relevance ranking).
func TestSearchEndpointTools_EmptyQueryBrowsesAllowedAlphabetically(t *testing.T) {
	allowed := grantTools("list_customers", "create_customer", "create_product", "list_products")
	got := resultSlugs(SearchEndpointTools("", allowed, 10))
	want := []string{"create_customer", "create_product", "list_customers", "list_products"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("empty query = %v, want %v (granted tools, alphabetical)", got, want)
	}

	// The browse is still capped by the limit.
	if capped := SearchEndpointTools("", grantAllTools(), 5); len(capped) != 5 {
		t.Errorf("empty-query browse with limit 5 returned %d, want 5", len(capped))
	}
}

// TestSearchEndpointTools_TieBreakIsAlphabetical: equal-scoring matches come back in slug order so
// results are stable across runs.
func TestSearchEndpointTools_TieBreakIsAlphabetical(t *testing.T) {
	// All three match "customer" as a whole slug segment, so they score equally and sort by slug.
	allowed := grantTools("update_customer", "retrieve_customer", "delete_customer")
	got := resultSlugs(SearchEndpointTools("customer", allowed, 10))
	if !sort.StringsAreSorted(got) {
		t.Errorf("tied results should be slug-sorted, got %v", got)
	}
}

// TestTokenize covers the normalization the search depends on: lowercasing, splitting on any run of
// non-alphanumeric characters, dropping single-character noise, and keeping digits.
func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Create_Customer", []string{"create", "customer"}},
		{"list open sales-orders!", []string{"list", "open", "sales", "orders"}},
		{"a customer x", []string{"customer"}},  // single-char tokens dropped
		{"v2 orders", []string{"v2", "orders"}}, // digits retained
		{"", nil},
		{"   ", nil},
	}
	for _, tc := range cases {
		if got := tokenize(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestCatalogCoversPublicEndpoints sanity-checks that the generated catalog is
// the broad public-endpoint set (flagged AgentTool) and carries embedded schemas
// — there is no DB seed, so the catalog must be self-sufficient.
func TestCatalogCoversPublicEndpoints(t *testing.T) {
	if len(EndpointTools) < 50 {
		t.Errorf("expected the public-endpoint catalog to be large, got %d tools", len(EndpointTools))
	}
	bySlug := map[string]EndpointToolDescriptor{}
	for _, d := range EndpointTools {
		if d.InputSchema == "" {
			t.Errorf("%s: missing embedded InputSchema", d.Slug)
		}
		bySlug[d.Slug] = d
	}

	// A known public read endpoint and a known write endpoint.
	if d, ok := bySlug["list_customers"]; !ok {
		t.Error("expected list_customers in catalog")
	} else if d.Mutating() {
		t.Error("list_customers (GET) must be non-mutating")
	}
	if d, ok := bySlug["create_customer"]; !ok {
		t.Error("expected create_customer in catalog")
	} else if !d.Mutating() {
		t.Error("create_customer (POST) must be mutating")
	}
}
