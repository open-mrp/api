//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dayUnitID is the system time unit lead times and burn rates are counted in.
const dayUnitID = "day"

// newProductItemIDs creates a product and returns its id and the linked item id.
// Registers cleanup to delete the product.
func newProductItemIDs(t *testing.T, namePrefix string) (productID, itemID string) {
	t.Helper()
	sku := uniqueName(namePrefix)
	resp, err := apiClient.PostFull(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID = jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID = jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)
	return productID, itemID
}

// newSellableProductIDs creates a product in the seeded line, which order customers can buy, and returns its product
// and item ids. Sales order lines still name products, so the product comes from the preview.9 route.
func newSellableProductIDs(t *testing.T, namePrefix string) (productID, itemID string) {
	t.Helper()
	body := validProductBody(uniqueName(namePrefix))
	body["product_line_id"] = SeedProductLineID
	status, respBody, err := apiClient.Post(productsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	productID = jsonField(created, "id")
	itemID = jsonField(jsonObject(created, "item"), "id")
	require.NotEmpty(t, itemID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })
	return productID, itemID
}

// forgeItemBody is the smallest valid create body for an item of the type, in a category of the right kind.
func forgeItemBody(itemType, sku string) map[string]any {
	categoryID := SeedItemCategoryID
	if itemType == "material" {
		categoryID = SeedMaterialCategoryID
	}
	return map[string]any{"type": itemType, "sku": sku, "category_id": categoryID}
}

// createForgeItem creates an item (path may carry a query) and deletes it after the test.
func createForgeItem(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	resp, err := apiClient.PostFull(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	itemID := jsonField(created, "id")
	require.NotEmpty(t, itemID)
	assertCreatedLocation(t, resp.Header, itemID)
	t.Cleanup(func() { apiClient.Delete(itemsPath + "/" + itemID) })
	return created
}

// newColorAttribute creates an attribute in the seeded Color property, which both Socks and Yarn carry.
func newColorAttribute(t *testing.T, status string) string {
	t.Helper()
	created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{
		"value":  uniqueName("e2e-itm-attr"),
		"status": status,
	})
	return jsonField(created, "id")
}

// newArchivedProductLine creates a product line and archives it.
func newArchivedProductLine(t *testing.T) string {
	t.Helper()
	lineID := jsonField(createAndCleanup(t, productLinesPath, map[string]any{
		"name": uniqueName("e2e-itm-arch-line"), "commission_policy": "applied", "freight_policy": "billed",
	}), "id")
	status, body, err := apiClient.Patch(productLinesPath+"/"+lineID, map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return lineID
}

func getItem(t *testing.T, itemID string, includes ...string) map[string]any {
	t.Helper()
	var params url.Values
	if len(includes) > 0 {
		params = url.Values{"include": {strings.Join(includes, ",")}}
	}
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func patchItem(t *testing.T, itemID string, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := apiClient.Patch(itemsPath+"/"+itemID, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

// assertItemPrice checks a UnitPrice by value, so "0" matches however the amount is formatted.
func assertItemPrice(t *testing.T, price map[string]any, amount, perUnitID, label string) {
	t.Helper()
	require.NotNil(t, price, "%s is always present", label)
	assertDecimalEqual(t, amount, jsonField(price, "amount"), label)
	assert.Equal(t, "usd", jsonField(price, "currency"), label)
	assertUnitShape(t, jsonObject(price, "per_unit"), perUnitID, label+".per_unit")
	assertNoValueIdentity(t, price, label)
}

// assertItemEnvelope checks the fields every item carries whatever its type.
func assertItemEnvelope(t *testing.T, got map[string]any, itemType, sku string) {
	t.Helper()
	assertIDFormat(t, jsonField(got, "id"), id.ItemIDPrefix)
	assertObjectField(t, got, "item")
	assert.Equal(t, itemType, jsonField(got, "type"))
	assert.Equal(t, sku, jsonField(got, "sku"))
	assertNilField(t, got, "default_manufacturing_method")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	for _, gone := range []string{"notes", "unit_value"} {
		assert.NotContains(t, got, gone, "forge items have no %s", gone)
	}
}

// attributeIDsOf reads the ids in an item's attributes list.
func attributeIDsOf(item map[string]any) []string {
	var out []string
	for _, raw := range jsonListData(item, "attributes") {
		if m, ok := raw.(map[string]any); ok {
			out = append(out, jsonField(m, "id"))
		}
	}
	return out
}

// --- CRUD ---

func TestItems_CRUD(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-item-crud")
	resp, err := apiClient.PostFull(itemsPath, forgeItemBody("product", sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	itemID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, itemID)
	deleted := false
	t.Cleanup(func() {
		if !deleted {
			apiClient.Delete(itemsPath + "/" + itemID)
		}
	})

	got := getItem(t, itemID)
	assert.Equal(t, sku, jsonField(got, "sku"))

	newSKU := uniqueName("e2e-item-crud-upd")
	updated := patchItem(t, itemID, map[string]any{"sku": newSKU, "description": "crud description"})
	assert.Equal(t, itemID, jsonField(updated, "id"))
	assert.Equal(t, newSKU, jsonField(updated, "sku"))
	assert.Equal(t, "crud description", jsonField(getItem(t, itemID), "description"))

	status, body, err := apiClient.Delete(itemsPath + "/" + itemID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	deleted = true

	status, _, err = apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a deleted item is gone")
}

func TestItems_RetrieveSeededProduct(t *testing.T) {
	t.Parallel()
	got := getItem(t, SeedItemID)
	assert.Equal(t, SeedItemID, jsonField(got, "id"))
	assertItemEnvelope(t, got, "product", SeedItemSKU)
	assert.Equal(t, SeedItemDescription, jsonField(got, "description"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assertNilField(t, got, "category")
	assertNilField(t, got, "attributes")
	assertNilField(t, got, "material")
	sales := jsonObject(got, "sales")
	require.NotNil(t, sales, "a product always has a sales section")
	assertNilField(t, sales, "product_line")
	assert.Contains(t, []string{"visible", "hidden"}, jsonField(sales, "portal_visibility"))
	require.NotNil(t, jsonObject(got, "unit_cost"))
	assertRateShape(t, jsonObject(got, "burn_rate"), "", "", dayUnitID, "burn_rate")
}

func TestItems_GetNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+mustGenID(t, id.ItemIDPrefix), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// --- Create: every type, every field ---

func TestItems_CreateMaterialAllFields(t *testing.T) {
	t.Parallel()
	attrID := newColorAttribute(t, "active")
	sku := uniqueName("e2e-item-mat-all")
	body := forgeItemBody("material", sku)
	body["description"] = "Merino yarn, 2/28"
	body["attribute_ids"] = []string{attrID}
	body["status"] = "active"
	body["supply"] = "buy"
	body["unit_cost"] = priceInputPer("3.25", SeedMaterialUnitID)
	body["material"] = map[string]any{
		"order_point": map[string]any{"value": "40", "unit_id": SeedMaterialUnitID},
		"lead_time":   map[string]any{"value": "5", "unit_id": dayUnitID},
	}

	got := createForgeItem(t, itemsPath+"?include=category,attributes", body)
	assertItemEnvelope(t, got, "material", sku)
	assert.Equal(t, "Merino yarn, 2/28", jsonField(got, "description"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, "buy", jsonField(got, "supply"))
	cat := jsonObject(got, "category")
	require.NotNil(t, cat)
	assert.Equal(t, SeedMaterialCategoryID, jsonField(cat, "id"))
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.Equal(t, []string{attrID}, attributeIDsOf(got))
	assertItemPrice(t, jsonObject(got, "unit_cost"), "3.25", SeedMaterialUnitID, "unit_cost")
	assertRateShape(t, jsonObject(got, "burn_rate"), "0", SeedMaterialUnitID, dayUnitID, "burn_rate")
	assertNilField(t, got, "sales")
	material := jsonObject(got, "material")
	require.NotNil(t, material, "a material has a material section")
	assertQuantityIn(t, jsonObject(material, "order_point"), "40", SeedMaterialUnitID, "material.order_point")
	assertQuantityIn(t, jsonObject(material, "lead_time"), "5", dayUnitID, "material.lead_time")

	// The read matches the create response.
	read := getItem(t, jsonField(got, "id"), "category", "attributes")
	assert.Equal(t, got, read)
}

// A part you sell carries the sales section it was created with.
func TestItems_CreatePartWithSales(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-item-part-sales")
	body := forgeItemBody("part", sku)
	body["supply"] = "make"
	body["sales"] = map[string]any{
		"unit_price":        priceInputPer("7.50", SeedUnitID),
		"product_line_id":   SeedProductLineID,
		"portal_visibility": "visible",
		"lead_time":         map[string]any{"value": "3", "unit_id": dayUnitID},
	}

	got := createForgeItem(t, itemsPath+"?include=sales.product_line", body)
	assertItemEnvelope(t, got, "part", sku)
	assertNilField(t, got, "description")
	assert.Equal(t, "make", jsonField(got, "supply"))
	assertNilField(t, got, "material")
	sales := jsonObject(got, "sales")
	require.NotNil(t, sales)
	assertItemPrice(t, jsonObject(sales, "unit_price"), "7.50", SeedUnitID, "sales.unit_price")
	line := jsonObject(sales, "product_line")
	require.NotNil(t, line, "sales.product_line with include")
	assert.Equal(t, SeedProductLineID, jsonField(line, "id"))
	assert.Equal(t, "product_line", jsonField(line, "object"))
	assert.Equal(t, "visible", jsonField(sales, "portal_visibility"))
	assertQuantityIn(t, jsonObject(sales, "lead_time"), "3", dayUnitID, "sales.lead_time")
}

func TestItems_CreateProductAllFields(t *testing.T) {
	t.Parallel()
	attrID := newColorAttribute(t, "active")
	sku := uniqueName("e2e-item-prod-all")
	body := forgeItemBody("product", sku)
	body["description"] = "Crew sock"
	body["attribute_ids"] = []string{attrID}
	body["status"] = "archived"
	body["supply"] = "stock"
	body["unit_cost"] = priceInputPer("1.10", SeedUnitID)
	body["sales"] = map[string]any{
		"unit_price":        priceInputPer("4.99", SeedUnitID),
		"product_line_id":   SeedProductLineID,
		"portal_visibility": "visible",
		"lead_time":         map[string]any{"value": "10", "unit_id": dayUnitID},
	}

	got := createForgeItem(t, itemsPath+"?include=category,attributes,sales.product_line", body)
	assertItemEnvelope(t, got, "product", sku)
	assert.Equal(t, "Crew sock", jsonField(got, "description"))
	assert.Equal(t, "archived", jsonField(got, "status"))
	assert.Equal(t, "stock", jsonField(got, "supply"))
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(got, "category"), "id"))
	assert.Equal(t, []string{attrID}, attributeIDsOf(got))
	assertItemPrice(t, jsonObject(got, "unit_cost"), "1.10", SeedUnitID, "unit_cost")
	assertRateShape(t, jsonObject(got, "burn_rate"), "0", SeedUnitID, dayUnitID, "burn_rate")
	assertNilField(t, got, "material")
	sales := jsonObject(got, "sales")
	require.NotNil(t, sales)
	assertItemPrice(t, jsonObject(sales, "unit_price"), "4.99", SeedUnitID, "sales.unit_price")
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(sales, "product_line"), "id"))
	assert.Equal(t, "visible", jsonField(sales, "portal_visibility"))
	assertQuantityIn(t, jsonObject(sales, "lead_time"), "10", dayUnitID, "sales.lead_time")
}

// A service is sold, so it always has a sales section, which defaults to hidden, unlined and free.
func TestItems_CreateServiceDefaults(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-item-svc")
	got := createForgeItem(t, itemsPath, forgeItemBody("service", sku))
	assertItemEnvelope(t, got, "service", sku)
	assert.Equal(t, "active", jsonField(got, "status"))
	assertNilField(t, got, "description")
	assertNilField(t, got, "supply")
	assertNilField(t, got, "category")
	assertNilField(t, got, "attributes")
	assertNilField(t, got, "material")
	assertItemPrice(t, jsonObject(got, "unit_cost"), "0", SeedUnitID, "unit_cost")
	sales := jsonObject(got, "sales")
	require.NotNil(t, sales, "a service always has a sales section")
	assertItemPrice(t, jsonObject(sales, "unit_price"), "0", SeedUnitID, "sales.unit_price")
	assertNilField(t, sales, "product_line")
	assert.Equal(t, "hidden", jsonField(sales, "portal_visibility"))
	assertNilField(t, sales, "lead_time")
}

// --- Omitted Fields ---

func TestItems_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("material without a material section has an empty one", func(t *testing.T) {
		got := createForgeItem(t, itemsPath, forgeItemBody("material", uniqueName("e2e-item-omit-mat")))
		assert.Equal(t, "active", jsonField(got, "status"))
		assertNilField(t, got, "description")
		assertNilField(t, got, "supply")
		assertNilField(t, got, "sales")
		assertItemPrice(t, jsonObject(got, "unit_cost"), "0", SeedMaterialUnitID, "unit_cost")
		material := jsonObject(got, "material")
		require.NotNil(t, material)
		assertNilField(t, material, "order_point")
		assertNilField(t, material, "lead_time")
	})

	t.Run("part has neither section", func(t *testing.T) {
		got := createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-item-omit-part")))
		assertNilField(t, got, "sales")
		assertNilField(t, got, "material")
	})

	t.Run("product gets a default sales section", func(t *testing.T) {
		got := createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-item-omit-prod")))
		sales := jsonObject(got, "sales")
		require.NotNil(t, sales)
		assert.Equal(t, "hidden", jsonField(sales, "portal_visibility"))
		assertNilField(t, sales, "product_line")
		assertNilField(t, sales, "lead_time")
		assertItemPrice(t, jsonObject(sales, "unit_price"), "0", SeedUnitID, "sales.unit_price")
	})

	t.Run("PATCH of one field keeps the rest", func(t *testing.T) {
		body := forgeItemBody("product", uniqueName("e2e-item-omit-patch"))
		body["description"] = "keep me"
		body["supply"] = "make"
		body["sales"] = map[string]any{"portal_visibility": "visible", "product_line_id": SeedProductLineID}
		created := createForgeItem(t, itemsPath, body)
		itemID := jsonField(created, "id")

		newSKU := uniqueName("e2e-item-omit-patch-sku")
		got := patchItem(t, itemID, map[string]any{"sku": newSKU})
		assert.Equal(t, newSKU, jsonField(got, "sku"))
		assert.Equal(t, "keep me", jsonField(got, "description"))
		assert.Equal(t, "make", jsonField(got, "supply"))
		assert.Equal(t, "active", jsonField(got, "status"))
		assert.Equal(t, jsonField(created, "created_at"), jsonField(got, "created_at"))
		sales := jsonObject(got, "sales")
		assert.Equal(t, "visible", jsonField(sales, "portal_visibility"))

		withLine := getItem(t, itemID, "sales.product_line")
		assert.Equal(t, SeedProductLineID, jsonField(jsonObject(jsonObject(withLine, "sales"), "product_line"), "id"))
	})
}

// --- Update ---

func TestItems_UpdateAllFields(t *testing.T) {
	t.Parallel()
	attrA := newColorAttribute(t, "active")
	body := forgeItemBody("material", uniqueName("e2e-item-upd-all"))
	body["attribute_ids"] = []string{attrA}
	created := createForgeItem(t, itemsPath, body)
	itemID := jsonField(created, "id")
	assertNilField(t, created, "sales")

	attrB := newColorAttribute(t, "active")
	newSKU := uniqueName("e2e-item-upd-all-sku")
	got := patchItem(t, itemID, map[string]any{
		"sku":           newSKU,
		"description":   "updated description",
		"attribute_ids": []string{attrB},
		"status":        "archived",
		"supply":        "make",
		"unit_cost":     priceInputPer("6.00", SeedMaterialUnitID),
		"material": map[string]any{
			"order_point": map[string]any{"value": "12", "unit_id": SeedMaterialUnitID},
			"lead_time":   map[string]any{"value": "8", "unit_id": dayUnitID},
		},
		// A material you sell gains a sales section the first time one is sent.
		"sales": map[string]any{
			"unit_price":        priceInputPer("9.00", SeedMaterialUnitID),
			"product_line_id":   SeedProductLineID,
			"portal_visibility": "visible",
			"lead_time":         map[string]any{"value": "2", "unit_id": dayUnitID},
		},
	})
	assert.Equal(t, itemID, jsonField(got, "id"))
	assertItemEnvelope(t, got, "material", newSKU)
	assert.Equal(t, "updated description", jsonField(got, "description"))
	assert.Equal(t, "archived", jsonField(got, "status"))
	assert.Equal(t, "make", jsonField(got, "supply"))
	assertItemPrice(t, jsonObject(got, "unit_cost"), "6.00", SeedMaterialUnitID, "unit_cost")
	material := jsonObject(got, "material")
	assertQuantityIn(t, jsonObject(material, "order_point"), "12", SeedMaterialUnitID, "material.order_point")
	assertQuantityIn(t, jsonObject(material, "lead_time"), "8", dayUnitID, "material.lead_time")
	sales := jsonObject(got, "sales")
	require.NotNil(t, sales, "the material gained a sales section")
	assertItemPrice(t, jsonObject(sales, "unit_price"), "9.00", SeedMaterialUnitID, "sales.unit_price")
	assertNilField(t, sales, "product_line")
	assert.Equal(t, "visible", jsonField(sales, "portal_visibility"))
	assertQuantityIn(t, jsonObject(sales, "lead_time"), "2", dayUnitID, "sales.lead_time")

	read := getItem(t, itemID, "category", "attributes", "sales.product_line")
	assert.Equal(t, []string{attrB}, attributeIDsOf(read), "attribute_ids replaces the list")
	assert.Equal(t, SeedMaterialCategoryID, jsonField(jsonObject(read, "category"), "id"))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(jsonObject(read, "sales"), "product_line"), "id"))
	assert.Equal(t, jsonField(created, "created_at"), jsonField(read, "created_at"))
}

func TestItems_UpdateNullClears(t *testing.T) {
	t.Parallel()

	t.Run("product", func(t *testing.T) {
		body := forgeItemBody("product", uniqueName("e2e-item-clear-prod"))
		body["description"] = "to clear"
		body["supply"] = "buy"
		body["sales"] = map[string]any{
			"product_line_id": SeedProductLineID,
			"lead_time":       map[string]any{"value": "4", "unit_id": dayUnitID},
		}
		itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

		got := patchItem(t, itemID, map[string]any{
			"description": nil,
			"supply":      nil,
			"sales":       map[string]any{"product_line_id": nil, "lead_time": nil},
		})
		assertNilField(t, got, "description")
		assertNilField(t, got, "supply")
		sales := jsonObject(got, "sales")
		require.NotNil(t, sales, "clearing fields keeps the section")
		assertNilField(t, sales, "lead_time")

		read := getItem(t, itemID, "sales.product_line")
		assertNilField(t, jsonObject(read, "sales"), "product_line")
	})

	t.Run("material", func(t *testing.T) {
		body := forgeItemBody("material", uniqueName("e2e-item-clear-mat"))
		body["material"] = map[string]any{
			"order_point": map[string]any{"value": "9", "unit_id": SeedMaterialUnitID},
			"lead_time":   map[string]any{"value": "6", "unit_id": dayUnitID},
		}
		itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

		got := patchItem(t, itemID, map[string]any{"material": map[string]any{"order_point": nil, "lead_time": nil}})
		material := jsonObject(got, "material")
		require.NotNil(t, material)
		assertNilField(t, material, "order_point")
		assertNilField(t, material, "lead_time")
	})

	t.Run("empty attribute_ids removes every attribute", func(t *testing.T) {
		attrID := newColorAttribute(t, "active")
		body := forgeItemBody("part", uniqueName("e2e-item-clear-attrs"))
		body["attribute_ids"] = []string{attrID}
		itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

		patchItem(t, itemID, map[string]any{"attribute_ids": []string{}})
		assert.Empty(t, attributeIDsOf(getItem(t, itemID, "attributes")))
	})
}

func TestItems_UpdateStatusArchiveAndUnarchive(t *testing.T) {
	t.Parallel()
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-item-arch"))), "id")

	assert.Equal(t, "archived", jsonField(patchItem(t, itemID, map[string]any{"status": "archived"}), "status"))
	assert.Equal(t, "archived", jsonField(getItem(t, itemID), "status"))
	assert.Equal(t, "active", jsonField(patchItem(t, itemID, map[string]any{"status": "active"}), "status"))
}

// Changing category keeps the item's units; the new category only has to suit the type and carry its attributes.
func TestItems_UpdateCategory(t *testing.T) {
	t.Parallel()

	t.Run("moves to another product category", func(t *testing.T) {
		itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-item-cat"))), "id")
		got := patchItem(t, itemID, map[string]any{"category_id": SeedPropertylessItemCategoryID})
		assertNilField(t, got, "category")
		assert.Equal(t, SeedPropertylessItemCategoryID, jsonField(jsonObject(getItem(t, itemID, "category"), "category"), "id"))
	})

	t.Run("a category of the wrong kind is refused", func(t *testing.T) {
		itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-item-cat-kind"))), "id")
		status, body, err := apiClient.Patch(itemsPath+"/"+itemID, map[string]any{"category_id": SeedMaterialCategoryID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "category_id")
	})

	t.Run("a category that would strand an attribute is refused", func(t *testing.T) {
		body := forgeItemBody("product", uniqueName("e2e-item-cat-strand"))
		body["attribute_ids"] = []string{SeedAttributeID}
		itemID := jsonField(createForgeItem(t, itemsPath, body), "id")
		status, resp, err := apiClient.Patch(itemsPath+"/"+itemID, map[string]any{"category_id": SeedPropertylessItemCategoryID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, resp)
		assertErrorParam(t, requireErrorResponse(t, resp, "", "invalid_request_error"), "category_id")
	})

	t.Run("replacing the attributes in the same write makes the move valid", func(t *testing.T) {
		body := forgeItemBody("product", uniqueName("e2e-item-cat-swap"))
		body["attribute_ids"] = []string{SeedAttributeID}
		itemID := jsonField(createForgeItem(t, itemsPath, body), "id")
		patchItem(t, itemID, map[string]any{"category_id": SeedPropertylessItemCategoryID, "attribute_ids": []string{}})
		got := getItem(t, itemID, "category", "attributes")
		assert.Equal(t, SeedPropertylessItemCategoryID, jsonField(jsonObject(got, "category"), "id"))
		assert.Empty(t, attributeIDsOf(got))
	})
}

func TestItems_UpdateRejections(t *testing.T) {
	t.Parallel()
	productID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-item-upd-rej"))), "id")
	archivedAttr := newColorAttribute(t, "archived")
	archivedLine := newArchivedProductLine(t)

	t.Run("type is immutable", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+productID, map[string]any{"type": "part"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
		assertErrorParam(t, errObj, "type")
	})

	t.Run("material section on a product", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+productID, map[string]any{
			"material": map[string]any{"order_point": map[string]any{"value": "1", "unit_id": SeedUnitID}},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "material")
	})

	t.Run("archived attribute", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+productID, map[string]any{"attribute_ids": []string{archivedAttr}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "attribute_ids")
	})

	t.Run("archived product line", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+productID, map[string]any{
			"sales": map[string]any{"product_line_id": archivedLine},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "sales.product_line_id")
	})

	t.Run("empty sku", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+productID, map[string]any{"sku": ""}, newIdempotencyKey())
		require.NoError(t, err)
		requireClientRejection(t, status, body)
	})

	t.Run("sku over max length", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+productID, map[string]any{"sku": strings.Repeat("a", 256)}, newIdempotencyKey())
		require.NoError(t, err)
		requireClientRejection(t, status, body)
	})

	t.Run("unknown item", func(t *testing.T) {
		status, body, err := apiClient.Patch(itemsPath+"/"+mustGenID(t, id.ItemIDPrefix), map[string]any{"description": "x"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})
}

// --- Validation ---

func TestItems_CreateValidation(t *testing.T) {
	t.Parallel()
	attrA := newColorAttribute(t, "active")
	attrB := newColorAttribute(t, "active")
	archivedAttr := newColorAttribute(t, "archived")
	archivedLine := newArchivedProductLine(t)

	with := func(itemType string, set map[string]any) map[string]any {
		body := forgeItemBody(itemType, uniqueName("e2e-item-val"))
		for k, v := range set {
			body[k] = v
		}
		return body
	}
	without := func(field string) map[string]any {
		body := forgeItemBody("part", uniqueName("e2e-item-val"))
		delete(body, field)
		return body
	}

	// A 422 that names its param; code "" accepts any 422 code.
	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"material in a product category", with("material", map[string]any{"category_id": SeedItemCategoryID}), "invalid_format", "category_id"},
		{"product in a material category", with("product", map[string]any{"category_id": SeedMaterialCategoryID}), "invalid_format", "category_id"},
		{"material section on a part", with("part", map[string]any{"material": map[string]any{
			"order_point": map[string]any{"value": "1", "unit_id": SeedUnitID},
		}}), "invalid_format", "material"},
		{"two attributes of one property", with("part", map[string]any{"attribute_ids": []string{attrA, attrB}}), "invalid_format", "attribute_ids"},
		{"attribute of a property the category lacks", with("part", map[string]any{
			"category_id": SeedPropertylessItemCategoryID, "attribute_ids": []string{SeedAttributeID},
		}), "invalid_format", "attribute_ids"},
		{"archived attribute", with("part", map[string]any{"attribute_ids": []string{archivedAttr}}), "resource_archived", "attribute_ids"},
		{"archived product line", with("product", map[string]any{
			"sales": map[string]any{"product_line_id": archivedLine},
		}), "resource_archived", "sales.product_line_id"},
		{"sales lead time not in time", with("product", map[string]any{
			"sales": map[string]any{"lead_time": map[string]any{"value": "5", "unit_id": SeedUnitID}},
		}), "invalid_format", "sales.lead_time.unit_id"},
		{"material lead time not in time", with("material", map[string]any{
			"material": map[string]any{"lead_time": map[string]any{"value": "5", "unit_id": SeedMaterialUnitID}},
		}), "invalid_format", "material.lead_time.unit_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(itemsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), tc.param)
		})
	}

	// Malformed or missing fields: a 400 parameter error or a 422, whichever the layer that catches it gives.
	malformed := []struct {
		name string
		body map[string]any
	}{
		{"missing category_id", without("category_id")},
		{"missing sku", without("sku")},
		{"missing type", without("type")},
		{"unknown type", with("widget", nil)},
		{"unknown status", with("part", map[string]any{"status": "deleted"})},
		{"unknown supply", with("part", map[string]any{"supply": "make_to_order"})},
		{"unknown portal visibility", with("product", map[string]any{"sales": map[string]any{"portal_visibility": "public"}})},
		{"unknown category", with("part", map[string]any{"category_id": "itcg_00000000000000000000000000"})},
		{"notes is gone", with("part", map[string]any{"notes": "x"})},
	}
	for _, tc := range malformed {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(itemsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireClientRejection(t, status, body)
		})
	}

	t.Run("duplicate sku conflicts", func(t *testing.T) {
		sku := uniqueName("e2e-item-dup")
		createForgeItem(t, itemsPath, forgeItemBody("part", sku))
		status, body, err := apiClient.Post(itemsPath, forgeItemBody("product", strings.ToUpper(sku)), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "sku")
	})
}

// --- Idempotency ---

func TestItems_CreateIdempotent(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("part", uniqueName("e2e-item-idem"))
	key := newIdempotencyKey()

	s1, b1, err := apiClient.Post(itemsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s1, b1)
	first := parseJSON(b1)
	t.Cleanup(func() { apiClient.Delete(itemsPath + "/" + jsonField(first, "id")) })

	s2, b2, err := apiClient.Post(itemsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s2, b2)
	assert.Equal(t, first, parseJSON(b2), "a replay answers the first response")

	list, _, err := apiClient.GetList(itemsPath, url.Values{"skus": {jsonField(body, "sku")}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "a replay creates nothing")
}

func TestItems_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-item-upd-idem"))), "id")
	key := newIdempotencyKey()
	body := map[string]any{"description": "first write"}

	s1, b1, err := apiClient.Patch(itemsPath+"/"+itemID, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)

	patchItem(t, itemID, map[string]any{"description": "second write"})

	s2, b2, err := apiClient.Patch(itemsPath+"/"+itemID, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)
	assert.Equal(t, parseJSON(b1), parseJSON(b2), "a replay answers the first response")
	assert.Equal(t, "second write", jsonField(getItem(t, itemID), "description"), "a replay writes nothing")
}

// --- Delete ---

func TestItems_DeleteReturnsStubThenGone(t *testing.T) {
	t.Parallel()
	resp, err := apiClient.PostFull(itemsPath, forgeItemBody("material", uniqueName("e2e-item-del")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	itemID := jsonField(parseJSON(resp.Body), "id")

	status, body, err := apiClient.Delete(itemsPath + "/" + itemID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	stub := parseJSON(body)
	assert.Equal(t, map[string]any{"id": itemID, "object": "item", "deleted": true}, stub)

	status, body, err = apiClient.Delete(itemsPath + "/" + itemID)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)

	status, _, err = apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestItems_DeleteNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(itemsPath + "/" + mustGenID(t, id.ItemIDPrefix))
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// An item on a sales order line can't be deleted; archiving retires it instead.
func TestItems_DeleteInUseConflicts(t *testing.T) {
	t.Parallel()
	productID, itemID := newSellableProductIDs(t, "e2e-item-inuse")
	orderID := createLifecycleOrder(t)
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  productID,
		"product_sku": uniqueName("E2E-LINE-SKU"),
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	status, body, err = apiClient.Delete(itemsPath + "/" + itemID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")

	assert.Equal(t, itemID, jsonField(getItem(t, itemID), "id"), "the item survives a refused delete")
}

// An archived item stays on existing records but can't go on a new sales order line.
func TestItems_ArchivedCannotBeAddedToSalesOrderLine(t *testing.T) {
	t.Parallel()
	productID, itemID := newSellableProductIDs(t, "e2e-item-arch-line")
	orderID := createLifecycleOrder(t)
	patchItem(t, itemID, map[string]any{"status": "archived"})

	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  productID,
		"product_sku": uniqueName("E2E-LINE-SKU"),
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "product_id")
}

// --- Expandable Fields ---

func TestItems_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	attrID := newColorAttribute(t, "active")
	body := forgeItemBody("product", uniqueName("e2e-item-noinc"))
	body["attribute_ids"] = []string{attrID}
	body["sales"] = map[string]any{"product_line_id": SeedProductLineID}
	created := createForgeItem(t, itemsPath, body)
	itemID := jsonField(created, "id")

	for label, got := range map[string]map[string]any{"create": created, "retrieve": getItem(t, itemID)} {
		assertNilField(t, got, "category")
		assertNilField(t, got, "attributes")
		assertNilField(t, got, "default_manufacturing_method")
		sales := jsonObject(got, "sales")
		require.NotNil(t, sales, label)
		assertNilField(t, sales, "product_line")
	}

	list, _, err := apiClient.GetList(itemsPath, url.Values{"skus": {jsonField(created, "sku")}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	m := parseJSON(list.Data[0])
	assertNilField(t, m, "category")
	assertNilField(t, m, "attributes")
	assertNilField(t, jsonObject(m, "sales"), "product_line")

	withIncludes := getItem(t, itemID, "category", "attributes", "sales.product_line")
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(withIncludes, "category"), "id"))
	assert.Equal(t, "list", jsonField(jsonObject(withIncludes, "attributes"), "object"))
	assert.Equal(t, []string{attrID}, attributeIDsOf(withIncludes))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(jsonObject(withIncludes, "sales"), "product_line"), "id"))
}

func TestItems_UpdateHonorsIncludes(t *testing.T) {
	t.Parallel()
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-item-updinc"))), "id")
	status, body, err := apiClient.Patch(itemsPath+"/"+itemID+"?include=category,attributes,sales.product_line",
		map[string]any{"sales": map[string]any{"product_line_id": SeedProductLineID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "item_category", jsonField(jsonObject(got, "category"), "object"))
	assert.Equal(t, "list", jsonField(jsonObject(got, "attributes"), "object"))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(jsonObject(got, "sales"), "product_line"), "id"))
}

func TestItems_IncludeAttributesProperty(t *testing.T) {
	t.Parallel()
	got := getItem(t, SeedItemID, "attributes", "attributes.property", "category.unit_group")
	cat := jsonObject(got, "category")
	require.NotNil(t, cat)
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotNil(t, jsonObject(cat, "unit_group"), "category.unit_group with include")
	for _, raw := range jsonListData(got, "attributes") {
		attr, _ := raw.(map[string]any)
		assert.Equal(t, "attribute", jsonField(attr, "object"))
		assert.Equal(t, "property", jsonField(jsonObject(attr, "property"), "object"))
	}
}

func TestItems_ListIncludeCategoryAndAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"category,attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, data)
	for _, raw := range data {
		item, _ := raw.(map[string]any)
		assert.Equal(t, "item_category", jsonField(jsonObject(item, "category"), "object"))
		assert.Equal(t, "list", jsonField(jsonObject(item, "attributes"), "object"))
	}
}

func TestItems_UnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	for _, include := range []string{"unit_value", "notes"} {
		status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {include}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "forge items have no %s include: %s", include, body)
	}
}

// assertItemUnitPriceShape checks an item price is a value shape: an amount in dollars per a unit, with no id of its own.
func assertItemUnitPriceShape(t *testing.T, price map[string]any) {
	t.Helper()
	assert.NotEmpty(t, jsonField(price, "amount"))
	assert.Equal(t, "usd", jsonField(price, "currency"))
	assert.NotEmpty(t, jsonField(jsonObject(price, "per_unit"), "id"), "per_unit is always present")
	assert.NotContains(t, price, "id")
	assert.NotContains(t, price, "object")
}

// assertItemRateShape checks a rate is a value shape: a value in one unit per another, with no id of its own.
func assertItemRateShape(t *testing.T, rate map[string]any) {
	t.Helper()
	assert.NotEmpty(t, jsonField(rate, "value"))
	assert.NotEmpty(t, jsonField(jsonObject(rate, "unit"), "id"), "unit is always present")
	assert.NotEmpty(t, jsonField(jsonObject(rate, "per_unit"), "id"), "per_unit is always present")
	assert.NotContains(t, rate, "id")
	assert.NotContains(t, rate, "object")
}
