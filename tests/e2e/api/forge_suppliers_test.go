//go:build e2e

package api_test

import (
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func supplierItemsPath(supplierID string) string {
	return suppliersPath + "/" + supplierID + "/items"
}

// forgeSupplier creates a supplier with name and number plus extra, and deletes it after the test.
func forgeSupplier(t *testing.T, prefix string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"name": uniqueName(prefix), "number": uniqueName("SUP")}
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, suppliersPath, body)
}

func forgeSupplierRead(t *testing.T, supplierID string, includes ...string) map[string]any {
	t.Helper()
	params := url.Values{}
	for _, inc := range includes {
		params.Add("include[]", inc)
	}
	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+supplierID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func forgeSupplierPatch(t *testing.T, supplierID string, body map[string]any) (int, []byte) {
	t.Helper()
	status, resp, err := apiClient.Patch(suppliersPath+"/"+supplierID, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a supplier update must not 5xx: %s", resp)
	return status, resp
}

// forgeMaterialItem creates a material item and returns its ID.
func forgeMaterialItem(t *testing.T) string {
	t.Helper()
	return jsonField(createForgeItem(t, itemsPath, forgeItemBody("material", uniqueName("e2e-si-mat"))), "id")
}

// forgeSupplierItem creates a supplier item and deletes it after the test.
func forgeSupplierItem(t *testing.T, supplierID, itemID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"item_id": itemID, "supplier_sku": uniqueName("SKU")}
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, supplierItemsPath(supplierID), body)
}

// --- Suppliers ---

func TestForgeSuppliers_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	name, number := uniqueName("e2e-fs-all"), uniqueName("FS")
	created := createAndCleanup(t, suppliersPath, map[string]any{
		"name":            name,
		"number":          number,
		"status":          "active",
		"instructions":    "Confirm every delivery date.",
		"contact":         map[string]any{"email": "po@fs.e2e.openmrp.ai", "phone": "555-0100", "url": "https://fs.e2e.openmrp.ai"},
		"carrier_billing": map[string]any{"type": "third_party", "account_number": "TP-9"},
		"defaults": map[string]any{
			"carrier_id": SeedCarrierID, "service_level_id": SeedServiceLevelID,
			"payment_term_id": SeedPaymentTermID, "shipping_term_id": SeedShippingTermID,
		},
		"bill_to_address": paritySupplierAddressBody(name+" Billing", "1 Supply Way"),
	})
	supplierID := jsonField(created, "id")
	assertIDFormat(t, supplierID, id.AccountIDPrefix)
	requireDefaultSite(t, supplierID, name)
	assertObjectField(t, created, "supplier")
	assert.Equal(t, "active", jsonField(created, "status"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, number, jsonField(created, "number"))
	assert.Equal(t, "Confirm every delivery date.", jsonField(created, "instructions"))
	assert.Equal(t, map[string]any{"email": "po@fs.e2e.openmrp.ai", "phone": "555-0100", "url": "https://fs.e2e.openmrp.ai"}, jsonObject(created, "contact"))
	assert.Equal(t, map[string]any{"type": "third_party", "account_number": "TP-9"}, jsonObject(created, "carrier_billing"))
	defaults := jsonObject(created, "defaults")
	require.NotNil(t, defaults, "defaults is always present")
	assert.NotContains(t, defaults, "object")
	for _, ref := range []string{"carrier", "service_level", "payment_term", "shipping_term", "bill_to_address", "ship_to_address"} {
		assert.Contains(t, defaults, ref)
		assert.Nil(t, defaults[ref], "defaults.%s is null unless included", ref)
	}
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	for _, gone := range []string{"note", "bill_to_address", "ship_to_address"} {
		assert.NotContains(t, created, gone)
	}
	assertNilField(t, created, "material_count")

	withRefs := forgeSupplierRead(t, supplierID, "defaults.carrier", "defaults.service_level", "defaults.payment_term", "defaults.shipping_term", "defaults.bill_to_address", "defaults.ship_to_address")
	refs := jsonObject(withRefs, "defaults")
	assert.Equal(t, SeedCarrierID, jsonField(jsonObject(refs, "carrier"), "id"))
	assert.Equal(t, SeedServiceLevelID, jsonField(jsonObject(refs, "service_level"), "id"))
	assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(refs, "payment_term"), "id"))
	assert.Equal(t, SeedShippingTermID, jsonField(jsonObject(refs, "shipping_term"), "id"))
	billTo := jsonObject(refs, "bill_to_address")
	require.NotNil(t, billTo)
	assert.Equal(t, "1 Supply Way", jsonField(billTo, "line_1"))
	assert.Equal(t, jsonField(billTo, "id"), jsonField(jsonObject(refs, "ship_to_address"), "id"), "a bill-to alone is also the ship-from address")

	// Section fields merge: only what is sent changes.
	status, resp := forgeSupplierPatch(t, supplierID, map[string]any{
		"status":          "archived",
		"instructions":    nil,
		"contact":         map[string]any{"phone": "555-0199"},
		"carrier_billing": map[string]any{"type": "sender"},
		"defaults":        map[string]any{"service_level_id": nil, "carrier_id": nil, "shipping_term_id": nil, "ship_to_address_id": nil},
	})
	requireStatus(t, 200, status, resp)
	updated := parseJSON(resp)
	assert.Equal(t, "archived", jsonField(updated, "status"))
	assertNilField(t, updated, "instructions")
	assert.Equal(t, map[string]any{"email": "po@fs.e2e.openmrp.ai", "phone": "555-0199", "url": "https://fs.e2e.openmrp.ai"}, jsonObject(updated, "contact"))
	assert.Equal(t, map[string]any{"type": "sender", "account_number": "TP-9"}, jsonObject(updated, "carrier_billing"))
	assert.Equal(t, name, jsonField(updated, "name"), "a field the update left out keeps its value")

	after := jsonObject(forgeSupplierRead(t, supplierID, "defaults.carrier", "defaults.payment_term", "defaults.shipping_term", "defaults.bill_to_address", "defaults.ship_to_address"), "defaults")
	assertNilField(t, after, "carrier")
	assertNilField(t, after, "shipping_term")
	assertNilField(t, after, "ship_to_address")
	assert.Equal(t, SeedPaymentTermID, jsonField(jsonObject(after, "payment_term"), "id"), "a default the update left out stays")
	assert.Equal(t, jsonField(billTo, "id"), jsonField(jsonObject(after, "bill_to_address"), "id"))

	status, resp = forgeSupplierPatch(t, supplierID, map[string]any{"status": "active", "defaults": map[string]any{"ship_to_address_id": jsonField(billTo, "id")}})
	requireStatus(t, 200, status, resp)
	assert.Equal(t, "active", jsonField(parseJSON(resp), "status"))
}

func TestForgeSuppliers_OmittedFieldsTakeDefaults(t *testing.T) {
	t.Parallel()

	created := forgeSupplier(t, "e2e-fs-min", nil)
	assert.Equal(t, "active", jsonField(created, "status"))
	assertNilField(t, created, "instructions")
	assert.Equal(t, map[string]any{"email": nil, "phone": nil, "url": nil}, jsonObject(created, "contact"))
	assert.Equal(t, map[string]any{"type": "sender", "account_number": nil}, jsonObject(created, "carrier_billing"))
	read := jsonObject(forgeSupplierRead(t, jsonField(created, "id"), "defaults.carrier", "defaults.payment_term", "defaults.bill_to_address"), "defaults")
	for _, ref := range []string{"carrier", "payment_term", "bill_to_address"} {
		assertNilField(t, read, ref)
	}
}

func TestForgeSuppliers_CreateValidation(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		body  map[string]any
		code  int
		param string
	}{
		"missing name":       {body: map[string]any{"number": uniqueName("SUP")}, code: 422, param: "name"},
		"note is gone":       {body: map[string]any{"name": "x", "number": uniqueName("SUP"), "note": "n"}, code: 400, param: "note"},
		"bad status":         {body: map[string]any{"name": "x", "number": uniqueName("SUP"), "status": "inactive"}, code: 422, param: "status"},
		"unknown carrier":    {body: map[string]any{"name": "x", "number": uniqueName("SUP"), "defaults": map[string]any{"carrier_id": mustGenID(t, "cr")}}, code: 404, param: "defaults.carrier_id"},
		"unknown pay term":   {body: map[string]any{"name": "x", "number": uniqueName("SUP"), "defaults": map[string]any{"payment_term_id": mustGenID(t, "pytm")}}, code: 404, param: "defaults.payment_term_id"},
		"bad billing type":   {body: map[string]any{"name": "x", "number": uniqueName("SUP"), "carrier_billing": map[string]any{"type": "collect"}}, code: 422, param: "carrier_billing.type"},
		"inline address bad": {body: map[string]any{"name": "x", "number": uniqueName("SUP"), "bill_to_address": map[string]any{"name": "x", "country": "usa"}}, code: 422, param: "bill_to_address.country"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(suppliersPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireErrorParam(t, tc.code, status, body, tc.param)
		})
	}
}

func TestForgeSuppliers_ArchivedReferenceIsRefused(t *testing.T) {
	t.Parallel()

	term := createAndCleanup(t, paymentTermsPath, map[string]any{"name": uniqueName("e2e-fs-term"), "status": "archived"})
	status, body, err := apiClient.Post(suppliersPath, map[string]any{
		"name": uniqueName("e2e-fs-arch"), "number": uniqueName("SUP"),
		"defaults": map[string]any{"payment_term_id": jsonField(term, "id")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 422, status, "%s", body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "defaults.payment_term_id")
}

func TestForgeSuppliers_CreateIdempotent(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	body := map[string]any{"name": uniqueName("e2e-fs-idem"), "number": uniqueName("SUP")}
	status1, resp1, err := apiClient.Post(suppliersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, resp1)
	id := jsonField(parseJSON(resp1), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + id) })

	status2, resp2, err := apiClient.Post(suppliersPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, resp2)
	assert.Equal(t, id, jsonField(parseJSON(resp2), "id"), "a replayed create returns the first supplier")
}

func TestForgeSuppliers_UpdateRefusesAnotherAccountsAddress(t *testing.T) {
	t.Parallel()

	supplierID := jsonField(forgeSupplier(t, "e2e-fs-addr", nil), "id")
	status, body := forgeSupplierPatch(t, supplierID, map[string]any{"defaults": map[string]any{"bill_to_address_id": mustGenID(t, "ad")}})
	requireErrorParam(t, 404, status, body, "defaults.bill_to_address_id")
}

func TestForgeSuppliers_DeleteReturnsStubAndRefusesInUse(t *testing.T) {
	t.Parallel()

	created := forgeSupplier(t, "e2e-fs-del", nil)
	supplierID := jsonField(created, "id")
	status, body, err := apiClient.Delete(suppliersPath + "/" + supplierID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	stub := parseJSON(body)
	assert.Equal(t, map[string]any{"id": supplierID, "object": "supplier", "deleted": true}, stub)
	again, againBody, err := apiClient.Delete(suppliersPath + "/" + supplierID)
	require.NoError(t, err)
	assert.Equal(t, 410, again, "%s", againBody)

	// The seeded supplier has purchase orders.
	status, body, err = apiClient.Delete(suppliersPath + "/" + SeedSupplierAccountID)
	require.NoError(t, err)
	require.Equal(t, 409, status, "%s", body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "purchase orders")

	withItem := jsonField(forgeSupplier(t, "e2e-fs-del-item", nil), "id")
	forgeSupplierItem(t, withItem, forgeMaterialItem(t), nil)
	status, body, err = apiClient.Delete(suppliersPath + "/" + withItem)
	require.NoError(t, err)
	require.Equal(t, 409, status, "%s", body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "supplier items")
}

func TestForgeSuppliers_ListFilters(t *testing.T) {
	t.Parallel()

	tok := searchToken("fsl")
	active := forgeSupplier(t, "e2e-fs-list", map[string]any{"name": "Active " + tok})
	archived := forgeSupplier(t, "e2e-fs-list", map[string]any{"name": "Archived " + tok, "status": "archived"})
	activeID, archivedID := jsonField(active, "id"), jsonField(archived, "id")
	itemID := forgeMaterialItem(t)
	forgeSupplierItem(t, activeID, itemID, nil)

	assert.ElementsMatch(t, []string{activeID, archivedID}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}}))
	assert.Equal(t, []string{archivedID}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "statuses[]": {"archived"}}))
	assert.Equal(t, []string{activeID}, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "statuses[]": {"active"}}))
	assert.Equal(t, []string{activeID}, listedSupplierIDs(t, apiClient, url.Values{"item_ids[]": {itemID}}))
	assert.Equal(t, []string{activeID}, listedSupplierIDs(t, apiClient, url.Values{"q": {jsonField(active, "number")}}), "q matches the number")

	createdAt, err := time.Parse(time.RFC3339, jsonField(active, "created_at"))
	require.NoError(t, err)
	assert.Empty(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "created_after": {createdAt.Add(time.Hour).UTC().Format(time.RFC3339)}}))
	assert.Len(t, listedSupplierIDs(t, apiClient, url.Values{"q": {tok}, "created_before": {createdAt.Add(time.Hour).UTC().Format(time.RFC3339)}}), 2)

	status, body, err := apiClient.GetListRaw(suppliersPath, url.Values{"limit": {"101"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "the page size is capped at 100: %s", body)
	status, body, err = apiClient.GetListRaw(suppliersPath, url.Values{"starts_at": {createdAt.UTC().Format(time.RFC3339)}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "starts_at is gone in 1.0.forge: %s", body)
}

func TestForgeSuppliers_Preview9KeepsItsShape(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-fs-p9")
	p9 := apiClient.WithAPIVersion(preview9Version)
	status, body, err := p9.Post(suppliersPath+"?include=bill_to_address", map[string]any{
		"name": name, "number": uniqueName("SUP"), "note": "net 30",
		"bill_to_address": paritySupplierAddressBody(name, "9 Preview St"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	supplierID := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + supplierID) })
	assert.Equal(t, "net 30", jsonField(got, "note"))
	require.NotNil(t, jsonObject(got, "bill_to_address"))
	assert.Equal(t, "0", jsonField(got, "material_count"))
	for _, gone := range []string{"status", "instructions", "contact", "carrier_billing", "defaults"} {
		assert.NotContains(t, got, gone)
	}

	assert.Equal(t, "net 30", jsonField(forgeSupplierRead(t, supplierID), "instructions"), "a preview.9 note is the supplier's instructions")

	status, body, err = p9.Patch(suppliersPath+"/"+supplierID, map[string]any{"update_note": true}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "note")
	assertNilField(t, forgeSupplierRead(t, supplierID), "instructions")
}

// --- Supplier items ---

func TestForgeSupplierItems_CRUD(t *testing.T) {
	t.Parallel()

	supplierID := jsonField(forgeSupplier(t, "e2e-fsi-crud", nil), "id")
	itemID := forgeMaterialItem(t)
	sku := uniqueName("SKU")
	created := forgeSupplierItem(t, supplierID, itemID, map[string]any{"supplier_sku": sku, "supplier_description": "16 ga sheet"})
	siID := jsonField(created, "id")
	assertIDFormat(t, siID, id.SupplierItemIDPrefix)
	assertObjectField(t, created, "supplier_item")
	assert.Equal(t, "active", jsonField(created, "status"))
	assert.Equal(t, sku, jsonField(created, "supplier_sku"))
	assert.Equal(t, "16 ga sheet", jsonField(created, "supplier_description"))
	assertNilField(t, created, "item")
	assertNilField(t, created, "supplier")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	for _, gone := range []string{"material", "supplier_part_number"} {
		assert.NotContains(t, created, gone)
	}

	path := supplierItemsPath(supplierID) + "/" + siID
	status, body, err := apiClient.GetListRaw(path, url.Values{"include[]": {"item", "supplier"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	read := parseJSON(body)
	assert.Equal(t, itemID, jsonField(jsonObject(read, "item"), "id"))
	assert.Equal(t, supplierID, jsonField(jsonObject(read, "supplier"), "id"))

	status, body, err = apiClient.Patch(path, map[string]any{"supplier_description": nil, "status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assert.Equal(t, "archived", jsonField(updated, "status"))
	assertNilField(t, updated, "supplier_description")
	assert.Equal(t, sku, jsonField(updated, "supplier_sku"), "a field the update left out keeps its value")

	// Another supplier's path does not reach it.
	other := jsonField(forgeSupplier(t, "e2e-fsi-other", nil), "id")
	status, body, err = apiClient.GetListRaw(supplierItemsPath(other)+"/"+siID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "%s", body)

	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, map[string]any{"id": siID, "object": "supplier_item", "deleted": true}, parseJSON(body))
	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	assert.Equal(t, 410, status, "%s", body)
	status, body, err = apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "%s", body)
}

func TestForgeSupplierItems_CreateRules(t *testing.T) {
	t.Parallel()

	supplierID := jsonField(forgeSupplier(t, "e2e-fsi-rules", nil), "id")
	itemID := forgeMaterialItem(t)
	forgeSupplierItem(t, supplierID, itemID, nil)

	post := func(supplier string, body map[string]any) (int, []byte) {
		status, resp, err := apiClient.Post(supplierItemsPath(supplier), body, newIdempotencyKey())
		require.NoError(t, err)
		return status, resp
	}

	status, body := post(supplierID, map[string]any{"item_id": itemID, "supplier_sku": "again"})
	assert.Equal(t, 409, status, "a supplier holds an item once: %s", body)

	part := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-fsi-part"))), "id")
	status, body = post(supplierID, map[string]any{"item_id": part, "supplier_sku": "x"})
	requireErrorParam(t, 422, status, body, "item_id")

	status, body = post(supplierID, map[string]any{"item_id": mustGenID(t, "it"), "supplier_sku": "x"})
	requireErrorParam(t, 404, status, body, "item_id")

	status, body = post(mustGenID(t, "ac"), map[string]any{"item_id": itemID, "supplier_sku": "x"})
	requireErrorParam(t, 404, status, body, "supplier_id")

	archivedItem := createForgeItem(t, itemsPath, func() map[string]any {
		b := forgeItemBody("material", uniqueName("e2e-fsi-arch"))
		b["status"] = "archived"
		return b
	}())
	status, body = post(supplierID, map[string]any{"item_id": jsonField(archivedItem, "id"), "supplier_sku": "x"})
	require.Equal(t, 422, status, "%s", body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "item_id")

	archivedSupplier := jsonField(forgeSupplier(t, "e2e-fsi-arch-sup", map[string]any{"status": "archived"}), "id")
	status, body = post(archivedSupplier, map[string]any{"item_id": forgeMaterialItem(t), "supplier_sku": "x"})
	require.Equal(t, 422, status, "%s", body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "supplier_id")

	status, body = post(supplierID, map[string]any{"supplier_sku": "x"})
	requireErrorParam(t, 422, status, body, "item_id")
}

func TestForgeSupplierItems_CreateIdempotent(t *testing.T) {
	t.Parallel()

	supplierID := jsonField(forgeSupplier(t, "e2e-fsi-idem", nil), "id")
	itemID := forgeMaterialItem(t)
	key := newIdempotencyKey()
	body := map[string]any{"item_id": itemID, "supplier_sku": uniqueName("SKU")}
	status1, resp1, err := apiClient.Post(supplierItemsPath(supplierID), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, resp1)
	siID := jsonField(parseJSON(resp1), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(supplierItemsPath(supplierID) + "/" + siID) })
	status2, resp2, err := apiClient.Post(supplierItemsPath(supplierID), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, resp2)
	assert.Equal(t, siID, jsonField(parseJSON(resp2), "id"))
}

func TestForgeSupplierItems_ListFilters(t *testing.T) {
	t.Parallel()

	supplierID := jsonField(forgeSupplier(t, "e2e-fsi-list", nil), "id")
	first, second := forgeMaterialItem(t), forgeMaterialItem(t)
	tok := searchToken("fsik")
	a := forgeSupplierItem(t, supplierID, first, map[string]any{"supplier_sku": "A-" + tok})
	b := forgeSupplierItem(t, supplierID, second, map[string]any{"supplier_sku": "B-x", "supplier_description": "has " + tok, "status": "archived"})

	ids := func(params url.Values) []string {
		list, status, err := apiClient.GetList(supplierItemsPath(supplierID), params)
		require.NoError(t, err)
		require.Equal(t, 200, status)
		var out []string
		for _, row := range list.Data {
			out = append(out, DataItemField(row, "id"))
		}
		return out
	}
	aID, bID := jsonField(a, "id"), jsonField(b, "id")
	assert.Equal(t, []string{bID, aID}, ids(nil), "newest first")
	assert.ElementsMatch(t, []string{aID, bID}, ids(url.Values{"q": {tok}}), "q matches the code or the description")
	assert.Equal(t, []string{bID}, ids(url.Values{"statuses[]": {"archived"}}))
	assert.Equal(t, []string{aID}, ids(url.Values{"item_ids[]": {first}}))
	assert.Equal(t, []string{bID}, ids(url.Values{"limit": {"1"}}))

	list, status, err := apiClient.GetList(supplierItemsPath(supplierID), url.Values{"include[]": {"item"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	for _, row := range list.Data {
		assert.NotNil(t, parseJSON(row)["item"], "item expands on the list")
	}
	status, body, err := apiClient.GetListRaw(supplierItemsPath(mustGenID(t, "ac")), nil)
	require.NoError(t, err)
	requireErrorParam(t, 404, status, body, "supplier_id")
}

// A material with a supplier item can't be deleted; the item reports what holds it.
func TestForgeSupplierItems_BlockItemDelete(t *testing.T) {
	t.Parallel()

	supplierID := jsonField(forgeSupplier(t, "e2e-fsi-itemdel", nil), "id")
	itemID := forgeMaterialItem(t)
	forgeSupplierItem(t, supplierID, itemID, nil)
	status, body, err := apiClient.Delete(itemsPath + "/" + itemID)
	require.NoError(t, err)
	require.Equal(t, 409, status, "%s", body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "supplier items")
}

func TestForgeSupplierItems_MaterialsRoutesArePreview9Only(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+SeedSupplierAccountID+"/items", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The client pins supplier material routes to preview.9, so ask the gateway directly with forge's version.
	req, err := http.NewRequest(http.MethodGet, apiClient.baseURL+suppliersPath+"/"+SeedSupplierAccountID+"/materials", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiClient.apiKey)
	req.Header.Set("OpenMRP-Account", apiClient.accountID)
	req.Header.Set("OpenMRP-Version", defaultAPIVersion)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 404, resp.StatusCode, "%s", raw)
}
