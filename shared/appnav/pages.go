// Package appnav exposes the dashboard's page catalog to the API — which pages the app has, and
// which API object type each record detail page shows.
//
// Agents reference app records and pages by writing `augno:` markdown links that the frontend
// resolves to real routes, so nothing here builds a URL; the catalog exists so an agent can be told
// what is linkable at all. The manifest is generated from the dashboard's own navigation
// (`dashboard/apps/frontend/scripts/generate-app-pages.ts`) and committed, which keeps a new page
// linkable without an edit on this side.
package appnav

import (
	"embed"
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"github.com/augno/api/shared/fuzzy"
)

//go:embed app_pages.json
var manifestFS embed.FS

// Page is a linkable dashboard page.
type Page struct {
	// Key identifies the page in an `augno:page/<key>` link (e.g. `customer-prices`, `agents/runs`).
	Key string `json:"key"`
	// Title is the page's name as the navigation shows it.
	Title string `json:"title"`
	// Path is the operator-shell route. Kept for diagnostics — links carry the key, not the path.
	Path string `json:"path"`
	// Section and Subsection locate the page in the sidebar, which is how users describe where things are.
	Section    string `json:"section"`
	Subsection string `json:"subsection,omitempty"`
	// Shells lists which app shells mount the page: `user` (operator) and/or `customer` (portal).
	Shells []string `json:"shells"`
	// RecordType is the API object type this page's detail view shows, when it has one.
	RecordType string `json:"record_type,omitempty"`
}

// RecordRoute records that an object type has a detail page, and which page it is.
type RecordRoute struct {
	// Type is the object type as it appears in a resource's `object` field.
	Type       string   `json:"type"`
	PageKey    string   `json:"page_key"`
	DetailBase string   `json:"detail_base"`
	Shells     []string `json:"shells"`
}

type manifest struct {
	Pages   []Page        `json:"pages"`
	Records []RecordRoute `json:"records"`
}

var loaded = func() manifest {
	var m manifest
	b, err := manifestFS.ReadFile("app_pages.json")
	if err != nil {
		// Embedded at build time: unreadable means the file is missing from the build, not a runtime condition.
		panic("appnav: reading embedded app_pages.json: " + err.Error())
	}
	if err := json.Unmarshal(b, &m); err != nil {
		panic("appnav: parsing embedded app_pages.json: " + err.Error())
	}
	return m
}()

var (
	pagesByKey = func() map[string]Page {
		m := make(map[string]Page, len(loaded.Pages))
		for _, p := range loaded.Pages {
			m[p.Key] = p
		}
		return m
	}()
	routesByType = func() map[string]RecordRoute {
		m := make(map[string]RecordRoute, len(loaded.Records))
		for _, r := range loaded.Records {
			m[r.Type] = r
		}
		return m
	}()
)

// Pages returns every linkable page.
func Pages() []Page { return loaded.Pages }

// PageByKey looks up a page by its link key.
func PageByKey(key string) (Page, bool) {
	p, ok := pagesByKey[strings.Trim(strings.ToLower(key), "/")]
	return p, ok
}

// RecordRouteFor reports whether an object type has a detail page, and which one.
func RecordRouteFor(objectType string) (RecordRoute, bool) {
	r, ok := routesByType[objectType]
	return r, ok
}

// Search ranks pages against a plain-language query, best first, capped at limit.
//
// Matching is weighted by where a term hits: the page's own title is what someone naming a page is
// naming, so it outranks the section it happens to sit under — otherwise a query like "pricing"
// returns every page in the Pricing subsection ahead of the pricing pages themselves. An empty
// query returns the whole catalog in sidebar order, which is how an agent browses rather than looks up.
func Search(query string, limit int) []Page {
	terms := tokenize(query)
	if len(terms) == 0 {
		if limit > 0 && limit < len(loaded.Pages) {
			return loaded.Pages[:limit]
		}
		return loaded.Pages
	}

	type scored struct {
		page  Page
		score int
	}
	var matches []scored
	for _, p := range loaded.Pages {
		if s := scorePage(p, terms); s > 0 {
			matches = append(matches, scored{page: p, score: s})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].page.Key < matches[j].page.Key
	})

	out := make([]Page, 0, min(limit, len(matches)))
	for _, m := range matches {
		if len(out) >= limit {
			break
		}
		out = append(out, m.page)
	}
	return out
}

func scorePage(p Page, terms []string) int {
	titleWords := strings.Fields(strings.ToLower(p.Title))
	title := strings.ToLower(p.Title)
	keyWords := strings.Split(strings.ToLower(p.Key), "/")
	key := strings.ToLower(p.Key)
	recordWords := strings.Split(p.RecordType, "_")
	context := strings.ToLower(p.Section + " " + p.Subsection)

	total := 0
	for _, t := range terms {
		best := 0
		switch {
		case containsWord(titleWords, t), containsWord(keyWords, t):
			best = 10
		case strings.Contains(title, t), strings.Contains(key, t):
			best = 8
		case containsWord(recordWords, t):
			best = 6
		case strings.Contains(context, t):
			best = 2
		}
		// Typo tolerance, always below the exact equivalent so a real match wins when one exists.
		if best < 8 {
			if fuzzy.AnyTypo(t, titleWords) || fuzzy.AnyTypo(t, keyWords) {
				best = max(best, 7)
			}
		}
		total += best
	}
	return total
}

func containsWord(words []string, term string) bool {
	return slices.Contains(words, term)
}

func tokenize(s string) []string {
	var terms []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if len(f) > 1 { // drop single-char noise
			terms = append(terms, f)
		}
	}
	return terms
}
