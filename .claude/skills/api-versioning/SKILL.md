---
name: api-versioning
description: >-
  How OpenMRP versions the public API: what counts as a breaking change, gateway-edge
  transformers, ForcedIncludes, ObjectType coverage, and the shipping checklist. Use when
  changing an existing public request or response shape, adding a version transformer,
  touching OpenMRP-Version, versiontransforms/, shared/version, or version-compat e2e tests.
---

# API versioning

A pinned `OpenMRP-Version` never changes wire shape. The backend speaks **`Latest` only**; older versions exist as gateway-edge transforms in `services/api-gateway/internal/versiontransforms/`. Human spec: `docs/patterns/api-versioning-patterns.md`.

When in doubt, treat the change as breaking. An extra transformer is cheap; a silently broken consumer is not.

## Breaking vs not

**Breaking — new version + transformer:**

- Remove or rename a request or response field
- Change a field's type, format, nullability, or enum value set
- Move previously-unconditional data behind `?include=`
- Change a default, status code, error `type`/`param`, or pagination semantics
- Tighten validation so a previously-accepted body is now rejected
- Change the meaning of an existing field or parameter at the same shape

**Not breaking — ships on `Latest`:**

- New endpoints, new **response** fields, new **optional** request fields or query params
- New include keys, new values on a field documented as an open enum
- Bug fixes where the documented contract was always the fixed behavior

## Invariants

- No version conditionals below the gateway edge.
- Transformers bridge **adjacent** versions (`FromVersion` = new, `ToVersion` = previous). The registry chains them.
- Transformers **reshape real data and never fabricate**. If the old shape needs data the new shape gates, implement `ForcedIncludes` — do not invent defaults.
- Transformers are immutable once their `ToVersion` has shipped (bug fixes ok; evolving meaning is a new version).
- An endpoint without `ObjectType` is invisible to the transformer chain and leaks the new shape.
- OpenAPI/SDKs document `Latest` only.

## Shipping a shape change

1. Declare the version in `shared/version/version.go` (constant, prepend `Supported`, repoint `Latest`). Update `version_test.go`.
2. Change the backend to the new shape only.
3. Write transformer(s) in `versiontransforms/`, one file per resource, `init()`-registered. Recursive walk on `"object"`. List **every parent object type** that embeds the resource in `ObjectTypes()`. Identity `TransformRequest` if requests did not change.
4. Unit-test single, list, nested, and data-missing payloads, plus a registry end-to-end test.
5. Audit `ObjectType` on every endpoint that returns or accepts the changed resource (including action endpoints).
6. Bump `defaultAPIVersion` in e2e; rewrite latest-shape assertions.
7. Add `tests/e2e/api/version_compat_<resource>_test.go` pinned to the previous version.
8. `make openapi`. Changelog the new shape and the migration path.

## Forced includes

Old clients cannot ask for an include they have never heard of.

- Root key (no dot) → applied unconditionally (old version always had the data).
- Nested key (`defaults.sales_rep.user`) → applied **only when the client requested the parent path**.

## Removing a version

Remove it from `Supported` (gateway then 400s), delete its transformers and compat tests, drop `MinVersion` guards and ForcedIncludes that existed solely for it. Never serve `Latest` under an old version string.
