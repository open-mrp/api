package repository

import (
	gosql "database/sql"
	"strings"
	"unicode/utf8"

	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

// The pick list is a deferred join: buildPickListQuery pages pick ids using only the pick table, and
// the page is hydrated by id afterwards. Every filter is a pick column (buyer_account_id is
// denormalized from the order), so an (account_id, [filter], <sort>, id) index can serve the ORDER
// BY and stop at LIMIT, and the order/customer/carrier joins run for one page instead of for every
// candidate row.
const (
	pickShipByIndex       = "pick_account_ship_by_idx"
	pickOpenShipByIndex   = "pick_account_finished_ship_by_idx"
	pickCreatedIndex      = "pick_account_created_idx"
	pickOpenCreatedIndex  = "pick_account_finished_created_idx"
	pickBuyerShipByIndex  = "pick_account_buyer_ship_by_idx"
	pickBuyerCreatedIndex = "pick_account_buyer_created_idx"
	// The open variants lead with finished_at after the buyer, for the same reason as the account ones.
	pickBuyerOpenShipByIndex  = "pick_account_buyer_finished_ship_by_idx"
	pickBuyerOpenCreatedIndex = "pick_account_buyer_finished_created_idx"
	pickAccountNumberIndex    = "pick_account_number_idx"
)

// Shorter terms are too common as substrings to page quickly ("22" is in ~9k of the largest
// account's picks), so they match pick numbers by prefix instead.
const pickSubstringSearchMinRunes = 3

type pickSearch struct {
	// NumberPrefix is a LIKE pattern anchored at the start of the pick number.
	NumberPrefix string
	// Phrase is an ngram boolean-mode phrase, matched as a substring of the pick number, the order's
	// PO number, and the customer's name and number.
	Phrase string
}

func newPickSearch(q *string) pickSearch {
	if q == nil || *q == "" {
		return pickSearch{}
	}
	if utf8.RuneCountInString(*q) < pickSubstringSearchMinRunes {
		return pickSearch{NumberPrefix: db.EscapeLike(*q) + "%"}
	}
	return pickSearch{Phrase: db.NewNgramSearch(q).Fulltext.String}
}

type pickListQuery struct {
	AccountID    string
	SortByShipBy bool
	Search       pickSearch
	Status       *string
	// BuyerIDs, when non-nil, limits the list to these customers: the customer filter intersected
	// with the customer-group filter's members.
	BuyerIDs []string
	// DriveFromBuyers reads BuyerIDs from the buyer index in one scan instead of walking the sort
	// index: always for one customer (the index is in list order), and for a set too large to merge
	// when it has few picks (see pickBuyerScanCap). Sets of 2..pickBuyerMergeMax are merged instead.
	DriveFromBuyers bool
	ProductLineIDs  []string
	StartDate       gosql.NullTime
	EndDate         gosql.NullTime
	Direction       pagination.Direction
	CursorAt        gosql.NullTime
	CursorID        gosql.NullString
	Limit           int32
}

// openOnly reports the open filter; any status other than "closed" is open.
func (q pickListQuery) openOnly() bool {
	return q.Status != nil && *q.Status != "closed"
}

// indexHint names the indexes the scan may drive from, so MySQL cannot pick one that filesorts the
// account or walks it for a rare filter value. status=open pins finished_at because open picks are a
// tiny slice of a mostly-closed table. Whether a customer set drives from the buyer index is decided
// by counting its picks (DriveFromBuyers), because MySQL cannot estimate it. A number prefix lets
// MySQL choose from its range estimate. A phrase search drives from its match set and takes no hint.
func (q pickListQuery) indexHint() []string {
	if q.Search.Phrase != "" {
		return nil
	}

	sortIndex := pickCreatedIndex
	switch {
	case q.SortByShipBy && q.openOnly():
		sortIndex = pickOpenShipByIndex
	case q.SortByShipBy:
		sortIndex = pickShipByIndex
	case q.openOnly():
		sortIndex = pickOpenCreatedIndex
	}

	switch {
	case q.DriveFromBuyers:
		return []string{q.buyerIndex()}
	case q.Search.NumberPrefix != "":
		return []string{sortIndex, pickAccountNumberIndex}
	default:
		return []string{sortIndex}
	}
}

// buyerIndex serves the sort for one customer's picks, pinning status=open when it is set.
func (q pickListQuery) buyerIndex() string {
	switch {
	case q.SortByShipBy && q.openOnly():
		return pickBuyerOpenShipByIndex
	case q.SortByShipBy:
		return pickBuyerShipByIndex
	case q.openOnly():
		return pickBuyerOpenCreatedIndex
	default:
		return pickBuyerCreatedIndex
	}
}

// buildPickListQuery returns the query for one page of pick ids (Limit rows, in list order) and its
// bind args. Only the predicates the caller set are emitted, so a bare ORDER BY can land on an index.
//
// Ship-by pages forward ascending (soonest first), created-at forward descending (newest first);
// backward is the reverse, undone by BuildPageString.
func buildPickListQuery(q pickListQuery) (string, []any) {
	if q.mergesBuyers() {
		return buildPickListMerge(q)
	}

	args := make([]any, 0, 16+len(q.BuyerIDs)+len(q.ProductLineIDs))
	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN p.id FROM ")
	if q.Search.Phrase != "" {
		b.WriteString("(" + pickPhraseMatches + ") matched JOIN pick p ON p.id = matched.id")
		for range 4 {
			args = append(args, q.AccountID, q.Search.Phrase)
		}
	} else {
		b.WriteString("pick p FORCE INDEX (" + strings.Join(q.indexHint(), ", ") + ")")
	}
	b.WriteString(" WHERE p.account_id = ?")
	args = append(args, q.AccountID)
	if len(q.BuyerIDs) > 0 {
		b.WriteString(" AND p.buyer_account_id IN (" + iclPlaceholders(len(q.BuyerIDs)) + ")")
		for _, id := range q.BuyerIDs {
			args = append(args, id)
		}
	}
	args = q.writeFilters(&b, args)

	col, order := q.sortColumn(), q.sortOrder()
	b.WriteString(" ORDER BY " + col + " " + order + ", p.id " + order + " LIMIT ?")
	args = append(args, q.Limit)
	return b.String(), args
}

// pickBuyerMergeMax is the most customers listed by merging one sort-ordered read per customer.
// MySQL reads an IN list of customers from the buyer index as ranges it then has to sort in full,
// and walking the sort index instead reads every other customer's picks too, which for customers
// that stopped ordering years ago was most of the account (119ms for three of them). Each merged
// read stops at the page size, so the whole list costs at most customers x page size rows.
const pickBuyerMergeMax = 50

func (q pickListQuery) mergesBuyers() bool {
	return q.Search.Phrase == "" && len(q.BuyerIDs) >= 2 && len(q.BuyerIDs) <= pickBuyerMergeMax
}

// buildPickListMerge pages each customer's picks from the buyer index in list order and merges the
// pages. Every read carries the same filters and cursor, so each returns that customer's next page.
func buildPickListMerge(q pickListQuery) (string, []any) {
	col, order := q.sortColumn(), q.sortOrder()
	args := make([]any, 0, len(q.BuyerIDs)*(8+len(q.ProductLineIDs))+1)

	var b strings.Builder
	b.WriteString("SELECT id FROM (")
	for i, buyerID := range q.BuyerIDs {
		if i > 0 {
			b.WriteString(" UNION ALL ")
		}
		b.WriteString("(SELECT p.id, " + col + " AS sort_at FROM pick p FORCE INDEX (" + q.buyerIndex() + ")" +
			" WHERE p.account_id = ? AND p.buyer_account_id = ?")
		args = append(args, q.AccountID, buyerID)
		args = q.writeFilters(&b, args)
		b.WriteString(" ORDER BY " + col + " " + order + ", p.id " + order + " LIMIT ?)")
		args = append(args, q.Limit)
	}
	b.WriteString(") merged ORDER BY sort_at " + order + ", id " + order + " LIMIT ?")
	args = append(args, q.Limit)
	return b.String(), args
}

// writeFilters appends every predicate other than the account and customer scope: search prefix,
// status, product lines, the date window and the keyset cursor.
func (q pickListQuery) writeFilters(b *strings.Builder, args []any) []any {
	if q.Search.NumberPrefix != "" {
		b.WriteString(" AND p.number LIKE ?")
		args = append(args, q.Search.NumberPrefix)
	}
	if q.Status != nil {
		if q.openOnly() {
			b.WriteString(" AND p.finished_at IS NULL")
		} else {
			b.WriteString(" AND p.finished_at IS NOT NULL")
		}
	}
	if len(q.ProductLineIDs) > 0 {
		b.WriteString(" AND EXISTS (SELECT 1 FROM pick_line pl2" +
			" JOIN sales_order_line sol2 ON sol2.id = pl2.sales_order_line_id" +
			" JOIN product prod ON prod.id = sol2.product_id" +
			" WHERE pl2.pick_id = p.id AND prod.product_line_id IN (" + iclPlaceholders(len(q.ProductLineIDs)) + "))")
		for _, id := range q.ProductLineIDs {
			args = append(args, id)
		}
	}
	if q.StartDate.Valid {
		b.WriteString(" AND p.created_at >= ?")
		args = append(args, q.StartDate.Time)
	}
	if q.EndDate.Valid {
		b.WriteString(" AND p.created_at <= ?")
		args = append(args, q.EndDate.Time)
	}
	// Keyset over (sort column, id). For ship-by, CAST keeps the param a DATE while the column stays
	// bare, so the composite is still usable.
	if q.CursorAt.Valid {
		col, cmp, param := q.sortColumn(), q.cursorComparison(), "?"
		if q.SortByShipBy {
			param = "CAST(? AS DATE)"
		}
		b.WriteString(" AND (" + col + " " + cmp + " " + param + " OR (" + col + " = " + param + " AND p.id " + cmp + " ?))")
		args = append(args, q.CursorAt.Time, q.CursorAt.Time, q.CursorID.String)
	}
	return args
}

func (q pickListQuery) sortColumn() string {
	if q.SortByShipBy {
		return "p.ship_by_sort_date"
	}
	return "p.created_at"
}

func (q pickListQuery) ascending() bool {
	return q.SortByShipBy == (q.Direction == pagination.DirectionForward)
}

func (q pickListQuery) sortOrder() string {
	if q.ascending() {
		return "ASC"
	}
	return "DESC"
}

func (q pickListQuery) cursorComparison() string {
	if q.ascending() {
		return ">"
	}
	return "<"
}

// pickPhraseMatches is the set of the account's pick ids whose number, PO number, customer name or
// customer number contains the phrase. Each arm is its own MATCH in a UNION because an OR of MATCH
// across joined tables cannot use any of the FULLTEXT indexes. Placeholders per arm: account id,
// then phrase.
const pickPhraseMatches = `SELECT pk.id FROM pick pk` +
	` WHERE pk.account_id = ? AND MATCH(pk.number) AGAINST(? IN BOOLEAN MODE)` +
	` UNION SELECT pk.id FROM sales_order pso JOIN pick pk ON pk.sales_order_id = pso.id` +
	` WHERE pk.account_id = ? AND MATCH(pso.customer_po_number) AGAINST(? IN BOOLEAN MODE)` +
	` UNION SELECT pk.id FROM account nba` +
	` JOIN account_relation nar ON nar.owner_account_id = ? AND nar.counterparty_account_id = nba.id` +
	` JOIN pick pk ON pk.account_id = nar.owner_account_id AND pk.buyer_account_id = nba.id` +
	` WHERE MATCH(nba.name) AGAINST(? IN BOOLEAN MODE)` +
	` UNION SELECT pk.id FROM account_relation rar` +
	` JOIN pick pk ON pk.account_id = rar.owner_account_id AND pk.buyer_account_id = rar.counterparty_account_id` +
	` WHERE rar.owner_account_id = ? AND MATCH(rar.external_number) AGAINST(? IN BOOLEAN MODE)`

// pickBuyerScanCap is the most picks a customer set too large to merge may have and still be read
// from the buyer index. That path reads every one of the set's picks and sorts them, so it is only
// cheap for few picks; with more, the set matches often enough that walking the sort index fills a
// page quickly. A rare customer group walked the other way read all 123k of the largest account's
// picks (1.5s).
const pickBuyerScanCap = 2000

// buildPickBuyerCountQuery counts the set's picks, stopping at pickBuyerScanCap, from the covering
// buyer index.
func buildPickBuyerCountQuery(accountID string, buyerIDs []string) (string, []any) {
	args := make([]any, 0, len(buyerIDs)+2)
	args = append(args, accountID)
	for _, id := range buyerIDs {
		args = append(args, id)
	}
	args = append(args, pickBuyerScanCap)
	return "SELECT COUNT(*) FROM (SELECT 1 FROM pick FORCE INDEX (" + pickBuyerCreatedIndex + ")" +
		" WHERE account_id = ? AND buyer_account_id IN (" + iclPlaceholders(len(buyerIDs)) + ") LIMIT ?) capped", args
}
