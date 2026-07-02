//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Additional coverage for /v1/catalog/item-categories closing gaps left by
// crud_item_categories_test.go: response-shape ID-prefix assertion, create
// validation (invalid enum + nonexistent FK), update validation (null/blank
// name+notes, too-long name), not-found on update, and the three action
// endpoints' not-found/conflict/idempotency paths. Live-verified via curl
// against the running e2e stack before writing any assertion below.

// covCatalogItemCategoriesRealPropertyID is a real, seeded property ("Size")
// that is never attached to any category created in these tests. Used to
// distinguish "property exists but isn't on this category" (no-op 200) from
// "property_id doesn't exist at all" (404).
const covCatalogItemCategoriesRealPropertyID = "pp_01k0a7ntn1egx9jjek42zsstrz"

// --- Create Validation ---

func TestCovCatalogItemCategories_CreateValidation_InvalidType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          uniqueName("e2e-itcg-badtype"),
		"type":          "bogus_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.False(t, status >= 500, "invalid type should never 5xx, got %d: %s", status, string(body))
	assert.True(t, status == 400 || status == 422,
		"Invalid type enum value should return 400 or 422, got %d: %s", status, string(body))

	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

func TestCovCatalogItemCategories_CreateValidation_NonexistentUnitGroupID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          uniqueName("e2e-itcg-badug"),
		"type":          "material_category",
		"unit_group_id": "ungp_doesnotexist00",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.False(t, status >= 500, "nonexistent unit_group_id should never 5xx, got %d: %s", status, string(body))
	assert.Equal(t, 404, status,
		"Create with nonexistent unit_group_id should return 404, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovCatalogItemCategories_CreateResponseShape_IDFormat(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-idfmt")
	createResp, err := apiClient.PostFull(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	assertIDFormat(t, id, "itcg")
}

// --- Update Validation ---

func TestCovCatalogItemCategories_UpdateValidation_NullName(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-nullname")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"name": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "Explicit null name should return 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogItemCategories_UpdateValidation_BlankName(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-blankname")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "Blank name should return 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogItemCategories_UpdateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-longname")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"name": longName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "256-char name should return 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogItemCategories_UpdateValidation_NullNotes(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-nullnotes")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"notes": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "Explicit null notes should return 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "notes")
}

func TestCovCatalogItemCategories_UpdateValidation_BlankNotes(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-blanknotes")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"notes": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "Blank notes should return 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "notes")
}

func TestCovCatalogItemCategories_UpdateNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(itemCategoriesPath+"/itcg_000000000000", map[string]any{
		"name": uniqueName("e2e-itcg-notfound"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "PATCH on nonexistent item category should return 404, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- List Validation ---

func TestCovCatalogItemCategories_ListFilterByType_InvalidValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath, url.Values{"type": {"bogus_type"}})
	require.NoError(t, err)
	require.False(t, status >= 500, "invalid type filter should never 5xx, got %d: %s", status, string(body))
	assert.Equal(t, 400, status,
		"List with invalid type filter should return 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovCatalogItemCategories_GetIncludeUnknown(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+SeedItemCategoryID, url.Values{"include": {"bogus_include"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "GET with an unknown ?include value should return 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// --- Property Management ---

func TestCovCatalogItemCategories_AddPropertyNotFound(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-addpropnf")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Put(itemCategoriesPath+"/"+id+"/properties/pp_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "Adding a nonexistent property should return 404, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovCatalogItemCategories_AddPropertyDuplicateConflict(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-dupprop")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	// First add succeeds.
	addStatus, addBody, err := apiClient.Put(itemCategoriesPath+"/"+id+"/properties/"+SeedPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, addStatus, addBody)

	// Adding the same property again is a documented conflict (same name already on the category).
	dupStatus, dupBody, err := apiClient.Put(itemCategoriesPath+"/"+id+"/properties/"+SeedPropertyID, nil)
	require.NoError(t, err)
	require.False(t, dupStatus >= 500, "duplicate property add should never 5xx, got %d: %s", dupStatus, string(dupBody))
	assert.Equal(t, 409, dupStatus,
		"Adding a duplicate property name should return 409, got %d: %s", dupStatus, string(dupBody))
	errObj := requireErrorResponse(t, dupBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "property_id")
}

func TestCovCatalogItemCategories_RemovePropertyNonexistentID(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-rmpropnf")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Delete(itemCategoriesPath + "/" + id + "/properties/pp_000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status, "Removing a property_id that doesn't exist at all should return 404, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovCatalogItemCategories_RemovePropertyNeverAddedIsNoOp(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-rmpropnoadd")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	// covCatalogItemCategoriesRealPropertyID exists as a real property but was never
	// added to this fresh category — removing it is a no-op, not a 404.
	status, body, err := apiClient.Delete(itemCategoriesPath + "/" + id + "/properties/" + covCatalogItemCategoriesRealPropertyID)
	require.NoError(t, err)
	assert.Equal(t, 200, status, "Removing a real property never associated with the category should be a 200 no-op, got %d: %s", status, string(body))
}

// --- Unit Group Management ---

func TestCovCatalogItemCategories_ChangeUnitGroupNotFound(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-chugnf")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	status, body, err := apiClient.Put(itemCategoriesPath+"/"+id+"/unit-groups/ungp_doesnotexist00", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "Changing to a nonexistent unit group should return 404, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")

	// Verify unit group was NOT changed.
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"), "unit group should remain unchanged after a failed change")
}

func TestCovCatalogItemCategories_ChangeUnitGroupRepeatCallIdempotent(t *testing.T) {
	t.Parallel()
	id := covCatalogItemCategoriesCreate(t, "e2e-itcg-chugrepeat")
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	// Sellable Socks unit group — same type (quantity) as SeedUnitGroupID (Socks).
	sameTypeUnitGroupID := "ungp_1gf7a8200f8x8jjpq5a9kdrhd"

	status1, body1, err := apiClient.Put(itemCategoriesPath+"/"+id+"/unit-groups/"+sameTypeUnitGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	// Repeating the same PUT should succeed identically with no error on the second call.
	status2, body2, err := apiClient.Put(itemCategoriesPath+"/"+id+"/unit-groups/"+sameTypeUnitGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup)
	assert.Equal(t, sameTypeUnitGroupID, jsonField(unitGroup, "id"))
}

// --- Shared helper ---

// covCatalogItemCategoriesCreate creates a minimal valid item category for use
// as a fixture in gap-fill tests and returns its id. Callers are responsible
// for cleanup via `defer apiClient.Delete(...)`.
func covCatalogItemCategoriesCreate(t *testing.T, namePrefix string) string {
	t.Helper()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          uniqueName(namePrefix),
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	return id
}
