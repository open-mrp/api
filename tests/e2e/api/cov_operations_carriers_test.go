//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes coverage gaps in the operations_carriers group
// (/v1/operations/carriers + nested /v1/operations/carriers/{carrier_id}/service-levels)
// left by crud_carriers_test.go and crud_service_levels_test.go. It reuses the
// carriersPath const and serviceLevelsPath helper already declared in those files.

// ──────────────────────────────────────────────
// Service Levels: full CRUD lifecycle
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_CRUD(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)

	// CREATE
	name := uniqueName("e2e-cov-sl-crud")
	code := uniqueName("e2e-cov-sl-crud-code")
	createResp, err := apiClient.PostFull(basePath, map[string]any{
		"name": name,
		"code": code,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "service_level", jsonField(created, "object"))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	// NOTE: the service-level create endpoint (like ~32 other create endpoints,
	// e.g. territories/supplier-materials) declares no LocationFunc, so a 201
	// here intentionally omits the Location header. Only the carrier create
	// endpoint sets one. Do NOT assert a Location header on this route.
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, code, jsonField(created, "service_level_token"))
	assert.Equal(t, "visible", jsonField(created, "customer_portal_visibility"))
	assert.Equal(t, "false", jsonField(created, "is_default"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(basePath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// UPDATE
	newName := uniqueName("e2e-cov-sl-crud-upd")
	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
		"name":                       newName,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "hidden", jsonField(updated, "customer_portal_visibility"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(basePath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// GET after delete → 404
	getStatus2, _, err := apiClient.GetListRaw(basePath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

// ──────────────────────────────────────────────
// Service Levels: allFields (create + update), explicitly asserting
// service_level_token, customer_portal_visibility, and is_default.
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	// Uses its own throwaway carrier (not SeedCarrierID) because this test
	// exercises is_default=true, and ClearDefaultsForCarrier clears defaults
	// account+carrier-wide with no per-test scoping.
	_, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	name := uniqueName("e2e-cov-sl-allf")
	code := uniqueName("e2e-cov-sl-allf-code")
	createResp, err := apiClient.PostFull(basePath, map[string]any{
		"name":                       name,
		"code":                       code,
		"customer_portal_visibility": "hidden",
		"is_default":                 false,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	// Service-level create declares no LocationFunc, so no Location header is
	// emitted on 201 (see CRUD test note). Don't assert one.
	defer apiClient.Delete(basePath + "/" + id)

	assert.Equal(t, "service_level", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, code, jsonField(got, "service_level_token"))
	assert.Equal(t, "hidden", jsonField(got, "customer_portal_visibility"))
	assert.Equal(t, "false", jsonField(got, "is_default"))
	assertNilField(t, got, "owner")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// UPDATE with different values, including is_default=true.
	updatedName := uniqueName("e2e-cov-sl-allf-u")
	updatedCode := uniqueName("e2e-cov-sl-allf-u-code")
	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
		"name":                       updatedName,
		"code":                       updatedCode,
		"customer_portal_visibility": "visible",
		"is_default":                 true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, updatedCode, jsonField(updated, "service_level_token"))
	assert.Equal(t, "visible", jsonField(updated, "customer_portal_visibility"))
	assert.Equal(t, "true", jsonField(updated, "is_default"))
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	// Clear is_default so the deferred Delete above doesn't hit the
	// default-service-levels-cannot-be-deleted guard.
	_, _, _ = apiClient.Patch(basePath+"/"+id, map[string]any{"is_default": false}, newIdempotencyKey())
}

// ──────────────────────────────────────────────
// Service Levels: omitted fields (defaults + update-preservation)
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_OmittedFields(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-cov-sl-omit")
		code := uniqueName("e2e-cov-sl-omit-code")
		status, body, err := apiClient.Post(basePath, map[string]any{
			"name": name,
			"code": code,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(basePath + "/" + id)

		assertObjectField(t, got, "service_level")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, code, jsonField(got, "service_level_token"), "service_level_token should default to the request's code")
		assert.Equal(t, "visible", jsonField(got, "customer_portal_visibility"), "customer_portal_visibility should default to visible")
		assert.Equal(t, "false", jsonField(got, "is_default"), "is_default should default to false")
		assertNilField(t, got, "owner")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-cov-sl-pres")
		code := uniqueName("e2e-cov-sl-pres-code")
		createStatus, createBody, err := apiClient.Post(basePath, map[string]any{
			"name":                       name,
			"code":                       code,
			"customer_portal_visibility": "hidden",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(basePath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name.
		newName := uniqueName("e2e-cov-sl-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, code, jsonField(got, "service_level_token"), "service_level_token should be preserved")
		assert.Equal(t, "hidden", jsonField(got, "customer_portal_visibility"), "customer_portal_visibility should be preserved")
		assert.Equal(t, "false", jsonField(got, "is_default"), "is_default should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

// ──────────────────────────────────────────────
// Service Levels: response shape (ID prefix, object, timestamps)
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_CreateResponseShape(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)

	name := uniqueName("e2e-cov-sl-shape")
	code := uniqueName("e2e-cov-sl-shape-code")
	createResp, err := apiClient.PostFull(basePath, map[string]any{
		"name":                       name,
		"code":                       code,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	// Service-level create declares no LocationFunc, so no Location header is
	// emitted on 201 (see CRUD test note). Don't assert one.
	// Freshly-created service levels get the "silv_" prefix
	// (shared/id.ServiceLevelIDPrefix = VocService+VocLevel). The seeded
	// rows use the legacy "crop_" ("carrier option") prefix, which only
	// exists on pre-seeded data, not on runtime creates.
	assertIDFormat(t, id, "silv")
	assert.Equal(t, "service_level", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, code, jsonField(got, "service_level_token"))
	assert.Equal(t, "hidden", jsonField(got, "customer_portal_visibility"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")

	apiClient.Delete(basePath + "/" + id)
}

// ──────────────────────────────────────────────
// Service Levels: list (basic, pagination, search, no-results)
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_List(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	list, _, err := apiClient.GetList(basePath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 2, "seeded carrier should expose at least 2 service levels")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedServiceLevelID {
			found = true
			break
		}
	}
	assert.True(t, found, "seeded service level %q should appear in the unfiltered list", SeedServiceLevelID)
}

func TestCovOperationsCarriers_ServiceLevel_ListSearchByName(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	list, _, err := apiClient.GetList(basePath, url.Values{"q": {"Ground"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "search for 'Ground' should return at least 1 result")

	for _, item := range list.Data {
		assert.Contains(t, DataItemField(item, "name"), "Ground")
	}
}

func TestCovOperationsCarriers_ServiceLevel_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	list, _, err := apiClient.GetList(basePath, url.Values{"q": {"zzzznotaservicelevel99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestCovOperationsCarriers_ServiceLevel_ListPagination(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	prefix := uniqueName("e2e-cov-sl-pg")
	var ids []string
	for i := 0; i < 2; i++ {
		status, body, err := apiClient.Post(basePath, map[string]any{
			"name": uniqueName(prefix),
			"code": uniqueName(prefix + "-code"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		id := jsonField(parseJSON(body), "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(basePath + "/" + id)
		ids = append(ids, id)
	}

	assertScopedCursorPagination(t, basePath, url.Values{"q": {prefix}}, ids)
}

func TestCovOperationsCarriers_ServiceLevel_ListUnknownCarrierReturnsEmpty(t *testing.T) {
	t.Parallel()
	// Confirmed live behavior: listing service levels under a syntactically
	// valid but non-existent carrier_id returns an empty page rather than
	// 404ing on the missing parent. This mirrors many other nested-collection
	// list endpoints in this API (list-under-missing-parent = empty, not 404).
	list, status, err := apiClient.GetList(serviceLevelsPath("car_e2e_cov_nonexistent"), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assertEmptyListData(t, list.Data)
}

func TestCovOperationsCarriers_ServiceLevel_CreateUnderUnknownCarrier404s(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(serviceLevelsPath("car_e2e_cov_nonexistent"), map[string]any{
		"name": uniqueName("e2e-cov-sl-ghost"),
		"code": uniqueName("e2e-cov-sl-ghost-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "create service level under unknown carrier should 404, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// Service Levels: expandable (owner) unknown/deep include rejection
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_RetrieveRejectsUnknownInclude(t *testing.T) {
	t.Parallel()
	path := serviceLevelsPath(SeedCarrierID) + "/" + SeedServiceLevelID
	status, body, err := apiClient.GetListRaw(path, url.Values{"include[]": {"carrier"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown include must be 400, got %d: %s", status, string(body))
}

func TestCovOperationsCarriers_ServiceLevel_RetrieveRejectsDeepInclude(t *testing.T) {
	t.Parallel()
	path := serviceLevelsPath(SeedCarrierID) + "/" + SeedServiceLevelID
	status, body, err := apiClient.GetListRaw(path, url.Values{"include[]": {"owner.account.branding"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "deep include outside allow-list must be 400, got %d: %s", status, string(body))
}

func TestCovOperationsCarriers_ServiceLevel_ListRejectsUnknownInclude(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.GetListRaw(basePath, url.Values{"include[]": {"carrier"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown include on list must be 400, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// Service Levels: idempotency (create + update)
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_CreateIdempotent(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	name := uniqueName("e2e-cov-sl-idem")
	code := uniqueName("e2e-cov-sl-idem-code")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(basePath, map[string]any{"name": name, "code": code}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(basePath, map[string]any{"name": name, "code": code}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(basePath + "/" + id1)
}

func TestCovOperationsCarriers_ServiceLevel_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	name := uniqueName("e2e-cov-sl-updidem")
	code := uniqueName("e2e-cov-sl-updidem-code")
	status, body, err := apiClient.Post(basePath, map[string]any{"name": name, "code": code}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(basePath + "/" + id)

	newName := uniqueName("e2e-cov-sl-updidem-upd")
	updKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(basePath+"/"+id, map[string]any{"name": newName}, updKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(basePath+"/"+id, map[string]any{"name": newName}, updKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, jsonField(parseJSON(body1), "updated_at"), jsonField(parseJSON(body2), "updated_at"),
		"repeated PATCH with the same idempotency key should replay the cached response")
}

// ──────────────────────────────────────────────
// Service Levels: validation
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name": "",
		"code": uniqueName("e2e-cov-sl-badname-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "missing_field", "invalid_request_error")
}

func TestCovOperationsCarriers_ServiceLevel_CreateValidation_EmptyCode(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-badcode"),
		"code": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "missing_field", "invalid_request_error")
}

func TestCovOperationsCarriers_ServiceLevel_CreateValidation_InvalidVisibility(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name":                       uniqueName("e2e-cov-sl-badvis"),
		"code":                       uniqueName("e2e-cov-sl-badvis-code"),
		"customer_portal_visibility": "public",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "customer_portal_visibility")
}

func TestCovOperationsCarriers_ServiceLevel_CreateDuplicateCodeFails(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	code := uniqueName("e2e-cov-sl-dupcode")

	status1, body1, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-dup-a"),
		"code": code,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id := jsonField(parseJSON(body1), "id")
	defer apiClient.Delete(basePath + "/" + id)

	status2, body2, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-dup-b"),
		"code": code,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status2, "duplicate code within the same carrier should return 409, got %d: %s", status2, string(body2))
	errObj := requireErrorResponse(t, body2, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "code")
}

func TestCovOperationsCarriers_ServiceLevel_UpdateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-updblank"),
		"code": uniqueName("e2e-cov-sl-updblank-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(basePath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{"name": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, patchStatus, patchBody)
	requireErrorResponse(t, patchBody, "invalid_format", "invalid_request_error")
}

func TestCovOperationsCarriers_ServiceLevel_UpdateValidation_EmptyCode(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-updblankcode"),
		"code": uniqueName("e2e-cov-sl-updblankcode-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(basePath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{"code": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, patchStatus, patchBody)
	requireErrorResponse(t, patchBody, "invalid_format", "invalid_request_error")
}

func TestCovOperationsCarriers_ServiceLevel_UpdateValidation_InvalidVisibility(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-updbadvis"),
		"code": uniqueName("e2e-cov-sl-updbadvis-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(basePath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
		"customer_portal_visibility": "public",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, patchStatus, patchBody)
	errObj := requireErrorResponse(t, patchBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "customer_portal_visibility")
}

func TestCovOperationsCarriers_ServiceLevel_UpdateValidation_DuplicateCode(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)

	codeA := uniqueName("e2e-cov-sl-updup-a")
	statusA, bodyA, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-updup-a"),
		"code": codeA,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, statusA, bodyA)
	idA := jsonField(parseJSON(bodyA), "id")
	defer apiClient.Delete(basePath + "/" + idA)

	statusB, bodyB, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-updup-b"),
		"code": uniqueName("e2e-cov-sl-updup-b"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, statusB, bodyB)
	idB := jsonField(parseJSON(bodyB), "id")
	defer apiClient.Delete(basePath + "/" + idB)

	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+idB, map[string]any{"code": codeA}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, patchStatus, "updating to a code already used by a sibling service level should 409, got %d: %s", patchStatus, string(patchBody))
}

// ──────────────────────────────────────────────
// Service Levels: business rules — default clearing + default delete guard
//
// is_default is cleared account+carrier-wide (ClearDefaultsForCarrier has no
// per-test scoping), so these tests each provision their own throwaway
// carrier rather than sharing SeedCarrierID — otherwise parallel runs of
// these tests would race on which service level is "the" default. Deleting
// the throwaway carrier cascades to (hard-deletes) all of its service
// levels (see carrierSvcImpl.DeleteCarrier → DeleteOptionsByCarrierID), so a
// single deferred carrier delete is sufficient cleanup even for a default
// service level that would otherwise refuse direct deletion.
// ──────────────────────────────────────────────

// covOperationsCarriersCreateThrowawayCarrier creates a fresh account-owned,
// non-Shippo carrier for tests that need is_default isolation, and returns
// its id plus its service-levels base path. Registers cleanup via t.Cleanup.
func covOperationsCarriersCreateThrowawayCarrier(t *testing.T) (carrierID, basePath string) {
	t.Helper()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-throwaway"),
		"code": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(carriersPath + "/" + id) })
	return id, serviceLevelsPath(id)
}

func TestCovOperationsCarriers_ServiceLevel_CreateIsDefaultClearsPriorDefault(t *testing.T) {
	t.Parallel()
	_, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	statusA, bodyA, err := apiClient.Post(basePath, map[string]any{
		"name":       uniqueName("e2e-cov-sl-clrdef-a"),
		"code":       uniqueName("e2e-cov-sl-clrdef-a-code"),
		"is_default": true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, statusA, bodyA)
	idA := jsonField(parseJSON(bodyA), "id")
	assert.Equal(t, "true", jsonField(parseJSON(bodyA), "is_default"))

	statusB, bodyB, err := apiClient.Post(basePath, map[string]any{
		"name":       uniqueName("e2e-cov-sl-clrdef-b"),
		"code":       uniqueName("e2e-cov-sl-clrdef-b-code"),
		"is_default": true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, statusB, bodyB)
	assert.Equal(t, "true", jsonField(parseJSON(bodyB), "is_default"))

	// Re-fetch A: creating B as the new default must have cleared A's.
	getStatus, getBody, err := apiClient.GetListRaw(basePath+"/"+idA, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "false", jsonField(parseJSON(getBody), "is_default"),
		"creating a new default service level should clear the carrier's previous default")
}

func TestCovOperationsCarriers_ServiceLevel_UpdateIsDefaultClearsPriorDefault(t *testing.T) {
	t.Parallel()
	_, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	statusA, bodyA, err := apiClient.Post(basePath, map[string]any{
		"name":       uniqueName("e2e-cov-sl-updclrdef-a"),
		"code":       uniqueName("e2e-cov-sl-updclrdef-a-code"),
		"is_default": true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, statusA, bodyA)
	idA := jsonField(parseJSON(bodyA), "id")

	statusB, bodyB, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-updclrdef-b"),
		"code": uniqueName("e2e-cov-sl-updclrdef-b-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, statusB, bodyB)
	idB := jsonField(parseJSON(bodyB), "id")

	// Promote B to default via PATCH; this should clear A's default.
	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+idB, map[string]any{"is_default": true}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "true", jsonField(parseJSON(patchBody), "is_default"))

	getStatus, getBody, err := apiClient.GetListRaw(basePath+"/"+idA, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "false", jsonField(parseJSON(getBody), "is_default"),
		"promoting a sibling service level to default should clear this one's default")
}

func TestCovOperationsCarriers_ServiceLevel_DefaultCannotBeDeleted(t *testing.T) {
	t.Parallel()
	_, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	status, body, err := apiClient.Post(basePath, map[string]any{
		"name":       uniqueName("e2e-cov-sl-defdel"),
		"code":       uniqueName("e2e-cov-sl-defdel-code"),
		"is_default": true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(basePath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 400, delStatus, "deleting a default service level should return 400, got %d: %s", delStatus, string(delBody))
	requireErrorResponse(t, delBody, "validation_failed", "invalid_request_error")
	// No further cleanup needed: the throwaway carrier's deferred delete
	// cascades to this (still-default) service level.
}

// ──────────────────────────────────────────────
// Service Levels: cross-carrier / already-deleted edge cases (some are
// SUSPECTED BACKEND BUGS — see confirmedBugs; these tests assert correct
// behavior and will fail red until the underlying issue is fixed).
// ──────────────────────────────────────────────

// TestCovOperationsCarriers_ServiceLevel_RetrieveCrossCarrier404s asserts that
// retrieving a real service level ID through the WRONG carrier_id path
// segment 404s, matching the scoping enforced by update/delete (both of
// which call IsInCarrier before returning the resource). SUSPECTED BACKEND
// BUG: the gateway's GetServiceLevel loads purely by ID
// (resourceloaders.LoadServiceLevels via loadServiceLevelByID) and never
// validates req.CarrierID, so any carrier_id in the path currently returns
// 200 with the resource. Verified live: GET
// /v1/operations/carriers/{SeedSystemCarrierID}/service-levels/{SeedServiceLevelID}
// (a service level that belongs to SeedCarrierID, not SeedSystemCarrierID)
// returns 200, not 404.
func TestCovOperationsCarriers_ServiceLevel_RetrieveCrossCarrier404s(t *testing.T) {
	t.Parallel()
	wrongPath := serviceLevelsPath(SeedSystemCarrierID) + "/" + SeedServiceLevelID
	status, body, err := apiClient.GetListRaw(wrongPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"retrieving a service level through a carrier_id that doesn't own it should 404, got %d: %s", status, string(body))
}

// TestCovOperationsCarriers_ServiceLevel_AlreadyDeleted410 asserts that
// deleting an already-deleted service level returns 410 Gone, matching the
// carrier delete-already-deleted behavior (TestCarriers has no equivalent
// name but the same NewAlreadyDeletedError pattern is used in
// service_level_service.go's DeleteServiceLevel). SUSPECTED BACKEND BUG: the
// IsInCarrier pre-check in DeleteServiceLevel runs before the
// Get+DeletedRecordRepo fallback that would produce 410, and IsInCarrier
// always returns false once the row is hard-deleted from carrier_option, so
// the 410 branch is unreachable — the second delete returns 404 "Service
// level not found." instead of 410.
func TestCovOperationsCarriers_ServiceLevel_AlreadyDeleted410(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)

	status, body, err := apiClient.Post(basePath, map[string]any{
		"name": uniqueName("e2e-cov-sl-gone"),
		"code": uniqueName("e2e-cov-sl-gone-code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(basePath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	redeleteStatus, redeleteBody, err := apiClient.Delete(basePath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, redeleteStatus,
		"deleting an already-deleted service level should return 410, got %d: %s", redeleteStatus, string(redeleteBody))
}

// ──────────────────────────────────────────────
// Carriers: gaps called out in the task doc (invalid code enum, ups/usps
// account_number requirement, already-deleted-410, PATCH with unknown
// code/account_number fields).
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_Carrier_CreateValidation_InvalidCodeEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-badcode"),
		"code": "not_a_carrier",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "code")
}

func TestCovOperationsCarriers_Carrier_CreateValidation_InvalidVisibilityEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name":                       uniqueName("e2e-cov-carr-badvis"),
		"customer_portal_visibility": "public",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "customer_portal_visibility")
}

// TestCovOperationsCarriers_Carrier_CreateUPSWithoutAccountNumber asserts the
// documented business rule (services/core-service/internal/service/carrier_service.go)
// that code=ups/usps without account_number returns 400
// ("Account number is required for this carrier."). SUSPECTED BACKEND BUG:
// verified live this currently returns 500 internal_error instead. The
// validation branch that would produce the 400 only runs when
// !accountCtx.IsSandbox, and in this environment the create instead falls
// through to a real Shippo-connection attempt that errors out as a 500
// (getShippoClient / crypto.DecryptAESGCM path) before ever reaching the
// account_number check for the actual failure mode observed.
func TestCovOperationsCarriers_Carrier_CreateUPSWithoutAccountNumber(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-ups-noacct"),
		"code": "ups",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status,
		"code=ups without account_number should return 400, got %d: %s", status, string(body))
	if status == 201 {
		id := jsonField(parseJSON(body), "id")
		apiClient.Delete(carriersPath + "/" + id)
	}
}

// TestCovOperationsCarriers_Carrier_CreateFedexDoesNotError asserts that
// creating a carrier with a Shippo-managed code (fedex, which doesn't
// require account_number since it uses OAuth) at least doesn't 500.
// SUSPECTED BACKEND BUG: verified live this currently returns 500
// internal_error (see the ups/usps sibling test doc comment for the
// suspected root cause).
func TestCovOperationsCarriers_Carrier_CreateFedexDoesNotError(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-fedex"),
		"code": "fedex",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.NotEqual(t, 500, status,
		"creating a carrier with a Shippo-managed code should not 500, got %d: %s", status, string(body))
	if status == 201 {
		id := jsonField(parseJSON(body), "id")
		apiClient.Delete(carriersPath + "/" + id)
	}
}

func TestCovOperationsCarriers_Carrier_CreateAlreadyDeleted410(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-carr-gone")
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": name,
		"code": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(carriersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	redeleteStatus, redeleteBody, err := apiClient.Delete(carriersPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, redeleteStatus,
		"deleting an already-deleted carrier should return 410, got %d: %s", redeleteStatus, string(redeleteBody))
}

// TestCovOperationsCarriers_Carrier_UpdateRejectsUnknownCodeAndAccountNumber
// asserts the live behavior for PATCH bodies containing `code`/`account_number`:
// these fields aren't part of UpdateCarrierRequest, and the gateway's strict
// JSON decoding rejects unknown fields with 400 parameter_unknown rather than
// silently ignoring them.
func TestCovOperationsCarriers_Carrier_UpdateRejectsUnknownCodeAndAccountNumber(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-carr-unkfield")
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": name,
		"code": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(carriersPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"code": "delivery",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "PATCH with unknown field code should return 400, got %d: %s", patchStatus, string(patchBody))
	errObj := requireErrorResponse(t, patchBody, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "code")

	patchStatus2, patchBody2, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"account_number": "999",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus2, "PATCH with unknown field account_number should return 400, got %d: %s", patchStatus2, string(patchBody2))
	errObj2 := requireErrorResponse(t, patchBody2, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj2, "account_number")
}

// ──────────────────────────────────────────────
// Service Levels list: query-param matrix (limit, cursor)
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_ListInvalidLimitZero(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.GetListRaw(basePath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "limit=0 should return 400, got %d: %s", status, string(body))
}

func TestCovOperationsCarriers_ServiceLevel_ListInvalidLimitTooLarge(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.GetListRaw(basePath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "limit=1001 should return 400, got %d: %s", status, string(body))
}

func TestCovOperationsCarriers_ServiceLevel_ListInvalidCursor(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.GetListRaw(basePath, url.Values{"cursor": {"not-a-valid-cursor"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "malformed cursor should return 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}
