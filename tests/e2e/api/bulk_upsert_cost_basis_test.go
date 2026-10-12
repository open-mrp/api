//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A bulk upsert that moves an existing row's unit cost restates the cost of what is built from it, as a single update does; one that leaves the cost alone does not.
func TestBulkUpsert_UnitCostChangeRestatesCostBasis(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		basePath string
		category string
		upsert   func(t *testing.T, row map[string]any) (created, updated []string)
		cleanup  func(ids []string)
	}{
		{"materials", materialsPath, SeedMaterialCategoryID, func(t *testing.T, row map[string]any) ([]string, []string) {
			return bulkUpsertMaterialIDs(t, row)
		}, cleanupMaterialIDs},
		{"parts", partsPath, SeedItemCategoryID, func(t *testing.T, row map[string]any) ([]string, []string) {
			return bulkUpsertPartIDs(t, row)
		}, cleanupPartIDs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			markOutbox(t)
			sku := uniqueName("e2e-bup-cost-" + tc.name)
			row := func(cost string) map[string]any {
				return map[string]any{"sku": sku, "category": map[string]any{"id": tc.category}, "unit_cost": bulkCurrencyRate(cost)}
			}

			created, _ := tc.upsert(t, row("0.75"))
			require.Len(t, created, 1)
			defer tc.cleanup(created)
			itemID := jsonField(catalogItem(t, tc.basePath, created[0]), "id")
			before := len(costBasisReasons(t, itemID))

			_, updated := tc.upsert(t, row("0.750"))
			require.Equal(t, created, updated)
			assert.Len(t, costBasisReasons(t, itemID), before, "an unchanged cost restates nothing")

			_, updated = tc.upsert(t, row("1.25"))
			require.Equal(t, created, updated)
			requireNewCostBasisReason(t, itemID, before, "unit_cost_updated")
		})
	}
}
