//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forgePaymentTerm creates a payment term with the extra fields given and deletes it when the test ends.
func forgePaymentTerm(t *testing.T, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"name": uniqueName("e2e-forge-pt")}
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, paymentTermsPath, body)
}

// forgeShippingTerm creates a carrier_rate shipping term with the extra fields given and deletes it when the test ends.
func forgeShippingTerm(t *testing.T, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"name": uniqueName("e2e-forge-st"), "type": "carrier_rate"}
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, shippingTermsPath, body)
}

func forgeDollars(amount string) map[string]any {
	return map[string]any{"amount": amount, "currency": "usd"}
}

// --- Payment terms ---

func TestForgePaymentTerms_Status(t *testing.T) {
	t.Parallel()
	created := forgePaymentTerm(t, map[string]any{"status": "archived"})
	id := jsonField(created, "id")
	assert.Equal(t, "archived", jsonField(created, "status"))

	status, body, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "active", jsonField(parseJSON(body), "status"))

	status, body, err = apiClient.Post(paymentTermsPath, map[string]any{"name": uniqueName("e2e-forge-pt-bad"), "status": "inactive"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "status")
}

func TestForgePaymentTerms_StatusesFilter(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-forge-pt-filter")
	archived := forgePaymentTerm(t, map[string]any{"name": name, "status": "archived"})

	list, _, err := apiClient.GetList(paymentTermsPath, url.Values{"q": {name}, "statuses[]": {"archived"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, jsonField(archived, "id"), DataItemField(list.Data[0], "id"))

	list, _, err = apiClient.GetList(paymentTermsPath, url.Values{"q": {name}, "statuses[]": {"active"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)

	list, _, err = apiClient.GetList(paymentTermsPath, url.Values{"q": {strings.ToUpper(name)}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "q is a case-insensitive substring match; omitted statuses return every status")
}

func TestForgePaymentTerms_ArchivedCantBeNewlyAssigned(t *testing.T) {
	t.Parallel()
	archivedID := jsonField(forgePaymentTerm(t, map[string]any{"status": "archived"}), "id")

	body := validCustomerBody(uniqueName("e2e-forge-pt-archcust"))
	body["default_payment_term_id"] = archivedID
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + id) })
	}
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "default_payment_term_id")

	customerID := dashCustomersNew(t, "e2e-forge-pt-cust", nil)
	status, resp, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"default_payment_term_id": archivedID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "default_payment_term_id")

	order := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	order["payment_term_id"] = archivedID
	status, resp, err = apiClient.Post(salesOrdersPath, order, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		deleteOrder(t, jsonField(parseJSON(resp), "id"))
	}
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "payment_term_id")

	orderID := createOrderForCustomer(t, SeedCustomerAccountID, nil)
	status, resp, err = apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{"payment_term_id": archivedID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "payment_term_id")
}

func TestForgePaymentTerms_ArchivingKeepsExistingReferences(t *testing.T) {
	t.Parallel()
	termID := jsonField(forgePaymentTerm(t, nil), "id")
	customerID := dashCustomersNew(t, "e2e-forge-pt-keep", map[string]any{"default_payment_term_id": termID})

	status, body, err := apiClient.Patch(paymentTermsPath+"/"+termID, map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"name": uniqueName("e2e-forge-pt-keep-u")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestForgePaymentTerms_DeleteInUse(t *testing.T) {
	t.Parallel()
	termID := jsonField(forgePaymentTerm(t, nil), "id")
	customerID := dashCustomersNew(t, "e2e-forge-pt-inuse", map[string]any{"default_payment_term_id": termID})

	status, body, err := apiClient.Delete(paymentTermsPath + "/" + termID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "customers")
	assert.Contains(t, jsonField(errObj, "message"), "archived")

	status, body, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"default_payment_term_id": SeedPaymentTermID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Delete(paymentTermsPath + "/" + termID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, map[string]any{"id": termID, "object": "payment_term", "deleted": true}, parseJSON(body))
}

func TestForgePaymentTerms_Preview9ReadsInactive(t *testing.T) {
	t.Parallel()
	id := jsonField(forgePaymentTerm(t, map[string]any{"status": "archived"}), "id")
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(paymentTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "inactive", jsonField(parseJSON(body), "status"))
}

// --- Shipping terms ---

func TestForgeShippingTerms_CreateEveryField(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-forge-st-all")
	created := createAndCleanup(t, shippingTermsPath+"?include=free_shipping.service_levels", map[string]any{
		"name":      name,
		"type":      "flat_rate",
		"status":    "archived",
		"flat_rate": forgeDollars("12.50"),
		"free_shipping": map[string]any{
			"minimum_order_value": forgeDollars("250"),
			"service_level_ids":   []string{SeedServiceLevelID},
		},
	})
	assertIDFormat(t, jsonField(created, "id"), "shtm")
	assert.Equal(t, "shipping_term", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "flat_rate", jsonField(created, "type"))
	assert.Equal(t, "archived", jsonField(created, "status"))
	assertShippingMoney(t, jsonObject(created, "flat_rate"), "12.50", "flat_rate")
	fs := jsonObject(created, "free_shipping")
	require.NotNil(t, fs)
	assertShippingMoney(t, jsonObject(fs, "minimum_order_value"), "250.00", "free_shipping.minimum_order_value")
	assert.Equal(t, "selected", jsonField(fs, "service_level_scope"))
	levels := jsonListData(fs, "service_levels")
	require.Len(t, levels, 1)
	assert.Equal(t, SeedServiceLevelID, jsonField(levels[0].(map[string]any), "id"))
	assertNilField(t, created, "owner")
	assert.NotContains(t, created, "minimum_order_value")
	assert.NotContains(t, created, "free_shipping_service_levels")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// Free shipping on every service level.
	all := forgeShippingTerm(t, map[string]any{"free_shipping": map[string]any{"minimum_order_value": forgeDollars("99")}})
	allFS := jsonObject(all, "free_shipping")
	assert.Equal(t, "all", jsonField(allFS, "service_level_scope"))
	assertNilField(t, allFS, "service_levels")
}

func TestForgeShippingTerms_ChargesFollowTheType(t *testing.T) {
	t.Parallel()
	post := func(body map[string]any) (int, []byte) {
		body["name"] = uniqueName("e2e-forge-st-rule")
		status, resp, err := apiClient.Post(shippingTermsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			id := jsonField(parseJSON(resp), "id")
			t.Cleanup(func() { _, _, _ = apiClient.Delete(shippingTermsPath + "/" + id) })
		}
		return status, resp
	}

	status, body := post(map[string]any{"type": "flat_rate"})
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "missing_field", "invalid_request_error"), "flat_rate")

	status, body = post(map[string]any{"type": "carrier_rate", "flat_rate": forgeDollars("5")})
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "flat_rate")

	status, body = post(map[string]any{"type": "free", "free_shipping": map[string]any{"minimum_order_value": forgeDollars("5")}})
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "free_shipping")

	status, body = post(map[string]any{"type": "carrier_rate", "free_shipping": map[string]any{"service_level_ids": []string{SeedServiceLevelID}}})
	requireStatus(t, 422, status, body)

	// Moving a term to free drops its free shipping; setting it in the same request is refused.
	term := forgeShippingTerm(t, map[string]any{"free_shipping": map[string]any{"minimum_order_value": forgeDollars("50")}})
	id := jsonField(term, "id")
	status, body, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"type": "free", "free_shipping": map[string]any{"minimum_order_value": forgeDollars("60")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	status, body, err = apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{"type": "free"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "free_shipping")
}

func TestForgeShippingTerms_NameIsUnique(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-forge-st-dup")
	forgeShippingTerm(t, map[string]any{"name": name})

	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{"name": strings.ToUpper(name), "type": "free"}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(body), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(shippingTermsPath + "/" + id) })
	}
	requireStatus(t, 409, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_exists", "invalid_request_error"), "name")

	other := forgeShippingTerm(t, nil)
	status, body, err = apiClient.Patch(shippingTermsPath+"/"+jsonField(other, "id"), map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_exists", "invalid_request_error")
}

func TestForgeShippingTerms_StatusesFilter(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-forge-st-filter")
	archived := forgeShippingTerm(t, map[string]any{"name": name, "status": "archived"})

	list, _, err := apiClient.GetList(shippingTermsPath, url.Values{"q": {name}, "statuses[]": {"archived"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, jsonField(archived, "id"), DataItemField(list.Data[0], "id"))

	list, _, err = apiClient.GetList(shippingTermsPath, url.Values{"q": {name}, "statuses[]": {"active"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)

	status, body, err := apiClient.Patch(shippingTermsPath+"/"+jsonField(archived, "id"), map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "active", jsonField(parseJSON(body), "status"))
}

func TestForgeShippingTerms_ArchivedCantBeNewlyAssigned(t *testing.T) {
	t.Parallel()
	archivedID := jsonField(forgeShippingTerm(t, map[string]any{"status": "archived"}), "id")

	body := validCustomerBody(uniqueName("e2e-forge-st-archcust"))
	body["default_shipping_term_id"] = archivedID
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + id) })
	}
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "default_shipping_term_id")

	order := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	order["shipping_term_id"] = archivedID
	status, resp, err = apiClient.Post(salesOrdersPath, order, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		deleteOrder(t, jsonField(parseJSON(resp), "id"))
	}
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "shipping_term_id")

	orderID := createOrderForCustomer(t, SeedCustomerAccountID, nil)
	status, resp, err = apiClient.Patch(salesOrdersPath+"/"+orderID, map[string]any{"shipping_term_id": archivedID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "shipping_term_id")
}

func TestForgeShippingTerms_DeleteInUse(t *testing.T) {
	t.Parallel()
	termID := jsonField(forgeShippingTerm(t, nil), "id")
	customerID := dashCustomersNew(t, "e2e-forge-st-inuse", map[string]any{"default_shipping_term_id": termID})

	status, body, err := apiClient.Delete(shippingTermsPath + "/" + termID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "customers")

	status, body, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"default_shipping_term_id": SeedShippingTermID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Delete(shippingTermsPath + "/" + termID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, map[string]any{"id": termID, "object": "shipping_term", "deleted": true}, parseJSON(body))
}

func TestForgeShippingTerms_Preview9Shape(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	status, body, err := preview.Post(shippingTermsPath+"?include=free_shipping_service_levels", map[string]any{
		"name":                            uniqueName("e2e-forge-st-p9"),
		"type":                            "carrier_rate_freight",
		"minimum_order_value":             forgeDollars("75"),
		"free_shipping_service_level_ids": []string{SeedServiceLevelID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	id := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(shippingTermsPath + "/" + id) })

	assert.Equal(t, "carrier_rate_freight", jsonField(got, "type"))
	assert.NotContains(t, got, "free_shipping")
	assert.Equal(t, "75.00", jsonField(jsonObject(got, "minimum_order_value"), "value"))
	levels := jsonListData(got, "free_shipping_service_levels")
	require.Len(t, levels, 1)
	assert.Equal(t, SeedServiceLevelID, jsonField(levels[0].(map[string]any), "id"))

	// The same term reads in the forge shape.
	status, body, err = apiClient.GetListRaw(shippingTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	forge := parseJSON(body)
	assert.Equal(t, "carrier_rate", jsonField(forge, "type"))
	assert.Equal(t, "selected", jsonField(jsonObject(forge, "free_shipping"), "service_level_scope"))

	status, body, err = preview.Patch(shippingTermsPath+"/"+id, map[string]any{"minimum_order_value": nil, "free_shipping_service_level_ids": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["minimum_order_value"])
}

// --- Units ---

func TestForgeUnits_CurrencyUnitsAreHidden(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitsPath+"/dollar", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"Dollar"}})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.NotEqual(t, "currency", DataItemField(item, "dimension"))
	}
}

func TestForgeUnits_AbbreviationMaxLength(t *testing.T) {
	t.Parallel()
	body := covCatalogUnitsCreateBody(uniqueName("e2e-forge-unit-long"), strings.Repeat("a", 33))
	status, resp, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "validation_failed", "invalid_request_error"), "abbreviation")
}

func TestForgeUnits_Preview9Shape(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	body := covCatalogUnitsCreateBody(uniqueName("e2e-forge-unit-p9"), uniqueName("p9"))
	delete(body, "dimension")
	body["type"] = "mass"
	body["is_base_unit"] = false
	status, resp, err := preview.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	got := parseJSON(resp)
	id := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(unitsPath + "/" + id) })
	assert.Equal(t, "mass", jsonField(got, "type"))
	assert.Equal(t, false, got["is_base_unit"])
	assert.NotContains(t, got, "dimension")

	status, resp, err = preview.GetListRaw(unitsPath+"/gram", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	assert.Equal(t, true, parseJSON(resp)["is_base_unit"])

	list, _, err := preview.GetList(unitsPath, url.Values{"type": {"time"}})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data)
	for _, item := range list.Data {
		assert.Equal(t, "time", DataItemField(item, "type"))
	}
}
