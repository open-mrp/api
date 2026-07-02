//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file adds targeted coverage gaps identified for /v1/finance/payment-terms
// on top of the existing tests/e2e/api/crud_payment_terms_test.go suite: the
// duplicate-name 409 business rule (create + update), per-field validation
// (empty/too-long/null name), nested owner.account expandable hydration on
// both GET and List, response-shape id-prefix assertion, DELETE-not-found,
// unknown include value rejection, and pinned (non-loose) status codes for
// mutating a system-owned default payment term.

// --- Validation: Create ---

func TestCovFinancePaymentTerms_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "empty-string name should return 400: %s", string(body))

	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovFinancePaymentTerms_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("a", 256)
	status, body, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": longName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "256-char name should return 400: %s", string(body))

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovFinancePaymentTerms_CreateValidation_NameWrongType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": 123,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "numeric name should return 400: %s", string(body))

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// --- Validation: Duplicate name conflict (409) ---

func TestCovFinancePaymentTerms_CreateDuplicateNameConflict(t *testing.T) {
	t.Parallel()

	// Collide with an account-owned seed term.
	status1, body1, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": "Net 30",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status1, "duplicate of account-owned term name should return 409: %s", string(body1))
	errObj1 := requireErrorResponse(t, body1, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj1, "name")

	// Collide with a system-owned default term (proves cross-owner uniqueness scoping).
	status2, body2, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": "Due on Receipt",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status2, "duplicate of system-owned default term name should return 409: %s", string(body2))
	errObj2 := requireErrorResponse(t, body2, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj2, "name")
}

func TestCovFinancePaymentTerms_UpdateDuplicateNameConflict(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-pt-updconflict")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(paymentTermsPath + "/" + id)

	// Rename the scratch term to collide with a pre-existing seed term.
	patchStatus, patchBody, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": "Net 30",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, patchStatus, "update to a name that collides with an existing term should return 409: %s", string(patchBody))
	errObj := requireErrorResponse(t, patchBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")

	// The scratch term's own name must be unchanged after the failed rename.
	getStatus, getBody, err := apiClient.GetListRaw(paymentTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"), "name should be unchanged after a rejected duplicate-name update")
}

// --- Validation: Update ---

func TestCovFinancePaymentTerms_UpdateValidation_EmptyName(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-pt-updempty")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(paymentTermsPath + "/" + id)

	status, body, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "empty-string name on update should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovFinancePaymentTerms_UpdateValidation_NameTooLong(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-pt-updlong")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(paymentTermsPath + "/" + id)

	longName := strings.Repeat("b", 256)
	status, body, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": longName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "256-char name on update should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovFinancePaymentTerms_UpdateValidation_NullNameRejected(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-pt-updnull")
	createStatus, createBody, err := apiClient.Post(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(paymentTermsPath + "/" + id)

	// name is field.Optional[string] (not Clearable) -- explicit null must be
	// rejected since the underlying column is non-nullable.
	status, body, err := apiClient.Patch(paymentTermsPath+"/"+id, map[string]any{
		"name": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "explicit null name on update should be rejected (non-clearable field): %s", string(body))
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")

	// Verify the name was left unchanged.
	getStatus, getBody, err := apiClient.GetListRaw(paymentTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))
}

// --- Delete ---

func TestCovFinancePaymentTerms_DeleteNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(paymentTermsPath + "/pytm_000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status, "DELETE of a nonexistent payment term should return 404: %s", string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- System-owned default term: pinned status codes (tightened from the loose ---
// --- 400||403||409||422 assertions in crud_payment_terms_test.go) ---

func TestCovFinancePaymentTerms_UpdateDefaultTermPinnedStatus(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(paymentTermsPath+"/"+SeedDefaultPaymentTermID, map[string]any{
		"name": uniqueName("e2e-default-upd-pinned"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "updating a default payment term should return exactly 400: %s", string(body))
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovFinancePaymentTerms_DeleteDefaultTermPinnedStatus(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(paymentTermsPath + "/" + SeedDefaultPaymentTermID)
	require.NoError(t, err)
	assert.Equal(t, 400, status, "deleting a default payment term should return exactly 400: %s", string(body))
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

// --- Response shape ---

func TestCovFinancePaymentTerms_CreateResponseShape_IDFormat(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pt-idformat")
	createResp, err := apiClient.PostFull(paymentTermsPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	defer apiClient.Delete(paymentTermsPath + "/" + id)

	assertIDFormat(t, id, "pytm")
	assertObjectField(t, created, "payment_term")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	assertNilField(t, created, "owner")
}

// --- Expandable: nested owner.account ---

func TestCovFinancePaymentTerms_IncludeOwnerAccount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(paymentTermsPath+"/"+SeedPaymentTermID, url.Values{"include": {"owner.account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedPaymentTermID, jsonField(got, "id"))
	assert.Equal(t, "Net 30", jsonField(got, "name"))

	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner.account")
	assertObjectField(t, owner, "owner")
	assert.Equal(t, "account", jsonField(owner, "type"), "SeedPaymentTermID is account-owned")

	account := jsonObject(owner, "account")
	require.NotNil(t, account, "owner.account should be populated with ?include=owner.account for an account-owned term")
	assertObjectField(t, account, "account")
	assert.Equal(t, SeedAccountID, jsonField(account, "id"))
}

func TestCovFinancePaymentTerms_IncludeOwnerAccount_SystemOwnedIsNil(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(paymentTermsPath+"/"+SeedDefaultPaymentTermID, url.Values{"include": {"owner.account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner.account")
	assert.Equal(t, "system", jsonField(owner, "type"), "SeedDefaultPaymentTermID is system-owned")
	assertNilField(t, owner, "account")
}

// --- Expandable: list-level hydration (owner and nested owner.account) ---

func TestCovFinancePaymentTerms_ListIncludeOwner(t *testing.T) {
	t.Parallel()
	// listFindByField pages through the full list (not just the first page) so
	// this holds even once the seed row falls off the newest-first front page.
	item := listFindByField(t, paymentTermsPath, url.Values{"include": {"owner"}}, "id", SeedPaymentTermID)
	require.NotNil(t, item, "SeedPaymentTermID should be present in the list to verify owner hydration")

	m := parseJSON(item)
	owner := jsonObject(m, "owner")
	require.NotNil(t, owner, "owner should be populated on list items with ?include=owner")
	assertObjectField(t, owner, "owner")
	assert.Equal(t, "account", jsonField(owner, "type"))
}

func TestCovFinancePaymentTerms_ListIncludeOwnerAccount(t *testing.T) {
	t.Parallel()
	item := listFindByField(t, paymentTermsPath, url.Values{"include": {"owner.account"}}, "id", SeedPaymentTermID)
	require.NotNil(t, item, "SeedPaymentTermID should be present in the list to verify owner.account hydration")

	m := parseJSON(item)
	owner := jsonObject(m, "owner")
	require.NotNil(t, owner, "owner should be populated on list items with ?include=owner.account")
	assert.Equal(t, "account", jsonField(owner, "type"))

	account := jsonObject(owner, "account")
	require.NotNil(t, account, "owner.account should be populated on list items with ?include=owner.account (per-item hydration, not just single-GET)")
	assertObjectField(t, account, "account")
	assert.Equal(t, SeedAccountID, jsonField(account, "id"))
}

// --- Query param: unknown include value ---

func TestCovFinancePaymentTerms_IncludeUnknownValueRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(paymentTermsPath+"/"+SeedPaymentTermID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown include value should be rejected with 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}
