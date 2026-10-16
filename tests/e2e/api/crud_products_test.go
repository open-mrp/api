//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The /v1/catalog/products routes answer only 1.0.forge-preview.9 callers; on forge a product is an item with a sales
// section (crud_items_test.go). These tests pin what preview.9 callers still get from them.

// createPreview9Product creates a product through the preview.9 route (path may carry a query) and deletes it after the
// test.
func createPreview9Product(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	preview := apiClient.WithAPIVersion(preview9Version)
	status, respBody, err := preview.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	productID := jsonField(created, "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { preview.Delete(productsPath + "/" + productID) })
	return created
}

func getPreview9Product(t *testing.T, productID string, includes ...string) map[string]any {
	t.Helper()
	var params url.Values
	if len(includes) > 0 {
		params = url.Values{"include": includes}
	}
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(productsPath+"/"+productID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// assertProductsPreview9Price checks a price as preview.9 presents it: a rate in an unincluded currency per a unit.
func assertProductsPreview9Price(t *testing.T, rate map[string]any, value, perUnitID, label string) {
	t.Helper()
	require.NotNil(t, rate, "%s is present", label)
	assert.Equal(t, "rate", jsonField(rate, "object"), label)
	assertDecimalEqual(t, value, jsonField(rate, "value"), label+".value")
	assert.Nil(t, rate["numerator_unit"], "%s: the currency unit is not included", label)
	assert.Equal(t, perUnitID, jsonField(jsonObject(rate, "denominator_unit"), "id"), label+".denominator_unit")
	assert.NotEmpty(t, jsonField(rate, "display_value"), label+".display_value")
	for _, forgeKey := range []string{"amount", "currency", "per_unit"} {
		assert.NotContains(t, rate, forgeKey, "%s is a preview.9 rate", label)
	}
}

// ──────────────────────────────────────────────
// Includes
// ──────────────────────────────────────────────

func TestProductsPreview9_Includes(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	t.Run("expandables null without include", func(t *testing.T) {
		got := getPreview9Product(t, SeedProductID)
		assertNilField(t, got, "product_line")
		assertNilField(t, got, "item")

		list, _, err := preview.GetList(productsPath, nil)
		require.NoError(t, err)
		require.NotEmpty(t, list.Data)
		for _, raw := range list.Data {
			m := parseJSON(raw)
			assertNilField(t, m, "product_line")
			assertNilField(t, m, "item")
		}
	})

	t.Run("retrieve", func(t *testing.T) {
		got := getPreview9Product(t, SeedProductID, "product_line.unit_group", "item.category", "item.attributes")
		assertObjectField(t, got, "product")

		pl := jsonObject(got, "product_line")
		require.NotNil(t, pl)
		assertObjectField(t, pl, "product_line")
		ug := jsonObject(pl, "unit_group")
		require.NotNil(t, ug, "product_line.unit_group")
		assertObjectField(t, ug, "unit_group")

		item := jsonObject(got, "item")
		require.NotNil(t, item, "item.category reaches through item")
		assertPreview9ItemShape(t, item)
		assertObjectField(t, jsonObject(item, "category"), "item_category")
		assert.NotEmpty(t, jsonListData(item, "attributes"), "the seeded item has attributes")

		// An item's prices and burn rate are always present, as preview.9 rates.
		for _, key := range []string{"unit_cost", "burn_rate"} {
			rate := jsonObject(item, key)
			require.NotNil(t, rate, key)
			assert.Equal(t, "rate", jsonField(rate, "object"), key)
			assert.NotEmpty(t, jsonField(rate, "value"), key)
			assert.NotNil(t, jsonObject(rate, "denominator_unit"), key)
		}
		assert.NotNil(t, jsonObject(jsonObject(item, "burn_rate"), "numerator_unit"), "a burn rate counts its item units")
		assert.Equal(t, "rate", jsonField(jsonObject(item, "unit_value"), "object"), "a product is sold, so it has a unit_value")
	})

	t.Run("list", func(t *testing.T) {
		// Seeded products only: products other tests create and delete meanwhile can lose their item mid-read.
		seeded, err := time.Parse(time.RFC3339, jsonField(getPreview9Product(t, SeedProductID), "created_at"))
		require.NoError(t, err)
		status, body, err := preview.GetListRaw(productsPath, url.Values{
			"include":          {"item", "product_line.unit_group"},
			"product_line_ids": {SeedProductLineID},
			"ends_at":          {seeded.Add(time.Minute).Format(time.RFC3339)},
			"limit":            {"5"},
		})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		data := jsonArray(parseJSON(body), "data")
		require.NotEmpty(t, data)
		for _, raw := range data {
			row := raw.(map[string]any)
			assertPreview9ItemShape(t, jsonObject(row, "item"))
			pl := jsonObject(row, "product_line")
			require.NotNil(t, pl)
			assert.Equal(t, SeedProductLineID, jsonField(pl, "id"))
			assertObjectField(t, jsonObject(pl, "unit_group"), "unit_group")
		}
	})

	// The item's value fields are no longer includes of their own; asking for them leaves the item unexpanded rather
	// than failing the request.
	t.Run("value includes are accepted", func(t *testing.T) {
		got := getPreview9Product(t, SeedProductID, "item.unit_value", "item.unit_cost", "item.burn_rate")
		assert.Equal(t, SeedProductID, jsonField(got, "id"))
	})
}

// ──────────────────────────────────────────────
// Validate (bulk SKU resolution)
// ──────────────────────────────────────────────

const productsValidatePath = "/v1/catalog/products/actions/validate"

var productValidateIncludes = []string{
	"product_line",
	"product_line.unit_group",
	"product_line.unit_group.base_unit",
	"product_line.unit_group.associated_units",
	"product_line.unit_group.associated_units.unit",
	"item",
	"item.category",
	"item.category.unit_group",
	"item.category.unit_group.base_unit",
	"item.category.unit_group.associated_units",
	"item.category.unit_group.associated_units.unit",
	"item.attributes",
}

func productValidateIncludeQueryValues() url.Values {
	v := url.Values{}
	for _, inc := range productValidateIncludes {
		v.Add("include", inc)
	}
	return v
}

func validatedProduct(m map[string]any, rowKey string) map[string]any {
	products, ok := m["products"].(map[string]any)
	if !ok {
		return nil
	}
	p, _ := products[rowKey].(map[string]any)
	return p
}

func associatedUnitFingerprintsFromUnitGroup(ug map[string]any) []string {
	if ug == nil {
		return nil
	}
	list := jsonObject(ug, "associated_units")
	if list == nil {
		return nil
	}
	data, ok := list["data"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(data))
	for _, raw := range data {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		unitID := ""
		if ut := jsonObject(row, "unit"); ut != nil {
			unitID = jsonField(ut, "id")
		}
		out = append(out, fmt.Sprintf("%s|%s|%s|%s",
			jsonField(row, "id"),
			jsonField(row, "discount_percentage"),
			jsonField(row, "discount_fixed"),
			unitID,
		))
	}
	sort.Strings(out)
	return out
}

func categoryAssociatedUnitFingerprints(product map[string]any) []string {
	cat := jsonObject(jsonObject(product, "item"), "category")
	return associatedUnitFingerprintsFromUnitGroup(jsonObject(cat, "unit_group"))
}

func productLineAssociatedUnitFingerprints(product map[string]any) []string {
	return associatedUnitFingerprintsFromUnitGroup(jsonObject(jsonObject(product, "product_line"), "unit_group"))
}

func validatePreview9Products(t *testing.T, params url.Values, skus map[string]string) map[string]any {
	t.Helper()
	status, body, err := apiClient.WithAPIVersion(preview9Version).PutRaw(productsValidatePath, params, map[string]any{"products_map": skus})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	require.Equal(t, "map", jsonField(got, "object"))
	return got
}

func TestProductsPreview9_Validate_KnownSKU(t *testing.T) {
	t.Parallel()
	p := validatedProduct(validatePreview9Products(t, url.Values{"include": {"item"}}, map[string]string{"0": "SCK-001"}), "0")
	require.NotNil(t, p, "products[0] should be present")
	assertObjectField(t, p, "product")
	item := jsonObject(p, "item")
	assertPreview9ItemShape(t, item)
	assert.Equal(t, "SCK-001", jsonField(item, "sku"))
}

func TestProductsPreview9_Validate_IncludesUnitGroupAssociatedUnits(t *testing.T) {
	t.Parallel()
	p := validatedProduct(validatePreview9Products(t, productValidateIncludeQueryValues(), map[string]string{"0": "SCK-001"}), "0")
	require.NotNil(t, p)
	assertIncludePopulated(t, p, "item.category.unit_group.associated_units")

	ug := jsonObject(jsonObject(p, "product_line"), "unit_group")
	require.NotNil(t, ug, "product_line.unit_group should be present")
	assoc := jsonObject(ug, "associated_units")
	require.NotNil(t, assoc, "associated_units should be present")
	assert.Equal(t, "list", jsonField(assoc, "object"))
	assert.NotEmpty(t, jsonListData(ug, "associated_units"), "seed Socks product line unit group should have conversions")
}

func TestProductsPreview9_Validate_ParityWithGet(t *testing.T) {
	t.Parallel()
	q := productValidateIncludeQueryValues()
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(productsPath+"/"+SeedProductID, q)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	reference := parseJSON(body)
	sku := jsonField(jsonObject(reference, "item"), "sku")
	require.NotEmpty(t, sku)

	p := validatedProduct(validatePreview9Products(t, q, map[string]string{"0": sku}), "0")
	require.NotNil(t, p)
	assert.Equal(t, categoryAssociatedUnitFingerprints(reference), categoryAssociatedUnitFingerprints(p),
		"category unit_group associated_units should match GET for the same SKU and includes")
	if jsonObject(reference, "product_line") != nil {
		assert.Equal(t, productLineAssociatedUnitFingerprints(reference), productLineAssociatedUnitFingerprints(p),
			"product_line unit_group associated_units should match GET for the same SKU and includes")
	}
}

// ──────────────────────────────────────────────
// CRUD & lifecycle
// ──────────────────────────────────────────────

// Create with every field → retrieve → update → list → delete → replayed delete, all as a preview.9 caller. Notes are
// accepted but no longer stored, so they read back null.
func TestProductsPreview9_Lifecycle(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	sku := uniqueName("e2e-p9-prod-life")
	created := createPreview9Product(t, productsPath, map[string]any{
		"sku":               sku,
		"type":              "sale",
		"category_id":       SeedItemCategoryID,
		"product_line_id":   SeedProductLineID,
		"portal_visibility": "visible",
		"description":       "lifecycle description",
		"notes":             "lifecycle notes",
		"attribute_ids":     []string{SeedAttributeID},
		"unit_price":        priceInputPer("12.34", nonCurrencyUnitID),
		"unit_cost":         priceInputPer("4.50", nonCurrencyUnitID),
	})
	productID := jsonField(created, "id")
	assertIDFormat(t, productID, "pd")
	assertObjectField(t, created, "product")
	assert.Equal(t, "sale", jsonField(created, "type"))
	assert.Equal(t, "visible", jsonField(created, "portal_visibility"))
	assertNilField(t, created, "item")
	assertNilField(t, created, "product_line")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	createdAt := jsonField(created, "created_at")

	got := getPreview9Product(t, productID, "item", "item.attributes", "product_line")
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(got, "product_line"), "id"))
	item := jsonObject(got, "item")
	assertPreview9ItemShape(t, item)
	itemID := jsonField(item, "id")
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, "product", jsonField(item, "type"))
	assert.Equal(t, "lifecycle description", jsonField(item, "description"))
	assert.Equal(t, []string{SeedAttributeID}, attributeIDsOf(item))
	assertProductsPreview9Price(t, jsonObject(item, "unit_value"), "12.34", nonCurrencyUnitID, "item.unit_value")
	assertProductsPreview9Price(t, jsonObject(item, "unit_cost"), "4.50", nonCurrencyUnitID, "item.unit_cost")

	// forge reads the same item, with the price in its sales section.
	forgeItem := getItem(t, itemID, "sales.product_line")
	assertItemEnvelope(t, forgeItem, "product", sku)
	sales := jsonObject(forgeItem, "sales")
	assertItemPrice(t, jsonObject(sales, "unit_price"), "12.34", nonCurrencyUnitID, "sales.unit_price")
	assert.Equal(t, "visible", jsonField(sales, "portal_visibility"))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(sales, "product_line"), "id"))

	newSKU := uniqueName("e2e-p9-prod-life-u")
	status, body, err := preview.Patch(productsPath+"/"+productID+"?include=item", map[string]any{
		"sku":               newSKU,
		"description":       "updated description",
		"notes":             "updated notes",
		"portal_visibility": "hidden",
		"unit_price":        priceInputPer("99.01", nonCurrencyUnitID),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assert.Equal(t, productID, jsonField(updated, "id"))
	assert.Equal(t, "hidden", jsonField(updated, "portal_visibility"))
	assert.Equal(t, createdAt, jsonField(updated, "created_at"), "created_at must not change")
	item = jsonObject(updated, "item")
	assertPreview9ItemShape(t, item)
	assert.Equal(t, newSKU, jsonField(item, "sku"))
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assertProductsPreview9Price(t, jsonObject(item, "unit_value"), "99.01", nonCurrencyUnitID, "item.unit_value")

	// An update leaves what it omits alone.
	status, body, err = preview.Patch(productsPath+"/"+productID, map[string]any{"portal_visibility": "visible"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	item = jsonObject(getPreview9Product(t, productID, "item"), "item")
	assert.Equal(t, newSKU, jsonField(item, "sku"))
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assertProductsPreview9Price(t, jsonObject(item, "unit_value"), "99.01", nonCurrencyUnitID, "item.unit_value")

	list, _, err := preview.GetList(productsPath, url.Values{
		"portal_visibility": {"visible"},
		"category_ids":      {SeedItemCategoryID},
		"attribute_ids":     {SeedAttributeID},
		"limit":             {"100"},
	})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(list, productID), "the filtered list finds the product")

	status, body, err = preview.Delete(productsPath + "/" + productID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, map[string]any{}, parseJSON(body), "preview.9 answers a delete with an empty object")

	status, body, err = preview.Delete(productsPath + "/" + productID)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)

	status, _, err = preview.GetListRaw(productsPath+"/"+productID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "GET after delete")
	status, _, err = apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "deleting a product deletes its item")
}

// A product on a sales order line can't be deleted.
func TestProductsPreview9_DeleteInUseConflicts(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	productID, itemID := newSellableProductIDs(t, "e2e-p9-prod-inuse")
	orderID := createLifecycleOrder(t)
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  productID,
		"product_sku": uniqueName("E2E-LINE-SKU"),
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	status, body, err = preview.Delete(productsPath + "/" + productID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requirePreview9Error(t, body, "resource_conflict")

	assert.Equal(t, productID, jsonField(getPreview9Product(t, productID), "id"), "the product survives a refused delete")
	assert.Equal(t, itemID, jsonField(getItem(t, itemID), "id"), "so does its item")
}

func TestProductsPreview9_CreateIdempotent(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	sku := uniqueName("e2e-p9-prod-idem")
	key := newIdempotencyKey()

	status1, body1, err := preview.Post(productsPath, validProductBody(sku), key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	t.Cleanup(func() { preview.Delete(productsPath + "/" + id1) })

	status2, body2, err := preview.Post(productsPath, validProductBody(sku), key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))
}

func TestProductsPreview9_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	productID := jsonField(createPreview9Product(t, productsPath, validProductBody(uniqueName("e2e-p9-prod-idem-upd"))), "id")
	newSKU := uniqueName("e2e-p9-prod-idem-upd2")
	key := newIdempotencyKey()
	path := productsPath + "/" + productID + "?include=item"

	status1, body1, err := preview.Patch(path, map[string]any{"sku": newSKU}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	status2, body2, err := preview.Patch(path, map[string]any{"sku": newSKU}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	for _, body := range [][]byte{body1, body2} {
		got := parseJSON(body)
		assert.Equal(t, productID, jsonField(got, "id"))
		assert.Equal(t, newSKU, jsonField(jsonObject(got, "item"), "sku"))
	}
}

// The change-product-line action is preview.9's; forge moves a product with PATCH sales.product_line_id.
func TestProductsPreview9_ChangeProductLine(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	body := validProductBody(uniqueName("e2e-p9-prod-chgpl"))
	body["product_line_id"] = SeedProductLineID
	productID := jsonField(createPreview9Product(t, productsPath, body), "id")

	// SeedIncludePutProductLineChangeTargetID has a live unit group, so the product_line include resolves.
	target := SeedIncludePutProductLineChangeTargetID
	route := productsPath + "/" + productID + "/product-line/" + target
	for i := 0; i < 2; i++ {
		status, respBody, err := preview.Put(route, map[string]any{})
		require.NoError(t, err)
		requireStatus(t, 200, status, respBody)
	}
	assert.Equal(t, target, jsonField(jsonObject(getPreview9Product(t, productID, "product_line"), "product_line"), "id"),
		"repeat change-product-line calls are idempotent")

	t.Run("unknown product", func(t *testing.T) {
		status, respBody, err := preview.Put(productsPath+"/"+covCatalogProductsNonexistentID+"/product-line/"+SeedProductLineID, map[string]any{})
		require.NoError(t, err)
		requireStatus(t, 404, status, respBody)
	})

	t.Run("unknown product line", func(t *testing.T) {
		status, respBody, err := preview.Put(productsPath+"/"+productID+"/product-line/pdln_00000000000000000000000000", map[string]any{})
		require.NoError(t, err)
		assert.Contains(t, []int{400, 404}, status, "a nonexistent product line is refused: %s", respBody)
	})
}
