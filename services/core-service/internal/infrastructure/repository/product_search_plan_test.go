//go:build plans

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// TestSearchProducts_ReadsItsMatches holds the SKU search to the items it must examine: a prefix's
// range of the SKU key, or, for a substring no key can find, the account's items.
func TestSearchProducts_ReadsItsMatches(t *testing.T) {
	ensureCatalogCorpus(t)
	for _, tc := range []struct{ name, pattern, scope string }{
		{"prefix", "PLC-0000123%", "SELECT COUNT(*) FROM item WHERE account_id = ? AND sku LIKE 'PLC-0000123%'"},
		{"substring", "%0000123%", "SELECT COUNT(*) FROM item WHERE account_id = ?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkSetPlan(t, []string{"item", "product"}, func(q *sqlc.Queries) {
				_, apiErr := NewProductRepo(q).SearchBySKU(context.Background(), planCatAccount, tc.pattern)
				require.Nil(t, apiErr)
			}, func(t *testing.T, db *sql.DB) []planBound {
				n := planCount(t, db, tc.scope, planCatAccount)
				return []planBound{{"i.sku LIKE", "i", n}, {"i.sku LIKE", "p", n}}
			})
		})
	}
}

// TestListProducts_ReadsAPage holds the unordered first hundred sale products to reading about that
// many of the account's product items.
func TestListProducts_ReadsAPage(t *testing.T) {
	ensureCatalogCorpus(t)
	checkSetPlan(t, []string{"item", "product"}, func(q *sqlc.Queries) {
		_, apiErr := NewProductRepo(q).ListByAccount(context.Background(), planCatAccount)
		require.Nil(t, apiErr)
	}, func(t *testing.T, db *sql.DB) []planBound {
		return []planBound{{"LIMIT 100", "i", 100}, {"LIMIT 100", "p", 100}}
	})
}
