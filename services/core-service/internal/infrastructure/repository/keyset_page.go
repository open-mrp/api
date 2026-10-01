package repository

import (
	"slices"
	"strings"
)

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
//   - A long list (a search's matches) keeps its key: ranging just its rows and sorting them is the
//     best plan when it is sparse, walking another key when it is dense.
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
	if len(indexes) == 0 {
		indexes = append(indexes, p.createdIndex)
	}
	indexes = append(indexes, long...)

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
