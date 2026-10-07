package repository

import (
	gosql "database/sql"
	"slices"
	"strings"
	"time"
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
	// Phrase is matched as a substring of the pick number, the order's PO number, and the customer's
	// name and number.
	Phrase db.NgramSubstring
}

func (s pickSearch) hasPhrase() bool {
	return s.Phrase.Like != ""
}

func newPickSearch(q *string) pickSearch {
	if q == nil || *q == "" {
		return pickSearch{}
	}
	if utf8.RuneCountInString(*q) < pickSubstringSearchMinRunes {
		return pickSearch{NumberPrefix: db.EscapeLike(*q) + "%"}
	}
	return pickSearch{Phrase: db.NewNgramSubstring(*q)}
}

type pickListQuery struct {
	AccountID    string
	SortByShipBy bool
	Search       pickSearch
	// PhraseIDs, when non-nil, are the picks a phrase search matches, read up front because there are
	// few (see pickPhraseScanCap). A phrase with too many is collected again in the query.
	PhraseIDs []string
	// PhraseMatches, when non-nil, are PhraseIDs with the columns phrasePage filters and sorts on.
	PhraseMatches []pickPhraseMatch
	// DriveFromPhrase reads the page from PhraseIDs by primary key and sorts them.
	DriveFromPhrase bool
	Status          *string
	// BuyerIDs, when non-nil, limits the list to these customers: the customer filter intersected
	// with the customer-group filter's members.
	BuyerIDs []string
	// DriveFromBuyers reads BuyerIDs from the buyer index in one scan instead of walking the sort
	// index: always for one customer (the index is in list order), and for a set too large to merge
	// when it has few picks (see pickBuyerScanCap). Sets of 2..pickBuyerMergeMax are merged instead.
	DriveFromBuyers bool
	// DriveFromNumberPrefix reads the number prefix's range instead, for a prefix narrower than a
	// counted customer set: MySQL estimates an IN list of more than a couple of hundred customers from
	// index statistics, which overcount the customer set, and picks the prefix's range however wide.
	DriveFromNumberPrefix bool
	ProductLineIDs        []string
	// DriveFromProductLines reads the product lines' picks from their lines and sorts them, for product
	// lines on few picks (see pickProductLineScanCap); walking a sort index for them reads the account.
	DriveFromProductLines bool
	StartDate             gosql.NullTime
	EndDate               gosql.NullTime
	Direction             pagination.Direction
	CursorAt              gosql.NullTime
	CursorID              gosql.NullString
	Limit                 int32
}

// openOnly reports the open filter; any status other than "closed" is open.
func (q pickListQuery) openOnly() bool {
	return q.Status != nil && *q.Status != "closed"
}

// indexHint names the indexes the scan may drive from, so MySQL cannot pick one that filesorts the
// account or walks it for a rare filter value. status=open pins finished_at because open picks are a
// tiny slice of a mostly-closed table. Whether a customer set drives from the buyer index is decided
// by counting its picks (DriveFromBuyers), because MySQL cannot estimate it. A number prefix, and a
// creation window under the ship-by sort, are ranges whose keys are offered too, and MySQL chooses
// from its estimate of how many rows they hold.
func (q pickListQuery) indexHint() []string {
	if q.DriveFromNumberPrefix {
		return []string{pickAccountNumberIndex}
	}
	if q.DriveFromBuyers {
		// A counted set was already found narrower than the prefix, so the number key is not offered.
		return q.withRangeIndexes(q.buyerIndex(), q.buyerCreatedIndex(), len(q.BuyerIDs) == 1)
	}
	return q.withRangeIndexes(q.sortIndex(), q.createdIndex(), true)
}

func (q pickListQuery) withRangeIndexes(sortIndex, createdIndex string, offerNumber bool) []string {
	hint := []string{sortIndex}
	if q.SortByShipBy && (q.StartDate.Valid || q.EndDate.Valid) {
		hint = append(hint, createdIndex)
	}
	if offerNumber && q.Search.NumberPrefix != "" {
		hint = append(hint, pickAccountNumberIndex)
	}
	return hint
}

// sortIndex serves the sort for the account's picks, pinning status=open when it is set.
func (q pickListQuery) sortIndex() string {
	if q.SortByShipBy {
		if q.openOnly() {
			return pickOpenShipByIndex
		}
		return pickShipByIndex
	}
	return q.createdIndex()
}

func (q pickListQuery) createdIndex() string {
	if q.openOnly() {
		return pickOpenCreatedIndex
	}
	return pickCreatedIndex
}

// buyerIndex serves the sort for one customer's picks, pinning status=open when it is set.
func (q pickListQuery) buyerIndex() string {
	if q.SortByShipBy {
		if q.openOnly() {
			return pickBuyerOpenShipByIndex
		}
		return pickBuyerShipByIndex
	}
	return q.buyerCreatedIndex()
}

func (q pickListQuery) buyerCreatedIndex() string {
	if q.openOnly() {
		return pickBuyerOpenCreatedIndex
	}
	return pickBuyerCreatedIndex
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

	args := make([]any, 0, 16+len(q.PhraseIDs)+len(q.BuyerIDs)+len(q.ProductLineIDs))
	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN p.id FROM ")
	switch {
	case q.DriveFromPhrase:
		b.WriteString("pick p FORCE INDEX (PRIMARY)")
	case q.DriveFromProductLines:
		b.WriteString("(" + pickProductLineMatches(len(q.ProductLineIDs)) + ") matched JOIN pick p ON p.id = matched.id")
		args = append(args, stringArgs(q.ProductLineIDs)...)
	default:
		b.WriteString("pick p FORCE INDEX (" + strings.Join(q.indexHint(), ", ") + ")")
	}
	b.WriteString(" WHERE p.account_id = ?")
	args = append(args, q.AccountID)
	switch {
	case q.PhraseIDs != nil:
		b.WriteString(" AND p.id IN (" + iclPlaceholders(len(q.PhraseIDs)) + ")")
		args = append(args, stringArgs(q.PhraseIDs)...)
	case q.Search.hasPhrase():
		// Too many to read up front, so the phrase's matches are collected once and probed per row.
		b.WriteString(" AND p.id IN (SELECT id FROM (" + pickPhraseMatches(q.Search.Phrase) + ") phrased)")
		args = q.appendPhraseArgs(args)
	}
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

// mergesBuyers reports a set of a few customers with nothing narrower to read from. A phrase search
// is not merged: each merged read would collect the phrase's matches again.
func (q pickListQuery) mergesBuyers() bool {
	return !q.Search.hasPhrase() && !q.DriveFromProductLines && !q.DriveFromBuyers && !q.DriveFromNumberPrefix &&
		len(q.BuyerIDs) >= 2 && len(q.BuyerIDs) <= pickBuyerMergeMax
}

func (q pickListQuery) appendPhraseArgs(args []any) []any {
	for range pickPhraseArmCount {
		args = append(args, q.AccountID)
		args = append(args, q.Search.Phrase.Args()...)
	}
	return args
}

// buildPickListMerge pages each customer's picks from the buyer index in list order and merges the
// pages. Every read carries the same filters and cursor, so each returns that customer's next page.
func buildPickListMerge(q pickListQuery) (string, []any) {
	col, order := q.sortColumn(), q.sortOrder()
	hint := strings.Join(q.withRangeIndexes(q.buyerIndex(), q.buyerCreatedIndex(), true), ", ")
	args := make([]any, 0, len(q.BuyerIDs)*(8+len(q.ProductLineIDs))+1)

	var b strings.Builder
	b.WriteString("SELECT id FROM (")
	for i, buyerID := range q.BuyerIDs {
		if i > 0 {
			b.WriteString(" UNION ALL ")
		}
		b.WriteString("(SELECT p.id, " + col + " AS sort_at FROM pick p FORCE INDEX (" + hint + ")" +
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
	if len(q.ProductLineIDs) > 0 && !q.DriveFromProductLines {
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

const pickPhraseArmCount = 4

// pickPhraseArms each select the account's pick ids whose number, PO number, customer name or
// customer number contains the phrase. Each arm is its own MATCH because an OR of MATCH across joined
// tables cannot use any of the FULLTEXT indexes. Placeholders per arm: account id, then the phrase's.
//
// With pagesInMemory, every arm also selects pickPhraseMatch's columns. The customer arms otherwise
// read only a buyer key, so they force the one that covers those columns: MySQL picks a key that
// does not when the select list grows, and reads every pick of the customer.
func pickPhraseArms(phrase db.NgramSubstring, q pickListQuery, pagesInMemory bool) []string {
	numberIndex := ""
	if !phrase.Indexed() {
		// The account's number key covers a LIKE, where MySQL would read every pick's row.
		numberIndex = " FORCE INDEX (" + pickAccountNumberIndex + ")"
	}
	columns, buyerIndex := "pk.id", ""
	if pagesInMemory {
		columns = "pk.id, pk.buyer_account_id, pk.finished_at, " + strings.Replace(q.sortColumn(), "p.", "pk.", 1)
		buyerIndex = " FORCE INDEX (" + q.phraseBuyerIndex() + ")"
	}
	return []string{
		`SELECT ` + columns + ` FROM pick pk` + numberIndex +
			` WHERE pk.account_id = ? AND ` + phrase.Where("pk.number"),
		`SELECT ` + columns + ` FROM sales_order pso JOIN pick pk ON pk.sales_order_id = pso.id` +
			` WHERE pk.account_id = ? AND ` + phrase.Where("pso.customer_po_number"),
		`SELECT ` + columns + ` FROM account nba` +
			` JOIN account_relation nar ON nar.owner_account_id = ? AND nar.counterparty_account_id = nba.id` +
			` JOIN pick pk` + buyerIndex + ` ON pk.account_id = nar.owner_account_id AND pk.buyer_account_id = nba.id` +
			` WHERE ` + phrase.Where("nba.name"),
		`SELECT ` + columns + ` FROM account_relation rar` +
			` JOIN pick pk` + buyerIndex + ` ON pk.account_id = rar.owner_account_id AND pk.buyer_account_id = rar.counterparty_account_id` +
			` WHERE rar.owner_account_id = ? AND ` + phrase.Where("rar.external_number"),
	}
}

// pickPhraseMatches is the set of picks the phrase matches.
func pickPhraseMatches(phrase db.NgramSubstring) string {
	return strings.Join(pickPhraseArms(phrase, pickListQuery{}, false), " UNION ")
}

// phraseBuyerIndex is the buyer key that holds every column a phrase page reads.
func (q pickListQuery) phraseBuyerIndex() string {
	if q.SortByShipBy {
		return pickBuyerOpenShipByIndex
	}
	return pickBuyerOpenCreatedIndex
}

// phrasePagesInMemory reports whether a phrase's few matches can be paged from the columns read with
// them. A product-line filter is a child-table EXISTS, and a creation window under the ship-by sort
// reads a column no buyer key holds alongside the sort, so those page in SQL.
func (q pickListQuery) phrasePagesInMemory() bool {
	return len(q.ProductLineIDs) == 0 && !(q.SortByShipBy && (q.StartDate.Valid || q.EndDate.Valid))
}

// pickPhraseMatch is a pick a phrase matched, with the columns its page is filtered and sorted on.
type pickPhraseMatch struct {
	ID         string
	BuyerID    gosql.NullString
	FinishedAt gosql.NullTime
	// SortAt is the list's sort column: ship_by_sort_date (a DATE, so midnight UTC) or created_at.
	SortAt time.Time
}

// phrasePage is buildPickListQuery's page over PhraseMatches, computed from the columns already read
// instead of reading every match's row again. It applies the same predicates and order. Ids compare
// as bytes, which is the column collation's order for ids: lowercase letters, digits, '-' and '_' at
// matching positions.
func (q pickListQuery) phrasePage() []string {
	var buyers map[string]bool
	if q.BuyerIDs != nil {
		buyers = make(map[string]bool, len(q.BuyerIDs))
		for _, id := range q.BuyerIDs {
			buyers[id] = true
		}
	}
	cursorAt := q.CursorAt.Time
	if q.SortByShipBy {
		// The SQL compares against CAST(? AS DATE) of the UTC value.
		y, m, d := cursorAt.UTC().Date()
		cursorAt = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	asc := q.ascending()
	after := func(at time.Time, id string) bool {
		c := at.Compare(cursorAt)
		if c == 0 {
			c = strings.Compare(id, q.CursorID.String)
		}
		if asc {
			return c > 0
		}
		return c < 0
	}

	page := make([]pickPhraseMatch, 0, len(q.PhraseMatches))
	for _, m := range q.PhraseMatches {
		switch {
		case buyers != nil && (!m.BuyerID.Valid || !buyers[m.BuyerID.String]):
		case q.Status != nil && q.openOnly() == m.FinishedAt.Valid:
		case q.StartDate.Valid && m.SortAt.Before(q.StartDate.Time):
		case q.EndDate.Valid && m.SortAt.After(q.EndDate.Time):
		case q.CursorAt.Valid && !after(m.SortAt, m.ID):
		default:
			page = append(page, m)
		}
	}
	slices.SortFunc(page, func(a, b pickPhraseMatch) int {
		c := a.SortAt.Compare(b.SortAt)
		if c == 0 {
			c = strings.Compare(a.ID, b.ID)
		}
		if !asc {
			c = -c
		}
		return c
	})
	ids := make([]string, 0, min(len(page), int(q.Limit)))
	for _, m := range page[:min(len(page), int(q.Limit))] {
		ids = append(ids, m.ID)
	}
	return ids
}

// pickPhraseScanCap is the most picks a phrase may match and still be read up front to drive the
// list. A phrase matching more (one in every customer's name matches every pick) is common enough
// that walking an index with the phrase as a residual fills a page first, where driving from it
// reads and sorts every match.
const pickPhraseScanCap = 2000

// buildPickPhraseIDsQuery reads the phrase's matches, stopping one past pickPhraseScanCap to tell a
// set that fits from one that does not. UNION ALL streams and stops at the limit, where UNION would
// collect every match to deduplicate first; a pick matching several arms is deduplicated by the caller.
func buildPickPhraseIDsQuery(q pickListQuery) (string, []any) {
	args := q.appendPhraseArgs(make([]any, 0, 13))
	args = append(args, pickPhraseScanCap+1)
	return "(" + strings.Join(pickPhraseArms(q.Search.Phrase, q, q.phrasePagesInMemory()), ") UNION ALL (") + ") LIMIT ?", args
}

// buildPickPrefixCountQuery counts the account's picks whose number matches the LIKE prefix, stopping
// at limit, from the covering number index.
func buildPickPrefixCountQuery(accountID, prefix string, limit int) (string, []any) {
	return "SELECT COUNT(*) FROM (SELECT 1 FROM pick FORCE INDEX (" + pickAccountNumberIndex + ")" +
		" WHERE account_id = ? AND number LIKE ? LIMIT ?) capped", []any{accountID, prefix, limit}
}

// pickProductLineLines is the pick lines for a product in any of n product lines, read from the lines'
// products.
func pickProductLineLines(n int) string {
	return "product prod JOIN sales_order_line sol ON sol.product_id = prod.id" +
		" JOIN pick_line pl ON pl.sales_order_line_id = sol.id" +
		" WHERE prod.product_line_id IN (" + iclPlaceholders(n) + ")"
}

// pickProductLineMatches is the ids of picks with a line for a product in any of n product lines.
func pickProductLineMatches(n int) string {
	return "SELECT DISTINCT pl.pick_id AS id FROM " + pickProductLineLines(n)
}

// pickProductLineScanCap is the most pick lines product lines may have and still drive the read. That
// path reads all their picks and sorts them, so it is only cheap for few; with more, they are on enough
// picks that walking the sort index with them as a residual fills a page first. A product line on 98 of
// the largest account's 123k picks walked the other way read them all.
const pickProductLineScanCap = 2000

// buildPickProductLineCountQuery counts the product lines' pick lines, stopping at
// pickProductLineScanCap. Lines bound their picks from above, and counting them stops at the cap where
// collecting distinct picks would read every match first.
func buildPickProductLineCountQuery(productLineIDs []string) (string, []any) {
	args := make([]any, 0, len(productLineIDs)+1)
	args = append(args, stringArgs(productLineIDs)...)
	args = append(args, pickProductLineScanCap)
	return "SELECT COUNT(*) FROM (SELECT 1 FROM " + pickProductLineLines(len(productLineIDs)) + " LIMIT ?) capped", args
}

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
