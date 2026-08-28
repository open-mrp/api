package db

import (
	"testing"
)

func TestNewCatalogSearch_Empty(t *testing.T) {
	t.Parallel()
	cs := NewCatalogSearch(nil)
	if cs.Contains.Valid || cs.Exact.Valid || cs.Prefix.Valid {
		t.Fatalf("expected all invalid, got Contains=%v Exact=%v Prefix=%v", cs.Contains.Valid, cs.Exact.Valid, cs.Prefix.Valid)
	}
	q := ""
	cs = NewCatalogSearch(&q)
	if cs.Contains.Valid || cs.Exact.Valid || cs.Prefix.Valid {
		t.Fatal("expected empty query to yield invalid params")
	}
}

func TestCatalogSearchRank(t *testing.T) {
	t.Parallel()
	q := "622"
	cs := NewCatalogSearch(&q)
	if g, w := CatalogSearchRank("622", cs), int32(0); g != w {
		t.Fatalf("exact sku: got %d want %d", g, w)
	}
	if g, w := CatalogSearchRank("Greige 622", cs), int32(1); g != w {
		t.Fatalf("token sku: got %d want %d", g, w)
	}
	if g, w := CatalogSearchRank("6220", cs), int32(2); g != w {
		t.Fatalf("prefix sku: got %d want %d", g, w)
	}
	if g, w := CatalogSearchRank("x622x", cs), int32(3); g != w {
		t.Fatalf("substring only: got %d want %d", g, w)
	}
	empty := NewCatalogSearch(nil)
	if g, w := CatalogSearchRank("anything", empty), int32(0); g != w {
		t.Fatalf("no search: got %d want %d", g, w)
	}
}

func TestCatalogSearchRank_LoosePartial621(t *testing.T) {
	t.Parallel()
	q := "621"
	cs := NewCatalogSearch(&q)
	if g, w := CatalogSearchRank("56214", cs), int32(3); g != w {
		t.Fatalf("loose partial: got %d want %d", g, w)
	}
}

func TestCatalogSearchRank_EscapesLikeMeta(t *testing.T) {
	t.Parallel()
	q := `50%_off`
	cs := NewCatalogSearch(&q)
	if !cs.Prefix.Valid {
		t.Fatal("expected prefix pattern")
	}
	if g, w := CatalogSearchRank(`50%_off`, cs), int32(0); g != w {
		t.Fatalf("exact with meta: got %d want %d", g, w)
	}
}

// The rank is the cursor's match_tier, so it has to agree with the CASE expression in the
// catalog list SQL row for row: a tier the SQL would not have assigned makes page 2 skip
// every row the cursor's tier no longer selects.
func TestCatalogSearchRank_Tiers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		query string
		sku   string
		want  int32
	}{
		{name: "exact", query: "622", sku: "622", want: 0},
		{name: "token in the middle", query: "622", sku: "Greige 622 Red", want: 1},
		{name: "token at the start", query: "622", sku: "622 Red", want: 1},
		{name: "token at the end", query: "622", sku: "Greige 622", want: 1},
		{name: "prefix", query: "622", sku: "6220", want: 2},
		{name: "prefix of a longer token run", query: "622", sku: "622A Greige", want: 2},
		{name: "substring only", query: "622", sku: "x622x", want: 3},
		{name: "no match", query: "622", sku: "Greige", want: 3},
		{name: "exact with like metacharacters", query: `50%_off`, sku: `50%_off`, want: 0},
		{name: "substring with like metacharacters", query: `50%_off`, sku: `x50%_offx`, want: 3},
		{name: "metacharacter query against an unrelated sku", query: `50%_off`, sku: `Greige 622`, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs := NewCatalogSearch(&tt.query)
			if got := CatalogSearchRank(tt.sku, cs); got != tt.want {
				t.Fatalf("CatalogSearchRank(%q, %q) = %d, want %d", tt.sku, tt.query, got, tt.want)
			}
		})
	}
}

// The LIKE patterns are bound straight into the query, so an unescaped metacharacter would
// turn a user's search term into a wildcard.
func TestNewCatalogSearch_PatternsEscapeLikeMeta(t *testing.T) {
	t.Parallel()
	q := `50%_off`
	cs := NewCatalogSearch(&q)

	if got, want := cs.Contains.String, `%50\%\_off%`; got != want {
		t.Fatalf("Contains = %q, want %q", got, want)
	}
	if got, want := cs.Prefix.String, `50\%\_off%`; got != want {
		t.Fatalf("Prefix = %q, want %q", got, want)
	}
	if got, want := cs.Exact.String, q; got != want {
		t.Fatalf("Exact = %q, want %q", got, want)
	}
}

func TestNullTierInt64Param(t *testing.T) {
	t.Parallel()
	if got := NullTierInt64Param(nil); got.Valid {
		t.Fatalf("nil tier should bind NULL, got %+v", got)
	}
	tier := 3
	got := NullTierInt64Param(&tier)
	if !got.Valid || got.Int64 != 3 {
		t.Fatalf("NullTierInt64Param(3) = %+v, want {3 true}", got)
	}
	zero := 0
	if got := NullTierInt64Param(&zero); !got.Valid || got.Int64 != 0 {
		t.Fatalf("tier 0 must bind as a value, not NULL, got %+v", got)
	}
}
