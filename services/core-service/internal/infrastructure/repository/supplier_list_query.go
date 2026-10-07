package repository

import (
	"strings"
	"unicode"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

// supplierSearchWordLimit bounds how many predicates a search emits; a supplier search is a name or a number, not prose.
const supplierSearchWordLimit = 16

// supplierListPageQuery selects one page of supplier account IDs in list order, emitting only the predicates the
// request uses. ListSuppliersByIDs reads the page's rows.
//
// Every word of a search must appear in the supplier's name, number or notes, each word matched on its own.
func supplierListPageQuery(params domain.ListSuppliersParams, cursor *pagination.StringCursor, limit int32) (string, []any) {
	var f listFilter
	f.add("ar.owner_account_id = ?", params.OwnerAccountID)
	f.add("ar.account_relation_role_code = 'supplier'")

	join := ""
	if words := supplierSearchWords(params.Query); len(words) > 0 {
		join = "\nJOIN account a ON a.id = ar.counterparty_account_id"
		for _, w := range words {
			like := "%" + db.EscapeLike(w) + "%"
			f.add("(COALESCE(NULLIF(ar.alias, ''), a.name) LIKE ? OR ar.external_number LIKE ? OR ar.notes LIKE ?)", like, like, like)
		}
	}
	if params.StartDate != nil {
		f.add("ar.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		f.add("ar.created_at <= ?", *params.EndDate)
	}
	if len(params.ItemIDs) > 0 {
		f.add("EXISTS (SELECT 1 FROM supplier_material sm JOIN material m ON m.id = sm.material_id"+
			" WHERE sm.supplier_account_id = ar.counterparty_account_id AND sm.owner_account_id = ar.owner_account_id"+
			" AND m.item_id IN ("+placeholders(len(params.ItemIDs))+"))", stringArgs(params.ItemIDs)...)
	}

	order := f.keyset(cursor, "ar.created_at", "ar.counterparty_account_id")
	// The owner's customers share these rows; this key holds only its suppliers, already in list order.
	return "SELECT ar.counterparty_account_id FROM account_relation ar FORCE INDEX (account_relation_owner_role_created_idx)" +
		join + f.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", append(f.args, limit)
}

// supplierSearchWords splits a search on whitespace and trims the punctuation around each word, so "Acme, Inc."
// matches "Acme Inc" while a number such as "SUP-001" stays whole. Repeats are dropped.
func supplierSearchWords(query *string) []string {
	if query == nil {
		return nil
	}
	notWordRune := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }
	var words []string
	seen := map[string]bool{}
	for _, field := range strings.Fields(*query) {
		w := strings.TrimFunc(field, notWordRune)
		key := strings.ToLower(w)
		if w == "" || seen[key] {
			continue
		}
		seen[key] = true
		words = append(words, w)
		if len(words) == supplierSearchWordLimit {
			break
		}
	}
	return words
}
