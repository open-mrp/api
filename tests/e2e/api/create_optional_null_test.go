//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCreateOptionalFieldExplicitNullRejected documents that nullable create fields
// still reject explicit JSON null (RejectExplicitJSONNulls); only PATCH clearable fields accept null.
func TestCreateOptionalFieldExplicitNullRejected(t *testing.T) {
	t.Parallel()

	t.Run("material_description", func(t *testing.T) {
		t.Parallel()
		status, _, err := apiClient.Post("/v1/catalog/materials", map[string]any{
			"sku":         uniqueName("e2e-mat-null"),
			"category_id": SeedMaterialCategoryID,
			"description": nil,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.True(t, status == 400 || status == 422,
			"explicit null on optional create field should be rejected, got %d", status)
	})

	t.Run("address_input_phone", func(t *testing.T) {
		t.Parallel()
		status, _, err := apiClient.Post(addressesPath, map[string]any{
			"name":          uniqueName("e2e-addr-null"),
			"phone":         nil,
			"street_line_1": "1 Null St",
			"locality":      "Denver",
			"state":         "CO",
			"postal_code":   "80202",
			"country":       "US",
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.True(t, status == 400 || status == 422,
			"explicit null on optional create field should be rejected, got %d", status)
	})
}
