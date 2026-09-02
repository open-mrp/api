---
name: api-resources
description: >-
  API resource and request-field conventions: Object field, no omitempty on responses,
  sub-objects not bare IDs, List[T] for arrays, expandable includes (null unless requested,
  never fabricate), Sample* fixtures, and request presence types (field.Optional vs
  *field.Clearable, omitzero). Use when adding or changing an endpoint, resource struct,
  request body, include key, presenter, or OpenAPI sample.
---

# API resources and request fields

Human specs: `docs/patterns/api-resource-conventions.md`, `docs/patterns/nullable-field-patterns.md`. After any request-struct change, `make openapi` and commit the spec.

## Response resources

- Every resource has `Object constants.ObjectType` (`json:"object"`). Register the type in `shared/constants/object_type.go`.
- **Never `omitempty` on a response field.** Pointers serialize as `null`; value types always emit. Slices and `json.RawMessage` serialize as `null` when nil.
- JSON names are `snake_case`. Status enums are `status`, not `status_code` (exception: numeric HTTP `status_code`). Datetimes end in `_at`.
- Nested resources are sub-objects (`role: {id, object, ...}`), never `role_id`.
- Arrays of resources are `*List[T]`, never a raw slice. Embedded (non-paginated) lists: `NewList(items, PageInfo{})`. Export endpoints may return a file instead.
- Every resource has `Sample<Resource>` + `SchemaExample()`. Sample IDs are real `id.GenID` output (`go run ./tools/apidocs/gensampleids.go`), never hand-typed filler. Reuse `Sample*ID` constants for foreign keys.

## Expandable includes (non-negotiable)

`expandable:"true"` fields are **`null` unless the client passed `?include=<key>`**. When included, populate from a real `BatchGet<X>ByIDs` loader. **Never fabricate** stubs or default enum/status fields.

Mechanically: presenter leaves expandables `nil`, stashes the FK id in load meta (`resourcekit.GetLoadMeta(ctx).Set(...)`). Register a `SubField` with `Target`, `ExtractIDs`, `Populate`. The resolver only `Populate`s requested keys.

- Register fields in `pkg/endpoint/include_definitions.go`.
- Declare the allowed set on the endpoint via `IncludesFor`. Unlisted keys → `ParameterInvalidError`.
- Fetch expensive data only when `appctx.IsIncludeRequested(ctx, key)` (nil include context returns true — backward compat).

## Request field types

The type encodes presence. A bare `*T` cannot tell absent from `null`. **Never use a bare `*T` for an optional request field.**

| Context | Required / always present | Optional, not clearable | Clearable (accepts `null`) |
|---|---|---|---|
| Create / action | `T` + `validate:"required"`, no omit tag | `field.Optional[T]` + `,omitzero` | — |
| Update / PATCH | `T` (path params) | `field.Optional[T]` + `,omitzero` | `*field.Clearable[T]` + `,omitzero` |
| Response | `T` + `validate:"required"` | `*T` (nullable, **no** omit tag) | `*T` (nullable, **no** omit tag) |

- Every **request** field uses `,omitzero` (never `,omitempty`). `validate:"omitempty,..."` is a separate validator keyword and stays.
- Create slices: `[]T` + `,omitzero`. PATCH replace-the-collection: `field.Optional[[]T]`.
- Shared fragments (`QuantityInput`, `RateInput`, `AddressInput`) stay composed; do not flatten into sibling scalars. PATCH a fragment as a whole (`field.Optional[apirequest.RateInput]`).
- Fixed value sets use `constants.X`, never `string`. See the `constants-enums` skill.

Net accept/reject:

| Type | absent | `null` | `""` | value |
|---|---|---|---|---|
| `T` + required | 400 | 400 | depends | set |
| `field.Optional[T]` | unset | **400 cannot be null** | **400 must not be blank** | set |
| `*field.Clearable[T]` | unset | **clear** | set `""` | set |

Service-layer: `Value() (T, bool)`, `Ptr() *T`; Clearable via `field.Coalesce` / `*ClearablePtrToProto`. Samples: `field.Some(v)` / `field.Set(v)`, never `&field.Optional[T]{}`.
