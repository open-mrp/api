//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The parts routes answer 1.0.forge-preview.9 callers only; forge manages parts as items of type part. These tests
// pin preview.9 and check the shapes it still answers with.

const partsPath = "/v1/catalog/parts"

// partPreview9Price is a preview.9 price input: a rate with the dollar numerator.
func partPreview9Price(value, perUnitID string) map[string]any {
	return map[string]any{"value": value, "numerator_unit_id": currencyUnitID, "denominator_unit_id": perUnitID}
}

// assertPartPreview9Price checks a preview.9 price: a rate with no currency unit, per the given unit.
func assertPartPreview9Price(t *testing.T, rate map[string]any, value, perUnitID, label string) {
	t.Helper()
	require.NotNil(t, rate, "%s is present", label)
	assert.Equal(t, "rate", jsonField(rate, "object"), label)
	assertDecimalEqual(t, value, jsonField(rate, "value"), label+".value")
	assert.Nil(t, rate["numerator_unit"], "%s: preview.9 reads the currency unit as not included", label)
	assert.Equal(t, perUnitID, jsonField(jsonObject(rate, "denominator_unit"), "id"), label+".denominator_unit")
	assert.NotEmpty(t, jsonField(rate, "display_value"), label+".display_value")
	assert.NotContains(t, rate, "amount", "%s is not a forge UnitPrice", label)
}

func preview9PartPost(t *testing.T, path string, body map[string]any) (int, []byte) {
	t.Helper()
	status, respBody, err := apiClient.WithAPIVersion(preview9Version).Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, respBody
}

func preview9PartPatch(t *testing.T, path string, body map[string]any) (int, []byte) {
	t.Helper()
	status, respBody, err := apiClient.WithAPIVersion(preview9Version).Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, respBody
}

func preview9PartGet(t *testing.T, path string, params url.Values) (int, []byte) {
	t.Helper()
	status, respBody, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(path, params)
	require.NoError(t, err)
	return status, respBody
}

// createPreview9Part creates a part on preview.9 (path may carry a query) and deletes it after the test.
func createPreview9Part(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	status, respBody := preview9PartPost(t, path, body)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	partID := jsonField(created, "id")
	require.NotEmpty(t, partID)
	t.Cleanup(func() { apiClient.WithAPIVersion(preview9Version).Delete(partsPath + "/" + partID) })
	return created
}

// partListHasID reports whether a list response body holds the id.
func partListHasID(t *testing.T, body []byte, want string) bool {
	t.Helper()
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if m, ok := raw.(map[string]any); ok && jsonField(m, "id") == want {
			return true
		}
	}
	return false
}

// A part's whole life on preview.9: create with every field, read, update, list, delete, and a replayed delete.
func TestParts_Preview9Lifecycle(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	const includes = "?include=item,item.category,item.attributes"

	sku := uniqueName("e2e-part-life")
	body := validPartBody(sku)
	body["description"] = "deep groove ball bearing"
	body["notes"] = "lubricated"
	body["unit_price"] = partPreview9Price("5.00", nonCurrencyUnitID)
	body["unit_cost"] = partPreview9Price("2.50", nonCurrencyUnitID)
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := preview.PostFull(partsPath+includes, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	partID := jsonField(created, "id")
	assertIDFormat(t, partID, id.PartIDPrefix)
	assertCreatedLocation(t, resp.Header, partID)
	assertObjectField(t, created, "part")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	item := jsonObject(created, "item")
	require.NotNil(t, item, "item is included")
	itemID := jsonField(item, "id")
	assertIDFormat(t, itemID, id.ItemIDPrefix)
	assertPreview9ItemShape(t, item)
	assert.Equal(t, "part", jsonField(item, "type"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, "deep groove ball bearing", jsonField(item, "description"))
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(item, "category"), "id"))
	assert.Equal(t, []string{SeedAttributeID}, attributeIDsOf(item))
	assertPartPreview9Price(t, jsonObject(item, "unit_value"), "5.00", nonCurrencyUnitID, "item.unit_value")
	assertPartPreview9Price(t, jsonObject(item, "unit_cost"), "2.50", nonCurrencyUnitID, "item.unit_cost")
	burnRate := jsonObject(item, "burn_rate")
	require.NotNil(t, burnRate, "burn_rate is always present")
	assert.Equal(t, "rate", jsonField(burnRate, "object"))
	assertDecimalEqual(t, "0", jsonField(burnRate, "value"), "a new part burns nothing")
	assert.Equal(t, dayUnitID, jsonField(jsonObject(burnRate, "denominator_unit"), "id"))

	t.Run("retrieve", func(t *testing.T) {
		status, raw := preview9PartGet(t, partsPath+"/"+partID, nil)
		requireStatus(t, 200, status, raw)
		got := parseJSON(raw)
		assert.Equal(t, partID, jsonField(got, "id"))
		assertNilField(t, got, "item")

		status, raw = preview9PartGet(t, partsPath+"/"+partID, url.Values{"include": {"item"}})
		requireStatus(t, 200, status, raw)
		got = parseJSON(raw)
		gotItem := jsonObject(got, "item")
		require.NotNil(t, gotItem)
		assert.Equal(t, itemID, jsonField(gotItem, "id"))
		assertPreview9ItemShape(t, gotItem)
		assertNilField(t, gotItem, "category")
		assertNilField(t, gotItem, "attributes")
	})

	t.Run("forge reads the part as an item", func(t *testing.T) {
		forgeItem := getItem(t, itemID)
		assertItemEnvelope(t, forgeItem, "part", sku)
		assertNilField(t, forgeItem, "sales") // a stored price alone doesn't make a part sold
		assertItemPrice(t, jsonObject(forgeItem, "unit_cost"), "2.50", nonCurrencyUnitID, "unit_cost")
	})

	newSKU := uniqueName("e2e-part-life-upd")
	t.Run("update", func(t *testing.T) {
		status, raw := preview9PartPatch(t, partsPath+"/"+partID+includes, map[string]any{
			"sku":   newSKU,
			"notes": "ignored",
		})
		requireStatus(t, 200, status, raw)
		updated := parseJSON(raw)
		assert.Equal(t, partID, jsonField(updated, "id"))
		assertObjectField(t, updated, "part")
		updatedItem := jsonObject(updated, "item")
		require.NotNil(t, updatedItem)
		assert.Equal(t, newSKU, jsonField(updatedItem, "sku"))
		assert.Equal(t, "deep groove ball bearing", jsonField(updatedItem, "description"), "an omitted field keeps its value")
		assertNilField(t, updatedItem, "notes")
		assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(updatedItem, "category"), "id"))
		assertPartPreview9Price(t, jsonObject(updatedItem, "unit_value"), "5.00", nonCurrencyUnitID, "item.unit_value")
	})

	t.Run("list filters", func(t *testing.T) {
		for name, params := range map[string]url.Values{
			"q":             {"q": {newSKU}},
			"category_ids":  {"q": {newSKU}, "category_ids": {SeedItemCategoryID}},
			"attribute_ids": {"q": {newSKU}, "attribute_ids": {SeedAttributeID}},
		} {
			status, raw := preview9PartGet(t, partsPath, params)
			requireStatus(t, 200, status, raw)
			assert.True(t, partListHasID(t, raw, partID), "%s filter finds the part", name)
		}
		status, raw := preview9PartGet(t, partsPath, url.Values{"q": {newSKU}, "category_ids": {SeedPropertylessItemCategoryID}})
		requireStatus(t, 200, status, raw)
		assert.False(t, partListHasID(t, raw, partID), "another category's filter leaves the part out")
	})

	t.Run("delete", func(t *testing.T) {
		status, raw, err := preview.Delete(partsPath + "/" + partID)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		assert.Equal(t, map[string]any{}, parseJSON(raw), "preview.9 answers a delete with an empty object")

		status, raw, err = preview.Delete(partsPath + "/" + partID)
		require.NoError(t, err)
		requireStatus(t, 410, status, raw)
		errObj := requirePreview9Error(t, raw, "resource_gone")
		assert.Nil(t, errObj["param"])

		status, raw = preview9PartGet(t, partsPath+"/"+partID, nil)
		assert.Equal(t, 404, status, "a deleted part is gone: %s", raw)
		status, raw, err = apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "deleting a part deletes its item: %s", raw)
	})
}

// A part made with only the required fields has no description and no notes.
func TestParts_Preview9OmittedFields(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-part-omit")
	created := createPreview9Part(t, partsPath+"?include=item", validPartBody(sku))
	item := jsonObject(created, "item")
	require.NotNil(t, item)
	assertPreview9ItemShape(t, item)
	assert.Equal(t, sku, jsonField(item, "sku"))
	assertNilField(t, item, "description")
	assertNilField(t, item, "notes")
}

func TestParts_Preview9CreateIdempotent(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	key := newIdempotencyKey()
	body := validPartBody(uniqueName("e2e-part-idem"))

	s1, b1, err := preview.Post(partsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s1, b1)
	partID := jsonField(parseJSON(b1), "id")
	require.NotEmpty(t, partID)
	t.Cleanup(func() { preview.Delete(partsPath + "/" + partID) })

	s2, b2, err := preview.Post(partsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s2, b2)
	assert.Equal(t, partID, jsonField(parseJSON(b2), "id"), "a replay answers the first part")
}

func TestParts_Preview9UpdateIdempotent(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	partID := jsonField(createPreview9Part(t, partsPath, validPartBody(uniqueName("e2e-part-patch-idem"))), "id")
	key := newIdempotencyKey()
	patch := map[string]any{"sku": uniqueName("e2e-part-patch-idem-sku")}

	s1, b1, err := preview.Patch(partsPath+"/"+partID, patch, key)
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)
	s2, b2, err := preview.Patch(partsPath+"/"+partID, patch, key)
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)
	assert.Equal(t, parseJSON(b1), parseJSON(b2), "a replay answers the first response")
}

func TestParts_Preview9ListPagination(t *testing.T) {
	t.Parallel()
	// Scoped by a unique search term so parallel tests can't shift the cursor window. The client pins the parts
	// routes to preview.9.
	prefix := uniqueName("e2e-part-pg")
	var ids []string
	for i := 0; i < 2; i++ {
		ids = append(ids, jsonField(createPreview9Part(t, partsPath, validPartBody(fmt.Sprintf("%s-%d", prefix, i))), "id"))
	}
	assertScopedCursorPagination(t, partsPath, url.Values{"q": {prefix}}, ids)
}
