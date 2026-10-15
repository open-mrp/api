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
	assert.Equal(t, code, jsonField(created, "code"))
	assert.Equal(t, "visible", jsonField(created, "portal_visibility"))
	assert.Equal(t, "active", jsonField(created, "status"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(basePath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// UPDATE
	newName := uniqueName("e2e-cov-sl-crud-upd")
	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
		"name":              newName,
		"portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "hidden", jsonField(updated, "portal_visibility"))

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
// Service Levels: allFields (create + update).
// ──────────────────────────────────────────────

func TestCovOperationsCarriers_ServiceLevel_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	_, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	name := uniqueName("e2e-cov-sl-allf")
	code := uniqueName("e2e-cov-sl-allf-code")
	createResp, err := apiClient.PostFull(basePath, map[string]any{
		"name":                   name,
		"code":                   code,
		"portal_visibility":      "hidden",
		"status":                 "archived",
		"estimated_transit_days": 4,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(basePath + "/" + id)

	assert.Equal(t, "service_level", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, code, jsonField(got, "code"))
	assert.Equal(t, "hidden", jsonField(got, "portal_visibility"))
	assert.Equal(t, "archived", jsonField(got, "status"))
	assert.Equal(t, "4", jsonField(got, "estimated_transit_days"))
	assertNilField(t, got, "owner")
	assertNilField(t, got, "carrier")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	updatedName := uniqueName("e2e-cov-sl-allf-u")
	updatedCode := uniqueName("e2e-cov-sl-allf-u-code")
	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
		"name":                   updatedName,
		"code":                   updatedCode,
		"portal_visibility":      "visible",
		"status":                 "active",
		"estimated_transit_days": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, updatedCode, jsonField(updated, "code"))
	assert.Equal(t, "visible", jsonField(updated, "portal_visibility"))
	assert.Equal(t, "active", jsonField(updated, "status"))
	assertNilField(t, updated, "estimated_transit_days")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
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
		assert.Equal(t, code, jsonField(got, "code"), "code is the request's code")
		assert.Equal(t, "visible", jsonField(got, "portal_visibility"), "portal_visibility should default to visible")
		assert.Equal(t, "active", jsonField(got, "status"), "status should default to active")
		assertNilField(t, got, "estimated_transit_days")
		assertNilField(t, got, "carrier")
		assertNilField(t, got, "owner")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-cov-sl-pres")
		code := uniqueName("e2e-cov-sl-pres-code")
		createStatus, createBody, err := apiClient.Post(basePath, map[string]any{
			"name":              name,
			"code":              code,
			"portal_visibility": "hidden",
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
		assert.Equal(t, code, jsonField(got, "code"), "code should be preserved")
		assert.Equal(t, "hidden", jsonField(got, "portal_visibility"), "portal_visibility should be preserved")
		assert.Equal(t, "active", jsonField(got, "status"), "status should be preserved")
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
		"name":              name,
		"code":              code,
		"portal_visibility": "hidden",
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
	assert.Equal(t, code, jsonField(got, "code"))
	assert.Equal(t, "hidden", jsonField(got, "portal_visibility"))
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
	status, body, err := apiClient.GetListRaw(path, url.Values{"include[]": {"customer"}})
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
	status, body, err := apiClient.GetListRaw(basePath, url.Values{"include[]": {"customer"}})
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
	requireStatus(t, 422, status, body)
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
	requireStatus(t, 422, status, body)
	requireErrorResponse(t, body, "missing_field", "invalid_request_error")
}

func TestCovOperationsCarriers_ServiceLevel_CreateValidation_InvalidVisibility(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)
	status, body, err := apiClient.Post(basePath, map[string]any{
		"name":              uniqueName("e2e-cov-sl-badvis"),
		"code":              uniqueName("e2e-cov-sl-badvis-code"),
		"portal_visibility": "public",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "portal_visibility")
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
	requireStatus(t, 422, patchStatus, patchBody)
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
	requireStatus(t, 422, patchStatus, patchBody)
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
		"portal_visibility": "public",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, patchStatus, patchBody)
	errObj := requireErrorResponse(t, patchBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "portal_visibility")
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
// Carriers: default service level + the guards around it
//
// Each test provisions its own throwaway carrier, since a carrier has one
// default. Deleting the carrier deletes its service levels, so its cleanup
// covers them.
// ──────────────────────────────────────────────

// covOperationsCarriersCreateThrowawayCarrier creates a fresh account-owned,
// non-Shippo carrier for tests that need default isolation, and returns
// its id plus its service-levels base path. Registers cleanup via t.Cleanup.
func covOperationsCarriersCreateThrowawayCarrier(t *testing.T) (carrierID, basePath string) {
	t.Helper()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-throwaway"),
		"type": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(carriersPath+"/"+id, map[string]any{"default_service_level_id": nil}, newIdempotencyKey())
		apiClient.Delete(carriersPath + "/" + id)
	})
	return id, serviceLevelsPath(id)
}

func TestCovOperationsCarriers_Carrier_DefaultServiceLevel(t *testing.T) {
	t.Parallel()
	carrierID, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	idA := covOperationsCarriersCreateServiceLevel(t, basePath, nil)
	idB := covOperationsCarriersCreateServiceLevel(t, basePath, nil)

	status, body, err := apiClient.Patch(carriersPath+"/"+carrierID, map[string]any{"default_service_level_id": idA}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "default_service_level")

	status, body, err = apiClient.GetListRaw(carriersPath+"/"+carrierID, url.Values{"include[]": {"default_service_level"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	def := jsonObject(parseJSON(body), "default_service_level")
	require.NotNil(t, def)
	assert.Equal(t, idA, jsonField(def, "id"))
	assert.Equal(t, "service_level", jsonField(def, "object"))

	// Moving the default to another level replaces it.
	status, body, err = apiClient.Patch(carriersPath+"/"+carrierID+"?include[]=default_service_level", map[string]any{"default_service_level_id": idB}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, idB, jsonField(jsonObject(parseJSON(body), "default_service_level"), "id"))

	// An update that leaves default_service_level_id out keeps it.
	status, body, err = apiClient.Patch(carriersPath+"/"+carrierID+"?include[]=default_service_level", map[string]any{"name": uniqueName("e2e-cov-carr-keepdef")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, idB, jsonField(jsonObject(parseJSON(body), "default_service_level"), "id"))

	// null clears it.
	status, body, err = apiClient.Patch(carriersPath+"/"+carrierID+"?include[]=default_service_level", map[string]any{"default_service_level_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "default_service_level")
}

func TestCovOperationsCarriers_Carrier_DefaultServiceLevelMustBeOwnAndActive(t *testing.T) {
	t.Parallel()
	carrierID, basePath := covOperationsCarriersCreateThrowawayCarrier(t)

	// Another carrier's service level.
	status, body, err := apiClient.Patch(carriersPath+"/"+carrierID, map[string]any{"default_service_level_id": SeedServiceLevelID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "default_service_level_id")

	// An archived one of its own.
	archivedID := covOperationsCarriersCreateServiceLevel(t, basePath, map[string]any{"status": "archived"})
	status, body, err = apiClient.Patch(carriersPath+"/"+carrierID, map[string]any{"default_service_level_id": archivedID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_archived", "invalid_request_error"), "default_service_level_id")
}

func TestCovOperationsCarriers_ServiceLevel_DefaultCannotBeDeletedOrArchived(t *testing.T) {
	t.Parallel()
	carrierID, basePath := covOperationsCarriersCreateThrowawayCarrier(t)
	id := covOperationsCarriersCreateServiceLevel(t, basePath, nil)

	status, body, err := apiClient.Patch(carriersPath+"/"+carrierID, map[string]any{"default_service_level_id": id}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	delStatus, delBody, err := apiClient.Delete(basePath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 409, delStatus, delBody)
	requireErrorResponse(t, delBody, "resource_in_use", "invalid_request_error")

	archStatus, archBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, archStatus, archBody)
	assertErrorParam(t, requireErrorResponse(t, archBody, "invalid_format", "invalid_request_error"), "status")

	// Once the carrier names no default, the service level can go.
	status, body, err = apiClient.Patch(carriersPath+"/"+carrierID, map[string]any{"default_service_level_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	delStatus, delBody, err = apiClient.Delete(basePath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)
	assertDeletedStub(t, delBody, id, "service_level")
}

// covOperationsCarriersCreateServiceLevel adds a service level under basePath with the extra fields given and returns its id.
func covOperationsCarriersCreateServiceLevel(t *testing.T, basePath string, extra map[string]any) string {
	t.Helper()
	body := map[string]any{
		"name": uniqueName("e2e-cov-sl"),
		"code": uniqueName("e2e-cov-sl-code"),
	}
	for k, v := range extra {
		body[k] = v
	}
	status, resp, err := apiClient.Post(basePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	id := jsonField(parseJSON(resp), "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// Service Levels: cross-carrier / already-deleted edge cases
// ──────────────────────────────────────────────

// Retrieving a real service level through a carrier_id that does not own it 404s, matching the scoping update and delete already enforce via IsInCarrier.
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
// carrier delete-already-deleted behavior, via the same NewAlreadyDeletedError pattern in
// service_level_service.go's DeleteServiceLevel.
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
		"type": "not_a_carrier",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

func TestCovOperationsCarriers_Carrier_CreateValidation_InvalidVisibilityEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name":              uniqueName("e2e-cov-carr-badvis"),
		"type":              "will_call",
		"portal_visibility": "public",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "portal_visibility")
}

// TestCovOperationsCarriers_Carrier_CreateUPSWithoutAccountNumber asserts the
// documented business rule (services/core-service/internal/service/carrier_service.go)
// that type=ups/usps without account_number returns 422
// ("Account number is required for this carrier.").
func TestCovOperationsCarriers_Carrier_CreateUPSWithoutAccountNumber(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-ups-noacct"),
		"type": "ups",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 422, status,
		"type=ups without account_number should return 422, got %d: %s", status, string(body))
	if status == 201 {
		id := jsonField(parseJSON(body), "id")
		apiClient.Delete(carriersPath + "/" + id)
	}
}

// A Shippo-managed code (fedex needs no account_number, since it authenticates over OAuth) does not 500 on create.
func TestCovOperationsCarriers_Carrier_CreateFedexDoesNotError(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": uniqueName("e2e-cov-carr-fedex"),
		"type": "fedex",
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
		"type": "will_call",
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

// TestCovOperationsCarriers_Carrier_UpdateRejectsTypeAndAccountNumber
// asserts the live behavior for PATCH bodies containing `type`/`account_number`:
// these fields aren't part of UpdateCarrierRequest, and the gateway's strict
// JSON decoding rejects unknown fields with 400 parameter_unknown rather than
// silently ignoring them.
func TestCovOperationsCarriers_Carrier_UpdateRejectsTypeAndAccountNumber(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-carr-unkfield")
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": name,
		"type": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	defer apiClient.Delete(carriersPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"type": "local_delivery",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "PATCH with unknown field type should return 400, got %d: %s", patchStatus, string(patchBody))
	errObj := requireErrorResponse(t, patchBody, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "type")

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
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}
