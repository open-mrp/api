---
name: e2e-tests
description: >-
  How to write API e2e tests: required CRUD categories, assert-every-field,
  omitted-field defaults, expandable null-without-include, idempotency, and the
  no-skip-5xx rule. Use when writing or fixing tests in tests/e2e/api/.
---

# E2E tests

CRUD files live in `tests/e2e/api/` with `//go:build e2e`. Human spec (full templates): `docs/patterns/e2e-test-patterns.md`.

E2e exists to expose production bugs. A 5xx is a backend bug — fix the service/repo/gRPC layer. Never `t.Skip` 5xx, guard `status >= 500`, or paper over microservice bugs at the edge. Prefer root-cause fixes over in-memory filter/sort, retries, or gateway-only cosmetics.

## Required categories (priority order)

1. `TestResource_CRUD` — create → get → update → delete → 404
2. `TestResource_CreateAndUpdateAllFields` — every response-struct field asserted; then patch and assert updated + preserved
3. `TestResource_OmittedFields` — create defaults, missing-required 400/422, PATCH one field preserves the rest
4. `TestResource_CreateResponseShape` — id prefix, `object`, timestamps
5. List / pagination / search / search-no-results
6. Expandable: `null` without `?include`; populated with `?include`
7. Idempotency: same key on POST (and PATCH) returns the same id
8. Validation: empty/missing required fields

Look up the resource in `services/api-gateway/pkg/resource/` and assert **every** `json` field in `CreateAndUpdateAllFields`.

## Assertions

| Field | Assert |
|---|---|
| required string | `jsonField(got, "field")` |
| optional unset / expandable without include | `assertNilField` |
| bool | `assert.Equal(t, "false", jsonField(...))` — `jsonField` stringifies bools |
| timestamp | `assertValidTimestamp` |
| expandable with include | `jsonObject` → `id` + `object` |

Helpers: `uniqueName`, `newIdempotencyKey`, `parseJSON`, `jsonField`, `jsonObject`, `requireStatus`, `assertIDFormat`, `assertObjectField`, `assertEmptyListData`. Every `Post`/`Patch` takes an idempotency key.

## Conventions

- `t.Parallel()` on every top-level test. Subtests sharing a resource ID do **not**.
- `defer apiClient.Delete(...)` unless the test is the delete path.
- Seed IDs in `seed.go` for GET/list and FKs. Never mutate seeds in a way that breaks parallel tests.
- File sections: `// --- List ---`, `// --- CRUD ---`, `// --- Expandable Fields ---`, …

```bash
make e2e                  # up → test → down
make test-e2e             # against an already-running stack
go test -tags=e2e -run TestMyResource_OmittedFields ./tests/e2e/api/ -timeout 120s
```
