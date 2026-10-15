//go:build e2e

package api_test

// Currency / non-currency unit IDs from shared/db/seed/0005_measures.sql.
// `dollar` is the global currency base unit; `each` is a global quantity unit.
const (
	currencyUnitID    = "dollar"
	nonCurrencyUnitID = "each"
)

// validMaterialBody returns a minimal valid preview.9 CreateMaterialRequest body. Tests
// override individual fields by writing into the returned map before posting.
func validMaterialBody(sku string) map[string]any {
	return map[string]any{
		"sku":         sku,
		"category_id": SeedMaterialCategoryID,
	}
}
