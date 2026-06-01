//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Currency / non-currency unit IDs from shared/db/seed/0005_measures.sql.
// `dollar` is the global currency base unit; `each` is a global quantity unit.
const (
	currencyUnitID    = "dollar"
	nonCurrencyUnitID = "each"
)

// validMaterialBody returns a minimal valid CreateMaterialRequest body. Tests
// override individual fields by writing into the returned map before posting.
func validMaterialBody(sku string) map[string]any {
	return map[string]any{
		"sku":         sku,
		"category_id": SeedItemCategoryID,
	}
}

func TestMaterials_Create_Basic_DefaultsRates(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-mat")
	body := validMaterialBody(sku)

	resp, err := apiClient.PostFull(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + id) })

	assert.Equal(t, "material", jsonField(got, "object"))
}

func TestMaterials_Create_WithValidRates(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-mat-rates")
	body := validMaterialBody(sku)
	body["unit_price"] = map[string]any{
		"value":               "1.50",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["unit_cost"] = map[string]any{
		"value":               "0.75",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}

	resp, err := apiClient.PostFull(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + id) })
}

func TestMaterials_Create_RejectsNonCurrencyNumeratorOnUnitCost(t *testing.T) {
	t.Parallel()

	body := validMaterialBody(uniqueName("e2e-mat-bad-num"))
	body["unit_cost"] = map[string]any{
		"value":               "0.50",
		"numerator_unit_id":   nonCurrencyUnitID, // wrong: must be currency
		"denominator_unit_id": nonCurrencyUnitID,
	}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_cost.numerator_unit_id")
}

func TestMaterials_Create_RejectsCurrencyDenominatorOnUnitPrice(t *testing.T) {
	t.Parallel()

	body := validMaterialBody(uniqueName("e2e-mat-bad-den"))
	body["unit_price"] = map[string]any{
		"value":               "1.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": currencyUnitID, // wrong: must not be currency
	}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_price.denominator_unit_id")
}

func TestMaterials_Create_LinksAttributes(t *testing.T) {
	t.Parallel()

	body := validMaterialBody(uniqueName("e2e-mat-attrs"))
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := apiClient.PostFull(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + id) })

	materialStatus, materialBody, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, materialStatus, materialBody)
	itemID := jsonField(jsonObject(parseJSON(materialBody), "item"), "id")
	require.NotEmpty(t, itemID)

	// Verify the attribute is reachable via the underlying item. Materials wrap
	// items, and attributes are exposed through the item include.
	itemStatus, itemBody, err := apiClient.GetListRaw(
		"/v1/catalog/items/"+itemID,
		url.Values{"include": {"attributes"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, itemStatus, itemBody)

	itemDoc := parseJSON(itemBody)
	attrs := jsonListData(itemDoc, "attributes")
	require.NotEmpty(t, attrs, "expected at least one attribute on the new material's item")

	found := false
	for _, raw := range attrs {
		obj, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(obj, "id") == SeedAttributeID {
			found = true
			break
		}
	}
	assert.True(t, found, "expected linked attribute %s in item.attributes", SeedAttributeID)
}
