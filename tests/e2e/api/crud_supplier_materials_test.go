//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func supplierMaterialsPath() string {
	return suppliersPath + "/" + SeedSupplierAccountID + "/materials"
}

// firstSupplierMaterialID returns the id of the first supplier material under
// the seeded supplier. Fails loudly if none exist.
func firstSupplierMaterialID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(supplierMaterialsPath(), nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "supplier materials list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one supplier material must be seeded for supplier %q", SeedSupplierAccountID)
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// SupplierMaterial — Include Tests
// ──────────────────────────────────────────────

func TestSupplierMaterials_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstSupplierMaterialID(t)

	status, body, err := apiClient.GetListRaw(supplierMaterialsPath()+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["material"], "material should be null without ?include=material")

	list, _, err := apiClient.GetList(supplierMaterialsPath(), nil)
	require.NoError(t, err)
	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["material"], "material should be null on list items without ?include=material")
	}
}

func TestSupplierMaterials_IncludeMaterial(t *testing.T) {
	t.Parallel()
	id := firstSupplierMaterialID(t)

	status, body, err := apiClient.GetListRaw(supplierMaterialsPath()+"/"+id, url.Values{"include": {"material"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	material := jsonObject(got, "material")
	require.NotNil(t, material, "material should be present with ?include=material")
	assert.Equal(t, "material", jsonField(material, "object"))
	assert.NotEmpty(t, jsonField(material, "id"))
	// nested item should remain null without nested include
	assert.Nil(t, material["item"], "material.item should be null without ?include=material.item")
}

func TestSupplierMaterials_IncludeMaterialItem(t *testing.T) {
	t.Parallel()
	id := firstSupplierMaterialID(t)

	status, body, err := apiClient.GetListRaw(supplierMaterialsPath()+"/"+id, url.Values{"include": {"material.item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	material := jsonObject(got, "material")
	require.NotNil(t, material, "material should be present with ?include=material.item")
	item := jsonObject(material, "item")
	require.NotNil(t, item, "material.item should be present with ?include=material.item")
	assert.Equal(t, "item", jsonField(item, "object"))
}
