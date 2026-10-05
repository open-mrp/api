package repository

import (
	"strconv"
	"strings"

	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

// territoryListPageQuery selects one page of territory IDs in list order, emitting only the predicates the
// request uses. Its joins match GetTerritoriesByIDs, which reads the page's rows, so no listed ID goes unread.
//
// A search matches the state, the rep's name or email and the product line name; a ZIP code also matches the
// territories covering it, as sales rep assignment (FindSalesRepByZipcode) does.
func territoryListPageQuery(accountID string, search *string, cursor *pagination.StringCursor, limit int32) (string, []any) {
	var f listFilter
	f.add("t.account_id = ?", accountID)

	joins := "JOIN account_user au ON au.id = t.sales_rep_id\nJOIN `user` u ON u.id = au.user_id"
	if search != nil && *search != "" {
		joins += "\nLEFT JOIN product_line pl ON pl.id = t.product_line_id AND (pl.account_id = t.account_id OR pl.account_id IS NULL)"
		like := "%" + db.EscapeLike(*search) + "%"
		match := "t.state LIKE ? OR u.name LIKE ? OR u.email LIKE ? OR pl.name LIKE ?"
		args := []any{like, like, like, like}
		if zip, ok := territoryZipcodeQuery(*search); ok {
			match += " OR (t.start_zipcode <= ? AND t.end_zipcode >= ?) OR (t.start_zipcode = ? AND t.end_zipcode IS NULL)"
			args = append(args, zip, zip, zip)
		}
		f.add("("+match+")", args...)
	}

	order := f.keyset(cursor, "t.created_at", "t.id")
	return "SELECT t.id FROM territory t\n" + joins + f.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", append(f.args, limit)
}

// territoryZipcodeQuery reads a search term as a ZIP code, leading zeros allowed.
func territoryZipcodeQuery(search string) (int32, bool) {
	trimmed := strings.TrimLeft(search, "0")
	if trimmed == "" {
		return 0, false
	}
	zip, err := strconv.ParseInt(trimmed, 10, 32)
	if err != nil || zip < 501 || zip > 99999 {
		return 0, false
	}
	return int32(zip), true
}
