package repository

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
)

// Helpers shared by the list queries built in Go (sales orders, invoices).

// listFilter accumulates a WHERE clause and its arguments.
type listFilter struct {
	where []string
	args  []any
}

func (f *listFilter) add(clause string, args ...any) {
	f.where = append(f.where, clause)
	f.args = append(f.args, args...)
}

func (f *listFilter) in(column string, values []string) {
	if len(values) > 0 {
		f.add(column+" IN ("+placeholders(len(values))+")", stringArgs(values)...)
	}
}

// keyset applies a cursor on (atCol, idCol) and returns the page's ORDER BY: newest first, or oldest
// first after a backward cursor (BuildPageString reverses those).
func (f *listFilter) keyset(cursor *pagination.StringCursor, atCol, idCol string) string {
	clause, args, order := keysetPredicate(cursor, atCol, idCol)
	if clause != "" {
		f.add(clause, args...)
	}
	return atCol + " " + order + ", " + idCol + " " + order
}

func (f *listFilter) whereSQL() string {
	if len(f.where) == 0 {
		return ""
	}
	return "\nWHERE " + strings.Join(f.where, "\nAND ")
}

func decodeListCursor(cursor *string) (*pagination.StringCursor, *apierror.APIError) {
	if cursor == nil {
		return nil, nil
	}
	c, err := pagination.DecodeStringCursor(*cursor)
	if err != nil {
		return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
	}
	return &c, nil
}

// inPageOrder returns byID's values in the order of ids, dropping ids it lacks.
func inPageOrder[T any](ids []string, byID map[string]T) []T {
	out := make([]T, 0, len(ids))
	for _, id := range ids {
		if v, ok := byID[id]; ok {
			out = append(out, v)
		}
	}
	return out
}

// selectStrings runs a query selecting one string column.
func selectStrings(ctx context.Context, q sqlc.DBTX, query string, args ...any) ([]string, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// productsInLines is the products in any of productLineIDs. A product line filter is read as its
// products so the planner sees, per product, how many order lines it has; through a join to product
// it averages them, and a common line's lines look rare.
func productsInLines(ctx context.Context, q sqlc.DBTX, productLineIDs []string) ([]string, error) {
	return selectStrings(ctx, q, "SELECT id FROM product WHERE product_line_id IN ("+placeholders(len(productLineIDs))+")",
		stringArgs(productLineIDs)...)
}

// orderLineSubqueries select the ids of the orders with a line on any of itemIDs, and with a line of any of
// productIDs, for each set given.
func orderLineSubqueries(itemIDs, productIDs []string) (queries []string, args [][]any) {
	if len(itemIDs) > 0 {
		queries = append(queries, "SELECT sol.sales_order_id AS id FROM sales_order_line sol WHERE sol.item_id IN ("+placeholders(len(itemIDs))+")")
		args = append(args, stringArgs(itemIDs))
	}
	if productIDs != nil {
		queries = append(queries, "SELECT sol.sales_order_id AS id FROM sales_order_line sol WHERE sol.product_id IN ("+placeholders(len(productIDs))+")")
		args = append(args, stringArgs(productIDs))
	}
	return queries, args
}

// groupOrderProbe is how many of a customer group's orders groupIsLarge counts before calling it
// large. Fewer, and reading them all to sort is cheap; more, and walking the list in order meets a
// page of them within a few thousand rows.
const groupOrderProbe = 1000

// groupIsLarge reports whether the account's customers in groupIDs hold at least groupOrderProbe of its
// orders. The planner orders joins without regard to LIMIT, so it reads a group's customers first and
// sorts every order they hold, however many; which way to read a group is decided here instead.
func groupIsLarge(ctx context.Context, q sqlc.DBTX, accountID string, groupIDs []string) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
    SELECT 1 FROM account_relation gr
    JOIN sales_order AS gso ON gso.owner_account_id = gr.owner_account_id AND gso.buyer_account_id = gr.counterparty_account_id
    WHERE gr.owner_account_id = ? AND gr.account_group_id IN (`+placeholders(len(groupIDs))+`)
    LIMIT ?) probe`, append(append([]any{accountID}, stringArgs(groupIDs)...), groupOrderProbe)...).Scan(&n)
	return n >= groupOrderProbe, err
}
