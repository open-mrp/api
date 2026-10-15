//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forgeCarrier creates a will_call carrier with the extra fields given and deletes it when the test ends.
func forgeCarrier(t *testing.T, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"name": uniqueName("e2e-forge-carrier"), "type": "will_call"}
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, carriersPath, body)
}

func TestForgeCarriers_CreateEveryField(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-forge-carr-all")
	got := createAndCleanup(t, carriersPath, map[string]any{
		"name":              name,
		"type":              "ltl",
		"scac":              "ABCD",
		"account_number":    "LTL-001",
		"portal_visibility": "hidden",
		"status":            "archived",
	})

	assertIDFormat(t, jsonField(got, "id"), "cr")
	assert.Equal(t, "carrier", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "ltl", jsonField(got, "type"))
	assert.Equal(t, "ABCD", jsonField(got, "scac"))
	assert.Equal(t, "LTL-001", jsonField(got, "account_number"))
	assert.Equal(t, "hidden", jsonField(got, "portal_visibility"))
	assert.Equal(t, "archived", jsonField(got, "status"))
	assertNilField(t, got, "default_service_level")
	assertNilField(t, got, "owner")
	assertNilField(t, got, "service_levels")
	assert.NotContains(t, got, "code")
	assert.NotContains(t, got, "deleted_at")
	assert.NotContains(t, got, "customer_portal_visibility")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func TestForgeCarriers_TypeIsRequired(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{"name": uniqueName("e2e-forge-carr-notype")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "missing_field", "invalid_request_error"), "type")
}

func TestForgeCarriers_SCAC(t *testing.T) {
	t.Parallel()

	for _, bad := range []string{"A", "ABCDE", "abcd", "AB1"} {
		status, body, err := apiClient.Post(carriersPath, map[string]any{"name": uniqueName("e2e-forge-carr-scac"), "type": "ltl", "scac": bad}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "scac")
	}

	carrier := forgeCarrier(t, nil)
	assertNilField(t, carrier, "scac")
	path := carriersPath + "/" + jsonField(carrier, "id")

	status, body, err := apiClient.Patch(path, map[string]any{"scac": "FDEG"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "FDEG", jsonField(parseJSON(body), "scac"))

	status, body, err = apiClient.Patch(path, map[string]any{"name": uniqueName("e2e-forge-carr-scac-keep")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "FDEG", jsonField(parseJSON(body), "scac"), "an update that leaves scac out keeps it")

	status, body, err = apiClient.Patch(path, map[string]any{"scac": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "scac")
}

func TestForgeCarriers_StatusAndFilter(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-forge-carr-st")
	active := forgeCarrier(t, map[string]any{"name": prefix + "-a"})
	archived := forgeCarrier(t, map[string]any{"name": prefix + "-b", "status": "archived"})
	assert.Equal(t, "active", jsonField(active, "status"))

	ids := func(statuses ...string) []string {
		params := url.Values{"q": {prefix}}
		for _, s := range statuses {
			params.Add("statuses[]", s)
		}
		list, _, err := apiClient.GetList(carriersPath, params)
		require.NoError(t, err)
		var out []string
		for _, row := range list.Data {
			out = append(out, jsonField(parseJSON(row), "id"))
		}
		return out
	}
	assert.ElementsMatch(t, []string{jsonField(active, "id"), jsonField(archived, "id")}, ids())
	assert.ElementsMatch(t, []string{jsonField(archived, "id")}, ids("archived"))
	assert.ElementsMatch(t, []string{jsonField(active, "id")}, ids("active"))

	status, body, err := apiClient.Patch(carriersPath+"/"+jsonField(archived, "id"), map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "active", jsonField(parseJSON(body), "status"))

	status, body, err = apiClient.GetListRaw(carriersPath, url.Values{"statuses[]": {"retired"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestForgeCarriers_ArchivedCantBeNewlyAssigned(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, map[string]any{"status": "archived"})

	body := validCustomerBody(uniqueName("e2e-forge-carr-archcust"))
	customerDefaults(body)["carrier_id"] = jsonField(carrier, "id")
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + id) })
	}
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "default_carrier_id")

	// A customer already on the carrier keeps it after the carrier is archived.
	live := forgeCarrier(t, nil)
	customerID := dashCustomersNew(t, "e2e-forge-carr-keep", map[string]any{"default_carrier_id": jsonField(live, "id")})
	status, resp, err = apiClient.Patch(carriersPath+"/"+jsonField(live, "id"), map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	status, resp, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"name": uniqueName("e2e-forge-carr-keep-u")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	carrierID, _ := customerDefaultRouting(t, customerID)
	assert.Equal(t, jsonField(live, "id"), carrierID)
}

func TestForgeCarriers_DeleteInUse(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, nil)
	carrierID := jsonField(carrier, "id")
	dashCustomersNew(t, "e2e-forge-carr-inuse", map[string]any{"default_carrier_id": carrierID})

	status, body, err := apiClient.Delete(carriersPath + "/" + carrierID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "customers")

	status, body, err = apiClient.GetListRaw(carriersPath+"/"+carrierID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestForgeCarriers_DeleteReturnsStub(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{"name": uniqueName("e2e-forge-carr-del"), "type": "other"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	assert.Equal(t, "other", jsonField(parseJSON(body), "type"))

	// Its service levels go with it.
	slID := covOperationsCarriersCreateServiceLevel(t, serviceLevelsPath(id), nil)

	status, body, err = apiClient.Delete(carriersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDeletedStub(t, body, id, "carrier")

	status, _, err = apiClient.GetListRaw(serviceLevelsPath(id)+"/"+slID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
	status, _, err = apiClient.Delete(carriersPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, status)
}

func TestForgeCarriers_EmbeddedServiceLevelsPage(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, nil)
	carrierID := jsonField(carrier, "id")
	for i := 0; i < 11; i++ {
		covOperationsCarriersCreateServiceLevel(t, serviceLevelsPath(carrierID), map[string]any{"name": fmt.Sprintf("e2e-forge-sl-%02d", i)})
	}

	status, body, err := apiClient.GetListRaw(carriersPath+"/"+carrierID, url.Values{"include[]": {"service_levels"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	levels := jsonObject(parseJSON(body), "service_levels")
	require.NotNil(t, levels)
	data, _ := levels["data"].([]any)
	assert.Len(t, data, 10)
	page := jsonObject(levels, "page_info")
	require.NotNil(t, page)
	assert.Equal(t, serviceLevelsPath(carrierID), jsonField(page, "next_page_url"))

	// The embedded ten are the list's first page.
	list, _, err := apiClient.GetList(serviceLevelsPath(carrierID), url.Values{"limit": {"10"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 10)
	for i, row := range list.Data {
		assert.Equal(t, jsonField(parseJSON(row), "id"), jsonField(data[i].(map[string]any), "id"))
	}
}

func TestForgeServiceLevels_EveryFieldAndInclude(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, nil)
	carrierID := jsonField(carrier, "id")
	id := covOperationsCarriersCreateServiceLevel(t, serviceLevelsPath(carrierID), map[string]any{"estimated_transit_days": 2})

	status, body, err := apiClient.GetListRaw(serviceLevelsPath(carrierID)+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertIDFormat(t, id, "silv")
	assert.Equal(t, "service_level", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "code"))
	assert.Equal(t, "visible", jsonField(got, "portal_visibility"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, "2", jsonField(got, "estimated_transit_days"))
	assertNilField(t, got, "owner")
	assertNilField(t, got, "carrier")
	for _, gone := range []string{"service_level_token", "is_default", "default_transit_days", "customer_portal_visibility"} {
		assert.NotContains(t, got, gone)
	}

	status, body, err = apiClient.GetListRaw(serviceLevelsPath(carrierID)+"/"+id, url.Values{"include[]": {"carrier"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, carrierID, jsonField(jsonObject(parseJSON(body), "carrier"), "id"))
}

func TestForgeServiceLevels_StatusFilter(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, nil)
	path := serviceLevelsPath(jsonField(carrier, "id"))
	activeID := covOperationsCarriersCreateServiceLevel(t, path, nil)
	archivedID := covOperationsCarriersCreateServiceLevel(t, path, map[string]any{"status": "archived"})

	ids := func(status string) []string {
		list, _, err := apiClient.GetList(path, url.Values{"statuses[]": {status}})
		require.NoError(t, err)
		var out []string
		for _, row := range list.Data {
			out = append(out, jsonField(parseJSON(row), "id"))
		}
		return out
	}
	assert.Equal(t, []string{activeID}, ids("active"))
	assert.Equal(t, []string{archivedID}, ids("archived"))
}

func TestForgeServiceLevels_ArchivedCantBeNewlyAssigned(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, nil)
	carrierID := jsonField(carrier, "id")
	archivedID := covOperationsCarriersCreateServiceLevel(t, serviceLevelsPath(carrierID), map[string]any{"status": "archived"})

	body := validCustomerBody(uniqueName("e2e-forge-sl-archcust"))
	customerDefaults(body)["carrier_id"] = carrierID
	customerDefaults(body)["service_level_id"] = archivedID
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + id) })
	}
	requireStatus(t, 422, status, resp)
	assertErrorParam(t, requireErrorResponse(t, resp, "resource_archived", "invalid_request_error"), "default_service_level_id")
}

func TestForgeServiceLevels_DeleteInUse(t *testing.T) {
	t.Parallel()
	carrier := forgeCarrier(t, nil)
	carrierID := jsonField(carrier, "id")
	slID := covOperationsCarriersCreateServiceLevel(t, serviceLevelsPath(carrierID), nil)
	dashCustomersNew(t, "e2e-forge-sl-inuse", map[string]any{"default_carrier_id": carrierID, "default_service_level_id": slID})

	status, body, err := apiClient.Delete(serviceLevelsPath(carrierID) + "/" + slID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assert.Contains(t, jsonField(requireErrorResponse(t, body, "resource_in_use", "invalid_request_error"), "message"), "customers")

	// Archiving is the way out.
	status, body, err = apiClient.Patch(serviceLevelsPath(carrierID)+"/"+slID, map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "archived", jsonField(parseJSON(body), "status"))
}
