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
	pickShipByIndex        = "pick_account_ship_by_idx"
	pickOpenShipByIndex    = "pick_account_finished_ship_by_idx"
	pickCreatedIndex       = "pick_account_created_idx"
	pickOpenCreatedIndex   = "pick_account_finished_created_idx"
	pickBuyerShipByIndex   = "pick_account_buyer_ship_by_idx"
	pickBuyerCreatedIndex  = "pick_account_buyer_created_idx"
	pickAccountNumberIndex = "pick_account_number_idx"
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
	AccountID        string
	SortByShipBy     bool
	Search           pickSearch
	Status           *string
	CustomerIDs      []string
	CustomerGroupIDs []string
	ProductLineIDs   []string
	StartDate        gosql.NullTime
	EndDate          gosql.NullTime
	Direction        pagination.Direction
	CursorAt         gosql.NullTime
	CursorID         gosql.NullString
	Limit            int32
}

// openOnly reports the open filter; any status other than "closed" is open.
func (q pickListQuery) openOnly() bool {
	return q.Status != nil && *q.Status != "closed"
}

// indexHint names the indexes the scan may drive from, so MySQL cannot pick one that filesorts the
// account or walks it for a rare filter value. A customer filter pins its buyer; status=open pins
// finished_at because open picks are a tiny slice of a mostly-closed table. Where the better choice
// depends on how many rows match (a number prefix, a customer group), MySQL chooses among the listed
// indexes from its range estimates. A phrase search drives from its match set and takes no hint.
func (q pickListQuery) indexHint() []string {
	if q.Search.Phrase != "" {
		return nil
	}

	sortIndex, buyerIndex := pickCreatedIndex, pickBuyerCreatedIndex
	switch {
	case q.SortByShipBy && q.openOnly():
		sortIndex, buyerIndex = pickOpenShipByIndex, pickBuyerShipByIndex
	case q.SortByShipBy:
		sortIndex, buyerIndex = pickShipByIndex, pickBuyerShipByIndex
	case q.openOnly():
		sortIndex = pickOpenCreatedIndex
	}

	switch {
	case len(q.CustomerIDs) > 0:
		return []string{buyerIndex}
	case q.Search.NumberPrefix != "":
		return []string{sortIndex, pickAccountNumberIndex}
	case len(q.CustomerGroupIDs) > 0:
		return []string{sortIndex, buyerIndex}
	default:
		return []string{sortIndex}
	}
}

// buildPickListQuery returns the query for one page of pick ids (Limit rows, in list order) and its
// bind args. Only the predicates the caller set are emitted, so a bare ORDER BY can land on an index.
//
// Ship-by pages forward ascending (soonest first), created-at forward descending (newest first);
// backward is the reverse, undone by BuildPageString.
func buildPickListQuery(q pickListQuery) (string, []any) {
	args := make([]any, 0, 16+len(q.CustomerIDs)+len(q.CustomerGroupIDs)+len(q.ProductLineIDs))

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
	if len(q.CustomerIDs) > 0 {
		b.WriteString(" AND p.buyer_account_id IN (" + iclPlaceholders(len(q.CustomerIDs)) + ")")
		for _, id := range q.CustomerIDs {
			args = append(args, id)
		}
	}
	if len(q.CustomerGroupIDs) > 0 {
		b.WriteString(" AND p.buyer_account_id IN (SELECT gar.counterparty_account_id FROM account_relation gar" +
			" WHERE gar.owner_account_id = ? AND gar.account_group_id IN (" + iclPlaceholders(len(q.CustomerGroupIDs)) + "))")
		args = append(args, q.AccountID)
		for _, id := range q.CustomerGroupIDs {
			args = append(args, id)
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

	ascending := q.SortByShipBy == (q.Direction == pagination.DirectionForward)
	cmp, order := "<", "DESC"
	if ascending {
		cmp, order = ">", "ASC"
	}

	// Keyset over (sort column, id). For ship-by, CAST keeps the param a DATE while the column stays
	// bare, so the composite is still usable.
	col, param := "p.created_at", "?"
	if q.SortByShipBy {
		col, param = "p.ship_by_sort_date", "CAST(? AS DATE)"
	}
	if q.CursorAt.Valid {
		b.WriteString(" AND (" + col + " " + cmp + " " + param + " OR (" + col + " = " + param + " AND p.id " + cmp + " ?))")
		args = append(args, q.CursorAt.Time, q.CursorAt.Time, q.CursorID.String)
	}

	b.WriteString(" ORDER BY " + col + " " + order + ", p.id " + order + " LIMIT ?")
	args = append(args, q.Limit)

	return b.String(), args
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
