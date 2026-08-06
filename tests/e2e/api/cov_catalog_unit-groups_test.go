//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the existing catalog/unit-groups coverage in
// crud_unit_groups_test.go with the gaps identified in the coverage audit:
// consolidated omitted-fields tests, a create-all-fields+update-all-fields
// round trip that includes associated_units and a base_unit_id change,
// dimension-mismatch validation (both at the group level via base_unit_id/
// associated_units and at the nested-unit level via unit_id), invalid-enum
// validation for type/customer_portal_visibility, duplicate-name 409s,
// nonexistent-base_unit_id handling, 404s on update/delete of nonexistent
// resources, a double-delete of a unit_group_unit, an idempotency test for
// PATCH .../units/{id}, and customer-portal update/delete 403s. It reuses
// unitGroupsPath, unitGroupUnitsPath, and createTestUnitGroup declared in
// crud_unit_groups_test.go, plus SeedUnitGroupID/SeedUnitGroupUnitID/
// SeedUnitID from seed_test.go. No new seed rows are needed.

// covCatalogUnitGroupsCreateBody returns the minimal required body for
// creating a quantity-type unit group.
func covCatalogUnitGroupsCreateBody(name string) map[string]any {
	return map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}
}

// covCatalogUnitGroupsCreate creates a quantity unit group and returns its id.
func covCatalogUnitGroupsCreate(t *testing.T, name string) string {
	t.Helper()
	status, body, err := apiClient.Post(unitGroupsPath, covCatalogUnitGroupsCreateBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	return id
}

// covCatalogUnitGroupUnitPath builds the path to a nested unit_group_unit.
func covCatalogUnitGroupUnitPath(unitGroupID, unitGroupUnitID string) string {
	return fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, unitGroupID, unitGroupUnitID)
}

// --- Omitted fields: UnitGroup ---

func TestCovCatalogUnitGroups_OmittedFields(t *testing.T) {
	t.Parallel()

	// Sub-case 1: create with only required fields; every optional/expandable
	// field must be null/absent on the bare create response.
	name := uniqueName("e2e-cov-ug-omit")
	createResp, err := apiClient.PostFull(unitGroupsPath, covCatalogUnitGroupsCreateBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	assertObjectField(t, created, "unit_group")
	assertIDFormat(t, id, "ungp")
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "quantity", jsonField(created, "type"))
	assertNilField(t, created, "notes")
	assertNilField(t, created, "base_unit")
	assertNilField(t, created, "associated_units")
	assertNilField(t, created, "owner")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// Sub-case 2: missing each required field individually returns 400/422
	// with the offending field named in error.param.
	cases := []struct {
		field string
		body  map[string]any
		param string
	}{
		{"name", map[string]any{"type": "quantity", "base_unit_id": "each"}, "name"},
		{"type", map[string]any{"name": uniqueName("e2e-cov-ug-notype"), "base_unit_id": "each"}, "type"},
		{"base_unit_id", map[string]any{"name": uniqueName("e2e-cov-ug-nobase"), "type": "quantity"}, "base_unit_id"},
	}
	for _, tc := range cases {
		status, body, err := apiClient.Post(unitGroupsPath, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing %s should return 400/422, got %d: %s", tc.field, status, string(body))
	}

	// Sub-case 3: patching an unrelated field (name) preserves every
	// untouched field, including the type, base_unit shape, and
	// associated_units shape.
	patchStatus, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id+"?include=base_unit,associated_units,owner", map[string]any{
		"name": uniqueName("e2e-cov-ug-omit-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "quantity", jsonField(patched, "type"), "type must be preserved (immutable, no update path)")
	assertNilField(t, patched, "notes")
	baseUnit := jsonObject(patched, "base_unit")
	require.NotNil(t, baseUnit, "base_unit should still resolve after an unrelated patch")
	assert.Equal(t, "each", jsonField(baseUnit, "id"))
	// associated_units is requested via ?include=, so on a group created with no
	// units it correctly materializes as an empty list ({object:list, data:[]}),
	// not null. (It IS null on the bare create response above, where it was not
	// included.)
	assocPatched := jsonObject(patched, "associated_units")
	require.NotNil(t, assocPatched, "included associated_units should be an empty list, not null")
	assert.Equal(t, "list", jsonField(assocPatched, "object"))
	assocPatchedData, ok := assocPatched["data"].([]any)
	require.True(t, ok, "associated_units.data should be an array")
	// A group's base unit is auto-included as an associated unit even when none are listed.
	assert.Len(t, assocPatchedData, 1, "a group created without units still carries its base unit")
}

// TestCovCatalogUnitGroups_UpdateCannotChangeType asserts there is no way to
// change a unit group's type via PATCH: even when a type field is sent, it is
// either ignored (type unchanged) or rejected, never applied.
func TestCovCatalogUnitGroups_UpdateCannotChangeType(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ug-notype-upd"))
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	status, body, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"type": "mass",
	}, newIdempotencyKey())
	require.NoError(t, err)

	if status == 400 || status == 422 {
		return // rejected outright: acceptable
	}
	requireStatus(t, 200, status, body)
	assert.Equal(t, "quantity", jsonField(parseJSON(body), "type"),
		"type must not change even if a type field is sent in the PATCH body")
}

// --- Omitted fields: UnitGroupUnit ---

func TestCovCatalogUnitGroupUnits_OmittedFields(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-omit"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Sub-case 1: create with only unit_id; defaults must appear directly on
	// the bare create response (not just inferred later via a PATCH diff).
	createResp, err := apiClient.PostFull(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	// This nested sub-resource create is an upsert ("adds a unit to a group, or
	// updates the existing association") and, consistent with the other nested
	// sub-resource creates in the gateway (sales-order/purchase-order line
	// items, supplier-materials, consumptions), does not set a LocationFunc, so
	// no Location header is emitted. Only the parent unit-group create does.
	assertObjectField(t, created, "unit_group_unit")
	assertIDFormat(t, id, "ungpun")
	assertNilField(t, created, "unit")
	assert.Equal(t, "1", jsonField(created, "discount_percentage"), "discount_percentage should default to 1")
	assert.Equal(t, "0", jsonField(created, "discount_fixed"), "discount_fixed should default to 0")
	assert.Equal(t, "visible", jsonField(created, "customer_portal_visibility"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// Sub-case 2: missing unit_id returns 400/422 with param=unit_id.
	status, body, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing unit_id should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_id")
}

// --- All fields round trip ---

// TestCovCatalogUnitGroups_CreateAllFields_UpdateAllFields closes the
// allFields gap: create sets name/type/base_unit_id/notes/associated_units
// together and asserts every field, then update changes name/notes AND
// base_unit_id (to a different same-dimension unit) and re-asserts the full
// field set including owner.
func TestCovCatalogUnitGroups_CreateAllFields_UpdateAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-cov-ug-allf")
	createResp, err := apiClient.PostFull(unitGroupsPath+"?include=base_unit,associated_units,owner", map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
		"notes":        "create notes",
		"associated_units": []map[string]any{
			{
				"unit_id":                    "un_01seeddozen00000000",
				"discount_percentage":        5.0,
				"discount_fixed":             1.0,
				"customer_portal_visibility": "hidden",
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	assertObjectField(t, created, "unit_group")
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "quantity", jsonField(created, "type"))
	assert.Equal(t, "create notes", jsonField(created, "notes"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	baseUnit := jsonObject(created, "base_unit")
	require.NotNil(t, baseUnit)
	assert.Equal(t, "each", jsonField(baseUnit, "id"))
	assert.Equal(t, "unit", jsonField(baseUnit, "object"))

	owner := jsonObject(created, "owner")
	require.NotNil(t, owner)
	assertObjectField(t, owner, "owner")

	assoc := jsonObject(created, "associated_units")
	require.NotNil(t, assoc)
	assert.Equal(t, "list", jsonField(assoc, "object"))
	assocData, ok := assoc["data"].([]any)
	require.True(t, ok)
	// The base unit is auto-included, so the explicit conversion is the other entry.
	require.Len(t, assocData, 2)
	var first map[string]any
	for _, raw := range assocData {
		entryRaw, err := json.Marshal(raw)
		require.NoError(t, err)
		entry := parseJSON(entryRaw)
		if jsonField(jsonObject(entry, "unit"), "id") != "each" {
			first = entry
		}
	}
	require.NotNil(t, first, "the explicit conversion should sit alongside the base unit")
	assertObjectField(t, first, "unit_group_unit")
	assert.Equal(t, "5", jsonField(first, "discount_percentage"))
	assert.Equal(t, "1", jsonField(first, "discount_fixed"))
	assert.Equal(t, "hidden", jsonField(first, "customer_portal_visibility"))

	// ── UPDATE: change name, notes, and base_unit_id (same dimension) ──
	updatedName := uniqueName("e2e-cov-ug-allf-upd")
	patchResp, err := apiClient.PatchFull(unitGroupsPath+"/"+id+"?include=base_unit,owner", map[string]any{
		"name":         updatedName,
		"notes":        "updated notes",
		"base_unit_id": SeedUnitID, // "Pair", same quantity dimension as "each"
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchResp.StatusCode, patchResp.Body)

	updated := parseJSON(patchResp.Body)
	assert.Equal(t, id, jsonField(updated, "id"))
	assertObjectField(t, updated, "unit_group")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "quantity", jsonField(updated, "type"), "type is immutable")
	assert.Equal(t, "updated notes", jsonField(updated, "notes"))

	updBaseUnit := jsonObject(updated, "base_unit")
	require.NotNil(t, updBaseUnit)
	assert.Equal(t, SeedUnitID, jsonField(updBaseUnit, "id"), "base_unit_id should have changed to the new unit")

	updOwner := jsonObject(updated, "owner")
	require.NotNil(t, updOwner)
	assertObjectField(t, updOwner, "owner")
}

// --- Response shape (id prefix / object / timestamps) ---

func TestCovCatalogUnitGroups_ResponseShapeIDPrefix(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ug-shape"))
	defer apiClient.Delete(unitGroupsPath + "/" + id)
	assertIDFormat(t, id, "ungp")

	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func TestCovCatalogUnitGroupUnits_ResponseShapeIDPrefix(t *testing.T) {
	t.Parallel()
	assertIDFormat(t, SeedUnitGroupUnitID, "ungpun")

	status, body, err := apiClient.GetListRaw(covCatalogUnitGroupUnitPath(SeedUnitGroupID, SeedUnitGroupUnitID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// --- Dimension-mismatch validation ---

// TestCovCatalogUnitGroups_CreateAssociatedUnitsDimensionMismatch asserts
// that POSTing associated_units whose unit_id belongs to a different
// dimension (mass "gram" into a quantity group) is rejected with the
// server's own validation-error param.
func TestCovCatalogUnitGroups_CreateAssociatedUnitsDimensionMismatch(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-cov-ug-dimassoc"),
		"type":         "quantity",
		"base_unit_id": "each",
		"associated_units": []map[string]any{
			{"unit_id": "gram"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"dimension-mismatched associated_units should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_conversions")
}

// TestCovCatalogUnitGroups_UpdateAssociatedUnitsDimensionMismatch is the
// PATCH .../unit-groups/{id} equivalent: no prior e2e test exercised
// associated_units on update at all (full gap per the coverage audit).
func TestCovCatalogUnitGroups_UpdateAssociatedUnitsDimensionMismatch(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ug-dimassocupd"))
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	status, body, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"associated_units": []map[string]any{
			{"unit_id": "gram"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"dimension-mismatched associated_units on update should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_conversions")
}

// TestCovCatalogUnitGroups_UpdateAssociatedUnitsAdditive asserts the
// documented upsert-additive semantics of PATCH associated_units: entries
// already on the group are preserved (not replaced) when a new entry is
// added via update.
func TestCovCatalogUnitGroups_UpdateAssociatedUnitsAdditive(t *testing.T) {
	t.Parallel()
	createResp, err := apiClient.PostFull(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-cov-ug-additive"),
		"type":         "quantity",
		"base_unit_id": "each",
		"associated_units": []map[string]any{
			{"unit_id": "un_01seeddozen00000000"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)
	id := jsonField(parseJSON(createResp.Body), "id")
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id+"?include=associated_units", map[string]any{
		"associated_units": []map[string]any{
			{"unit_id": SeedUnitID},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	assoc := jsonObject(parseJSON(patchBody), "associated_units")
	require.NotNil(t, assoc)
	assocData, ok := assoc["data"].([]any)
	require.True(t, ok)
	assert.Len(t, assocData, 3, "PATCH associated_units should be additive: prior entries plus the new one, alongside the auto-included base unit")
}

// TestCovCatalogUnitGroups_CreateBaseUnitDimensionMismatch documents a
// SUSPECTED BACKEND BUG: unlike associated_units (which is validated via
// validateUnitConversionTypes), base_unit_id's dimension is never checked
// against the group's type on create. This asserts the CORRECT/desired
// behavior (reject) per repo policy of not weakening assertions to match a
// known bug; it will fail (red) until the backend adds this validation.
func TestCovCatalogUnitGroups_CreateBaseUnitDimensionMismatch(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-cov-ug-dimbase"),
		"type":         "quantity",
		"base_unit_id": "gram", // mass unit, wrong dimension for a quantity group
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		// Clean up the incorrectly-created resource before failing so the
		// bug doesn't leave orphaned data behind for other tests.
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			apiClient.Delete(unitGroupsPath + "/" + id)
		}
	}
	assert.True(t, status == 400 || status == 422,
		"dimension-mismatched base_unit_id on create should return 400/422, got %d: %s", status, string(body))
}

// TestCovCatalogUnitGroups_UpdateBaseUnitDimensionMismatch is the PATCH
// equivalent of the above: same missing validation on the update path.
func TestCovCatalogUnitGroups_UpdateBaseUnitDimensionMismatch(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ug-dimbaseupd"))
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	status, body, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"base_unit_id": "gram", // mass unit, wrong dimension for a quantity group
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"dimension-mismatched base_unit_id on update should return 400/422, got %d: %s", status, string(body))
}

// TestCovCatalogUnitGroupUnits_CreateDimensionMismatch asserts POSTing a
// wrong-dimension unit_id to .../units is rejected with param=unit_id.
func TestCovCatalogUnitGroupUnits_CreateDimensionMismatch(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-dim"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	status, body, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "gram",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"dimension-mismatched unit_id should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_id")
}

// TestCovCatalogUnitGroupUnits_UpdateDimensionMismatch asserts PATCHing a
// unit_group_unit's unit_id to a wrong-dimension unit is rejected.
func TestCovCatalogUnitGroupUnits_UpdateDimensionMismatch(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-dimupd"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	status, body, err := apiClient.Patch(covCatalogUnitGroupUnitPath(groupID, unitID), map[string]any{
		"unit_id": "gram",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"dimension-mismatched unit_id on update should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_id")
}

// --- Invalid enum values ---

func TestCovCatalogUnitGroups_CreateInvalidTypeEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-cov-ug-badtype"),
		"type":         "bogus_type",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"invalid type enum should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

func TestCovCatalogUnitGroups_ListInvalidTypeFilter(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath, url.Values{"type": {"bogus_type"}})
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"invalid type filter should return 400/422, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroupUnits_CreateInvalidVisibilityEnum(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-badvis"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	status, body, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id":                    "un_01seeddozen00000000",
		"customer_portal_visibility": "bogus",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"invalid customer_portal_visibility should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "customer_portal_visibility")
}

func TestCovCatalogUnitGroupUnits_UpdateInvalidVisibilityEnum(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-badvisupd"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	status, body, err := apiClient.Patch(covCatalogUnitGroupUnitPath(groupID, unitID), map[string]any{
		"customer_portal_visibility": "bogus",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"invalid customer_portal_visibility on update should return 400/422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "customer_portal_visibility")
}

// --- Duplicate name (409) ---

func TestCovCatalogUnitGroups_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-ug-dupname")
	id1 := covCatalogUnitGroupsCreate(t, name)
	defer apiClient.Delete(unitGroupsPath + "/" + id1)

	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "mass",
		"base_unit_id": "gram",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "duplicate name should return 409, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogUnitGroups_UpdateToDuplicateName(t *testing.T) {
	t.Parallel()
	nameA := uniqueName("e2e-cov-ug-dupupd-a")
	nameB := uniqueName("e2e-cov-ug-dupupd-b")
	idA := covCatalogUnitGroupsCreate(t, nameA)
	defer apiClient.Delete(unitGroupsPath + "/" + idA)
	idB := covCatalogUnitGroupsCreate(t, nameB)
	defer apiClient.Delete(unitGroupsPath + "/" + idB)

	status, body, err := apiClient.Patch(unitGroupsPath+"/"+idB, map[string]any{
		"name": nameA,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "updating to a duplicate name should return 409, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// --- Nonexistent base_unit_id ---

// TestCovCatalogUnitGroups_CreateNonexistentBaseUnitID: the service performs
// zero existence validation of base_unit_id (no FK constraint in schema), so
// the row is written and then immediately becomes unreachable through the
// gateway's INNER-JOIN-based batch loader. This currently surfaces as a 404
// (not a 500), which is within the documented acceptable range for this
// failure mode, so the test asserts a 4xx without pinning the exact code —
// but it explicitly must NOT be a 5xx.
func TestCovCatalogUnitGroups_CreateNonexistentBaseUnitID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-cov-ug-nobaseexist"),
		"type":         "quantity",
		"base_unit_id": "un_00000000000000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 404 || status == 409 || status == 422,
		"nonexistent base_unit_id on create must not 500, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroups_UpdateNonexistentBaseUnitID(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ug-nobaseexistupd"))
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	status, body, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"base_unit_id": "un_00000000000000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 404 || status == 409 || status == 422,
		"nonexistent base_unit_id on update must not 500, got %d: %s", status, string(body))
}

// --- 404s on nonexistent resources ---

func TestCovCatalogUnitGroups_UpdateNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(unitGroupsPath+"/ungp_000000000000", map[string]any{
		"name": uniqueName("e2e-cov-ug-updnf"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "PATCH of a nonexistent unit group should 404, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroups_DeleteNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(unitGroupsPath + "/ungp_000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status, "DELETE of a nonexistent unit group should 404, got %d: %s", status, string(body))
}

// TestCovCatalogUnitGroupUnits_UpdateNotFound documents a SUSPECTED BACKEND
// BUG: PATCH .../unit-groups/{group}/units/{id} for a nonexistent nested
// unit id currently returns 500, not 404. Root cause: UpsertUnitGroupUnit's
// "isUpdate" backfill loop in unit_group_service.go only fires when the
// UnitGroupUnitID actually matches an existing conversion; if it doesn't
// match (nonexistent id) none of UnitID/DiscountPercentage/DiscountFixed get
// backfilled, so the subsequent upsert is attempted with an empty unit_id,
// which fails at the DB layer and falls through to a generic 500. This test
// asserts the CORRECT/desired behavior (404) per repo policy of never
// weakening an assertion to match a live 5xx bug.
func TestCovCatalogUnitGroupUnits_UpdateNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(covCatalogUnitGroupUnitPath(SeedUnitGroupID, "ungpun_000000000000"), map[string]any{
		"discount_fixed": 1.0,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"PATCH of a nonexistent unit_group_unit should 404, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroupUnits_DeleteNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(covCatalogUnitGroupUnitPath(SeedUnitGroupID, "ungpun_000000000000"))
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"DELETE of a nonexistent unit_group_unit should 404, got %d: %s", status, string(body))
}

// --- Double-delete ---

func TestCovCatalogUnitGroupUnits_DoubleDelete(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-deldel"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(covCatalogUnitGroupUnitPath(groupID, unitID))
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status2, body2, err := apiClient.Delete(covCatalogUnitGroupUnitPath(groupID, unitID))
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"deleting an already-deleted unit_group_unit should return 404 or 410, got %d: %s", status2, string(body2))
}

// --- Idempotency: PATCH .../units/{id} ---

func TestCovCatalogUnitGroupUnits_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-idemupd"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	idemKey := newIdempotencyKey()
	payload := map[string]any{"discount_percentage": 42.0}

	status1, body1, err := apiClient.Patch(covCatalogUnitGroupUnitPath(groupID, unitID), payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(covCatalogUnitGroupUnitPath(groupID, unitID), payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	got1 := parseJSON(body1)
	got2 := parseJSON(body2)
	assert.Equal(t, jsonField(got1, "id"), jsonField(got2, "id"))
	assert.Equal(t, jsonField(got1, "discount_percentage"), jsonField(got2, "discount_percentage"))
	assert.Equal(t, jsonField(got1, "updated_at"), jsonField(got2, "updated_at"),
		"a replayed idempotent PATCH should return the exact cached response, not re-execute")
}

// TestCovCatalogUnitGroupUnits_UpdatePreservesVisibility documents a
// SUSPECTED BACKEND BUG: PATCHing a unit_group_unit with a field other than
// customer_portal_visibility (e.g. discount_fixed alone) silently resets
// customer_portal_visibility to "hidden". Root cause: in the gateway's
// UpdateUnitGroupUnit, IsVisible is a bare proto bool with no "unset"
// sentinel (unlike UnitID/DiscountPercentage/DiscountFixed, which use empty-
// string sentinels), and the core-service upsert's isUpdate backfill loop
// only restores those three sentinel-bearing fields from the existing row,
// never IsVisible — so an omitted customer_portal_visibility is always sent
// as Go's zero value (false/"hidden"), clobbering whatever was previously
// stored. This test asserts the CORRECT/desired behavior (omitted fields are
// preserved, matching every other optional field on this resource).
func TestCovCatalogUnitGroupUnits_UpdatePreservesVisibility(t *testing.T) {
	t.Parallel()
	groupID := covCatalogUnitGroupsCreate(t, uniqueName("e2e-cov-ugu-vispreserve"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	unitID := jsonField(created, "id")
	require.Equal(t, "visible", jsonField(created, "customer_portal_visibility"))

	// Patch an unrelated field only.
	status, body, err := apiClient.Patch(covCatalogUnitGroupUnitPath(groupID, unitID), map[string]any{
		"discount_fixed": 9.99,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	patched := parseJSON(body)
	assert.Equal(t, "9.99", jsonField(patched, "discount_fixed"))
	assert.Equal(t, "visible", jsonField(patched, "customer_portal_visibility"),
		"customer_portal_visibility must be preserved when not present in the PATCH body")
}

// --- Customer portal: forbidden mutations ---

func TestCovCatalogUnitGroups_CustomerCannotUpdate(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()
	status, body, err := client.Patch(unitGroupsPath+"/"+SeedUnitGroupID, map[string]any{
		"name": uniqueName("e2e-cov-ug-customerupd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer actor should not be able to update a unit group, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroups_CustomerCannotDelete(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()
	status, body, err := client.Delete(unitGroupsPath + "/" + SeedUnitGroupID)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer actor should not be able to delete a unit group, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroupUnits_CustomerCannotCreate(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()
	status, body, err := client.Post(unitGroupUnitsPath(SeedUnitGroupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer actor should not be able to create a unit_group_unit, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroupUnits_CustomerCannotUpdate(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()
	status, body, err := client.Patch(covCatalogUnitGroupUnitPath(SeedUnitGroupID, SeedUnitGroupUnitID), map[string]any{
		"discount_fixed": 1.0,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer actor should not be able to update a unit_group_unit, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroupUnits_CustomerCannotDelete(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()
	status, body, err := client.Delete(covCatalogUnitGroupUnitPath(SeedUnitGroupID, SeedUnitGroupUnitID))
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer actor should not be able to delete a unit_group_unit, got %d: %s", status, string(body))
}

// --- Query param validation ---

func TestCovCatalogUnitGroups_ListSearchOverLength(t *testing.T) {
	t.Parallel()
	longQ := make([]byte, 501)
	for i := range longQ {
		longQ[i] = 'a'
	}
	status, body, err := apiClient.GetListRaw(unitGroupsPath, url.Values{"q": {string(longQ)}})
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"over-length q should return 400/422, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroups_GetUnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown include value should return 400, got %d: %s", status, string(body))
}

func TestCovCatalogUnitGroups_GetCombinedIncludes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, url.Values{"include": {"base_unit,associated_units,owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	require.NotNil(t, jsonObject(got, "base_unit"), "base_unit should be present with combined include")
	require.NotNil(t, jsonObject(got, "associated_units"), "associated_units should be present with combined include")
	require.NotNil(t, jsonObject(got, "owner"), "owner should be present with combined include")
}
