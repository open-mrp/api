//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The materials routes answer 1.0.forge-preview.9 callers only; forge manages materials as items of type material
// (crud_items_test.go). These tests pin the preview.9 contract: the material object, its preview.9 item and value
// shapes, and its errors.
const materialsPath = "/v1/catalog/materials"

// priceInputPer is a UnitPriceInput in dollars per perUnitID.
func priceInputPer(amount, perUnitID string) map[string]any {
	return map[string]any{"amount": amount, "currency": "usd", "per_unit_id": perUnitID}
}

// moneyInputUSD is a MoneyInput in dollars.
func moneyInputUSD(amount string) map[string]any {
	return map[string]any{"amount": amount, "currency": "usd"}
}

// assertUnitShape checks a value shape's unit, which is always the full unit.
func assertUnitShape(t *testing.T, unit map[string]any, unitID, label string) {
	t.Helper()
	require.NotNil(t, unit, "%s: a value shape always carries its full unit", label)
	assert.Equal(t, "unit", jsonField(unit, "object"), label)
	if unitID != "" {
		assert.Equal(t, unitID, jsonField(unit, "id"), label)
	} else {
		assert.NotEmpty(t, jsonField(unit, "id"), label)
	}
	assert.NotEmpty(t, jsonField(unit, "abbreviation"), label)
}

// assertNoValueIdentity checks a value shape carries none of the record fields values used to have.
func assertNoValueIdentity(t *testing.T, v map[string]any, label string) {
	t.Helper()
	for _, field := range []string{"id", "object", "display_value", "created_at", "updated_at"} {
		_, has := v[field]
		assert.False(t, has, "%s: a value shape has no %s", label, field)
	}
}

// assertPricePer checks a UnitPrice; an empty amount only requires one.
func assertPricePer(t *testing.T, price map[string]any, amount, perUnitID, label string) {
	t.Helper()
	require.NotNil(t, price, "%s should be set", label)
	if amount != "" {
		assert.Equal(t, amount, jsonField(price, "amount"), label)
	} else {
		assert.NotEmpty(t, jsonField(price, "amount"), label)
	}
	assert.Equal(t, "usd", jsonField(price, "currency"), label)
	assertUnitShape(t, jsonObject(price, "per_unit"), perUnitID, label+".per_unit")
	assertNoValueIdentity(t, price, label)
}

// assertMoneyUSD checks a Money; an empty amount only requires one.
func assertMoneyUSD(t *testing.T, money map[string]any, amount, label string) {
	t.Helper()
	require.NotNil(t, money, "%s should be set", label)
	if amount != "" {
		assert.Equal(t, amount, jsonField(money, "amount"), label)
	} else {
		assert.NotEmpty(t, jsonField(money, "amount"), label)
	}
	assert.Equal(t, "usd", jsonField(money, "currency"), label)
	assertNoValueIdentity(t, money, label)
}

// assertQuantityIn checks a Quantity; an empty value only requires one, an empty unitID any unit.
func assertQuantityIn(t *testing.T, q map[string]any, value, unitID, label string) {
	t.Helper()
	require.NotNil(t, q, "%s should be set", label)
	if value != "" {
		assertDecimalEqual(t, value, jsonField(q, "value"), label)
	} else {
		assert.NotEmpty(t, jsonField(q, "value"), label)
	}
	assertUnitShape(t, jsonObject(q, "unit"), unitID, label+".unit")
	assertNoValueIdentity(t, q, label)
}

// assertRateShape checks a Rate; empty ids accept any unit.
func assertRateShape(t *testing.T, r map[string]any, value, unitID, perUnitID, label string) {
	t.Helper()
	require.NotNil(t, r, "%s should be set", label)
	if value != "" {
		assertDecimalEqual(t, value, jsonField(r, "value"), label)
	} else {
		assert.NotEmpty(t, jsonField(r, "value"), label)
	}
	assertUnitShape(t, jsonObject(r, "unit"), unitID, label+".unit")
	assertUnitShape(t, jsonObject(r, "per_unit"), perUnitID, label+".per_unit")
	assertNoValueIdentity(t, r, label)
}

// --- preview.9 helpers ---

// materialP9Fields are the keys of a preview.9 material, and nothing more.
var materialP9Fields = []string{"id", "object", "item", "order_point", "lead_time", "created_at", "updated_at"}

// materialsP9 is a client pinned to preview.9, the only version the materials routes answer.
func materialsP9() *Client { return apiClient.WithAPIVersion(preview9Version) }

// createMaterialP9 creates a material on preview.9 (path may carry a query) and deletes it after the test.
func createMaterialP9(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	resp, err := materialsP9().PostFull(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	materialID := jsonField(created, "id")
	require.NotEmpty(t, materialID)
	assertCreatedLocation(t, resp.Header, materialID)
	t.Cleanup(func() { materialsP9().Delete(materialsPath + "/" + materialID) })
	return created
}

// getMaterialP9 reads a material on preview.9 with the given includes.
func getMaterialP9(t *testing.T, materialID string, includes ...string) map[string]any {
	t.Helper()
	var params url.Values
	if len(includes) > 0 {
		params = url.Values{"include": {strings.Join(includes, ",")}}
	}
	status, body, err := materialsP9().GetListRaw(materialsPath+"/"+materialID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// assertMaterialP9Envelope checks a preview.9 material carries exactly its preview.9 fields.
func assertMaterialP9Envelope(t *testing.T, got map[string]any) {
	t.Helper()
	assertIDFormat(t, jsonField(got, "id"), id.MaterialIDPrefix)
	assertObjectField(t, got, "material")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	assert.ElementsMatch(t, materialP9Fields, keys, "a preview.9 material has exactly its preview.9 fields")
}

// assertMaterialP9Quantity checks a preview.9 quantity: an object with a value, its unit and a display_value.
func assertMaterialP9Quantity(t *testing.T, q map[string]any, value, unitID, label string) {
	t.Helper()
	require.NotNil(t, q, "%s should be set", label)
	assert.Equal(t, "quantity", jsonField(q, "object"), label)
	assertDecimalEqual(t, value, jsonField(q, "value"), label)
	assert.Equal(t, unitID, jsonField(jsonObject(q, "unit"), "id"), label+".unit")
	assert.NotEmpty(t, jsonField(q, "display_value"), label)
}

// assertMaterialP9Rate checks a preview.9 rate. An empty numeratorUnitID is a dollar price, whose currency unit is
// null on preview.9.
func assertMaterialP9Rate(t *testing.T, r map[string]any, value, numeratorUnitID, denominatorUnitID, label string) {
	t.Helper()
	require.NotNil(t, r, "%s should be set", label)
	assert.Equal(t, "rate", jsonField(r, "object"), label)
	assertDecimalEqual(t, value, jsonField(r, "value"), label)
	if numeratorUnitID == "" {
		assert.Nil(t, r["numerator_unit"], "%s: a price's currency unit is null on preview.9", label)
	} else {
		assert.Equal(t, numeratorUnitID, jsonField(jsonObject(r, "numerator_unit"), "id"), label+".numerator_unit")
	}
	assert.Equal(t, denominatorUnitID, jsonField(jsonObject(r, "denominator_unit"), "id"), label+".denominator_unit")
	assert.NotEmpty(t, jsonField(r, "display_value"), label)
	for _, forgeKey := range []string{"unit", "per_unit", "amount", "currency"} {
		assert.NotContains(t, r, forgeKey, "%s: preview.9 rates have no %s", label, forgeKey)
	}
}

// --- lifecycle ---

// Create with every field, read with every include, update, list by filter, delete, and replay the delete. Notes
// are accepted but no longer stored.
func TestMaterials_Preview9Lifecycle(t *testing.T) {
	t.Parallel()
	p9 := materialsP9()
	attrID := newColorAttribute(t, "active")
	sku := uniqueName("e2e-mat-p9")

	created := createMaterialP9(t, materialsPath+"?include=item", map[string]any{
		"sku":           sku,
		"category_id":   SeedMaterialCategoryID,
		"description":   "Merino yarn, 2/28",
		"notes":         "not stored",
		"order_point":   map[string]any{"value": "10", "unit_id": SeedMaterialUnitID},
		"lead_time":     map[string]any{"value": "5", "unit_id": dayUnitID},
		"unit_price":    priceInputPer("4.00", SeedMaterialUnitID),
		"unit_cost":     priceInputPer("2.50", SeedMaterialUnitID),
		"attribute_ids": []string{attrID},
	})
	materialID := jsonField(created, "id")
	assertMaterialP9Envelope(t, created)
	assertMaterialP9Quantity(t, jsonObject(created, "order_point"), "10", SeedMaterialUnitID, "order_point")
	assertMaterialP9Quantity(t, jsonObject(created, "lead_time"), "5", dayUnitID, "lead_time")

	item := jsonObject(created, "item")
	require.NotNil(t, item, "item is expanded with ?include=item")
	assertPreview9ItemShape(t, item)
	itemID := jsonField(item, "id")
	assertIDFormat(t, itemID, id.ItemIDPrefix)
	assert.Equal(t, "material", jsonField(item, "type"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, "Merino yarn, 2/28", jsonField(item, "description"))
	assertMaterialP9Rate(t, jsonObject(item, "unit_value"), "4.00", "", SeedMaterialUnitID, "item.unit_value")
	assertMaterialP9Rate(t, jsonObject(item, "unit_cost"), "2.50", "", SeedMaterialUnitID, "item.unit_cost")
	assertMaterialP9Rate(t, jsonObject(item, "burn_rate"), "0", SeedMaterialUnitID, dayUnitID, "item.burn_rate")
	assertNilField(t, item, "category")
	assertNilField(t, item, "attributes")

	t.Run("retrieve with every include", func(t *testing.T) {
		got := getMaterialP9(t, materialID, "item", "item.category", "item.attributes")
		assertMaterialP9Envelope(t, got)
		gotItem := jsonObject(got, "item")
		require.NotNil(t, gotItem)
		assert.Equal(t, SeedMaterialCategoryID, jsonField(jsonObject(gotItem, "category"), "id"))
		assert.Equal(t, []string{attrID}, attributeIDsOf(gotItem))

		assertNilField(t, getMaterialP9(t, materialID), "item")
	})

	t.Run("forge reads the item it made", func(t *testing.T) {
		forgeItem := getItem(t, itemID)
		assertItemEnvelope(t, forgeItem, "material", sku)
		material := jsonObject(forgeItem, "material")
		require.NotNil(t, material)
		assertQuantityIn(t, jsonObject(material, "order_point"), "10", SeedMaterialUnitID, "material.order_point")
		assertItemPrice(t, jsonObject(forgeItem, "unit_cost"), "2.50", SeedMaterialUnitID, "unit_cost")
	})

	t.Run("update", func(t *testing.T) {
		newSKU := uniqueName("e2e-mat-p9-upd")
		status, body, err := p9.Patch(materialsPath+"/"+materialID+"?include=item", map[string]any{
			"sku":         newSKU,
			"description": "Merino yarn, 2/32",
			"notes":       "still not stored",
			"order_point": map[string]any{"value": "20", "unit_id": SeedMaterialUnitID},
			"lead_time":   map[string]any{"value": "8", "unit_id": dayUnitID},
			"unit_cost":   priceInputPer("3.00", SeedMaterialUnitID),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		updated := parseJSON(body)
		assertMaterialP9Envelope(t, updated)
		assert.Equal(t, materialID, jsonField(updated, "id"))
		assert.Equal(t, jsonField(created, "created_at"), jsonField(updated, "created_at"), "created_at never changes")
		assertMaterialP9Quantity(t, jsonObject(updated, "order_point"), "20", SeedMaterialUnitID, "order_point")
		assertMaterialP9Quantity(t, jsonObject(updated, "lead_time"), "8", dayUnitID, "lead_time")

		upItem := jsonObject(updated, "item")
		require.NotNil(t, upItem)
		assertPreview9ItemShape(t, upItem)
		assert.Equal(t, newSKU, jsonField(upItem, "sku"))
		assert.Equal(t, "Merino yarn, 2/32", jsonField(upItem, "description"))
		assertMaterialP9Rate(t, jsonObject(upItem, "unit_cost"), "3.00", "", SeedMaterialUnitID, "item.unit_cost")
		assertMaterialP9Rate(t, jsonObject(upItem, "unit_value"), "4.00", "", SeedMaterialUnitID, "an omitted unit_price is kept")
		assert.Equal(t, []string{attrID}, attributeIDsOf(jsonObject(getMaterialP9(t, materialID, "item.attributes"), "item")),
			"an update leaves the attributes alone")

		list, status, err := p9.GetList(materialsPath, url.Values{
			"q":             {newSKU},
			"category_ids":  {SeedMaterialCategoryID},
			"attribute_ids": {attrID},
			"starts_at":     {"2020-01-01T00:00:00Z"},
			"include":       {"item"},
		})
		require.NoError(t, err)
		require.Equal(t, 200, status)
		require.Len(t, list.Data, 1, "the filters select the material")
		row := parseJSON(list.Data[0])
		assert.Equal(t, materialID, jsonField(row, "id"))
		assert.Equal(t, newSKU, jsonField(jsonObject(row, "item"), "sku"))

		for name, params := range map[string]url.Values{
			"starts_at after it":  {"q": {newSKU}, "starts_at": {"2099-01-01T00:00:00Z"}},
			"ends_at before it":   {"q": {newSKU}, "ends_at": {"2020-01-01T00:00:00Z"}},
			"another attribute":   {"q": {newSKU}, "attribute_ids": {"at_" + strings.ReplaceAll(newIdempotencyKey(), "-", "")}},
			"an unknown category": {"q": {newSKU}, "category_ids": {"itcg_00000000000000000"}},
		} {
			list, status, err := p9.GetList(materialsPath, params)
			require.NoError(t, err)
			require.Equal(t, 200, status, name)
			assertEmptyListData(t, list.Data, name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		status, body, err := p9.Delete(materialsPath + "/" + materialID)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Equal(t, map[string]any{}, parseJSON(body), "a preview.9 delete answers an empty object")

		status, body, err = p9.Delete(materialsPath + "/" + materialID)
		require.NoError(t, err)
		requireStatus(t, 410, status, body)

		status, _, err = p9.GetListRaw(materialsPath+"/"+materialID, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status)
		status, _, err = apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "deleting a material deletes its item")
	})
}

// Omitted optional fields read back as null; a zero order point or lead time is no threshold, so it reads as null too.
func TestMaterials_Preview9OmittedFields(t *testing.T) {
	t.Parallel()

	created := createMaterialP9(t, materialsPath+"?include=item", validMaterialBody(uniqueName("e2e-mat-p9-omit")))
	assertMaterialP9Envelope(t, created)
	assertNilField(t, created, "order_point")
	assertNilField(t, created, "lead_time")
	item := jsonObject(created, "item")
	require.NotNil(t, item)
	assertPreview9ItemShape(t, item)
	assertNilField(t, item, "description")
	assertNilField(t, item, "unit_value")
	assertMaterialP9Rate(t, jsonObject(item, "burn_rate"), "0", SeedMaterialUnitID, dayUnitID, "item.burn_rate")

	body := validMaterialBody(uniqueName("e2e-mat-p9-zero"))
	body["order_point"] = map[string]any{"value": "0", "unit_id": SeedMaterialUnitID}
	body["lead_time"] = map[string]any{"value": "0", "unit_id": dayUnitID}
	zero := createMaterialP9(t, materialsPath, body)
	assertNilField(t, zero, "order_point")
	assertNilField(t, zero, "lead_time")
}

func TestMaterials_Preview9ListPagination(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-mat-pg")
	var ids []string
	for i := 0; i < 2; i++ {
		ids = append(ids, jsonField(createMaterialP9(t, materialsPath, validMaterialBody(fmt.Sprintf("%s-%d", prefix, i))), "id"))
	}
	assertScopedCursorPagination(t, materialsPath, url.Values{"q": {prefix}}, ids)

	list, _, err := materialsP9().GetList(materialsPath, url.Values{"q": {prefix}})
	require.NoError(t, err)
	require.Len(t, list.Data, 2)
	for _, raw := range list.Data {
		assertNilField(t, parseJSON(raw), "item")
	}
}

func TestMaterials_Preview9Idempotent(t *testing.T) {
	t.Parallel()
	p9 := materialsP9()
	body := validMaterialBody(uniqueName("e2e-mat-p9-idem"))
	key := newIdempotencyKey()

	s1, b1, err := p9.Post(materialsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s1, b1)
	materialID := jsonField(parseJSON(b1), "id")
	t.Cleanup(func() { p9.Delete(materialsPath + "/" + materialID) })
	s2, b2, err := p9.Post(materialsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s2, b2)
	assert.Equal(t, materialID, jsonField(parseJSON(b2), "id"), "a replayed create answers the first material")

	updateKey := newIdempotencyKey()
	update := map[string]any{"sku": uniqueName("e2e-mat-p9-idem-upd")}
	s1, b1, err = p9.Patch(materialsPath+"/"+materialID, update, updateKey)
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)
	s2, b2, err = p9.Patch(materialsPath+"/"+materialID, update, updateKey)
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)
	assert.JSONEq(t, string(b1), string(b2), "a replayed update answers the first response")
}

// --- validation ---

// preview.9 errors keep their preview.9 form: 400 for invalid input, the failing field's own code, no errors list.
func TestMaterials_Preview9Validation(t *testing.T) {
	t.Parallel()
	p9 := materialsP9()
	existingID := jsonField(createMaterialP9(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-p9-val"))), "id")

	with := func(set map[string]any) map[string]any {
		body := validMaterialBody(uniqueName("e2e-mat-p9-val"))
		for k, v := range set {
			body[k] = v
		}
		return body
	}
	longSKU := func() string {
		prefix := uniqueName("e2e-mat-max")
		return prefix + strings.Repeat("x", 256-len(prefix))
	}

	creates := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		param  string
	}{
		{"cost in another currency", with(map[string]any{"unit_cost": map[string]any{"amount": "0.50", "currency": "eur", "per_unit_id": nonCurrencyUnitID}}), 400, "currency_not_supported", "unit_cost.currency"},
		{"price per a currency", with(map[string]any{"unit_price": priceInputPer("1.00", currencyUnitID)}), 400, "invalid_format", "unit_price.per_unit_id"},
		{"product category", with(map[string]any{"category_id": SeedItemCategoryID}), 400, "invalid_format", "category_id"},
		{"unknown category", with(map[string]any{"category_id": "itcg_00000000000000000"}), 400, "invalid_format", "category_id"},
		{"sku over 255", with(map[string]any{"sku": longSKU()}), 400, "invalid_format", "sku"},
		{"missing sku", map[string]any{"category_id": SeedMaterialCategoryID}, 400, "missing_field", "sku"},
		{"sku taken", with(map[string]any{"sku": covCatalogMaterialsSeedYarnSKU}), 409, "resource_conflict", "sku"},
	}
	for _, tc := range creates {
		t.Run("create/"+tc.name, func(t *testing.T) {
			status, body, err := p9.Post(materialsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, tc.status, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, tc.code), tc.param)
		})
	}

	t.Run("create/unknown attribute", func(t *testing.T) {
		status, body, err := p9.Post(materialsPath, with(map[string]any{"attribute_ids": []string{"at_00000000000000000"}}), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
		requirePreview9Error(t, body, "resource_not_found")
	})

	updates := []struct {
		name   string
		body   map[string]any
		status int
		code   string
		param  string
	}{
		{"null sku", map[string]any{"sku": nil}, 400, "invalid_format", "sku"},
		{"null description", map[string]any{"description": nil}, 400, "invalid_format", "description"},
		{"cost in another currency", map[string]any{"unit_cost": map[string]any{"amount": "0.50", "currency": "eur", "per_unit_id": nonCurrencyUnitID}}, 400, "currency_not_supported", "unit_cost.currency"},
		{"sku taken", map[string]any{"sku": covCatalogMaterialsSeedYarnSKU}, 409, "resource_conflict", "sku"},
	}
	for _, tc := range updates {
		t.Run("update/"+tc.name, func(t *testing.T) {
			status, body, err := p9.Patch(materialsPath+"/"+existingID, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, tc.status, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, tc.code), tc.param)
		})
	}

	queries := []struct {
		name   string
		path   string
		params url.Values
		param  string
	}{
		{"unknown include", materialsPath + "/" + SeedMaterialID, url.Values{"include": {"bogus_field"}}, "include[]"},
		{"malformed starts_at", materialsPath, url.Values{"starts_at": {"not-a-date"}}, "starts_at"},
		{"malformed ends_at", materialsPath, url.Values{"ends_at": {"not-a-date"}}, "ends_at"},
	}
	for _, tc := range queries {
		t.Run("query/"+tc.name, func(t *testing.T) {
			status, body, err := p9.GetListRaw(tc.path, tc.params)
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, "parameter_invalid"), tc.param)
		})
	}

	t.Run("delete/unknown material", func(t *testing.T) {
		status, body, err := p9.Delete(materialsPath + "/" + mustGenID(t, id.MaterialIDPrefix))
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})
}

// covCatalogMaterialsSeedYarnSKU is the SKU of SeedMaterialID's underlying item, used as a stable conflict target for
// duplicate-SKU tests.
const covCatalogMaterialsSeedYarnSKU = "YRN-001"
