//go:build e2e

package api_test

import (
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createLocation creates a location and deletes it when the test ends (children first: cleanups run in reverse).
func createLocation(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Post(locationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	created := parseJSON(raw)
	id := jsonField(created, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(locationsPath + "/" + id) })
	return created
}

// createSiteAddress creates one of the account's addresses for a site, deleted after the site.
func createSiteAddress(t *testing.T) string {
	t.Helper()
	resp, err := apiClient.PostFull(addressesPath, map[string]any{
		"name": uniqueName("e2e-site-addr"), "type": "standard", "line_1": "1 Mill Rd", "city": "Burlington",
		"state": "NC", "postal_code": "27215", "country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	id := jsonField(parseJSON(resp.Body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(addressesPath + "/" + id) })
	return id
}

// requireDefaultSite asserts the account was born with its default site: an active, top-level site named wantName,
// at the account's shipping address (none when the account has none). Returns the site's ID.
func requireDefaultSite(t *testing.T, accountID, wantName string) string {
	t.Helper()
	var siteID, name, typeCode string
	var isActive, topLevel, atShippingAddress bool
	var shippingAddressID sql.NullString
	require.NoError(t, authDB(t).QueryRow(
		`SELECT sl.id, sl.name, sl.storage_location_type_code, sl.is_active, sl.parent_id IS NULL,
		        sl.address_id <=> a.default_shipping_address_id, a.default_shipping_address_id
		   FROM account a JOIN storage_location sl ON sl.id = a.default_site_id AND sl.account_id = a.id
		  WHERE a.id = ?`, accountID,
	).Scan(&siteID, &name, &typeCode, &isActive, &topLevel, &atShippingAddress, &shippingAddressID),
		"account %s has a default site", accountID)
	assert.Equal(t, wantName, name)
	assert.Equal(t, "site", typeCode)
	assert.True(t, isActive)
	assert.True(t, topLevel)
	assert.True(t, atShippingAddress, "the site is at the account's shipping address (%v)", shippingAddressID)
	return siteID
}

// --- Default site on new accounts ---

func TestForgeLocations_NewSandboxHasADefaultSite(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"blank", "seeded"} {
		name := uniqueName("e2e-site-sb-" + mode)
		status, raw, err := apiClient.Post(sandboxesPath, map[string]any{"name": name, "mode": mode}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, raw)
		typeID := jsonField(parseJSON(raw), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(sandboxesPath + "/" + typeID) })

		var accountID string
		require.NoError(t, authDB(t).QueryRow("SELECT account_id FROM sandbox_account WHERE type_id = ?", typeID).Scan(&accountID))
		siteID := requireDefaultSite(t, accountID, name)

		if mode == "seeded" {
			// The seeded building lands inside the default site, which stays the only site.
			var parentID sql.NullString
			eventually(t, 60*time.Second, time.Second, func() error {
				return authDB(t).QueryRow("SELECT parent_id FROM storage_location WHERE account_id = ? AND name = 'Main Building'", accountID).Scan(&parentID)
			})
			assert.Equal(t, siteID, parentID.String)
			var sites int
			require.NoError(t, authDB(t).QueryRow("SELECT COUNT(*) FROM storage_location WHERE account_id = ? AND storage_location_type_code = 'site'", accountID).Scan(&sites))
			assert.Equal(t, 1, sites)
		}
	}
}

func TestForgeLocations_NewSupplierAccountHasADefaultSite(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-site-sup")
	status, raw, err := apiClient.Post(suppliersPath, map[string]any{
		"name": name, "number": uniqueName("SUP"),
		"bill_to_address": map[string]any{"name": name, "line_1": "9 Supply Way", "city": "Dalton", "state": "GA", "postal_code": "30720", "country": "US"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	supplierAccountID := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(suppliersPath + "/" + supplierAccountID) })

	requireDefaultSite(t, supplierAccountID, name)
}

// --- Sites ---

func TestForgeLocations_SiteAllFields(t *testing.T) {
	t.Parallel()
	addressID := createSiteAddress(t)
	name := uniqueName("e2e-site")

	site := createLocation(t, map[string]any{"name": name, "type": "site", "address_id": addressID})
	id := jsonField(site, "id")
	assert.Equal(t, "location", jsonField(site, "object"))
	assert.Equal(t, name, jsonField(site, "name"))
	assert.Equal(t, "site", jsonField(site, "type"))
	assert.Equal(t, "active", jsonField(site, "status"), "status defaults to active")
	assertNilField(t, site, "address")
	assertNilField(t, site, "parent")
	assertNilField(t, site, "children")
	assertValidTimestamp(t, jsonField(site, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(site, "updated_at"), "updated_at")

	status, raw, err := apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include[]": {"address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	address := jsonObject(parseJSON(raw), "address")
	require.NotNil(t, address, "address expands with include: %s", raw)
	assert.Equal(t, addressID, jsonField(address, "id"))
	assertObjectField(t, address, "address")

	// A building inside the site; the site lists it as a child.
	building := createLocation(t, map[string]any{"name": uniqueName("e2e-site-bldg"), "type": "building", "parent_id": id})
	status, raw, err = apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include[]": {"children"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	children := jsonObject(parseJSON(raw), "children")
	require.NotNil(t, children)
	data, _ := children["data"].([]any)
	require.Len(t, data, 1)
	assert.Equal(t, jsonField(building, "id"), jsonField(data[0].(map[string]any), "id"))

	// Clearing the address.
	status, raw, err = apiClient.Patch(locationsPath+"/"+id, map[string]any{"address_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	status, raw, err = apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include[]": {"address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertNilField(t, parseJSON(raw), "address")
}

func TestForgeLocations_SiteRules(t *testing.T) {
	t.Parallel()
	addressID := createSiteAddress(t)
	parent := createLocation(t, map[string]any{"name": uniqueName("e2e-site-rule-p"), "type": "building"})
	site := createLocation(t, map[string]any{"name": uniqueName("e2e-site-rule-s"), "type": "site"})

	for _, tc := range []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"site under a parent", map[string]any{"name": uniqueName("e2e-site-rule"), "type": "site", "parent_id": jsonField(parent, "id")}, "parent_id"},
		{"address on a bin", map[string]any{"name": uniqueName("e2e-site-rule"), "type": "bin", "address_id": addressID}, "address_id"},
		{"site as a child", map[string]any{"name": uniqueName("e2e-site-rule"), "type": "building", "child_ids": []string{jsonField(site, "id")}}, "child_ids"},
		{"unknown address", map[string]any{"name": uniqueName("e2e-site-rule"), "type": "site", "address_id": "adr_doesnotexist000"}, "address_id"},
	} {
		status, raw, err := apiClient.Post(locationsPath, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, raw)
		errObj := requireErrorResponse(t, raw, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, tc.param)
	}

	// Turning a child into a site needs it detached first.
	child := createLocation(t, map[string]any{"name": uniqueName("e2e-site-rule-c"), "type": "section", "parent_id": jsonField(parent, "id")})
	status, raw, err := apiClient.Patch(locationsPath+"/"+jsonField(child, "id"), map[string]any{"type": "site"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "parent_id")

	status, raw, err = apiClient.Patch(locationsPath+"/"+jsonField(child, "id"), map[string]any{"type": "site", "parent_id": nil, "address_id": addressID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "site", jsonField(parseJSON(raw), "type"))
}

func TestForgeLocations_DefaultSiteStaysAnActiveSite(t *testing.T) {
	t.Parallel()

	for _, body := range []map[string]any{{"status": "archived"}, {"type": "building"}} {
		status, raw, err := apiClient.Patch(locationsPath+"/"+SeedLocationParentID, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, raw)
	}

	status, raw, err := apiClient.Delete(locationsPath + "/" + SeedLocationParentID)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	msg := jsonField(requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error"), "message")
	assert.Contains(t, msg, "account settings")
	assert.Contains(t, msg, "child locations")

	status, raw, err = apiClient.GetListRaw(locationsPath+"/"+SeedLocationParentID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assert.Equal(t, "site", jsonField(got, "type"))
	assert.Equal(t, "active", jsonField(got, "status"))
}

// --- Status ---

func TestForgeLocations_ArchiveAndStatusFilter(t *testing.T) {
	t.Parallel()
	archived := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-arch"), "type": "rack", "status": "archived"})
	id := jsonField(archived, "id")
	assert.Equal(t, "archived", jsonField(archived, "status"))

	found := listFindByField(t, locationsPath, url.Values{"statuses[]": {"archived"}, "q": {jsonField(archived, "name")}}, "id", id)
	assert.NotNil(t, found, "statuses[]=archived lists it")
	list, _, err := apiClient.GetList(locationsPath, url.Values{"statuses[]": {"active"}, "q": {jsonField(archived, "name")}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)

	// An archived location can't be a new parent.
	status, raw, err := apiClient.Post(locationsPath, map[string]any{"name": uniqueName("e2e-loc-arch-c"), "type": "bin", "parent_id": id}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "parent_id")

	status, raw, err = apiClient.Patch(locationsPath+"/"+id, map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "active", jsonField(parseJSON(raw), "status"))
	assert.Equal(t, "rack", jsonField(parseJSON(raw), "type"), "a status-only update keeps the type")
}

// --- Hierarchy ---

func TestForgeLocations_ParentFilterAndEmbeddedChildrenPage(t *testing.T) {
	t.Parallel()
	parent := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-many"), "type": "aisle"})
	parentID := jsonField(parent, "id")
	childIDs := map[string]bool{}
	for range 11 {
		child := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-many-c"), "type": "bin", "parent_id": parentID})
		childIDs[jsonField(child, "id")] = true
	}

	status, raw, err := apiClient.GetListRaw(locationsPath+"/"+parentID, url.Values{"include[]": {"children"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	children := jsonObject(parseJSON(raw), "children")
	require.NotNil(t, children)
	data, _ := children["data"].([]any)
	assert.Len(t, data, 10, "an embedded list holds the first 10")
	pageInfo := jsonObject(children, "page_info")
	require.NotNil(t, pageInfo)
	next := jsonField(pageInfo, "next_page_url")
	assert.True(t, strings.Contains(next, "/v1/operations/locations?") && strings.Contains(next, url.QueryEscape("parent_ids[]")+"="+parentID),
		"next_page_url lists the children: %s", next)

	seen := map[string]bool{}
	params := url.Values{"parent_ids[]": {parentID}, "limit": {"100"}}
	list, _, err := apiClient.GetList(locationsPath, params)
	require.NoError(t, err)
	for _, item := range list.Data {
		seen[DataItemField(item, "id")] = true
	}
	assert.Equal(t, childIDs, seen, "parent_ids[] lists exactly the direct children")
}

func TestForgeLocations_NoCycles(t *testing.T) {
	t.Parallel()
	top := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-cyc-t"), "type": "building"})
	mid := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-cyc-m"), "type": "section", "parent_id": jsonField(top, "id")})
	low := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-cyc-l"), "type": "shelf", "parent_id": jsonField(mid, "id")})

	status, raw, err := apiClient.Patch(locationsPath+"/"+jsonField(top, "id"), map[string]any{"parent_id": jsonField(low, "id")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "parent_id")

	status, raw, err = apiClient.Patch(locationsPath+"/"+jsonField(low, "id"), map[string]any{"child_ids": []string{jsonField(top, "id")}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "invalid_format", "invalid_request_error"), "child_ids")
}

// --- Delete ---

func TestForgeLocations_DeleteInUseAndStub(t *testing.T) {
	t.Parallel()
	parent := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-del-p"), "type": "building"})
	child := createLocation(t, map[string]any{"name": uniqueName("e2e-loc-del-c"), "type": "bin", "parent_id": jsonField(parent, "id")})

	status, raw, err := apiClient.Delete(locationsPath + "/" + jsonField(parent, "id"))
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	msg := jsonField(requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error"), "message")
	assert.Contains(t, msg, "child locations")
	assert.Contains(t, msg, "Archive the location instead")

	// The seeded building holds a delivery.
	status, raw, err = apiClient.Delete(locationsPath + "/" + SeedLocationID)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	assert.Contains(t, jsonField(requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error"), "message"), "deliveries")

	status, raw, err = apiClient.Delete(locationsPath + "/" + jsonField(child, "id"))
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertDeletedStub(t, raw, jsonField(child, "id"), "location")
}

// --- Version compatibility ---

func TestVersionCompatLocations_Preview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	// The default site reads as the building it was.
	status, raw, err := preview.GetListRaw(locationsPath+"/"+SeedLocationParentID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "building", jsonField(parseJSON(raw), "type"))

	// Location types stay for preview.9, without site; forge has no such route.
	list, _, err := preview.GetList(locationTypesPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.NotEqual(t, "site", DataItemField(item, "code"))
	}
	status, _, err = preview.GetListRaw(locationTypesPath+"/site", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)

	status, _, err = apiClient.GetListRaw(locationTypesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}
