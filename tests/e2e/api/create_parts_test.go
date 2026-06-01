//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validPartBody returns a minimal valid CreatePartRequest body.
func validPartBody(sku string) map[string]any {
	return map[string]any{
		"sku":         sku,
		"category_id": SeedItemCategoryID,
	}
}

func TestParts_Create_Basic_DefaultsRates(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(partsPath, validPartBody(uniqueName("e2e-part")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	assert.Equal(t, "part", jsonField(got, "object"))
}

func TestParts_Create_PersistsNotes(t *testing.T) {
	t.Parallel()

	notes := "deep groove ball bearing, lubricated"
	body := validPartBody(uniqueName("e2e-part-notes"))
	body["notes"] = notes

	resp, err := apiClient.PostFull(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	// Notes live on the wrapped item record. Round-trip via GET to confirm.
	getStatus, getBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, notes, jsonField(jsonObject(parseJSON(getBody), "item"), "notes"))
}

func TestParts_Create_WithValidRates(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-rates"))
	body["unit_price"] = map[string]any{
		"value":               "5.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["unit_cost"] = map[string]any{
		"value":               "2.50",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}

	resp, err := apiClient.PostFull(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	// Verify unit_cost was persisted with the supplied currency numerator by
	// expanding the nested rate via ?include=item.unit_cost.
	rateStatus, rateBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, rateStatus, rateBody)

	uc := jsonObject(jsonObject(parseJSON(rateBody), "item"), "unit_cost")
	require.NotNil(t, uc, "item.unit_cost should be populated with ?include=item.unit_cost")
	assert.Equal(t, "rate", jsonField(uc, "object"))
	assert.Equal(t, "2.50", jsonField(uc, "value"))
}

func TestParts_Create_RejectsNonCurrencyNumeratorOnUnitCost(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-bad-num"))
	body["unit_cost"] = map[string]any{
		"value":               "1.00",
		"numerator_unit_id":   nonCurrencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}

	status, respBody, err := apiClient.Post(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_cost.numerator_unit_id")
}

func TestParts_Create_RejectsCurrencyDenominatorOnUnitPrice(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-bad-den"))
	body["unit_price"] = map[string]any{
		"value":               "1.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": currencyUnitID,
	}

	status, respBody, err := apiClient.Post(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_price.denominator_unit_id")
}

func TestParts_Create_LinksAttributes(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-attrs"))
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := apiClient.PostFull(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	partStatus, partBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, partStatus, partBody)
	itemID := jsonField(jsonObject(parseJSON(partBody), "item"), "id")
	require.NotEmpty(t, itemID)

	itemStatus, itemBody, err := apiClient.GetListRaw(
		"/v1/catalog/items/"+itemID,
		url.Values{"include": {"attributes"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, itemStatus, itemBody)

	attrs := jsonListData(parseJSON(itemBody), "attributes")
	require.NotEmpty(t, attrs)

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

func TestParts_Create_MissingSKU(t *testing.T) {
	t.Parallel()

	status, respBody, err := apiClient.Post(partsPath, map[string]any{
		"category_id": SeedItemCategoryID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestParts_Create_EmptySKU(t *testing.T) {
	t.Parallel()

	status, respBody, err := apiClient.Post(partsPath, map[string]any{
		"sku":         "",
		"category_id": SeedItemCategoryID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestParts_Create_MissingCategoryID(t *testing.T) {
	t.Parallel()

	status, respBody, err := apiClient.Post(partsPath, map[string]any{
		"sku": uniqueName("e2e-part-no-cat"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "category_id")
}
