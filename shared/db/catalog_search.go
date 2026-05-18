package db

import (
	"database/sql"
	"strings"
)

// CatalogSearch binds parameters for catalog list queries that filter by item SKU
// and description and rank exact / prefix SKU matches ahead of substring matches.
type CatalogSearch struct {
	// Contains is a LIKE pattern "%escaped_query%" for substring matches.
	Contains sql.NullString
	// Exact is the raw query string for SKU equality (tier 0) and MATCH expressions.
	Exact sql.NullString
	// Prefix is a LIKE pattern "escaped_query%" for prefix SKU matches (tier 1).
	Prefix sql.NullString
}

// NewCatalogSearch builds bind args for catalog search. If q is nil or empty, all
// fields are invalid (no search).
func NewCatalogSearch(q *string) CatalogSearch {
	if q == nil || *q == "" {
		return CatalogSearch{}
	}
	escaped := EscapeLike(*q)
	return CatalogSearch{
		Contains: sql.NullString{String: "%" + escaped + "%", Valid: true},
		Exact:    sql.NullString{String: *q, Valid: true},
		Prefix:   sql.NullString{String: escaped + "%", Valid: true},
	}
}

// CatalogSearchRank returns the SKU tier used for search ordering (0 exact, 1 prefix, 2 other).
// It mirrors the CASE expression in catalog list SQL.
func CatalogSearchRank(sku string, cs CatalogSearch) int32 {
	if !cs.Contains.Valid {
		return 0
	}
	if cs.Exact.Valid && sku == cs.Exact.String {
		return 0
	}
	if cs.Prefix.Valid {
		pat := cs.Prefix.String
		if len(pat) > 0 && strings.HasSuffix(pat, "%") {
			prefix := strings.TrimSuffix(pat, "%")
			if strings.HasPrefix(sku, prefix) {
				return 1
			}
		}
	}
	return 2
}

// NullTierInt64Param binds cursor_match_tier for sqlc (nullable integer tier 0–2).
func NullTierInt64Param(t *int) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*t), Valid: true}
}
