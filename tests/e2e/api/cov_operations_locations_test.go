//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes gaps identified in the operations_locations e2e coverage
// review (TASK-operations_locations.md): invalid `type` on create/update,
// nonexistent parent_id/child_ids references on create and update,
// self-parent rejection on update, duplicate `name` conflict, PATCH/DELETE
// on a never-existed id, an id-format assertion on create, list-level
// ?include=parent/children, invalid ?include value (list + retrieve), the
// name>255 boundary, the PATCH {name:""} edge case, child_ids preservation
// across a name-only PATCH, and the internal-actor-only gate. It reuses
// `locationsPath` from crud_locations_test.go (same package) rather than
// redeclaring it.

// ──────────────────────────────────────────────
// type — invalid enum value rejected (create + update)
// ──────────────────────────────────────────────

func TestCovOperationsLocations_CreateInvalidTypeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name": uniqueName("e2e-cov-loc-badtype"),
		"type": "not_a_real_type",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

func TestCovOperationsLocations_UpdateInvalidTypeRejected(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name": uniqueName("e2e-cov-loc-updbadtype"),
		"type": "building",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"type": "garbage_type",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "type")

	// Confirm the invalid update did not silently persist.
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "building", jsonField(parseJSON(getBody), "type"), "type should be unchanged after a rejected update")
}

// ──────────────────────────────────────────────
// parent_id / child_ids — nonexistent references rejected (create + update)
// ──────────────────────────────────────────────

func TestCovOperationsLocations_CreateNonexistentParentIDRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name":      uniqueName("e2e-cov-loc-badparent"),
		"type":      "section",
		"parent_id": "lc_doesnotexist0000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "parent_id")
}

func TestCovOperationsLocations_CreateNonexistentChildIDRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name":      uniqueName("e2e-cov-loc-badchild"),
		"type":      "building",
		"child_ids": []string{"lc_doesnotexist0000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "child_ids")
}

func TestCovOperationsLocations_UpdateNonexistentParentIDRejected(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name": uniqueName("e2e-cov-loc-updbadparent"),
		"type": "section",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"parent_id": "lc_doesnotexist0000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "parent_id")
}

func TestCovOperationsLocations_UpdateNonexistentChildIDRejected(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name": uniqueName("e2e-cov-loc-updbadchild"),
		"type": "building",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"child_ids": []string{"lc_doesnotexist0000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "child_ids")
}

// ──────────────────────────────────────────────
// parent_id — self-parent rejected on update
// ──────────────────────────────────────────────

func TestCovOperationsLocations_UpdateSelfParentRejected(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name": uniqueName("e2e-cov-loc-selfparent"),
		"type": "building",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"parent_id": id,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "parent_id")

	// The location must still have no parent.
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assertNilField(t, parseJSON(getBody), "parent")
}

// ──────────────────────────────────────────────
// name — duplicate within account conflicts (409)
// ──────────────────────────────────────────────

func TestCovOperationsLocations_CreateDuplicateNameConflict(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-loc-dupname")
	createAndCleanup(t, locationsPath, map[string]any{
		"name": name,
		"type": "building",
	})

	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "section",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	errObj := requireErrorResponse(t, body, "resource_exists", "invalid_request_error")
	assert.Nil(t, errObj["param"])
}

// ──────────────────────────────────────────────
// 404 — PATCH/DELETE on an id that was never created
// ──────────────────────────────────────────────

func TestCovOperationsLocations_UpdateNonexistentIDNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(locationsPath+"/lc_neverexisted0000", map[string]any{
		"name": uniqueName("e2e-cov-loc-updnf"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovOperationsLocations_DeleteNeverExistedIDNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(locationsPath + "/lc_neverexisted0001")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// name — >255 chars rejected on create
// ──────────────────────────────────────────────

func TestCovOperationsLocations_CreateNameTooLongRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name": strings.Repeat("a", 256),
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// ──────────────────────────────────────────────
// name — empty string rejected on update (prodBugSuspect #2: confirmed the
// backend correctly rejects this; the suspected bypass does not reproduce)
// ──────────────────────────────────────────────

func TestCovOperationsLocations_UpdateEmptyStringNameRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-loc-emptyname")
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name": name,
		"type": "building",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")

	// The name must remain unchanged.
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"), "name should be unchanged after a rejected empty-string update")
}

// ──────────────────────────────────────────────
// child_ids — preserved across a PATCH that omits the field
// ──────────────────────────────────────────────

func TestCovOperationsLocations_UpdateChildIDsPreservedWhenOmitted(t *testing.T) {
	t.Parallel()

	childName := uniqueName("e2e-cov-loc-preschild")
	child := createAndCleanup(t, locationsPath, map[string]any{
		"name": childName,
		"type": "section",
	})
	childID := jsonField(child, "id")

	parentName := uniqueName("e2e-cov-loc-presparent")
	parent := createAndCleanup(t, locationsPath, map[string]any{
		"name":      parentName,
		"type":      "building",
		"child_ids": []string{childID},
	})
	parentID := jsonField(parent, "id")

	// Name-only PATCH must not touch child_ids.
	newParentName := uniqueName("e2e-cov-loc-presparent-renamed")
	patchStatus, patchBody, err := apiClient.Patch(locationsPath+"/"+parentID, map[string]any{
		"name": newParentName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+parentID, url.Values{"include": {"children"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	children := jsonObject(got, "children")
	require.NotNil(t, children, "children should be preserved after a name-only PATCH")
	childData, ok := children["data"].([]any)
	require.True(t, ok, "children.data should be an array")
	require.Len(t, childData, 1)
	childMap, ok := childData[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, childID, jsonField(childMap, "id"))
}

// ──────────────────────────────────────────────
// Response shape — id prefix
// ──────────────────────────────────────────────

func TestCovOperationsLocations_CreateResponseShapeIDPrefix(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, locationsPath, map[string]any{
		"name": uniqueName("e2e-cov-loc-idfmt"),
		"type": "building",
	})
	id := jsonField(created, "id")
	assertIDFormat(t, id, "lc")
	assertObjectField(t, created, "location")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// include — list-level ?include=parent / ?include=children
// ──────────────────────────────────────────────

func TestCovOperationsLocations_ListIncludeParent(t *testing.T) {
	t.Parallel()

	parentName := uniqueName("e2e-cov-loc-listinclp")
	parent := createAndCleanup(t, locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	})
	parentID := jsonField(parent, "id")

	childName := uniqueName("e2e-cov-loc-listinclc")
	child := createAndCleanup(t, locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	})
	childID := jsonField(child, "id")

	item := listFindByField(t, locationsPath, url.Values{"include": {"parent"}}, "id", childID)
	require.NotNil(t, item, "created child should appear in the list with ?include=parent")

	m := parseJSON(item)
	parentObj := jsonObject(m, "parent")
	require.NotNil(t, parentObj, "parent should be populated on the list item with ?include=parent")
	assert.Equal(t, parentID, jsonField(parentObj, "id"))
	assert.Equal(t, "location", jsonField(parentObj, "object"))
}

func TestCovOperationsLocations_ListIncludeChildren(t *testing.T) {
	t.Parallel()

	parentName := uniqueName("e2e-cov-loc-listinclchp")
	parent := createAndCleanup(t, locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	})
	parentID := jsonField(parent, "id")

	childName := uniqueName("e2e-cov-loc-listinclchc")
	child := createAndCleanup(t, locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	})
	childID := jsonField(child, "id")

	item := listFindByField(t, locationsPath, url.Values{"include": {"children"}}, "id", parentID)
	require.NotNil(t, item, "created parent should appear in the list with ?include=children")

	m := parseJSON(item)
	childrenObj := jsonObject(m, "children")
	require.NotNil(t, childrenObj, "children should be populated on the list item with ?include=children")
	assert.Equal(t, "list", jsonField(childrenObj, "object"))
	childData, ok := childrenObj["data"].([]any)
	require.True(t, ok, "children.data should be an array")
	found := false
	for _, raw := range childData {
		cm, ok := raw.(map[string]any)
		require.True(t, ok)
		if jsonField(cm, "id") == childID {
			found = true
			break
		}
	}
	assert.True(t, found, "created child %q should appear in the parent's included children", childID)
}

// ──────────────────────────────────────────────
// include — invalid value rejected (list + retrieve)
// ──────────────────────────────────────────────

func TestCovOperationsLocations_GetInvalidIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(locationsPath+"/"+SeedLocationID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assert.Equal(t, "include[]", errObj["param"])
}

func TestCovOperationsLocations_ListInvalidIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(locationsPath, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assert.Equal(t, "include[]", errObj["param"])
}

// ──────────────────────────────────────────────
// Auth — internal-actor-only gate on a Public:true route
// ──────────────────────────────────────────────

// TestCovOperationsLocations_CustomerActorForbidden confirms non-internal
// (customer portal) actors cannot reach this group at all: the service layer
// calls identity.CheckIsInternalActor() unconditionally, so even a valid
// customer-scoped API key for the same account is rejected with 403, despite
// the route being marked Public:true in the OpenAPI surface (prodBugSuspect
// #5 — flagged as worth a sanity check, confirmed here as the actual,
// consistent behavior rather than a flaky 5xx).
func TestCovOperationsLocations_CustomerActorForbidden(t *testing.T) {
	t.Parallel()
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.GetListRaw(locationsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}
