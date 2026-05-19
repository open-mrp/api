# Nullable and PATCH Field Patterns

PATCH endpoints need to distinguish three states for updatable fields:

| State | JSON | Meaning |
|-------|------|---------|
| Unset | key absent | Leave the existing value unchanged |
| Clear | `"field": null` | Set the column to NULL (where supported) |
| Set | `"field": "value"` | Update to the new value |

## Request struct types

### `*patch.Field[T]` — clearable optional fields

Use for any PATCH field that accepts explicit `null` to clear. The pointer means the key may be absent (unset); the inner field encodes clear vs set when present:

```go
type UpdateAccountGroupRequest struct {
	Description *patch.Field[string] `json:"description,omitempty,omitzero"`
}
```

- Use `json:"...,omitempty"` (and `omitzero` for value-type `*patch.Field`) on the pointer so absent keys stay unset and examples omit unset fields.
- `validate:"omitempty,..."` on the pointer is still correct (nil skips validation).
- OpenAPI: inner type (`string`, `QuantityInput`, `array`, …), `nullable: true`, `x-nullable-clear: true`; PATCH request schemas must not list the field in `required` (all body fields are optional).
- Runtime: absent key → nil pointer; `ApplyPtrFieldNulls` maps explicit JSON `null` to inner clear before service logic runs.
- Proto: map with `StringPatch`, `QuantityPatch`, `StringListPatch`, etc. via `patch.StringFieldPtrToProto`, etc.

### `patch.Nullable[T]` — optional input documented as nullable

Use on create (and other non-clearable) request fields that should appear as `nullable: true` in OpenAPI but must not accept explicit JSON `null` at runtime.

Always use the **value type** with `json:"...,omitzero"`. Do not use `*patch.Nullable[T]` (null would decode as a nil pointer and skip `UnmarshalJSON`, so explicit `null` would not be rejected).

```go
type CreateMaterialRequest struct {
	Description patch.Nullable[string] `json:"description,omitzero"`
	Phone       patch.Nullable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
}
```

- Unset when the key is absent (`omitzero` + `IsZero()`); set when a value is provided. Explicit JSON `null` is rejected at unmarshal (`patch.ErrExplicitNull`).
- Add `validate:"omitempty,..."` only when other validators apply to the inner value; do not use `validate:"omitempty"` alone (unset is already handled by the custom validator).
- OpenAPI: inner type, `nullable: true`, no `x-nullable-clear`.
- Service layer: use `field.Ptr()` to obtain `*T` for proto mapping when set, or `field.Value()` when you need the value and presence.
- Samples: `patch.SetNullable(v)` or `patch.PtrNullable(&v)`; never `&patch.Nullable[T]`.

### `*T` with `json:"...,omitempty"` — optional, not clearable

Use for PATCH fields that may be omitted but must not be sent as `null`:

```go
type UpdateAccountGroupRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
}
```

- OpenAPI: not nullable
- Runtime: `RejectExplicitJSONNulls` rejects explicit `null` and blank strings for `*string`

### `*T` without `omitempty` — response nullable fields

Use on **response** resources for fields that are always present but may be `null`:

```go
type AccountGroup struct {
	Description *string `json:"description"`
}
```

- OpenAPI: `nullable: true` (no `x-nullable-clear`)
- Never use `omitempty` on response resource fields

## Layer mapping

| Layer | Clearable field | Nullable input | Optional non-clearable |
|-------|-----------------|----------------|------------------------|
| HTTP request | `*patch.Field[T]` + `omitempty` | `patch.Nullable[T]` (value) + `omitzero` | `*T` + `omitempty` |
| OpenAPI | inner type, nullable, `x-nullable-clear` | inner type, nullable | inner type, not nullable |
| Proto | `StringPatch` / `QuantityPatch` / … | `optional` scalar via `.Ptr()` | `optional` scalar |
| Domain | `patch.Field[T]` | `*T` after backfill | `*T` after backfill |
| SQL | `patch.StringToNullString` etc. | `COALESCE` / backfill | `COALESCE` / backfill |

## Gateway handler pipeline

After JSON decode on PATCH/POST bodies:

1. `ApplyPtrFieldNulls` — maps explicit JSON `null` on `*patch.Field` keys to inner clear (encoding/json leaves them nil)
2. `ApplySlicePresenceFlags` — legacy `Has*` slice companions only
3. `RejectExplicitJSONNulls` — rejects `null` on optional pointers (`omitempty`); skips `*patch.Field` and `patch.Nullable`

## Adding a new clearable field

1. Use `*patch.Field[T]` on the gateway update request struct with `json:"...,omitempty"`
2. Map to proto patch types in the gateway service (`patch.StringFieldPtrToProto`, etc.)
3. Use `patch.Field` in core-service domain params and service logic
4. Regenerate OpenAPI: `make openapi`

## Adding a new nullable input field

1. Use `patch.Nullable[T]` with `json:"...,omitzero"` on the gateway request struct
2. Map with `.Ptr()` in the gateway service when calling proto
3. Regenerate OpenAPI: `make openapi`
