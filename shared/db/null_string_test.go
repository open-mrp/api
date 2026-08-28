package db

import "testing"

func TestNullStringPtr_NilReturnsInvalid(t *testing.T) {
	t.Parallel()
	got := NullStringPtr(nil)
	if got.Valid {
		t.Fatalf("expected invalid null string for nil input")
	}
}

func TestNullStringPtr_EmptyReturnsInvalid(t *testing.T) {
	t.Parallel()
	empty := ""
	got := NullStringPtr(&empty)
	if got.Valid {
		t.Fatalf("expected invalid null string for empty input")
	}
}

func TestNullStringPtr_NonEmptyReturnsValid(t *testing.T) {
	t.Parallel()
	value := "hello"
	got := NullStringPtr(&value)
	if !got.Valid {
		t.Fatalf("expected valid null string for non-empty input")
	}
	if got.String != "hello" {
		t.Fatalf("expected value %q, got %q", "hello", got.String)
	}
}

// Queries shorter than InnoDB's minimum token size can never match a FULLTEXT index, so they
// have to fall back to LIKE or the search silently returns nothing.
func TestNewFulltextSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		query        *string
		wantFulltext string
		wantLike     string
	}{
		{name: "nil"},
		{name: "empty", query: ptr("")},
		{name: "one character falls back to like", query: ptr("p"), wantLike: "%p%"},
		{name: "below the token size falls back to like", query: ptr("pr"), wantLike: "%pr%"},
		{name: "at the token size uses fulltext", query: ptr("kil"), wantFulltext: "kil*"},
		{name: "above the token size uses fulltext", query: ptr("kilo"), wantFulltext: "kilo*"},
		{name: "like fallback escapes metacharacters", query: ptr("a%"), wantLike: `%a\%%`},
		{name: "fulltext strips boolean operators", query: ptr("+kilo -gram"), wantFulltext: "kilo gram*"},
		{name: "all operators leave nothing to search", query: ptr("+++")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewFulltextSearch(tt.query)

			if got.Fulltext.Valid != (tt.wantFulltext != "") || got.Fulltext.String != tt.wantFulltext {
				t.Fatalf("Fulltext = %+v, want %q", got.Fulltext, tt.wantFulltext)
			}
			if got.Like.Valid != (tt.wantLike != "") || got.Like.String != tt.wantLike {
				t.Fatalf("Like = %+v, want %q", got.Like, tt.wantLike)
			}
			// sqlc emits a second bind for the same AGAINST parameter; a mismatch binds NULL
			// to half the clause.
			if got.Fulltext2 != got.Fulltext {
				t.Fatalf("Fulltext2 = %+v, want it to equal Fulltext %+v", got.Fulltext2, got.Fulltext)
			}
		})
	}
}

func TestNewFulltextSearch_UsesEitherModeNotBoth(t *testing.T) {
	t.Parallel()
	for _, q := range []string{"p", "pr", "kil", "kilogram"} {
		got := NewFulltextSearch(&q)
		if got.Fulltext.Valid && got.Like.Valid {
			t.Fatalf("query %q bound both modes: %+v", q, got)
		}
	}
}

func ptr(s string) *string { return &s }
