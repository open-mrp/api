package appnav

import (
	"strings"
	"testing"
)

// TestManifestLoads guards the generated manifest against being committed empty or truncated, which
// would silently strip every agent link rather than fail anything.
func TestManifestLoads(t *testing.T) {
	if len(Pages()) < 50 {
		t.Fatalf("expected the dashboard's page catalog, got %d pages — regenerate with `bun run generate:app-pages` in dashboard/apps/frontend", len(Pages()))
	}
	for _, p := range Pages() {
		if p.Key == "" || p.Title == "" || p.Path == "" {
			t.Errorf("page missing key/title/path: %+v", p)
		}
		if strings.HasPrefix(p.Key, "/") {
			t.Errorf("page key %q should be path-relative (no leading slash)", p.Key)
		}
		if len(p.Shells) == 0 {
			t.Errorf("page %q claims no shell", p.Key)
		}
	}
}

// TestRecordRoutesCoverKnownTypes pins the record types the previous hand-written registry covered, so
// a regeneration that drops one is caught. These are the types agents link most.
func TestRecordRoutesCoverKnownTypes(t *testing.T) {
	for _, objectType := range []string{"sales_order", "purchase_order", "invoice", "customer", "product", "agent_run", "account_price"} {
		route, ok := RecordRouteFor(objectType)
		if !ok {
			t.Errorf("object type %q has no record route", objectType)
			continue
		}
		if route.DetailBase == "" {
			t.Errorf("object type %q has an empty detail base", objectType)
		}
	}
}

func TestPageByKey(t *testing.T) {
	p, ok := PageByKey("customer-prices")
	if !ok {
		t.Fatal("customer-prices page not found")
	}
	if p.Title != "Customer Prices" {
		t.Errorf("title = %q, want Customer Prices", p.Title)
	}
	if p.RecordType != "account_price" {
		t.Errorf("record type = %q, want account_price", p.RecordType)
	}
	if _, ok := PageByKey("no-such-page"); ok {
		t.Error("expected miss for an unknown key")
	}
}

// TestSearchRanksTitleOverSection covers the failure that made page links unreliable: a query naming a
// page must return that page, not the section it sits under.
func TestSearchRanksTitleOverSection(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{"customer prices", "customer-prices"},
		{"volume discounts", "volume-discounts"},
		{"sales orders", "sales-orders"},
		{"where do I set up discount codes", "order-discounts"},
		// Typo tolerance: the agent paraphrases, so near-misses must still land.
		{"custmer prices", "customer-prices"},
	}
	for _, c := range cases {
		results := Search(c.query, 5)
		if len(results) == 0 {
			t.Errorf("%q matched nothing", c.query)
			continue
		}
		if results[0].Key != c.want {
			t.Errorf("%q → %q, want %q (top 3: %v)", c.query, results[0].Key, c.want, keys(results))
		}
	}
}

func TestSearchEmptyQueryListsCatalog(t *testing.T) {
	if got := Search("", 5); len(got) != 5 {
		t.Errorf("limited empty query returned %d pages, want 5", len(got))
	}
	if got := Search("   ", 0); len(got) != len(Pages()) {
		t.Errorf("unlimited empty query returned %d pages, want all %d", len(got), len(Pages()))
	}
}

// TestSearchUnmatchedQueryReturnsNothing is the contract the model-facing tool rests on: a query the
// catalog cannot answer must come back empty, not as pages the agent will link anyway.
func TestSearchUnmatchedQueryReturnsNothing(t *testing.T) {
	t.Parallel()
	for _, q := range []string{"zzqqxx wombatron", "quixotic flibbertigibbet", "asdfgh qwerty"} {
		if got := Search(q, 5); len(got) != 0 {
			t.Errorf("%q returned %d pages (%v), want none", q, len(got), keys(got))
		}
	}
}

func TestSearchLimitCapsMatches(t *testing.T) {
	t.Parallel()
	all := Search("orders", 100)
	if len(all) < 3 {
		t.Fatalf("%q matched %d pages, expected the order pages", "orders", len(all))
	}
	capped := Search("orders", 2)
	if len(capped) != 2 {
		t.Fatalf("limit 2 returned %d pages (%v)", len(capped), keys(capped))
	}
	// The cap must keep the best matches, not an arbitrary slice of them.
	for i, p := range capped {
		if p.Key != all[i].Key {
			t.Errorf("capped[%d] = %q, want %q — the limit changed the ranking", i, p.Key, all[i].Key)
		}
	}
}

func TestPageByKeyNormalizesKey(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"/customer-prices", "customer-prices/", "Customer-Prices", "/Customer-Prices/"} {
		p, ok := PageByKey(key)
		if !ok {
			t.Errorf("PageByKey(%q) missed", key)
			continue
		}
		if p.Key != "customer-prices" {
			t.Errorf("PageByKey(%q) = %q, want customer-prices", key, p.Key)
		}
	}
}

func keys(pages []Page) []string {
	out := make([]string, 0, len(pages))
	for i, p := range pages {
		if i == 3 {
			break
		}
		out = append(out, p.Key)
	}
	return out
}
