//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const paymentTermsPath = "/v1/finance/payment-terms"

// --- List ---

func TestPaymentTerms_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(paymentTermsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded payment term")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedPaymentTermID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded payment term should appear in list")
}

func TestPaymentTerms_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(paymentTermsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "payment_term", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "status"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestPaymentTerms_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(paymentTermsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestPaymentTerms_ListCursorPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(paymentTermsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough payment terms for pagination test")
		return
	}
	require.NotNil(t, page1.PageInfo.NextPageURL, "next_page_url should be set when has_next_page is true")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	requirePageLen(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should return a different item than page 1")
}

func TestPaymentTerms_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(paymentTermsPath, url.Values{"q": {"Net"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Net' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "net"),
			"Search result %q should contain 'net'", name,
		)
	}
}

func TestPaymentTerms_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(paymentTermsPath, url.Values{"q": {"zzzznotapayterm99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestPaymentTerms_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(paymentTermsPath+"/"+SeedPaymentTermID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedPaymentTermID, jsonField(got, "id"))
	assert.Equal(t, "payment_term", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "status"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestPaymentTerms_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(paymentTermsPath+"/pytm_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- CRUD ---

func TestPaymentTerms_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-payterm")

	// Create
	createResp, err := apiClient.PostFull(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "payment_term", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(paymentTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-payterm-upd")
	patchStatus, patchBody, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(paymentTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(paymentTermsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(paymentTermsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- Create ---

func TestPaymentTerms_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE ──
	name := uniqueName("e2e-pt-allf")
	createResp, err := apiClient.PostFull(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(paymentTermsPath + "/" + id)

	assert.Equal(t, "payment_term", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// ── UPDATE ──
	updatedName := uniqueName("e2e-pt-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": updatedName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "active", jsonField(updated, "status"), "status should be preserved")
}

func TestPaymentTerms_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-payterm-shape")
	createResp, err := apiClient.PostFull(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "payment_term", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "active", jsonField(created, "status"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(paymentTermsPath + "/" + id)
}

func TestPaymentTerms_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(paymentTermsPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

// --- Idempotency ---

func TestPaymentTerms_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-payterm")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(paymentTermsPath + "/" + id1)
}

func TestPaymentTerms_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-payterm-idem-upd")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-payterm-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(paymentTermsPath + "/" + id)
}

// --- Update ---

func TestPaymentTerms_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-payterm-pname")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-payterm-pname2")
	patchStatus, patchBody, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "active", jsonField(patched, "status"), "status should be preserved when only name is updated")

	apiClient.Delete(paymentTermsPath + "/" + id)
}

func TestPaymentTerms_UpdateEmptyBodyRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-payterm-empty")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, _, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "Empty PATCH body should return 400")

	apiClient.Delete(paymentTermsPath + "/" + id)
}

func TestPaymentTerms_UpdateDefaultTermFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(paymentTermsPath+"/"+SeedDefaultPaymentTermID, map[string]any{
		"name": uniqueName("e2e-default-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Updating a default payment term should fail, got %d", status)
}

// --- Delete ---

func TestPaymentTerms_DeleteDefaultTermFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(paymentTermsPath + "/" + SeedDefaultPaymentTermID)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Deleting a default payment term should fail, got %d", status)
}

func TestPaymentTerms_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-payterm-deldel")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(paymentTermsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Second delete should fail
	status2, _, err := apiClient.Delete(paymentTermsPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted payment term should return 404 or 410, got %d", status2)
}

// --- Expandable Fields ---

func TestPaymentTerms_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(paymentTermsPath+"/"+SeedPaymentTermID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")

	// Test on List
	list, _, err := apiClient.GetList(paymentTermsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null on list items without ?include=owner")
	}
}

func TestPaymentTerms_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(paymentTermsPath+"/"+SeedPaymentTermID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}
