//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An update answers with what it saved: a role that may write a part, material or product but not read it gets the record back, includes and all, instead of a refusal after the save.
func TestCatalogUpdates_AnswerARoleThatMayOnlyWrite(t *testing.T) {
	t.Parallel()
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
		t.Run(tc.object, func(t *testing.T) {
			notes := uniqueName("saved by a writer")
			status, body, err := customRoleClient(t, tc.permission).Patch(withQuery(tc.path, url.Values{"include": {"item"}}), map[string]any{"notes": notes}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, http.StatusOK, status, body)
			got := parseJSON(body)
			assert.Equal(t, tc.object, jsonField(got, "object"))
			assert.Equal(t, notes, jsonField(jsonObject(got, "item"), "notes"), "the answer is the record as saved")
		})
	}
}
