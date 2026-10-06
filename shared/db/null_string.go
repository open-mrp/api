package db

import (
	"database/sql"
	"strings"
)

func NullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func NullStringPtr(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullStringLikePtr returns a NullString with the value wrapped in % wildcards for LIKE queries.
func NullStringLikePtr(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: "%" + EscapeLike(*s) + "%", Valid: true}
}

// NullStringFulltextPtr returns a NullString formatted for MySQL FULLTEXT BOOLEAN MODE search. It appends a wildcard (*) so the term matches any word that starts with the given value (e.g. "kilo" → "kilo*").
func NullStringFulltextPtr(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	sanitized := SanitizeFulltextBoolean(*s)
	if sanitized == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: sanitized + "*", Valid: true}
}

// EscapeLike escapes MySQL LIKE metacharacters in user-provided search terms.
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// AllWordsSearch splits search input so that every word must begin a word of the text, in any order.
// Words of at least innoDBMinTokenSize characters go into a BOOLEAN MODE query for the FULLTEXT index
// (see AllWordsPrefixQuery). The index holds no shorter word, so those go into a pattern for
// REGEXP_LIKE(col, pattern, 'i') with a word-start lookahead each: "QA init P1" becomes "+init*" and
// `^(?=.*\bQA)(?=.*\bP1)`. Emit each part only when it is valid, the pattern after the MATCH so it
// reads only the index's matches.
func AllWordsSearch(query *string) (fulltext, shortWords sql.NullString) {
	if query == nil {
		return
	}
	var long []string
	var short strings.Builder
	for _, w := range searchWords(*query) {
		if len(w) < innoDBMinTokenSize {
			short.WriteString(`(?=.*\b` + w + `)`)
		} else {
			long = append(long, "+"+w+"*")
		}
	}
	if len(long) > 0 {
		fulltext = sql.NullString{String: strings.Join(long, " "), Valid: true}
	}
	if short.Len() > 0 {
		shortWords = sql.NullString{String: "^" + short.String(), Valid: true}
	}
	return
}

// searchWords splits search input on anything that is not an ASCII letter or digit, which also leaves
// nothing a FULLTEXT or regular expression would read as an operator.
func searchWords(query string) []string {
	return strings.FieldsFunc(query, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})
}

// AllWordsPrefixQuery turns search input into a BOOLEAN MODE query that requires every word to
// begin a word of the indexed text, as the dashboard's PrismaUtils.sanitizeQuery did: the input is
// split on anything that is not an ASCII letter or digit, so "TX-0012 acme" becomes "+TX* +0012* +acme*".
// It returns "" when the input has no words.
func AllWordsPrefixQuery(query *string) string {
	if query == nil {
		return ""
	}
	words := searchWords(*query)
	for i, w := range words {
		words[i] = "+" + w + "*"
	}
	return strings.Join(words, " ")
}

// SanitizeFulltextBoolean strips MySQL BOOLEAN MODE operators from user input.
func SanitizeFulltextBoolean(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '+', '-', '>', '<', '(', ')', '~', '*', '"', '@':
			return -1
		}
		return r
	}, s)
}

// innoDBMinTokenSize is the default minimum word length for InnoDB FULLTEXT indexes. Queries shorter than this must fall back to LIKE.
const innoDBMinTokenSize = 3

// FulltextSearch holds parameters for a SQL clause that supports both FULLTEXT (MATCH/AGAINST) and LIKE search. Queries with at least innoDBMinTokenSize characters use FULLTEXT; shorter queries fall back to LIKE so that short abbreviations (e.g. "pr") are still matched.
//
// The SQL clause should be structured as:
//
//	AND (
//	    (sqlc.narg('search_query') IS NULL AND sqlc.narg('like_query') IS NULL)
//	    OR MATCH(...) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
//	    OR col LIKE sqlc.narg('like_query')
//	)
//
// Due to a sqlc bug, MATCH/AGAINST generates a duplicate parameter (SearchQuery_2). This helper populates both so callers don't need to know about the dedup issue.
//
// Usage:
//
//	ft := db.NewFulltextSearch(params.Query)
//	sqlc.ListFooParams{ SearchQuery: ft.Fulltext, SearchQuery_2: ft.Fulltext2, LikeQuery: ft.Like, ... }
type FulltextSearch struct {
	// Fulltext is the value for the FULLTEXT IS NULL guard and AGAINST clause.
	Fulltext sql.NullString
	// Fulltext2 is a duplicate of Fulltext required by a sqlc dedup bug.
	Fulltext2 sql.NullString
	// Like is the value for the LIKE fallback (set for short queries).
	Like sql.NullString
}

func NewFulltextSearch(s *string) FulltextSearch {
	if s == nil || *s == "" {
		return FulltextSearch{}
	}
	if len(*s) < innoDBMinTokenSize {
		sanitized := EscapeLike(*s)
		if sanitized == "" {
			return FulltextSearch{}
		}
		return FulltextSearch{
			Like: sql.NullString{String: "%" + sanitized + "%", Valid: true},
		}
	}
	ft := NullStringFulltextPtr(s)
	return FulltextSearch{Fulltext: ft, Fulltext2: ft}
}

// ngramTokenSize is the MySQL server's ngram_token_size (default 2).
const ngramTokenSize = 2

// ngramStopwords are the two-letter words on InnoDB's default stopword list that hold neither "a" nor "i".
var ngramStopwords = map[string]bool{"be": true, "by": true, "de": true, "en": true, "of": true, "on": true, "or": true, "to": true}

// NgramSubstring finds a term anywhere in a column that has an ngram FULLTEXT index: the index narrows
// to rows holding every token of the term, and LIKE confirms the term itself.
//
// It never asks the index for a phrase: InnoDB verifies a phrase against token positions it stored, and
// on production those disagree with some rows, which a phrase search then silently misses.
type NgramSubstring struct {
	// Tokens is a boolean-mode query requiring each token of the term the index can hold, or "" when it
	// can hold none of them.
	Tokens string
	// Like matches the term anywhere in the column.
	Like string
}

// NewNgramSubstring builds the search for a non-empty term.
func NewNgramSubstring(term string) NgramSubstring {
	runes := []rune(strings.ToLower(term))
	seen := map[string]bool{}
	var tokens []string
	for i := 0; i+ngramTokenSize <= len(runes); i++ {
		token := string(runes[i : i+ngramTokenSize])
		if !seen[token] && ngramIndexable(token) {
			seen[token] = true
			tokens = append(tokens, "+"+token)
		}
	}
	return NgramSubstring{Tokens: strings.Join(tokens, " "), Like: "%" + EscapeLike(term) + "%"}
}

// ngramIndexable reports whether the index can hold token, so requiring it does not exclude every row.
// The server's stopwords keep out any token containing "a" or "i" (themselves stopwords) and the
// two-letter stopwords; a token with anything but a letter or digit is whitespace the parser skips or
// an operator in boolean mode.
func ngramIndexable(token string) bool {
	for _, r := range token {
		if (r < '0' || r > '9') && (r < 'b' || r > 'z' || r == 'i') {
			return false
		}
	}
	return !ngramStopwords[token]
}

// Indexed reports whether the index can narrow the search; without it, only the LIKE remains.
func (s NgramSubstring) Indexed() bool {
	return s.Tokens != ""
}

// Where is the predicate that col contains the term; Args binds its placeholders.
func (s NgramSubstring) Where(col string) string {
	if !s.Indexed() {
		return col + " LIKE ?"
	}
	return "MATCH(" + col + ") AGAINST(? IN BOOLEAN MODE) AND " + col + " LIKE ?"
}

// Args are the values for Where's placeholders, in order.
func (s NgramSubstring) Args() []any {
	if !s.Indexed() {
		return []any{s.Like}
	}
	return []any{s.Tokens, s.Like}
}

func StringFromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// StringFromInterface extracts a string from an interface{} value. MySQL CASE expressions are typed as interface{} by sqlc and may arrive as []byte or string depending on the driver. Returns "" for nil.
func StringFromInterface(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return ""
	}
}
