//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const addressesPath = "/v1/sales/addresses"

func TestAddresses_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr")
	phone := "555-123-4567"
	email := name + "@e2e-test.augno.com"

	// CREATE
	createResp, err := apiClient.PostFull(addressesPath, map[string]any{
		"name":          name,
		"phone":         phone,
		"email":         email,
		"is_drop_ship":  true,
		"street_line_1": "100 Test Ave",
		"street_line_2": "Suite 200",
		"locality":      "Denver",
		"state":         "CO",
		"postal_code":   "80202",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "address", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, phone, jsonField(created, "phone"))
	assert.Equal(t, email, jsonField(created, "email"))
	assert.Equal(t, "true", jsonField(created, "is_drop_ship"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	// Geolocation sub-resource
	geo := jsonObject(created, "geolocation")
	require.NotNil(t, geo, "geolocation should be present")
	assert.NotEmpty(t, jsonField(geo, "id"))
	assert.Equal(t, "geolocation", jsonField(geo, "object"))
	assert.Equal(t, "100 Test Ave", jsonField(geo, "street_line_1"))
	assert.Equal(t, "Suite 200", jsonField(geo, "street_line_2"))
	assert.Equal(t, "Denver", jsonField(geo, "locality"))
	assert.Equal(t, "CO", jsonField(geo, "state"))
	assert.Equal(t, "80202", jsonField(geo, "postal_code"))
	assert.Equal(t, "US", jsonField(geo, "country"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(addressesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))

	// UPDATE
	newName := uniqueName("e2e-addr-upd")
	newPhone := "555-987-6543"
	patchStatus, patchBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"name":  newName,
		"phone": newPhone,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, newPhone, jsonField(updated, "phone"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(addressesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus2, _, err := apiClient.GetListRaw(addressesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestAddresses_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-addr-allf")
	email := name + "@e2e-test.augno.com"
	createResp, err := apiClient.PostFull(addressesPath, map[string]any{
		"name":          name,
		"phone":         "555-000-1234",
		"email":         email,
		"is_drop_ship":  true,
		"street_line_1": "100 Create Ave",
		"street_line_2": "Suite 200",
		"locality":      "Denver",
		"state":         "CO",
		"postal_code":   "80202",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	assert.Equal(t, "address", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "555-000-1234", jsonField(got, "phone"))
	assert.Equal(t, email, jsonField(got, "email"))
	assert.Equal(t, "true", jsonField(got, "is_drop_ship"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	geo := jsonObject(got, "geolocation")
	require.NotNil(t, geo, "geolocation must be set after create")
	assert.NotEmpty(t, jsonField(geo, "id"))
	assert.Equal(t, "geolocation", jsonField(geo, "object"))
	assert.Equal(t, "100 Create Ave", jsonField(geo, "street_line_1"))
	assert.Equal(t, "Suite 200", jsonField(geo, "street_line_2"))
	assert.Equal(t, "Denver", jsonField(geo, "locality"))
	assert.Equal(t, "CO", jsonField(geo, "state"))
	assert.Equal(t, "80202", jsonField(geo, "postal_code"))
	assert.Equal(t, "US", jsonField(geo, "country"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-addr-allf-u")
	updatedEmail := updatedName + "@e2e-test.augno.com"
	patchStatus, patchBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"name":          updatedName,
		"phone":         "555-999-8888",
		"email":         updatedEmail,
		"is_drop_ship":  false,
		"street_line_1": "200 Update Blvd",
		"street_line_2": "Floor 3",
		"locality":      "Seattle",
		"state":         "WA",
		"postal_code":   "98101",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "555-999-8888", jsonField(updated, "phone"))
	assert.Equal(t, updatedEmail, jsonField(updated, "email"))
	assert.Equal(t, "false", jsonField(updated, "is_drop_ship"))

	updGeo := jsonObject(updated, "geolocation")
	require.NotNil(t, updGeo, "geolocation should be present after update")
	assert.Equal(t, "200 Update Blvd", jsonField(updGeo, "street_line_1"))
	assert.Equal(t, "Floor 3", jsonField(updGeo, "street_line_2"))
	assert.Equal(t, "Seattle", jsonField(updGeo, "locality"))
	assert.Equal(t, "WA", jsonField(updGeo, "state"))
	assert.Equal(t, "98101", jsonField(updGeo, "postal_code"))
	assert.Equal(t, "US", jsonField(updGeo, "country"), "country should be preserved")
}

func TestAddresses_GetByID(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(addressesPath+"/"+SeedAddressID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, SeedAddressID, jsonField(got, "id"))
	assert.Equal(t, "address", jsonField(got, "object"))
	assert.Equal(t, "Global Manufacturing Solutions", jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	geo := jsonObject(got, "geolocation")
	require.NotNil(t, geo, "geolocation should be present")
	assert.NotEmpty(t, jsonField(geo, "id"))
	assert.Equal(t, "geolocation", jsonField(geo, "object"))
	assert.Equal(t, "789 Mission St", jsonField(geo, "street_line_1"))
	assert.Equal(t, "San Francisco", jsonField(geo, "locality"))
	assert.Equal(t, "CA", jsonField(geo, "state"))
	assert.Equal(t, "94103", jsonField(geo, "postal_code"))
	assert.Equal(t, "US", jsonField(geo, "country"))
}

func TestAddresses_GetNotFound(t *testing.T) {
	t.Parallel()
	getStatus, _, err := apiClient.GetListRaw(addressesPath+"/ad_000000000000000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestAddresses_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 4, "should have at least 4 seeded addresses")
}

func TestAddresses_ListWithLimit(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestAddresses_ListPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(addressesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.True(t, page1.PageInfo.HasNextPage, "should have a next page")
	require.NotNil(t, page1.PageInfo.NextCursor)

	page2, _, err := apiClient.GetList(addressesPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "pages should return different items")
}

func TestAddresses_ListSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressesPath, url.Values{"q": {"Global Manufacturing"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "search for 'Global Manufacturing' should return at least 1 result")
}

func TestAddresses_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressesPath, url.Values{"q": {"zzzznotanaddress99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestAddresses_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-addr")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"name":    name,
		"country": "US",
	}

	status1, body1, err := apiClient.Post(addressesPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(addressesPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(addressesPath + "/" + id1)
}

func TestAddresses_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    "",
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestAddresses_CreateValidation_MissingCountry(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name": uniqueName("e2e-addr-nocountry"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing country should return 400 or 422, got %d: %s", status, string(body))
}

func TestAddresses_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-dup-addr")

	status1, body1, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	id2 := jsonField(parseJSON(body2), "id")

	assert.NotEqual(t, id1, id2, "duplicate names should create separate addresses")

	apiClient.Delete(addressesPath + "/" + id1)
	apiClient.Delete(addressesPath + "/" + id2)
}

func TestAddresses_UpdateAddressFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-upd-fields")

	// Create
	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":          name,
		"street_line_1": "500 Original St",
		"locality":      "Austin",
		"state":         "TX",
		"postal_code":   "73301",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Update address-level fields only
	newName := uniqueName("e2e-addr-upd-fields2")
	patchStatus, patchBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"name":  newName,
		"phone": "555-111-2222",
		"email": newName + "@test.augno.com",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "555-111-2222", jsonField(updated, "phone"))
	assert.Equal(t, newName+"@test.augno.com", jsonField(updated, "email"))

	// Geolocation should be preserved
	geo := jsonObject(updated, "geolocation")
	require.NotNil(t, geo, "geolocation should still be present after address-only update")
	assert.Equal(t, "500 Original St", jsonField(geo, "street_line_1"))
	assert.Equal(t, "Austin", jsonField(geo, "locality"))
	assert.Equal(t, "US", jsonField(geo, "country"))

	apiClient.Delete(addressesPath + "/" + id)
}

func TestAddresses_UpdateGeolocationFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-upd-geo")

	// Create
	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":          name,
		"street_line_1": "600 Old Rd",
		"locality":      "Miami",
		"state":         "FL",
		"postal_code":   "33101",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Update geolocation fields
	patchStatus, patchBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"street_line_1": "700 New Blvd",
		"locality":      "Seattle",
		"state":         "WA",
		"postal_code":   "98101",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	// Name should be preserved
	assert.Equal(t, name, jsonField(updated, "name"))

	// Geolocation should reflect updates
	geo := jsonObject(updated, "geolocation")
	require.NotNil(t, geo, "geolocation should be present")
	assert.Equal(t, "700 New Blvd", jsonField(geo, "street_line_1"))
	assert.Equal(t, "Seattle", jsonField(geo, "locality"))
	assert.Equal(t, "WA", jsonField(geo, "state"))
	assert.Equal(t, "98101", jsonField(geo, "postal_code"))
	assert.Equal(t, "US", jsonField(geo, "country"))

	apiClient.Delete(addressesPath + "/" + id)
}

func TestAddresses_DeleteInUse(t *testing.T) {
	t.Parallel()
	// The seeded address is used as billing address on sales orders, invoices, and account defaults.
	delStatus, delBody, err := apiClient.Delete(addressesPath + "/" + SeedAddressID)
	require.NoError(t, err)
	assert.True(t, delStatus == 400 || delStatus == 409 || delStatus == 422,
		"Deleting in-use address should return 400, 409, or 422, got %d: %s", delStatus, string(delBody))
}
