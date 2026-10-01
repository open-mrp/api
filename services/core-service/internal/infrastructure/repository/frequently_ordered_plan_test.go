//go:build plans

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// TestFrequentlyOrderedProducts_ReadsItsWindow holds the frequently-ordered aggregate to the lines of
// the customer's most recent orders, the window it counts, for a customer with a long history and one
// with a short one.
func TestFrequentlyOrderedProducts_ReadsItsWindow(t *testing.T) {
	ensureCatalogCorpus(t)
	for _, customer := range []string{planCatCustomerID(0), planCatCustomerID(7)} {
		t.Run(customer, func(t *testing.T) {
			bounds := func(t *testing.T, db *sql.DB) []planBound {
				window := `SELECT id FROM (SELECT id FROM sales_order WHERE owner_account_id = ? AND buyer_account_id = ?
					ORDER BY created_at DESC, id DESC LIMIT 400) w`
				orders := planCount(t, db, "SELECT COUNT(*) FROM ("+window+") o", planCatAccount, customer)
				lines := planCount(t, db, "SELECT COUNT(*) FROM sales_order_line WHERE sales_order_id IN ("+window+")", planCatAccount, customer)
				return []planBound{
					{"FROM sales_order FORCE INDEX", "sales_order", orders},
					{"FROM sales_order FORCE INDEX", "sol", lines},
					{"FROM sales_order FORCE INDEX", "fg", lines},
				}
			}
			checkSetPlan(t, []string{"sales_order_line", "product"}, func(q *sqlc.Queries) {
				_, apiErr := NewCustomerRepo(q).GetFrequentlyOrderedProducts(context.Background(), planCatAccount, customer)
				require.Nil(t, apiErr)
			}, bounds)
		})
	}
}
