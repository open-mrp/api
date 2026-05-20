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
