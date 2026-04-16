//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// firstProductID returns the id of the first product in seed data. Fails
// loudly if the list endpoint errors or the seed is empty so missing fixtures
// surface as real failures rather than silent skips.
func firstProductID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(productsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "products list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one product must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// Product — Include Tests
// ──────────────────────────────────────────────

func TestProducts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstProductID(t)

	status, body, err := apiClient.GetListRaw(productsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["product_type"], "product_type should be null without ?include=product_type")
	assert.Nil(t, got["product_line"], "product_line should be null without ?include=product_line")
	assert.Nil(t, got["item"], "item should be null without ?include=item")

	list, _, err := apiClient.GetList(productsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["product_type"], "product_type should be null on list items without ?include=product_type")
		assert.Nil(t, m["product_line"], "product_line should be null on list items without ?include=product_line")
		assert.Nil(t, m["item"], "item should be null on list items without ?include=item")
	}
}

func TestProducts_IncludeProductType(t *testing.T) {
	t.Parallel()
	id := firstProductID(t)

	status, body, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"product_type"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pt := jsonObject(got, "product_type")
	require.NotNil(t, pt, "product_type should be present with ?include=product_type")
	assert.Equal(t, "product_type", jsonField(pt, "object"))
	assert.NotEmpty(t, jsonField(pt, "id"))
}

func TestProducts_IncludeProductLine(t *testing.T) {
	t.Parallel()
	id := firstProductID(t)

	status, body, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_line")
	require.NotNil(t, pl, "product_line should be present with ?include=product_line")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
}

func TestProducts_IncludeItem(t *testing.T) {
	t.Parallel()
	id := firstProductID(t)

	status, body, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}

func TestProducts_ListIncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	pl := jsonObject(first, "product_line")
	require.NotNil(t, pl, "product_line should be present on list items with ?include=product_line")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
}
