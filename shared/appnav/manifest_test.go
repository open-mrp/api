package appnav

import (
	"encoding/json"
	"os"
	"testing"
)

// rawManifest models the generated file in full, including the two diagnostic lists the package's own
// struct drops. A regeneration that leaves detail routes unmapped is otherwise invisible on this side.
type rawManifest struct {
	Pages                []Page            `json:"pages"`
	Records              []RecordRoute     `json:"records"`
	SkippedDetailRoutes  []json.RawMessage `json:"skipped_detail_routes"`
	UnmappedDetailRoutes []json.RawMessage `json:"unmapped_detail_routes"`
}

func readManifest(t *testing.T) rawManifest {
	t.Helper()
	b, err := os.ReadFile("app_pages.json")
	if err != nil {
		t.Fatalf("read app_pages.json: %v", err)
	}
	var m rawManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse app_pages.json: %v", err)
	}
	return m
}

// TestManifestCoverageFloor sits just under the catalog's real size, so a truncated or partly-failed
// regeneration fails here instead of quietly shrinking what agents can link to.
func TestManifestCoverageFloor(t *testing.T) {
	t.Parallel()
	const (
		minPages   = 80
		minRecords = 40
	)

	if len(Pages()) < minPages {
		t.Errorf("catalog has %d pages, want at least %d — regenerate with `bun run generate:app-pages` in dashboard/apps/frontend, or lower the floor deliberately", len(Pages()), minPages)
	}
	if got := len(readManifest(t).Records); got < minRecords {
		t.Errorf("catalog has %d record routes, want at least %d — a drop this large means the generator lost detail routes", got, minRecords)
	}
}

// TestManifestDiagnosticsAreClean reads the generator's own verdict: an unmapped detail route is a page
// the dashboard renders that no object type can be linked to.
func TestManifestDiagnosticsAreClean(t *testing.T) {
	t.Parallel()
	m := readManifest(t)

	if len(m.UnmappedDetailRoutes) > 0 {
		t.Errorf("generator reported %d unmapped detail routes: %s", len(m.UnmappedDetailRoutes), string(mustMarshal(t, m.UnmappedDetailRoutes)))
	}
	for i, s := range m.SkippedDetailRoutes {
		if len(s) == 0 || string(s) == "null" || string(s) == `""` {
			t.Errorf("skipped_detail_routes[%d] is empty", i)
		}
	}
}

// TestManifestLoadsEveryPage catches the drift the Go structs can hide: a renamed or dropped JSON tag
// zeroes a field, and a duplicate key silently collapses two pages into one lookup.
func TestManifestLoadsEveryPage(t *testing.T) {
	t.Parallel()
	m := readManifest(t)

	if len(Pages()) != len(m.Pages) {
		t.Fatalf("Pages() returned %d pages, file holds %d", len(Pages()), len(m.Pages))
	}

	seen := make(map[string]bool, len(m.Pages))
	for _, want := range m.Pages {
		if seen[want.Key] {
			t.Errorf("duplicate page key %q — one of the two is unreachable through PageByKey", want.Key)
		}
		seen[want.Key] = true

		got, ok := PageByKey(want.Key)
		if !ok {
			t.Errorf("page %q is in the file but not reachable through PageByKey", want.Key)
			continue
		}
		if got.Title != want.Title || got.Path != want.Path || got.Section != want.Section || got.RecordType != want.RecordType {
			t.Errorf("page %q loaded as %+v, file holds %+v", want.Key, got, want)
		}
	}
}

// TestRecordRoutesCoverEveryDetailPage replaces pinning a handful of known types: every detail page in
// the catalog must be reachable by object type, and every route must point at a page that exists.
func TestRecordRoutesCoverEveryDetailPage(t *testing.T) {
	t.Parallel()
	m := readManifest(t)

	for _, p := range m.Pages {
		if p.RecordType == "" {
			continue
		}
		route, ok := RecordRouteFor(p.RecordType)
		if !ok {
			t.Errorf("page %q shows record type %q, which has no record route", p.Key, p.RecordType)
			continue
		}
		if route.PageKey != p.Key {
			t.Errorf("record route %q points at page %q, want %q", p.RecordType, route.PageKey, p.Key)
		}
	}

	seen := make(map[string]bool, len(m.Records))
	for _, r := range m.Records {
		if seen[r.Type] {
			t.Errorf("duplicate record route for %q", r.Type)
		}
		seen[r.Type] = true

		if r.Type == "" || r.PageKey == "" {
			t.Errorf("record route missing type or page key: %+v", r)
			continue
		}
		if r.DetailBase == "" || r.DetailBase[0] != '/' {
			t.Errorf("record route %q has detail base %q, want an absolute route", r.Type, r.DetailBase)
		}
		if len(r.Shells) == 0 {
			t.Errorf("record route %q claims no shell", r.Type)
		}

		page, ok := PageByKey(r.PageKey)
		if !ok {
			t.Errorf("record route %q points at page %q, which is not in the catalog", r.Type, r.PageKey)
			continue
		}
		if page.RecordType != r.Type {
			t.Errorf("record route %q points at page %q, whose record type is %q", r.Type, r.PageKey, page.RecordType)
		}
		for _, shell := range r.Shells {
			if !containsWord(page.Shells, shell) {
				t.Errorf("record route %q claims shell %q that page %q does not mount (%v)", r.Type, shell, r.PageKey, page.Shells)
			}
		}
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
