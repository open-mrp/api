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

// ngramTokenSize is the MySQL server's ngram_token_size (default 2). An ngram FULLTEXT index tokenizes
// text into overlapping tokens of this length, so a query shorter than it has no token to match and
// must fall back to LIKE.
const ngramTokenSize = 2

// NewNgramSearch formats a search term for a MySQL ngram FULLTEXT index (substring search). A term of
// at least ngramTokenSize characters becomes a boolean-mode phrase ("term"), which forces the ngram
// tokens to appear consecutively — matching the term as a substring anywhere in the column. Shorter
// terms have no ngram token and fall back to LIKE.
//
// Inside a phrase every boolean operator (+, -, *, …) is literal, so — unlike NewFulltextSearch — the
// term must NOT be run through SanitizeFulltextBoolean: stripping the hyphen from a pick number like
// "PICK-002" changes its ngram tokens and matches nothing. The only character that must go is the
// double-quote, which would close the phrase early. A term that is all punctuation stays a (possibly
// empty) phrase, which matches nothing — never dropping the filter so the list returns every row.
func NewNgramSearch(s *string) FulltextSearch {
	if s == nil || *s == "" {
		return FulltextSearch{}
	}
	if len([]rune(*s)) < ngramTokenSize {
		return FulltextSearch{
			Like: sql.NullString{String: "%" + EscapeLike(*s) + "%", Valid: true},
		}
	}
	phrase := `"` + strings.ReplaceAll(*s, `"`, "") + `"`
	ft := sql.NullString{String: phrase, Valid: true}
	return FulltextSearch{Fulltext: ft, Fulltext2: ft}
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
