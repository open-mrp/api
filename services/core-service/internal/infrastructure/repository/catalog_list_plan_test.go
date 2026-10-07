//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// planBound holds one table of one statement to reading at most max rows: the rows the result it
// returns is built from, twice over, plus a page of slack.
type planBound struct {
	// statement is a fragment that picks the statement out, e.g. "FROM product_line pl".
	statement, alias string
	max              float64
}

// checkSetPlan runs a request that returns a whole set (no keyset page) under both statistics modes
// and holds each bounded table to the rows its result needs.
func checkSetPlan(t *testing.T, tables []string, run func(q *sqlc.Queries), bounds func(*testing.T, *sql.DB) []planBound) {
	t.Helper()
	db := planDB(t)
	edb := &explainingDB{db: db}
	q := sqlc.New(edb)
	for _, mode := range planStatsModes {
		t.Run("stats="+mode, func(t *testing.T) {
			for _, table := range tables {
				usePlanStats(t, db, table, mode)
				t.Cleanup(func() { usePlanStats(t, db, table, "analyzed") })
			}
			edb.statements = nil
			run(q)
			for _, b := range bounds(t, db) {
				var stmt *explainedStatement
				for i := range edb.statements {
					if strings.Contains(edb.statements[i].query, b.statement) {
						stmt = &edb.statements[i]
					}
				}
				if stmt == nil {
					continue // not issued: nothing to read it for
				}
				limit := 2*b.max + 26
				if a := tableAccess(stmt.plan, b.alias); a.rows > limit {
					t.Errorf("read %.0f rows of %s via %v; its result needs %.0f\n%s", a.rows, b.alias, a.indexes, b.max, stmt.plan)
				}
			}
		})
	}
}

// planSmallTenant is an account from the dev seed with a handful of products, for checking that a
// catalog read for it does not walk the large tenant's rows.
func planSmallTenant(t *testing.T, db *sql.DB) string {
	t.Helper()
	var id string
	require.NoError(t, db.QueryRow(`SELECT i.account_id FROM product p JOIN item i ON i.id = p.item_id
		WHERE i.account_id <> ? AND p.product_line_id IS NOT NULL AND p.is_portal_ready = 1
		GROUP BY i.account_id ORDER BY COUNT(*) DESC LIMIT 1`, planCatAccount).Scan(&id))
	return id
}

func catalogPortalItems(t *testing.T, db *sql.DB, accountID string) float64 {
	return planCount(t, db, `SELECT COUNT(*) FROM item WHERE account_id = ? AND item_type_code = 'product' AND deleted_at IS NULL`, accountID)
}

func TestCatalogProductLines_ReadsTheirProducts(t *testing.T) {
	ensureCatalogCorpus(t)
	tables := []string{"product", "item", "product_line"}
	for _, tenant := range []string{"large", "small"} {
		t.Run(tenant, func(t *testing.T) {
			account := planCatAccount
			if tenant == "small" {
				account = planSmallTenant(t, planDB(t))
			}
			bounds := func(t *testing.T, db *sql.DB) []planBound {
				n := catalogPortalItems(t, db, account)
				lines := planCount(t, db, "SELECT COUNT(*) FROM product_line WHERE account_id = ? OR account_id IS NULL", account)
				return []planBound{
					{"FROM product_line pl", "p", n}, {"FROM product_line pl", "it", n}, {"FROM product_line pl", "pl", lines},
				}
			}
			t.Run("merchant", func(t *testing.T) {
				checkSetPlan(t, tables, func(q *sqlc.Queries) {
					_, apiErr := NewCatalogRepo(q).ListProductLines(context.Background(), account)
					require.Nil(t, apiErr)
				}, bounds)
			})
			t.Run("customer", func(t *testing.T) {
				checkSetPlan(t, tables, func(q *sqlc.Queries) {
					_, apiErr := NewCatalogRepo(q).ListProductLinesForCustomer(context.Background(), account, planCatCustomerID(0))
					require.Nil(t, apiErr)
				}, bounds)
			})
		})
	}
}

func TestCatalogProducts_ReadsTheirLine(t *testing.T) {
	ensureCatalogCorpus(t)
	tables := []string{"product", "item", "_item_attributes"}
	for _, line := range []string{planCatLineID(0), planCatLineID(planCatLines - 1)} {
		t.Run(line, func(t *testing.T) {
			bounds := func(t *testing.T, db *sql.DB) []planBound {
				n := planCount(t, db, "SELECT COUNT(*) FROM product WHERE product_line_id = ?", line)
				attrs := planCount(t, db, `SELECT COUNT(*) FROM _item_attributes ia JOIN product p ON p.item_id = ia.B
					WHERE p.product_line_id = ? AND p.is_portal_ready = 1`, line)
				return []planBound{
					{"FROM product p", "p", n}, {"FROM product p", "it", n}, {"FROM product p", "ic", n},
					{"FROM _item_attributes ia", "ia", attrs}, {"FROM _item_attributes ia", "att", attrs},
				}
			}
			t.Run("merchant", func(t *testing.T) {
				checkSetPlan(t, tables, func(q *sqlc.Queries) {
					_, apiErr := NewCatalogRepo(q).ListProducts(context.Background(), planCatAccount, line)
					require.Nil(t, apiErr)
				}, bounds)
			})
			t.Run("customer", func(t *testing.T) {
				checkSetPlan(t, tables, func(q *sqlc.Queries) {
					_, apiErr := NewCatalogRepo(q).ListProductsForCustomer(context.Background(), planCatAccount, planCatCustomerID(0), line)
					require.Nil(t, apiErr)
				}, bounds)
			})
		})
	}
}

// legacyCatalogProductLinesSQL is the account's lines as read off its portal-ready products, which
// ListCatalogProductLines must keep returning: including a line with no account, which legacy rows have.
const legacyCatalogProductLinesSQL = `SELECT pl.id FROM (
  SELECT DISTINCT p.product_line_id FROM item it JOIN product p ON p.item_id = it.id
  WHERE it.account_id = ? AND it.item_type_code = 'product' AND it.deleted_at IS NULL AND p.is_portal_ready = 1
) portal_lines JOIN product_line pl ON pl.id = portal_lines.product_line_id ORDER BY pl.name, pl.id`

func TestCatalogProductLines_MatchTheAccountsProducts(t *testing.T) {
	ensureCatalogCorpus(t)
	db := planDB(t)
	ctx := context.Background()

	// A line with no account, holding one of the large tenant's portal-ready products.
	var productID, lineID string
	require.NoError(t, db.QueryRow(`SELECT p.id, p.product_line_id FROM product p JOIN item it ON it.id = p.item_id
		WHERE it.account_id = ? AND p.is_portal_ready = 1 AND it.deleted_at IS NULL AND it.item_type_code = 'product'
		ORDER BY p.id LIMIT 1`, planCatAccount).Scan(&productID, &lineID))
	_, err := db.Exec(`INSERT INTO product_line (id, name, unit_group_id, account_id, created_at) VALUES ('pdln_plancat_unowned', 'Plan unowned line', 'ungp_plancat', NULL, NOW())`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE product SET product_line_id = 'pdln_plancat_unowned' WHERE id = ?`, productID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE product SET product_line_id = ? WHERE id = ?`, lineID, productID)
		_, _ = db.Exec(`DELETE FROM product_line WHERE id = 'pdln_plancat_unowned'`)
	})

	for _, account := range []string{planCatAccount, planSmallTenant(t, db)} {
		want, err := selectStrings(ctx, db, legacyCatalogProductLinesSQL, account)
		require.NoError(t, err)
		lines, apiErr := NewCatalogRepo(sqlc.New(db)).ListProductLines(ctx, account)
		require.Nil(t, apiErr)
		got := make([]string, len(lines))
		for i, l := range lines {
			got[i] = l.ID
		}
		require.ElementsMatch(t, want, got, account)
		require.NotEmpty(t, got, account)
	}
}
