# E2E Test Patterns

This document describes how to write thorough end-to-end tests for API resources. All CRUD test files live in `tests/e2e/api/` and use the `//go:build e2e` build tag.

## Infrastructure Overview

| File | Purpose |
|------|---------|
| `harness_test.go` | `TestMain` entry point — waits for gateway health, verifies auth, discovers endpoints from OpenAPI spec |
| `client.go` | HTTP client with auth headers, retry logic (429 / transient 5xx), list parsing |
| `seed.go` | Seeded entity IDs and path parameter mappings |
| `helpers_test.go` | Assertion helpers and test utilities |
| `spec.go` | OpenAPI spec parser for dynamic endpoint discovery |

## Test File Structure

Each CRUD resource gets one file named `crud_{resource}_test.go`. Organize tests with section comments:

```go
//go:build e2e

package api_test

import (
    "net/url"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

const myResourcePath = "/v1/domain/my-resources"

// --- List ---
// --- Get ---
// --- CRUD ---
// --- Create ---
// --- Update ---
// --- Delete ---
// --- Expandable Fields ---
// --- Omitted Fields ---
```

## Required Test Categories

Every resource MUST have these test categories. The categories are listed in order of priority — implement top to bottom.

### 1. Basic CRUD Lifecycle

A single test that exercises create → get → update → delete → verify deletion:

```go
func TestMyResource_CRUD(t *testing.T) {
    t.Parallel()
    name := uniqueName("e2e-myres")

    // CREATE
    createStatus, createBody, err := apiClient.Post(myResourcePath, map[string]any{
        "name": name,
        // ... all required fields
    }, newIdempotencyKey())
    require.NoError(t, err)
    requireStatus(t, 201, createStatus, createBody)

    created := parseJSON(createBody)
    id := jsonField(created, "id")
    assert.NotEmpty(t, id)
    assert.Equal(t, "my_resource", jsonField(created, "object"))
    assert.Equal(t, name, jsonField(created, "name"))

    // GET
    getStatus, getBody, err := apiClient.GetListRaw(myResourcePath+"/"+id, nil)
    require.NoError(t, err)
    requireStatus(t, 200, getStatus, getBody)
    got := parseJSON(getBody)
    assert.Equal(t, id, jsonField(got, "id"))
    assert.Equal(t, name, jsonField(got, "name"))

    // UPDATE
    newName := uniqueName("e2e-myres-upd")
    patchStatus, patchBody, err := apiClient.Patch(myResourcePath+"/"+id, map[string]any{
        "name": newName,
    }, newIdempotencyKey())
    require.NoError(t, err)
    requireStatus(t, 200, patchStatus, patchBody)
    updated := parseJSON(patchBody)
    assert.Equal(t, newName, jsonField(updated, "name"))

    // DELETE
    delStatus, delBody, err := apiClient.Delete(myResourcePath + "/" + id)
    require.NoError(t, err)
    requireStatus(t, 200, delStatus, delBody)

    // Verify deletion
    getStatus2, _, err := apiClient.GetListRaw(myResourcePath+"/"+id, nil)
    require.NoError(t, err)
    assert.Equal(t, 404, getStatus2)
}
```

### 2. Create and Update All Fields

Create with every settable field, assert every response field. Then update changeable fields and assert both updated and preserved fields. This is the most important test — it proves the API returns everything correctly.

**Every field in the response struct must be asserted.** Look at the resource struct in `services/api-gateway/pkg/resource/` to find the full field list.

```go
func TestMyResource_CreateAndUpdateAllFields(t *testing.T) {
    t.Parallel()

    // ── CREATE with all fields ──
    name := uniqueName("e2e-myres-allf")
    createStatus, createBody, err := apiClient.Post(myResourcePath+"?include=sub_resource", map[string]any{
        "name":          name,
        "notes":         "Create notes",
        "sub_resource_id": SeedSubResourceID,
    }, newIdempotencyKey())
    require.NoError(t, err)
    requireStatus(t, 201, createStatus, createBody)

    got := parseJSON(createBody)
    id := jsonField(got, "id")
    require.NotEmpty(t, id)
    defer apiClient.Delete(myResourcePath + "/" + id)

    // Assert EVERY field
    assert.Equal(t, "my_resource", jsonField(got, "object"))
    assert.Equal(t, name, jsonField(got, "name"))
    assert.Equal(t, "Create notes", jsonField(got, "notes"))
    assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
    assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

    // Assert expandable fields included via ?include
    sub := jsonObject(got, "sub_resource")
    require.NotNil(t, sub, "sub_resource must be set after create")
    assert.Equal(t, SeedSubResourceID, jsonField(sub, "id"))
    assert.Equal(t, "sub_resource", jsonField(sub, "object"))

    // Assert expandable fields NOT included are nil
    assertNilField(t, got, "other_expandable")

    // ── UPDATE with different values ──
    updatedName := uniqueName("e2e-myres-allf-u")
    patchStatus, patchBody, err := apiClient.Patch(myResourcePath+"/"+id+"?include=sub_resource", map[string]any{
        "name":  updatedName,
        "notes": "Updated notes",
    }, newIdempotencyKey())
    require.NoError(t, err)
    requireStatus(t, 200, patchStatus, patchBody)

    updated := parseJSON(patchBody)
    // Assert updated fields
    assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
    assert.Equal(t, updatedName, jsonField(updated, "name"))
    assert.Equal(t, "Updated notes", jsonField(updated, "notes"))
    assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
    assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

    // Assert preserved fields
    updSub := jsonObject(updated, "sub_resource")
    require.NotNil(t, updSub, "sub_resource should be preserved")
    assert.Equal(t, SeedSubResourceID, jsonField(updSub, "id"))
}
```

### 3. Omitted Fields

A dedicated test that verifies:
- **Create defaults**: when optional fields are omitted, the response has correct defaults
- **Required field validation**: when required fields are missing, the API returns 400/422
- **Update preservation**: when a PATCH sends only one field, all other fields are unchanged

```go
func TestMyResource_OmittedFields(t *testing.T) {
    t.Parallel()

    t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
        name := uniqueName("e2e-myres-omit")
        status, body, err := apiClient.Post(myResourcePath, map[string]any{
            "name": name,
            // Only required fields — omit everything optional
        }, newIdempotencyKey())
        require.NoError(t, err)
        requireStatus(t, 201, status, body)

        got := parseJSON(body)
        id := jsonField(got, "id")
        require.NotEmpty(t, id)
        defer apiClient.Delete(myResourcePath + "/" + id)

        // Assert every field has its expected default
        assertObjectField(t, got, "my_resource")
        assert.Equal(t, name, jsonField(got, "name"))
        assertNilField(t, got, "notes")                  // optional string → null
        assert.Equal(t, "false", jsonField(got, "is_active")) // bool → false
        assertNilField(t, got, "owner")                   // expandable → null
        assertNilField(t, got, "sub_resource")            // expandable → null
        assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
        assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
    })

    t.Run("CreateMissingRequiredFields", func(t *testing.T) {
        // Test each required field individually
        status, body, err := apiClient.Post(myResourcePath, map[string]any{
            // missing "name"
        }, newIdempotencyKey())
        require.NoError(t, err)
        assert.True(t, status == 400 || status == 422,
            "Missing name should return 400 or 422, got %d: %s", status, string(body))
    })

    t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
        // 1. Create with ALL fields
        name := uniqueName("e2e-myres-pres")
        createStatus, createBody, err := apiClient.Post(myResourcePath, map[string]any{
            "name":  name,
            "notes": "Original notes",
            // ... all optional fields set
        }, newIdempotencyKey())
        require.NoError(t, err)
        requireStatus(t, 201, createStatus, createBody)

        created := parseJSON(createBody)
        id := jsonField(created, "id")
        require.NotEmpty(t, id)
        defer apiClient.Delete(myResourcePath + "/" + id)
        origCreatedAt := jsonField(created, "created_at")

        // 2. PATCH only ONE field
        newName := uniqueName("e2e-myres-pres-u")
        patchStatus, patchBody, err := apiClient.Patch(myResourcePath+"/"+id, map[string]any{
            "name": newName,
        }, newIdempotencyKey())
        require.NoError(t, err)
        requireStatus(t, 200, patchStatus, patchBody)

        got := parseJSON(patchBody)

        // 3. Assert updated field changed
        assert.Equal(t, newName, jsonField(got, "name"))

        // 4. Assert EVERY other field is unchanged
        assert.Equal(t, "Original notes", jsonField(got, "notes"), "notes should be preserved")
        assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
        assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
    })
}
```

### 4. Response Shape

Verify the create response shape when creating with minimal input. This focuses on field presence and types rather than values:

```go
func TestMyResource_CreateResponseShape(t *testing.T) {
    t.Parallel()
    name := uniqueName("e2e-myres-shape")
    status, body, err := apiClient.Post(myResourcePath, map[string]any{
        "name": name,
    }, newIdempotencyKey())
    require.NoError(t, err)
    requireStatus(t, 201, status, body)

    got := parseJSON(body)
    id := jsonField(got, "id")
    assert.NotEmpty(t, id)
    assertIDFormat(t, id, "myrs")      // Validate ID prefix
    assertObjectField(t, got, "my_resource")
    assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
    assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

    apiClient.Delete(myResourcePath + "/" + id)
}
```

### 5. List Tests

```go
func TestMyResource_List(t *testing.T) {
    t.Parallel()
    list, _, err := apiClient.GetList(myResourcePath, nil)
    require.NoError(t, err)
    assert.Equal(t, "list", list.Object)
    assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded resource")
}

func TestMyResource_ListPagination(t *testing.T) {
    t.Parallel()
    page1, _, err := apiClient.GetList(myResourcePath, url.Values{"limit": {"1"}})
    require.NoError(t, err)
    require.Len(t, page1.Data, 1)
    require.True(t, page1.PageInfo.HasNextPage)
    require.NotNil(t, page1.PageInfo.NextCursor)

    page2, _, err := apiClient.GetList(myResourcePath, url.Values{
        "limit":  {"1"},
        "cursor": {*page1.PageInfo.NextCursor},
    })
    require.NoError(t, err)
    require.Len(t, page2.Data, 1)

    id1 := DataItemField(page1.Data[0], "id")
    id2 := DataItemField(page2.Data[0], "id")
    assert.NotEqual(t, id1, id2, "pages should return different items")
}

func TestMyResource_ListSearch(t *testing.T) {
    t.Parallel()
    list, _, err := apiClient.GetList(myResourcePath, url.Values{"q": {"SeedName"}})
    require.NoError(t, err)
    assert.GreaterOrEqual(t, len(list.Data), 1)
}

func TestMyResource_ListSearchNoResults(t *testing.T) {
    t.Parallel()
    list, _, err := apiClient.GetList(myResourcePath, url.Values{"q": {"zzzznotaresource99999"}})
    require.NoError(t, err)
    assertEmptyListData(t, list.Data)
}
```

### 6. Expandable Fields

```go
func TestMyResource_ExpandableFieldsNullWithoutInclude(t *testing.T) {
    t.Parallel()
    status, body, err := apiClient.GetListRaw(myResourcePath+"/"+SeedMyResourceID, nil)
    require.NoError(t, err)
    requireStatus(t, 200, status, body)

    got := parseJSON(body)
    assertNilField(t, got, "owner")
    assertNilField(t, got, "sub_resource")
}

func TestMyResource_IncludeOwner(t *testing.T) {
    t.Parallel()
    status, body, err := apiClient.GetListRaw(myResourcePath+"/"+SeedMyResourceID, url.Values{"include": {"owner"}})
    require.NoError(t, err)
    requireStatus(t, 200, status, body)

    owner := jsonObject(parseJSON(body), "owner")
    require.NotNil(t, owner, "owner should be present with ?include=owner")
    assert.Equal(t, "owner", jsonField(owner, "object"))
}
```

### 7. Idempotency

```go
func TestMyResource_CreateIdempotent(t *testing.T) {
    t.Parallel()
    name := uniqueName("e2e-idem-myres")
    idemKey := newIdempotencyKey()

    status1, body1, err := apiClient.Post(myResourcePath, map[string]any{
        "name": name,
    }, idemKey)
    require.NoError(t, err)
    requireStatus(t, 201, status1, body1)
    id1 := jsonField(parseJSON(body1), "id")

    status2, body2, err := apiClient.Post(myResourcePath, map[string]any{
        "name": name,
    }, idemKey)
    require.NoError(t, err)
    requireStatus(t, 201, status2, body2)
    assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

    apiClient.Delete(myResourcePath + "/" + id1)
}
```

### 8. Validation

```go
func TestMyResource_CreateValidation_EmptyName(t *testing.T) {
    t.Parallel()
    status, body, err := apiClient.Post(myResourcePath, map[string]any{
        "name": "",
    }, newIdempotencyKey())
    require.NoError(t, err)
    assert.True(t, status == 400 || status == 422,
        "Empty name should return 400 or 422, got %d: %s", status, string(body))
}
```

## Field Assertion Rules

### Assert every field in the response struct

Look up the resource struct in `services/api-gateway/pkg/resource/{resource}_resource.go`. Every field with a `json` tag must be asserted in the `CreateAndUpdateAllFields` test.

| Field Type | How to Assert |
|-----------|---------------|
| `string` (required) | `assert.Equal(t, expected, jsonField(got, "field"))` |
| `*string` (optional, set) | `assert.Equal(t, expected, jsonField(got, "field"))` |
| `*string` (optional, not set) | `assertNilField(t, got, "field")` |
| `bool` | `assert.Equal(t, "true", jsonField(got, "field"))` (JSON bools come as strings via `jsonField`) |
| `time.Time` (required) | `assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")` |
| `*time.Time` (optional, set) | `assertValidTimestamp(t, jsonField(got, "field"), "field")` |
| `*time.Time` (optional, not set) | `assertNilField(t, got, "field")` |
| `*SubResource` (expandable, included) | `sub := jsonObject(got, "field"); require.NotNil(t, sub); assert.Equal(t, expectedID, jsonField(sub, "id"))` |
| `*SubResource` (expandable, not included) | `assertNilField(t, got, "field")` |
| `*List[T]` (expandable, not included) | `assertNilField(t, got, "field")` |

### Boolean field gotcha

`jsonField()` returns `"true"` or `"false"` as strings for boolean values. Always compare with string literals:

```go
assert.Equal(t, "false", jsonField(got, "is_active"))  // correct
assert.False(t, got["is_active"].(bool))                // wrong — type assertion will fail
```

### Expandable fields without `?include`

Expandable fields (tagged `expandable:"true"` in the resource struct) are always `null` unless explicitly requested via `?include=field_name`. Assert them as nil in tests that don't use `?include`:

```go
assertNilField(t, got, "owner")
assertNilField(t, got, "sub_resource")
```

### Expandable fields with `?include`

When using `?include`, assert the sub-resource is populated with at least `id` and `object`:

```go
sub := jsonObject(got, "sub_resource")
require.NotNil(t, sub, "sub_resource should be present with ?include=sub_resource")
assert.Equal(t, SeedSubResourceID, jsonField(sub, "id"))
assert.Equal(t, "sub_resource", jsonField(sub, "object"))
```

## Helper Functions Reference

| Function | Purpose |
|----------|---------|
| `uniqueName(prefix)` | Generate unique name with UUID suffix (e.g., `"e2e-myres-a1b2c3d4"`) |
| `newIdempotencyKey()` | Generate UUID v4 for POST/PATCH idempotency |
| `parseJSON(body)` | Unmarshal `[]byte` to `map[string]any` |
| `jsonField(m, key)` | Extract string value from map (bools become `"true"`/`"false"`) |
| `jsonObject(m, key)` | Extract nested `map[string]any` from map |
| `requireStatus(t, expected, actual, body)` | Fatal assert on HTTP status with body in error message |
| `requireErrorResponse(t, body, code, type)` | Validate error envelope and return error object |
| `assertErrorParam(t, errObj, param)` | Assert `error.param` field |
| `assertIDFormat(t, id, prefix)` | Assert ID starts with `prefix_` |
| `assertValidTimestamp(t, value, name)` | Assert RFC3339 timestamp |
| `assertObjectField(t, m, expected)` | Assert `object` field value |
| `assertNilField(t, m, field)` | Assert field is `null` |
| `assertEmptyListData(t, data)` | Assert list data is empty with readable failure |

## Conventions

### Test naming

- `TestResource_CRUD` — basic lifecycle
- `TestResource_CreateAndUpdateAllFields` — comprehensive field coverage
- `TestResource_OmittedFields` — default/preservation behavior (with subtests)
- `TestResource_CreateResponseShape` — field format validation
- `TestResource_List`, `TestResource_ListPagination`, `TestResource_ListSearch`
- `TestResource_ExpandableFieldsNullWithoutInclude`, `TestResource_IncludeFieldName`
- `TestResource_CreateIdempotent`, `TestResource_UpdateIdempotent`
- `TestResource_CreateValidation_MissingFieldName`, `TestResource_CreateValidation_EmptyFieldName`

### Parallelism

- Every top-level test must call `t.Parallel()`
- Subtests within `t.Run()` that share state (same resource ID) should NOT call `t.Parallel()`

### Cleanup

- Use `defer apiClient.Delete(path + "/" + id)` after creating resources in tests that don't explicitly test deletion
- For the basic CRUD test, deletion is part of the test flow

### Idempotency keys

- Every `apiClient.Post()` and `apiClient.Patch()` call needs an idempotency key
- Use `newIdempotencyKey()` for normal calls
- For idempotency tests, reuse the same key variable

### Seed data

- Seed entity IDs are in `seed.go`
- Use seeded entities for GET/list tests and as foreign key references in create payloads
- Never modify seeded entities in a way that breaks other parallel tests (or restore them after)

## Running Tests

```bash
make e2e              # Full stack: up → test → down
make test-e2e         # Run tests against already-running stack
make e2e-up           # Start the e2e Docker stack
make e2e-down         # Tear down the e2e Docker stack

# Run specific tests
go test -tags=e2e -run TestMyResource_OmittedFields ./tests/e2e/api/ -timeout 120s
```

## Checklist for New Resources

When adding e2e tests for a new resource:

1. [ ] Read the resource struct in `services/api-gateway/pkg/resource/` — note every field
2. [ ] Read the create/update request structs in `services/api-gateway/endpoints/` — note required vs optional fields
3. [ ] Add seed data IDs to `seed.go` if needed
4. [ ] Write `TestResource_CRUD` — basic lifecycle
5. [ ] Write `TestResource_CreateAndUpdateAllFields` — assert every response field
6. [ ] Write `TestResource_OmittedFields` — defaults, required validation, update preservation
7. [ ] Write `TestResource_CreateResponseShape` — ID format, timestamps
8. [ ] Write list tests — basic list, pagination, search, no results
9. [ ] Write expandable field tests — null without include, populated with include
10. [ ] Write idempotency tests — create and update
11. [ ] Write validation tests — missing/empty required fields
12. [ ] Verify every field in the response struct has at least one assertion across all tests
