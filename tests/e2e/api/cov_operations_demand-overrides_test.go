//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	demandOverridesPath     = "/v1/operations/demand-overrides"
	demandOverrideTypesPath = "/v1/operations/demand-override-types"
)

// overridePeriod returns a period well clear of any planning window the schedule tests look at, so creating overrides here cannot move their numbers.
func overridePeriod() (string, string) {
	start := time.Now().UTC().AddDate(-3, 0, 0)
	return rfc3339(start), rfc3339(start.AddDate(0, 3, 0))
}

func createOverride(t *testing.T, extra map[string]any) map[string]any {
	t.Helper()

	start, end := overridePeriod()
	body := map[string]any{
		"scope_type":       "item",
		"scope_ref_id":     SeedItemID,
		"period_starts_at": start,
		"period_ends_at":   end,
		"adjustment":       "delta_units",
		"value":            1500,
	}
	for k, v := range extra {
		body[k] = v
	}

	resp, err := apiClient.PostFull(demandOverridesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	created := parseJSON(resp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { deleteOverride(t, id) })
	return created
}

func deleteOverride(t *testing.T, id string) {
	t.Helper()
	status, body, err := apiClient.Delete(demandOverridesPath + "/" + id)
	require.NoError(t, err)
	// 404 is fine: a test that deletes explicitly still runs its deferred cleanup.
	if status != 200 && status != 204 && status != 404 {
		t.Fatalf("cleanup delete %s returned %d: %s", id, status, string(body))
	}
}

// ──────────────────────────────────────────────
// Types (the seeded, global taxonomy)
// ──────────────────────────────────────────────

func TestDemandOverrideTypes_ListReturnsSeededTaxonomy(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(demandOverrideTypesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	types := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, types, "the override type taxonomy must be seeded")

	seen := map[string]bool{}
	for _, raw := range types {
		entry, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "demand_override_type", jsonField(entry, "object"))
		assert.NotEmpty(t, jsonField(entry, "name"))
		seen[jsonField(entry, "code")] = true
	}

	// These three codes are the whole vocabulary the solver understands. A missing one means overrides of that kind silently cannot be created.
	for _, code := range []string{"absolute", "delta_units", "delta_percent"} {
		assert.True(t, seen[code], "override type %q must be seeded", code)
	}
}

// ──────────────────────────────────────────────
// Create
// ──────────────────────────────────────────────

func TestDemandOverrides_CreateItemScopedRoundTrips(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()
	created := createOverride(t, map[string]any{
		"reason": "new_customer",
		"note":   "e2e: large customer about to order",
	})

	assert.Equal(t, "demand_override", jsonField(created, "object"))
	assert.Equal(t, "item", jsonField(created, "scope_type"))
	assertNilField(t, created, "scope") // expandable, so null until asked for
	assert.Equal(t, "delta_units", jsonField(created, "adjustment"))
	assert.Equal(t, "new_customer", jsonField(created, "reason"))
	assert.Equal(t, "active", jsonField(created, "status"), "an override defaults to active")
	assert.NotEmpty(t, jsonField(created, "effective_at"), "effective_at defaults to now")

	value, ok := created["value"].(float64)
	require.True(t, ok, "value should be numeric")
	assert.InDelta(t, 1500, value, 0.0001)

	// Dates round-trip as the calendar days they were sent as.
	assert.Equal(t, start[:10], jsonField(created, "period_starts_at")[:10])
	assert.Equal(t, end[:10], jsonField(created, "period_ends_at")[:10])
}

func TestDemandOverrides_CreateProductLineScoped(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{
		"scope_type":   "product_line",
		"scope_ref_id": SeedProductLineID,
		"adjustment":   "delta_percent",
		"value":        12.5,
	})

	assert.Equal(t, "product_line", jsonField(created, "scope_type"))

	// The reference itself only comes back when the scope is expanded.
	status, body, err := apiClient.GetListRaw(demandOverridesPath+"/"+jsonField(created, "id"),
		url.Values{"include": []string{"scope"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	scope := jsonObject(parseJSON(body), "scope")
	require.NotNil(t, scope, "?include=scope must expand a line-scoped override")
	assert.Equal(t, SeedProductLineID, jsonField(scope, "id"))
}

// An override that matches nothing is worse than an error: the plan looks adjusted and is not. The API must reject an unknown scope reference rather than store it.
func TestDemandOverrides_CreateRejectsUnknownScopeRef(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()
	status, body, err := apiClient.Post(demandOverridesPath, map[string]any{
		"scope_type":       "item",
		"scope_ref_id":     "it_01doesnotexist00000",
		"period_starts_at": start,
		"period_ends_at":   end,
		"adjustment":       "delta_units",
		"value":            100,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "scope_ref_id")
}

func TestDemandOverrides_CreateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()

	cases := []struct {
		name  string
		patch map[string]any
	}{
		{
			// Below -100% the demand goes negative, which the solver would read as supply and plan against.
			name:  "percent below -100",
			patch: map[string]any{"adjustment": "delta_percent", "value": -150},
		},
		{
			name:  "negative absolute",
			patch: map[string]any{"adjustment": "absolute", "value": -10},
		},
		{
			name:  "period ends before it starts",
			patch: map[string]any{"period_starts_at": end, "period_ends_at": start},
		},
		{
			name:  "unknown scope",
			patch: map[string]any{"scope_type": "customer"},
		},
		{
			name:  "unknown type",
			patch: map[string]any{"adjustment": "multiply"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := map[string]any{
				"scope_type":       "item",
				"scope_ref_id":     SeedItemID,
				"period_starts_at": start,
				"period_ends_at":   end,
				"adjustment":       "delta_units",
				"value":            100,
			}
			for k, v := range tc.patch {
				body[k] = v
			}

			status, respBody, err := apiClient.Post(demandOverridesPath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, respBody)
		})
	}
}

// ──────────────────────────────────────────────
// Retrieve and includes
// ──────────────────────────────────────────────

func TestDemandOverrides_RetrieveExpandsScopeTarget(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	id := jsonField(created, "id")

	status, body, err := apiClient.GetListRaw(demandOverridesPath+"/"+id, url.Values{"include": []string{"scope"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	scope := jsonObject(parseJSON(body), "scope")
	require.NotNil(t, scope, "?include=scope must expand an item-scoped override")
	assert.Equal(t, SeedItemID, jsonField(scope, "id"))
	// type names which resource the id points at; object is always "entity".
	assert.Equal(t, "item", jsonField(scope, "type"))
	assert.Equal(t, "entity", jsonField(scope, "object"))
	assert.Equal(t, SeedItemSKU, jsonField(scope, "handle"), "an item scope carries its SKU")
}

func TestDemandOverrides_RetrieveExpandsProductLineScope(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{
		"scope_type":   "product_line",
		"scope_ref_id": SeedProductLineID,
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.GetListRaw(demandOverridesPath+"/"+id, url.Values{"include": []string{"scope"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The same include key resolves a different resource type — which is the whole reason the scope is one polymorphic reference rather than two typed fields.
	scope := jsonObject(parseJSON(body), "scope")
	require.NotNil(t, scope, "?include=scope must expand a line-scoped override")
	assert.Equal(t, SeedProductLineID, jsonField(scope, "id"))
	assert.Equal(t, "entity", jsonField(scope, "object"))
	assert.Equal(t, "product_line", jsonField(scope, "type"))
	assert.Equal(t, SeedProductLineName, jsonField(scope, "name"))
}

func TestDemandOverrides_RetrieveUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(demandOverridesPath+"/deov_01doesnotexist0000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// List filters
// ──────────────────────────────────────────────

func TestDemandOverrides_ListFiltersByScope(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{
		"scope_type":   "product_line",
		"scope_ref_id": SeedProductLineID,
	})
	wantID := jsonField(created, "id")

	params := url.Values{}
	params.Set("scope_types[]", "product_line")
	params.Set("limit", "100")

	status, body, err := apiClient.GetListRaw(demandOverridesPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, rows)

	found := false
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "product_line", jsonField(row, "scope_type"), "scope filter must exclude other scopes")
		if jsonField(row, "id") == wantID {
			found = true
		}
	}
	assert.True(t, found, "the created override should appear under its own scope filter")
}

func TestDemandOverrides_ListFiltersByActive(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{"active": false})
	wantID := jsonField(created, "id")

	params := url.Values{}
	params.Set("statuses[]", "inactive")
	params.Set("limit", "100")

	status, body, err := apiClient.GetListRaw(demandOverridesPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, rows)

	found := false
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "inactive", jsonField(row, "status"), "active=false must exclude active overrides")
		if jsonField(row, "id") == wantID {
			found = true
		}
	}
	assert.True(t, found, "the inactive override should appear under active=false")
}

// The period filters match on overlap, not containment: an override spanning a quarter is relevant to a question about one month inside it.
func TestDemandOverrides_ListPeriodFilterMatchesOverlap(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	wantID := jsonField(created, "id")

	start := time.Now().UTC().AddDate(-3, 0, 0)
	inside := start.AddDate(0, 1, 0)

	params := url.Values{}
	params.Set("period_start", rfc3339(inside))
	params.Set("period_end", rfc3339(inside.AddDate(0, 0, 1)))
	params.Set("limit", "100")

	status, body, err := apiClient.GetListRaw(demandOverridesPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := jsonArray(parseJSON(body), "data")
	found := false
	for _, raw := range rows {
		if row, ok := raw.(map[string]any); ok && jsonField(row, "id") == wantID {
			found = true
		}
	}
	assert.True(t, found, "a window inside the override's period must match it")

	// A window entirely before the period must not.
	before := start.AddDate(-1, 0, 0)
	params.Set("period_start", rfc3339(before))
	params.Set("period_end", rfc3339(before.AddDate(0, 0, 1)))

	status, body, err = apiClient.GetListRaw(demandOverridesPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row, ok := raw.(map[string]any); ok {
			assert.NotEqual(t, wantID, jsonField(row, "id"), "a window outside the period must not match")
		}
	}
}

func TestDemandOverrides_ListSearchMatchesNote(t *testing.T) {
	t.Parallel()

	marker := uniqueName("e2e-override-note")
	created := createOverride(t, map[string]any{"note": marker})
	wantID := jsonField(created, "id")

	params := url.Values{}
	params.Set("q", marker)

	status, body, err := apiClient.GetListRaw(demandOverridesPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := jsonArray(parseJSON(body), "data")
	require.Len(t, rows, 1, "the marker note should match exactly one override")
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, wantID, jsonField(row, "id"))
}

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

func TestDemandOverrides_UpdateChangesValueAndClearsNote(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{"note": "original note"})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{
		"value": 9000,
		"note":  nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	value, ok := updated["value"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 9000, value, 0.0001)
	assertNilField(t, updated, "note")
}

func TestDemandOverrides_UpdateDeactivates(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	id := jsonField(created, "id")
	require.Equal(t, "active", jsonField(created, "status"))

	status, body, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{"active": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.Equal(t, "inactive", jsonField(parseJSON(body), "status"))
}

// Switching the type has to re-validate the value it will be interpreted as, even when only the type is sent: +1500 units is fine, +1500 percent is not the same claim.
func TestDemandOverrides_UpdateValidatesTypeAgainstExistingValue(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{
		"adjustment": "delta_units",
		"value":      -500,
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{
		"adjustment": "absolute",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "value")
}

func TestDemandOverrides_UpdateRejectsInvertedPeriod(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	id := jsonField(created, "id")

	start, _ := overridePeriod()
	status, body, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{
		"period_ends_at": rfc3339(time.Now().UTC().AddDate(-5, 0, 0)),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "period_ends_at")
	_ = start
}

// ──────────────────────────────────────────────
// Delete
// ──────────────────────────────────────────────

func TestDemandOverrides_DeleteRemovesIt(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	id := jsonField(created, "id")

	status, body, err := apiClient.Delete(demandOverridesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(demandOverridesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestDemandOverrides_DeleteUnknownIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(demandOverridesPath + "/deov_01doesnotexist0000")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// CRUD lifecycle, shapes and idempotency
// ──────────────────────────────────────────────

func TestDemandOverrides_CRUD(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()

	// CREATE
	createStatus, createBody, err := apiClient.Post(demandOverridesPath, map[string]any{
		"scope_type":       "item",
		"scope_ref_id":     SeedItemID,
		"period_starts_at": start,
		"period_ends_at":   end,
		"adjustment":       "delta_units",
		"value":            1500,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assert.Equal(t, "demand_override", jsonField(created, "object"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(demandOverridesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, "item", jsonField(got, "scope_type"))

	// UPDATE
	patchStatus, patchBody, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{
		"value": 2000,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.EqualValues(t, 2000, updated["value"])

	// DELETE
	delStatus, delBody, err := apiClient.Delete(demandOverridesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus2, _, err := apiClient.GetListRaw(demandOverridesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestDemandOverrides_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()
	expiresAt := rfc3339(time.Now().UTC().AddDate(1, 0, 0))
	note := uniqueName("e2e-override-allf")

	createStatus, createBody, err := apiClient.Post(demandOverridesPath+"?include=scope", map[string]any{
		"scope_type":       "item",
		"scope_ref_id":     SeedItemID,
		"period_starts_at": start,
		"period_ends_at":   end,
		"adjustment":       "delta_units",
		"value":            1500,
		"unit_id":          SeedUnitID,
		"reason":           "new_customer",
		"note":             note,
		"expires_at":       expiresAt,
		"active":           true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { deleteOverride(t, id) })

	// Assert EVERY field of the resource.
	assertIDFormat(t, id, "deov")
	assertObjectField(t, got, "demand_override")
	assert.Equal(t, "item", jsonField(got, "scope_type"))
	scope := jsonObject(got, "scope")
	require.NotNil(t, scope, "scope must be present with ?include=scope")
	assert.Equal(t, SeedItemID, jsonField(scope, "id"))
	assert.Equal(t, "entity", jsonField(scope, "object"))
	assert.Equal(t, start[:10], jsonField(got, "period_starts_at")[:10])
	assert.Equal(t, end[:10], jsonField(got, "period_ends_at")[:10])
	assert.Equal(t, "delta_units", jsonField(got, "adjustment"))
	assert.EqualValues(t, 1500, got["value"])
	assertNilField(t, got, "unit") // expandable and not included
	assert.Equal(t, "new_customer", jsonField(got, "reason"))
	assert.Equal(t, note, jsonField(got, "note"))
	assertNilField(t, got, "created_by") // expandable and not included
	assertValidTimestamp(t, jsonField(got, "effective_at"), "effective_at")
	assertValidTimestamp(t, jsonField(got, "expires_at"), "expires_at")
	assert.Equal(t, "active", jsonField(got, "status"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// UPDATE with different values, asserting updated and preserved fields.
	patchStatus, patchBody, err := apiClient.Patch(demandOverridesPath+"/"+id+"?include=scope", map[string]any{
		"value":  2500,
		"note":   note + "-upd",
		"reason": "promotion",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.EqualValues(t, 2500, updated["value"])
	assert.Equal(t, note+"-upd", jsonField(updated, "note"))
	assert.Equal(t, "promotion", jsonField(updated, "reason"))
	assert.Equal(t, "delta_units", jsonField(updated, "adjustment"), "adjustment should be preserved")
	assert.Equal(t, start[:10], jsonField(updated, "period_starts_at")[:10], "the period should be preserved")
	updScope := jsonObject(updated, "scope")
	require.NotNil(t, updScope, "scope should be preserved")
	assert.Equal(t, SeedItemID, jsonField(updScope, "id"))
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
}

func TestDemandOverrides_OmittedFields(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()
	requiredBody := func() map[string]any {
		return map[string]any{
			"scope_type":       "item",
			"scope_ref_id":     SeedItemID,
			"period_starts_at": start,
			"period_ends_at":   end,
			"adjustment":       "delta_units",
			"value":            1500,
		}
	}

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		status, body, err := apiClient.Post(demandOverridesPath, requiredBody(), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		t.Cleanup(func() { deleteOverride(t, id) })

		assertObjectField(t, got, "demand_override")
		assertNilField(t, got, "note")
		assertNilField(t, got, "unit")
		assertNilField(t, got, "reason")
		assertNilField(t, got, "expires_at")
		assertNilField(t, got, "scope")
		assertNilField(t, got, "created_by")
		assert.Equal(t, "active", jsonField(got, "status"))
		assertValidTimestamp(t, jsonField(got, "effective_at"), "effective_at")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		for _, field := range []string{
			"scope_type", "scope_ref_id", "period_starts_at", "period_ends_at", "adjustment", "value",
		} {
			body := requiredBody()
			delete(body, field)

			status, respBody, err := apiClient.Post(demandOverridesPath, body, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"missing %s should return 400 or 422, got %d: %s", field, status, string(respBody))
		}
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		created := createOverride(t, map[string]any{
			"reason": "new_customer",
			"note":   "preserved note",
		})
		id := jsonField(created, "id")
		origCreatedAt := jsonField(created, "created_at")

		status, body, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{
			"value": 9000,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		assert.EqualValues(t, 9000, got["value"])
		assert.Equal(t, "preserved note", jsonField(got, "note"), "note should be preserved")
		assert.Equal(t, "new_customer", jsonField(got, "reason"), "reason should be preserved")
		assert.Equal(t, "item", jsonField(got, "scope_type"), "scope_type should be preserved")
		assert.Equal(t, jsonField(created, "period_starts_at"), jsonField(got, "period_starts_at"))
		assert.Equal(t, jsonField(created, "period_ends_at"), jsonField(got, "period_ends_at"))
		assert.Equal(t, "delta_units", jsonField(got, "adjustment"), "adjustment should be preserved")
		assert.Equal(t, "active", jsonField(got, "status"), "status should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

func TestDemandOverrides_CreateResponseShape(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	id := jsonField(created, "id")

	assertIDFormat(t, id, "deov")
	assertObjectField(t, created, "demand_override")
	assertValidTimestamp(t, jsonField(created, "effective_at"), "effective_at")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
}

func TestDemandOverrides_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	created := createOverride(t, map[string]any{"unit_id": SeedUnitID})
	id := jsonField(created, "id")

	status, body, err := apiClient.GetListRaw(demandOverridesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "scope")
	assertNilField(t, got, "unit")
	assertNilField(t, got, "created_by")
}

func TestDemandOverrides_CreateIdempotent(t *testing.T) {
	t.Parallel()

	start, end := overridePeriod()
	idemKey := newIdempotencyKey()
	body := map[string]any{
		"scope_type":       "item",
		"scope_ref_id":     SeedItemID,
		"period_starts_at": start,
		"period_ends_at":   end,
		"adjustment":       "delta_units",
		"value":            1500,
	}

	status1, body1, err := apiClient.Post(demandOverridesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	t.Cleanup(func() { deleteOverride(t, id1) })

	status2, body2, err := apiClient.Post(demandOverridesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"),
		"a replayed create must return the override the first call made")
}

func TestDemandOverrides_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	created := createOverride(t, nil)
	id := jsonField(created, "id")

	updKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{"value": 9000}, updKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(demandOverridesPath+"/"+id, map[string]any{"value": 9000}, updKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, jsonField(parseJSON(body1), "updated_at"), jsonField(parseJSON(body2), "updated_at"),
		"a replayed update must return the cached response rather than applying again")
}

// ──────────────────────────────────────────────
// List basics
// ──────────────────────────────────────────────

func TestDemandOverrides_List(t *testing.T) {
	t.Parallel()

	createOverride(t, nil)

	list, _, err := apiClient.GetList(demandOverridesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "the created override must be listable")
}

func TestDemandOverrides_ListPagination(t *testing.T) {
	t.Parallel()

	// Both rows share a marker note, so the walk is scoped to rows this test owns.
	marker := uniqueName("e2e-override-page")
	first := createOverride(t, map[string]any{"note": marker})
	second := createOverride(t, map[string]any{"note": marker})

	assertScopedCursorPagination(t, demandOverridesPath, url.Values{"q": {marker}},
		[]string{jsonField(first, "id"), jsonField(second, "id")})
}

func TestDemandOverrides_ListSearchNoResults(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(demandOverridesPath, url.Values{"q": {"zzzznotanoverride99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}
