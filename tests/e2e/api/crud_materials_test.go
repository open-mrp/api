//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const materialsPath = "/v1/operations/materials"

// firstMaterialID returns the id of the first material in seed data. Fails if
// no materials are seeded so the suite flags missing fixtures rather than skipping.
func firstMaterialID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(materialsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "materials list endpoint should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one material must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// Material — Include Tests
// ──────────────────────────────────────────────

func TestMaterials_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstMaterialID(t)

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["item"], "item should be null without ?include=item")

	list, _, err := apiClient.GetList(materialsPath, nil)
	require.NoError(t, err)
	for _, m := range list.Data {
		mm := parseJSON(m)
		assert.Nil(t, mm["item"], "item should be null on list items without ?include=item")
	}
}

func TestMaterials_IncludeItem(t *testing.T) {
	t.Parallel()
	id := firstMaterialID(t)

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}
