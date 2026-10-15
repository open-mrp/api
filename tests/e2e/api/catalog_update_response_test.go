//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An update answers with what it saved: a role that may write an item but not read it gets the record back, includes
// and all, instead of a refusal after the save. preview.9's part, material and product routes answer the same way.
func TestCatalogUpdates_AnswerARoleThatMayOnlyWrite(t *testing.T) {
	t.Parallel()

	t.Run("item", func(t *testing.T) {
		itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-writeonly-item"))), "id")
		description := uniqueName("saved by a writer")
		status, body, err := customRoleClient(t, "items:update").Patch(withQuery(itemsPath+"/"+itemID, url.Values{"include": {"category"}}),
			map[string]any{"description": description}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, http.StatusOK, status, body)
		got := parseJSON(body)
		assert.Equal(t, "item", jsonField(got, "object"))
		assert.Equal(t, description, jsonField(got, "description"), "the answer is the record as saved")
		assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(got, "category"), "id"), "with its includes")
	})

	partID, _ := partInSocks(t)
	materialID := dashItemsCreateAs(t, apiClient, materialsPath, validMaterialBody(uniqueName("e2e-writeonly-mat")))
	productID, _ := parityProduct(t, "")
	for _, tc := range []struct {
		path, permission, object string
	}{
		{partsPath + "/" + partID, "parts:update", "part"},
		{materialsPath + "/" + materialID, "materials:update", "material"},
		{productsPath + "/" + productID, "items:update", "product"},
	} {
		t.Run("preview.9 "+tc.object, func(t *testing.T) {
			description := uniqueName("saved by a writer")
			writer := customRoleClient(t, tc.permission).WithAPIVersion(preview9Version)
			status, body, err := writer.Patch(withQuery(tc.path, url.Values{"include": {"item"}}), map[string]any{"description": description}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, http.StatusOK, status, body)
			got := parseJSON(body)
			assert.Equal(t, tc.object, jsonField(got, "object"))
			item := jsonObject(got, "item")
			require.NotNil(t, item, "the include is answered")
			assert.Equal(t, description, jsonField(item, "description"), "the answer is the record as saved")
		})
	}
}
