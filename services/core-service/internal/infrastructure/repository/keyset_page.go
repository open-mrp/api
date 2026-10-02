package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"strings"
)

// keysetCountedRatio sets how many matches, in pages, make a long list common.
const keysetCountedRatio = 40

// keysetMaxArms is the most values of one filter a list reads as separate arms. Each arm reads at most
// a page, so this bounds a request to a few pages; a longer list (a search's matches) is one IN.
const keysetMaxArms = 8

// keysetFilter is one equality filter of a keyset list and its key, which yields one value's rows in
// list order. A filter matches column IN values, or column IS NULL when isNull.
type keysetFilter struct {
	column, index string
	values        []string
	isNull        bool
}

func (f keysetFilter) active() bool { return f.isNull || len(f.values) > 0 }

// singular reports whether the filter pins one value, so its key yields its rows in list order.
func (f keysetFilter) singular() bool { return f.isNull || len(f.values) == 1 }

// keysetPage chooses a page of a keyset list from its table alone: the ids of the page's rows, in list
// order, for the caller to join its columns to.
type keysetPage struct {
	// table and alias are the listed table, and sortColumn the list's time column (ordered with id).
	table, alias, sortColumn string
	// createdIndex yields the scope's rows in list order with no filter.
	createdIndex string
	// filters run from most to least selective.
	filters []keysetFilter
	// where and args are every other predicate: the scope, a date range, the cursor, residual filters.
	where []string
	args  []any
	desc  bool
	limit int32
	// longIndex is set by settle: the one long list's key to range, or createdIndex to walk instead.
	longIndex string
}

// longFilters is the long lists (a search's matches) the page would range with no single-valued or
// armed filter's key to walk instead.
func (p keysetPage) longFilters() []keysetFilter {
	var long []keysetFilter
	for i, f := range p.filters {
		n := len(f.values)
		if f.singular() || (n > 1 && n <= keysetMaxArms && p.firstActive() == i) {
			return nil
		}
		if n > keysetMaxArms {
			long = append(long, f)
		}
	}
	return long
}

// firstActive is the index of the most selective active filter, or -1.
func (p keysetPage) firstActive() int {
	for i, f := range p.filters {
		if f.active() {
			return i
		}
	}
	return -1
}

// settle decides, for a page only long lists filter, between ranging the rarest list's key and sorting
// its matches, or walking the unfiltered key in list order with the lists residual. The planner cannot
// make that call: offered both keys, it takes the walk as a full scan from the scope's far end rather
// than a range from the cursor. Each count is capped, reading at most that many index entries.
func (p *keysetPage) settle(ctx context.Context, q interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *gosql.Row
}) error {
	long := p.longFilters()
	if len(long) == 0 {
		return nil
	}
	capped := int64(keysetCountedRatio) * int64(p.limit)
	p.longIndex = p.createdIndex
	fewest := capped
	for _, f := range long {
		args := append(slices.Clone(p.args), stringArgs(f.values)...)
		args = append(args, capped)
		var n int64
		err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM (SELECT 1 FROM "+p.table+" "+p.alias+
			" FORCE INDEX ("+f.index+") WHERE "+strings.Join(p.where, " AND ")+
			" AND "+f.column+" IN ("+placeholders(len(f.values))+") LIMIT ?) matches", args...).Scan(&n)
		if err != nil {
			return err
		}
		if n < fewest {
			p.longIndex, fewest = f.index, n
		}
	}
	return nil
}

// sql is the page as a derived table's body, and its args.
//
// Left free, the planner reaches for single-column keys or merges them, or ranges a multi-valued
// filter's key and sorts every match. So the page is held to keys that yield it in list order:
//   - A single-valued filter's key yields its rows in list order, so it is offered. The unfiltered
//     key is offered only when no filter's is: it yields a superset in the same order, and offered
//     beside a filter's key, the planner trades that key's range for a full scan of it.
//   - A filter with a few values is yielded in order by no key. When it is the most selective filter,
//     it is read as one arm per value, each stopping at a page on that value's key, and the arms
//     merged. Otherwise it is residual to a more selective filter and its key withheld: an arm whose
//     value the other filter excludes would read a whole range to find nothing.
//   - A long list (a search's matches) beside a filter whose key is offered keeps its key too. Alone,
//     its key is ranged and its matches sorted when it is sparse, and the unfiltered key walked when it
//     is dense, as settle decided; unsettled, its key is ranged.
func (p keysetPage) sql() (string, []any) {
	filters := make([]keysetFilter, len(p.filters))
	for i, f := range p.filters {
		f.values = slices.Compact(slices.Sorted(slices.Values(f.values)))
		filters[i] = f
	}
	split := -1
	for i, f := range filters {
		if n := len(f.values); n > 1 && n <= keysetMaxArms {
			split = i
		}
		if f.active() {
			break
		}
	}
	var indexes, long []string
	for i, f := range filters {
		switch {
		case f.singular() || i == split:
			indexes = append(indexes, f.index)
		case len(f.values) > keysetMaxArms:
			long = append(long, f.index)
		}
	}
	switch {
	case len(indexes) > 0:
		indexes = append(indexes, long...)
	case p.longIndex != "":
		indexes = []string{p.longIndex}
	case len(long) > 0:
		indexes = long
	default:
		indexes = []string{p.createdIndex}
	}

	order := " DESC"
	if !p.desc {
		order = " ASC"
	}
	a := p.alias + "."
	var b strings.Builder
	var args []any
	arm := func(value string) {
		b.WriteString("SELECT " + a + "id, " + a + p.sortColumn + " FROM " + p.table + " " + p.alias)
		b.WriteString(" FORCE INDEX (" + strings.Join(indexes, ", ") + ") WHERE ")
		where := slices.Clone(p.where)
		args = append(args, p.args...)
		for i, f := range filters {
			switch {
			case f.isNull:
				where = append(where, f.column+" IS NULL")
			case i == split:
				where = append(where, f.column+" = ?")
				args = append(args, value)
			case len(f.values) > 0:
				where = append(where, f.column+" IN ("+placeholders(len(f.values))+")")
				args = append(args, stringArgs(f.values)...)
			}
		}
		b.WriteString(strings.Join(where, " AND "))
		b.WriteString(" ORDER BY " + a + p.sortColumn + order + ", " + a + "id" + order + " LIMIT ?")
		args = append(args, p.limit)
	}

	if split < 0 {
		arm("")
		return b.String(), args
	}
	for i, v := range filters[split].values {
		if i > 0 {
			b.WriteString(" UNION ALL ")
		}
		b.WriteString("(")
		arm(v)
		b.WriteString(")")
	}
	b.WriteString(" ORDER BY " + p.sortColumn + order + ", id" + order + " LIMIT ?")
	args = append(args, p.limit)
	return b.String(), args
}
