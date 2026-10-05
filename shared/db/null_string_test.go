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

func TestNewNgramSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		query        *string
		wantFulltext string
		wantLike     string
	}{
		{name: "nil"},
		{name: "empty", query: ptr("")},
		{name: "below the ngram token size falls back to like", query: ptr("p"), wantLike: "%p%"},
		{name: "one char like fallback escapes metacharacters", query: ptr("%"), wantLike: `%\%%`},
		{name: "at the ngram token size uses a phrase", query: ptr("po"), wantFulltext: `"po"`},
		{name: "longer term becomes a phrase for substring matching", query: ptr("23839"), wantFulltext: `"23839"`},
		// The hyphen must survive: a phrase keeps operators literal, and stripping it would change the
		// ngram tokens so a real pick number never matches.
		{name: "hyphenated pick number keeps the hyphen", query: ptr("PICK-002"), wantFulltext: `"PICK-002"`},
		{name: "operators stay literal inside the phrase", query: ptr("+2-38*"), wantFulltext: `"+2-38*"`},
		// Only the phrase delimiter is removed; an all-quote term becomes an empty phrase, which matches
		// nothing rather than dropping the filter and returning every row.
		{name: "embedded quotes are stripped", query: ptr(`a"b"c`), wantFulltext: `"abc"`},
		{name: "all-quote term becomes an empty phrase", query: ptr(`""`), wantFulltext: `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewNgramSearch(tt.query)

			if got.Fulltext.Valid != (tt.wantFulltext != "") || got.Fulltext.String != tt.wantFulltext {
				t.Fatalf("Fulltext = %+v, want %q", got.Fulltext, tt.wantFulltext)
			}
			if got.Like.Valid != (tt.wantLike != "") || got.Like.String != tt.wantLike {
				t.Fatalf("Like = %+v, want %q", got.Like, tt.wantLike)
			}
			if got.Fulltext.Valid && got.Like.Valid {
				t.Fatalf("query bound both modes: %+v", got)
			}
			// sqlc binds the same AGAINST parameter twice (once per UNION branch); a mismatch
			// would send NULL to one of the two MATCH clauses.
			if got.Fulltext2 != got.Fulltext {
				t.Fatalf("Fulltext2 = %+v, want it to equal Fulltext %+v", got.Fulltext2, got.Fulltext)
			}
		})
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
