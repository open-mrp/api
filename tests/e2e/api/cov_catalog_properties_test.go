//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake, non-existent-but-well-formed IDs for 404 test cases. propertiesPath,
// attributesPath, and attributePath are declared in crud_properties_test.go /
// crud_attributes_test.go (same package) and reused here.
const (
	covCatalogPropertiesFakePropertyID  = "pp_00000000000000000000000000"
	covCatalogPropertiesFakeAttributeID = "at_00000000000000000000"
)

// covCatalogPropertiesAssignableColors mirrors the non-default colors the
// server picks from at random when ColorCode is omitted on create
// (services/api-gateway/endpoints/properties/service.go assignableColors).
var covCatalogPropertiesAssignableColors = map[string]bool{
	"blue": true, "brown": true, "gray": true, "green": true,
	"orange": true, "pink": true, "purple": true, "red": true, "yellow": true,
}

// --- Property: Response Shape ---

func TestCovCatalogProperties_PropertyCreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covprop-shape")
	status, body, err := apiClient.Post(propertiesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	assert.NotEmpty(t, id)
	assertIDFormat(t, id, "pp")
	assertObjectField(t, got, "property")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	apiClient.Delete(propertiesPath + "/" + id)
}

// --- Property: Omitted Fields ---

func TestCovCatalogProperties_PropertyOmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateMissingNameField", func(t *testing.T) {
		status, body, err := apiClient.Post(propertiesPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"Missing name field should return 400 or 422, got %d: %s", status, string(body))
	})

	t.Run("UpdateWithEmptyBodyRejected", func(t *testing.T) {
		// An empty PATCH body ({}) is deliberately rejected with 400 by the
		// framework-wide RejectEmptyPatchBody guard
		// (services/api-gateway/pkg/endpoint/api_endpoint.go), which prevents
		// no-op updates from silently succeeding. There is no "preserve on
		// omit" success path for a body with zero body-bound fields.
		name := uniqueName("e2e-covprop-pres")
		created := createAndCleanup(t, propertiesPath, map[string]any{"name": name})
		id := jsonField(created, "id")

		patchStatus, patchBody, err := apiClient.Patch(propertiesPath+"/"+id, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 400, patchStatus, "empty PATCH body should be rejected with 400, got %d: %s", patchStatus, string(patchBody))
		requireErrorResponse(t, patchBody, "validation_failed", "invalid_request_error")
	})
}

// --- Property: Validation ---

func TestCovCatalogProperties_PropertyCreateValidation(t *testing.T) {
	t.Parallel()

	t.Run("NameTooLong", func(t *testing.T) {
		longName := make([]byte, 256)
		for i := range longName {
			longName[i] = 'a'
		}
		status, body, err := apiClient.Post(propertiesPath, map[string]any{"name": string(longName)}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"name >255 chars should return 400 or 422, got %d: %s", status, string(body))
	})

	t.Run("MissingNameKey", func(t *testing.T) {
		status, body, err := apiClient.Post(propertiesPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"Missing name should return 400 or 422, got %d: %s", status, string(body))
	})
}

func TestCovCatalogProperties_PropertyRetrieveInvalidIncludeValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+SeedPropertyID, url.Values{"include": {"not_a_real_include"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown include value should return 400, got %d: %s", status, string(body))
}

// --- Property: Not Found / Gone ---

func TestCovCatalogProperties_PropertyNotFound(t *testing.T) {
	t.Parallel()

	t.Run("Get", func(t *testing.T) {
		status, _, err := apiClient.GetListRaw(propertiesPath+"/"+covCatalogPropertiesFakePropertyID, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status)
	})

	t.Run("Patch", func(t *testing.T) {
		status, _, err := apiClient.Patch(propertiesPath+"/"+covCatalogPropertiesFakePropertyID, map[string]any{"name": "x"}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status)
	})

	t.Run("Delete", func(t *testing.T) {
		status, _, err := apiClient.Delete(propertiesPath + "/" + covCatalogPropertiesFakePropertyID)
		require.NoError(t, err)
		assert.Equal(t, 404, status)
	})
}

func TestCovCatalogProperties_PropertyDeleteAlreadyDeletedReturns410(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covprop-gone")
	status, body, err := apiClient.Post(propertiesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(propertiesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	delStatus2, delBody2, err := apiClient.Delete(propertiesPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, delStatus2, "deleting an already-deleted property should return 410, got %d: %s", delStatus2, string(delBody2))
}

// --- Property: Expandable Fields ---

func TestCovCatalogProperties_PropertyExpandableNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+SeedPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "attributes")
}

// TestCovCatalogProperties_PropertyIncludeAttributesEmptyWhenNoAttributes documents
// expected behavior: SUSPECTED BACKEND BUG. Requesting ?include=attributes on a property
// with zero attributes returns "attributes": null instead of a populated-but-empty list
// object ({"object":"list","data":[]}). This makes "included but empty" indistinguishable
// from "not included" for API consumers, and contradicts the field's `expandable:"true"`
// contract. Root cause: services/api-gateway/internal/resourceloaders/property_loader.go
// LoadProperties only calls meta.Set(...,"attributes_list",...) when len(p.Attributes) > 0.
func TestCovCatalogProperties_PropertyIncludeAttributesEmptyWhenNoAttributes(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covprop-noattrs")
	created := createAndCleanup(t, propertiesPath, map[string]any{"name": name})
	id := jsonField(created, "id")

	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+id, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be a populated (possibly empty) list object when ?include=attributes is requested, even with zero attributes — got null")
	assert.Equal(t, "list", jsonField(attrs, "object"))
	data := jsonArray(attrs, "data")
	assert.Empty(t, data)
}

// --- Property: Idempotency ---

func TestCovCatalogProperties_PropertyUpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covprop-idemupd")
	created := createAndCleanup(t, propertiesPath, map[string]any{"name": name})
	id := jsonField(created, "id")

	idemKey := newIdempotencyKey()
	updatedName := uniqueName("e2e-covprop-idemupd2")

	status1, body1, err := apiClient.Patch(propertiesPath+"/"+id, map[string]any{"name": updatedName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	updatedAt1 := jsonField(parseJSON(body1), "updated_at")

	status2, body2, err := apiClient.Patch(propertiesPath+"/"+id, map[string]any{"name": updatedName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	got2 := parseJSON(body2)
	assert.Equal(t, id, jsonField(got2, "id"))
	assert.Equal(t, updatedName, jsonField(got2, "name"))
	assert.Equal(t, updatedAt1, jsonField(got2, "updated_at"), "repeated PATCH with the same idempotency key should return the identical cached response")
}

// --- Property: List ---

func TestCovCatalogProperties_PropertyListPagination(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, propertiesPath, nil)
}

func TestCovCatalogProperties_PropertyListInvalidLimit(t *testing.T) {
	t.Parallel()

	t.Run("Zero", func(t *testing.T) {
		status, _, err := apiClient.GetListRaw(propertiesPath, url.Values{"limit": {"0"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status)
	})

	t.Run("AboveMax", func(t *testing.T) {
		status, _, err := apiClient.GetListRaw(propertiesPath, url.Values{"limit": {"1001"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status)
	})

	t.Run("Negative", func(t *testing.T) {
		status, _, err := apiClient.GetListRaw(propertiesPath, url.Values{"limit": {"-1"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status)
	})
}

func TestCovCatalogProperties_PropertyListQueryTooLong(t *testing.T) {
	t.Parallel()
	longQuery := make([]byte, 501)
	for i := range longQuery {
		longQuery[i] = 'a'
	}
	status, _, err := apiClient.GetListRaw(propertiesPath, url.Values{"q": {string(longQuery)}})
	require.NoError(t, err)
	assert.Equal(t, 400, status)
}

// --- Attribute: All Fields (incl. `property`) ---

// TestCovCatalogProperties_AttributeCreateAllFieldsAndPropertyNil creates an
// attribute under a dedicated (single-attribute) property so sort_order is
// deterministic, and asserts every json field on Attribute — including
// `property`, which is never populated by this endpoint group (no
// `expandable:"true"` tag on the struct field, so AssertExpandablesNil will
// not catch a regression here; this test is the explicit guard).
func TestCovCatalogProperties_AttributeCreateAllFieldsAndPropertyNil(t *testing.T) {
	t.Parallel()
	prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-attrall")})
	propID := jsonField(prop, "id")

	value := uniqueName("e2e-covattr-allf")
	createStatus, createBody, err := apiClient.Post(attributesPath(propID), map[string]any{
		"value": value,
		"color": "purple",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(attributePath(propID, id))

	assertObjectField(t, got, "attribute")
	assert.Equal(t, value, jsonField(got, "value"))
	assert.Equal(t, "purple", jsonField(got, "color"))
	assert.Equal(t, "1", jsonField(got, "sort_order"), "only attribute under a fresh property should default to sort_order 1")
	assertNilField(t, got, "property")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// Also assert `property` stays nil when the attribute is nested under the
	// parent Property's `attributes` list (?include=attributes).
	propStatus, propBody, err := apiClient.GetListRaw(propertiesPath+"/"+propID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, propStatus, propBody)
	attrs := jsonObject(parseJSON(propBody), "attributes")
	require.NotNil(t, attrs)
	data := jsonArray(attrs, "data")
	require.Len(t, data, 1)
	nested, ok := data[0].(map[string]any)
	require.True(t, ok)
	assertNilField(t, nested, "property")
}

// --- Attribute: Response Shape ---

func TestCovCatalogProperties_AttributeCreateResponseShape(t *testing.T) {
	t.Parallel()
	value := uniqueName("e2e-covattr-shape")
	status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": value}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	assert.NotEmpty(t, id)
	assertIDFormat(t, id, "at")
	assertObjectField(t, got, "attribute")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	apiClient.Delete(attributePath(SeedPropertyID, id))
}

// --- Attribute: Omitted Fields ---

func TestCovCatalogProperties_AttributeOmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateDefaultsForColorAndSortOrder", func(t *testing.T) {
		prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-omit")})
		propID := jsonField(prop, "id")

		value := uniqueName("e2e-covattr-omit")
		status, body, err := apiClient.Post(attributesPath(propID), map[string]any{"value": value}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(attributePath(propID, id))

		color := jsonField(got, "color")
		assert.True(t, covCatalogPropertiesAssignableColors[color],
			"omitted color should default to one of the 9 non-default colors, got %q", color)
		assert.Equal(t, "1", jsonField(got, "sort_order"), "omitted sort_order should default to the last position")
	})

	t.Run("CreateMissingValue", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"color": "blue"}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"Missing value should return 400 or 422, got %d: %s", status, string(body))
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		// Use a dedicated property, not the shared SeedPropertyID: deleting an
		// attribute renumbers its siblings' sort_order (ShiftOrdersDown), so a
		// parallel test that creates and deletes attributes on the seed property
		// would shift this attribute's sort_order out from under the assertion.
		prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-pres")})
		propID := jsonField(prop, "id")

		value := uniqueName("e2e-covattr-pres")
		created := createAndCleanup(t, attributesPath(propID), map[string]any{
			"value": value,
			"color": "purple",
		})
		id := jsonField(created, "id")
		origSortOrder := jsonField(created, "sort_order")

		newValue := uniqueName("e2e-covattr-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(attributePath(propID, id), map[string]any{"value": newValue}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newValue, jsonField(got, "value"))
		assert.Equal(t, "purple", jsonField(got, "color"), "color should be preserved when PATCH omits it")
		assert.Equal(t, origSortOrder, jsonField(got, "sort_order"), "sort_order should be preserved when PATCH omits it")
	})
}

// --- Attribute: List ---

func TestCovCatalogProperties_AttributeListPagination(t *testing.T) {
	t.Parallel()
	prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-pag")})
	propID := jsonField(prop, "id")

	createAndCleanup(t, attributesPath(propID), map[string]any{"value": uniqueName("e2e-covattr-pag1")})
	createAndCleanup(t, attributesPath(propID), map[string]any{"value": uniqueName("e2e-covattr-pag2")})

	// This property is exclusively owned by this test, so the attributes list
	// is fully scoped — no parallel test can add/remove rows underneath it.
	assertCursorPaginationAdvances(t, attributesPath(propID), nil)
}

// --- Attribute: Idempotency ---

func TestCovCatalogProperties_AttributeCreateIdempotent(t *testing.T) {
	t.Parallel()
	value := uniqueName("e2e-covattr-idem")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": value}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": value}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(attributePath(SeedPropertyID, id1))
}

func TestCovCatalogProperties_AttributeUpdateIdempotent(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-idemupd")})
	id := jsonField(created, "id")

	idemKey := newIdempotencyKey()
	status1, body1, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"color": "orange"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	updatedAt1 := jsonField(parseJSON(body1), "updated_at")

	status2, body2, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"color": "orange"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	got2 := parseJSON(body2)
	assert.Equal(t, id, jsonField(got2, "id"))
	assert.Equal(t, "orange", jsonField(got2, "color"))
	assert.Equal(t, updatedAt1, jsonField(got2, "updated_at"), "repeated PATCH with the same idempotency key should return the identical cached response")
}

// --- Attribute: Validation — value ---

func TestCovCatalogProperties_AttributeValidationValue(t *testing.T) {
	t.Parallel()

	t.Run("CreateMissingValue", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing value should return 400/422, got %d: %s", status, string(body))
	})

	t.Run("CreateEmptyValue", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": ""}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"empty value should return 400/422, got %d: %s", status, string(body))
	})

	// SUSPECTED BACKEND BUG: services/core-service/internal/service/attribute_service.go
	// CreateAttribute trims params.Value with strings.TrimSpace but never checks the
	// trimmed result for emptiness before persisting (unlike UpdateAttribute, which does
	// check — see the sibling UpdateBlankAfterTrim subtest below, which passes). A
	// whitespace-only value is accepted with 201 and stored/returned as "".
	t.Run("CreateBlankAfterTrim", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": "   "}, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			id := jsonField(parseJSON(body), "id")
			defer apiClient.Delete(attributePath(SeedPropertyID, id))
		}
		assert.True(t, status == 400 || status == 422,
			"whitespace-only value should be rejected as blank-after-trim (400/422), got %d: %s", status, string(body))
	})

	t.Run("UpdateBlankAfterTrim", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-blankupd")})
		id := jsonField(created, "id")

		status, body, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"value": "   "}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"whitespace-only value on update should be rejected as blank-after-trim, got %d: %s", status, string(body))
	})
}

// --- Attribute: Validation — color ---

func TestCovCatalogProperties_AttributeValidationColor(t *testing.T) {
	t.Parallel()

	t.Run("CreateInvalidColor", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{
			"value": uniqueName("e2e-covattr-badcolor"),
			"color": "not-a-real-color",
		}, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			id := jsonField(parseJSON(body), "id")
			defer apiClient.Delete(attributePath(SeedPropertyID, id))
		}
		assert.True(t, status == 400 || status == 422,
			"invalid color enum should return 400/422, got %d: %s", status, string(body))
	})

	t.Run("UpdateInvalidColor", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-badcolorupd")})
		id := jsonField(created, "id")

		status, body, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"color": "not-a-real-color"}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"invalid color enum on update should return 400/422, got %d: %s", status, string(body))
	})
}

// --- Attribute: Validation — sort_order ---

func TestCovCatalogProperties_AttributeValidationSortOrder(t *testing.T) {
	t.Parallel()

	// SUSPECTED BACKEND BUG: CreateAttributeRequest.SortOrder / UpdateAttributeRequest.SortOrder
	// are field.Optional[int32] with `validate:"omitempty,min=1"`. field.RegisterValidator's
	// custom type func unwraps a *set* Optional[int32] to its bare int32 value for validation —
	// but go-playground/validator's "omitempty" then treats an explicit 0 as the Go zero value
	// and SKIPS the "min=1" check entirely, so explicit sort_order=0 is silently accepted
	// instead of rejected. On create it's ignored (falls back to default last-position); on
	// update it is actually persisted, producing a non-contiguous, non-1-based sort_order that
	// violates the documented invariant ("Positions are kept contiguous... starting at 1").
	t.Run("CreateZero", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{
			"value":      uniqueName("e2e-covattr-sort0"),
			"sort_order": 0,
		}, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			id := jsonField(parseJSON(body), "id")
			defer apiClient.Delete(attributePath(SeedPropertyID, id))
		}
		assert.True(t, status == 400 || status == 422,
			"sort_order=0 should be rejected by validate:\"min=1\", got %d: %s", status, string(body))
	})

	t.Run("UpdateZero", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-sort0upd")})
		id := jsonField(created, "id")

		status, body, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"sort_order": 0}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"sort_order=0 on update should be rejected by validate:\"min=1\", got %d: %s", status, string(body))
	})

	t.Run("CreateNegative", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{
			"value":      uniqueName("e2e-covattr-sortneg"),
			"sort_order": -5,
		}, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			id := jsonField(parseJSON(body), "id")
			defer apiClient.Delete(attributePath(SeedPropertyID, id))
		}
		assert.True(t, status == 400 || status == 422,
			"negative sort_order should return 400/422, got %d: %s", status, string(body))
	})

	t.Run("UpdateNegative", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-sortnegupd")})
		id := jsonField(created, "id")

		status, body, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"sort_order": -5}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"negative sort_order on update should return 400/422, got %d: %s", status, string(body))
	})

	t.Run("CreateAboveMax", func(t *testing.T) {
		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{
			"value":      uniqueName("e2e-covattr-sorthigh"),
			"sort_order": 999,
		}, newIdempotencyKey())
		require.NoError(t, err)
		if status == 201 {
			id := jsonField(parseJSON(body), "id")
			defer apiClient.Delete(attributePath(SeedPropertyID, id))
		}
		assert.True(t, status == 400 || status == 422,
			"sort_order above count+1 should return 400/422, got %d: %s", status, string(body))
	})

	t.Run("UpdateAboveMax", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-sorthighupd")})
		id := jsonField(created, "id")

		status, body, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{"sort_order": 999}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"sort_order above current attribute count on update should return 400/422, got %d: %s", status, string(body))
	})
}

// --- Attribute: Validation — duplicate value (409) ---

func TestCovCatalogProperties_AttributeValidationDuplicateValue(t *testing.T) {
	t.Parallel()

	t.Run("CreateDuplicateSameProperty", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-dup1")})
		value := jsonField(created, "value")

		status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": value}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 409, status, "duplicate value within the same property should return 409, got %d: %s", status, string(body))
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "value")
	})

	t.Run("CreateDuplicateCrossProperty", func(t *testing.T) {
		created := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-dupx")})
		value := jsonField(created, "value")

		otherProp := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-dupx")})
		otherPropID := jsonField(otherProp, "id")

		// Attribute values are unique account-wide, not just within a property.
		status, body, err := apiClient.Post(attributesPath(otherPropID), map[string]any{"value": value}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 409, status, "duplicate value across two different properties should return 409, got %d: %s", status, string(body))
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "value")
	})

	t.Run("UpdateCollidesWithSibling", func(t *testing.T) {
		prop := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-dupupd")})
		propID := jsonField(prop, "id")

		a1 := createAndCleanup(t, attributesPath(propID), map[string]any{"value": uniqueName("e2e-covattr-sib1")})
		a1Value := jsonField(a1, "value")
		a2 := createAndCleanup(t, attributesPath(propID), map[string]any{"value": uniqueName("e2e-covattr-sib2")})
		a2ID := jsonField(a2, "id")

		status, body, err := apiClient.Patch(attributePath(propID, a2ID), map[string]any{"value": a1Value}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 409, status, "updating to a value that collides with a sibling attribute should return 409, got %d: %s", status, string(body))
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "value")
	})
}

// --- Attribute: Not Found / Gone / Cross-Property ---

func TestCovCatalogProperties_AttributeNotFound(t *testing.T) {
	t.Parallel()

	t.Run("Get", func(t *testing.T) {
		status, _, err := apiClient.GetListRaw(attributePath(SeedPropertyID, covCatalogPropertiesFakeAttributeID), nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status)
	})

	t.Run("Patch", func(t *testing.T) {
		status, _, err := apiClient.Patch(attributePath(SeedPropertyID, covCatalogPropertiesFakeAttributeID), map[string]any{"value": "x"}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status)
	})

	t.Run("Delete", func(t *testing.T) {
		status, _, err := apiClient.Delete(attributePath(SeedPropertyID, covCatalogPropertiesFakeAttributeID))
		require.NoError(t, err)
		assert.Equal(t, 404, status)
	})
}

func TestCovCatalogProperties_AttributeCreateUnderNonexistentPropertyReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(attributesPath(covCatalogPropertiesFakePropertyID), map[string]any{
		"value": uniqueName("e2e-covattr-noprop"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "creating an attribute under a nonexistent property_id should return 404, got %d: %s", status, string(body))
}

// TestCovCatalogProperties_AttributeCrossPropertyGetLeaksData documents a
// SUSPECTED BACKEND BUG: RetrieveAttributeEndpoint's service handler
// (services/api-gateway/endpoints/properties/service.go GetAttribute) calls
// loadAttributeByID(ctx, req.AttributeID) and never uses req.PropertyID at all,
// unlike UpdateAttribute/DeleteAttribute which forward PropertyId to the
// gRPC request and get correctly scoped 404s. As a result, GET
// /v1/catalog/properties/{wrong_property_id}/attributes/{id} returns 200 with
// the attribute's full data regardless of which property_id is in the path —
// a cross-property (and potentially cross-tenant scoping, though this
// specific attribute stays within the same account here) data leak.
func TestCovCatalogProperties_AttributeCrossPropertyGetLeaksData(t *testing.T) {
	t.Parallel()
	otherProp := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-leak")})
	otherPropID := jsonField(otherProp, "id")

	// SeedAttributeID belongs to SeedPropertyID, not otherPropID.
	status, body, err := apiClient.GetListRaw(attributePath(otherPropID, SeedAttributeID), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"GET attribute scoped to the wrong property_id must 404, not leak the attribute; got %d: %s", status, string(body))
}

func TestCovCatalogProperties_AttributeCrossPropertyPatchAndDeleteReturn404(t *testing.T) {
	t.Parallel()
	otherProp := createAndCleanup(t, propertiesPath, map[string]any{"name": uniqueName("e2e-covprop-xprop")})
	otherPropID := jsonField(otherProp, "id")

	attr := createAndCleanup(t, attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-covattr-xprop")})
	attrID := jsonField(attr, "id")

	patchStatus, patchBody, err := apiClient.Patch(attributePath(otherPropID, attrID), map[string]any{"value": "hacked-value"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, patchStatus, "PATCH via wrong property_id should 404, got %d: %s", patchStatus, string(patchBody))

	delStatus, delBody, err := apiClient.Delete(attributePath(otherPropID, attrID))
	require.NoError(t, err)
	assert.Equal(t, 404, delStatus, "DELETE via wrong property_id should 404, got %d: %s", delStatus, string(delBody))

	// Confirm the attribute is untouched via its real property.
	getStatus, getBody, err := apiClient.GetListRaw(attributePath(SeedPropertyID, attrID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.NotEqual(t, "hacked-value", jsonField(parseJSON(getBody), "value"), "attribute value should be unchanged after a wrong-property PATCH attempt")
}

func TestCovCatalogProperties_AttributeDeleteAlreadyDeletedReturns410(t *testing.T) {
	t.Parallel()
	value := uniqueName("e2e-covattr-gone")
	status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": value}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(attributePath(SeedPropertyID, id))
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	delStatus2, delBody2, err := apiClient.Delete(attributePath(SeedPropertyID, id))
	require.NoError(t, err)
	assert.Equal(t, 410, delStatus2, "deleting an already-deleted attribute should return 410, got %d: %s", delStatus2, string(delBody2))
}

// --- Attribute: Include (unsupported) ---

// TestCovCatalogProperties_AttributeIncludeQueryParamUnsupported documents
// actual (and correct, per spec §3/§9.3) behavior: attribute endpoints have
// no IncludeConfig at all, so ?include=property is rejected as an unknown
// query parameter rather than being silently ignored or populating the field.
func TestCovCatalogProperties_AttributeIncludeQueryParamUnsupported(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(attributePath(SeedPropertyID, SeedAttributeID), url.Values{"include": {"property"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "?include=property on an attribute endpoint (no IncludeConfig) should be rejected, got %d: %s", status, string(body))

	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "include")
}
