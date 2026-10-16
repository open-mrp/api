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
		itemType string
		category string
		unitID   string
	}{
		{"material", SeedMaterialCategoryID, SeedMaterialUnitID},
		{"part", SeedItemCategoryID, SeedUnitID},
	}
	for _, tc := range cases {
		t.Run(tc.itemType, func(t *testing.T) {
			markOutbox(t)
			sku := uniqueName("e2e-bup-cost-" + tc.itemType)
			row := func(cost string) map[string]any {
				return map[string]any{
					"type": tc.itemType, "sku": sku, "category_id": tc.category, "stocking_unit_id": tc.unitID,
					"unit_cost": priceInputPer(cost, tc.unitID),
				}
			}

			created, _ := jobResultIDs(bulkUpsertItemsJob(t, row("0.75")))
			require.Len(t, created, 1)
			itemID := created[0]
			before := len(costBasisReasons(t, itemID))

			_, updated := jobResultIDs(bulkUpsertItemsJob(t, row("0.750")))
			require.Equal(t, created, updated)
			assert.Len(t, costBasisReasons(t, itemID), before, "an unchanged cost restates nothing")

			_, updated = jobResultIDs(bulkUpsertItemsJob(t, row("1.25")))
			require.Equal(t, created, updated)
			requireNewCostBasisReason(t, itemID, before, "unit_cost_updated")
		})
	}
}
