package repository

import (
	"context"
	gosql "database/sql"
	"strings"

	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/pagination"
)

// lookupListQuery is one keyset page of a lookup list (carriers, service levels, payment and shipping
// terms, units, properties), ordered by created_at and id. Only the predicates the caller supplied are
// emitted: an `(? IS NULL OR ...)` guard is not sargable.
type lookupListQuery struct {
	// selectFrom is the SELECT list and FROM clause, joins included.
	selectFrom string
	// alias is the listed table's alias, whose created_at and id key the page.
	alias string
	where []string
	args  []any
}

func (q *lookupListQuery) and(predicate string, args ...any) {
	q.where = append(q.where, predicate)
	q.args = append(q.args, args...)
}

// sql is the page after cur (the first page when nil): forward pages older (DESC), backward pages
// newer (ASC).
func (q lookupListQuery) sql(cur *pagination.StringCursor, limit int32) (string, []any) {
	a := q.alias + "."
	where, args := q.where, q.args
	order := " DESC"
	if cur != nil {
		cmp := "<"
		if cur.Direction == pagination.DirectionBackward {
			cmp, order = ">", " ASC"
		}
		where = append(where[:len(where):len(where)], "("+a+"created_at "+cmp+" ? OR ("+a+"created_at = ? AND "+a+"id "+cmp+" ?))")
		args = append(args[:len(args):len(args)], cur.OccurredAt, cur.OccurredAt, cur.ID)
	}
	query := q.selectFrom + " WHERE " + strings.Join(where, " AND ") +
		" ORDER BY " + a + "created_at" + order + ", " + a + "id" + order + " LIMIT ?"
	return query, append(args[:len(args):len(args)], limit)
}

// decodeLookupCursor decodes a list cursor; nil when the request has none.
func decodeLookupCursor(cursor *string) (*pagination.StringCursor, *apierror.APIError) {
	if cursor == nil {
		return nil, nil
	}
	cur, err := pagination.DecodeStringCursor(*cursor)
	if err != nil {
		return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
	}
	return &cur, nil
}

// queryLookupRows runs a lookup page and scans each row with scan.
func queryLookupRows[T any](ctx context.Context, db interface {
	QueryContext(ctx context.Context, query string, args ...any) (*gosql.Rows, error)
}, query string, args []any, scan func(*gosql.Rows) (T, error)) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []T
	for rows.Next() {
		row, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
