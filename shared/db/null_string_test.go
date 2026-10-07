package db

import (
	"reflect"
	"strings"
	"testing"
)

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

func TestNewNgramSubstring(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		term       string
		wantTokens string
		wantLike   string
	}{
		{name: "one character has no token", term: "p", wantLike: "%p%"},
		{name: "every token is required, lowercased", term: "XYZ3614", wantTokens: "+xy +yz +z3 +36 +61 +14", wantLike: "%XYZ3614%"},
		{name: "a repeated token is required once", term: "1212", wantTokens: "+12 +21", wantLike: "%1212%"},
		{name: "tokens with a or i are left to like", term: "Milky", wantTokens: "+lk +ky", wantLike: "%Milky%"},
		{name: "two-letter stopwords are left to like", term: "bed", wantTokens: "+ed", wantLike: "%bed%"},
		{name: "a term of only unindexable tokens is like alone", term: "navi", wantLike: "%navi%"},
		// The hyphen is an operator in boolean mode; the LIKE still holds it, so the match stays exact.
		{name: "punctuation and whitespace stay out of the tokens", term: "K-24 9", wantTokens: "+24", wantLike: "%K-24 9%"},
		{name: "like escapes its metacharacters", term: "5%_", wantLike: `%5\%\_%`},
		{name: "non-ascii tokens are left to like", term: "ñé", wantLike: "%ñé%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewNgramSubstring(tt.term)
			if got.Tokens != tt.wantTokens || got.Like != tt.wantLike {
				t.Fatalf("NewNgramSubstring(%q) = %+v, want tokens %q like %q", tt.term, got, tt.wantTokens, tt.wantLike)
			}
			if placeholders := strings.Count(got.Where("c"), "?"); placeholders != len(got.Args()) {
				t.Fatalf("Where has %d placeholders but Args has %d values", placeholders, len(got.Args()))
			}
		})
	}
}

func TestNgramSubstring_Where(t *testing.T) {
	t.Parallel()

	indexed := NewNgramSubstring("123")
	if got, want := indexed.Where("p.number"), "MATCH(p.number) AGAINST(? IN BOOLEAN MODE) AND p.number LIKE ?"; got != want {
		t.Errorf("Where = %q, want %q", got, want)
	}
	if got, want := indexed.Args(), []any{"+12 +23", "%123%"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Args = %v, want %v", got, want)
	}

	unindexed := NewNgramSubstring("aia")
	if got, want := unindexed.Where("p.number"), "p.number LIKE ?"; got != want {
		t.Errorf("Where = %q, want %q", got, want)
	}
	if got, want := unindexed.Args(), []any{"%aia%"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Args = %v, want %v", got, want)
	}
}

func ptr(s string) *string { return &s }

func TestAllWordsPrefixQuery(t *testing.T) {
	tests := map[string]string{
		"":             "",
		"  ":           "",
		"-*()":         "",
		"TX-0012 acme": "+TX* +0012* +acme*",
		"12345":        "+12345*",
		"o'brien+co":   "+o* +brien* +co*",
	}
	for in, want := range tests {
		in := in
		if got := AllWordsPrefixQuery(&in); got != want {
			t.Errorf("AllWordsPrefixQuery(%q) = %q, want %q", in, got, want)
		}
	}
	if got := AllWordsPrefixQuery(nil); got != "" {
		t.Errorf("AllWordsPrefixQuery(nil) = %q", got)
	}
}

func TestAllWordsSearch(t *testing.T) {
	for in, want := range map[string][2]string{
		"":                  {"", ""},
		"  - ":              {"", ""},
		"knit large":        {"+knit* +large*", ""},
		"QA Init P1":        {"+Init*", `^(?=.*\bQA)(?=.*\bP1)`},
		"TX-0012":           {"+0012*", `^(?=.*\bTX)`},
		"a b":               {"", `^(?=.*\ba)(?=.*\bb)`},
		"sew (pair) 2% off": {"+sew* +pair* +off*", `^(?=.*\b2)`},
	} {
		fulltext, short := AllWordsSearch(&in)
		if fulltext.String != want[0] || fulltext.Valid != (want[0] != "") {
			t.Errorf("AllWordsSearch(%q) fulltext = %+v, want %q", in, fulltext, want[0])
		}
		if short.String != want[1] || short.Valid != (want[1] != "") {
			t.Errorf("AllWordsSearch(%q) short words = %+v, want %q", in, short, want[1])
		}
	}
	if f, s := AllWordsSearch(nil); f.Valid || s.Valid {
		t.Errorf("AllWordsSearch(nil) = %+v, %+v, want both NULL", f, s)
	}
}
