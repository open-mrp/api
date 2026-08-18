# Request Field Tags, Nullability, and PATCH Patterns

This is the canonical reference for how we model **request** field presence on the API gateway: which Go type to use, which `json` tag to pair it with, what each combination means on the wire, and how it flows through validation, OpenAPI, proto, and the service layer.

For the **response** side (resource structs), see [`docs/api-resource-conventions.md`](../api-resource-conventions.md). The short version of the response rule lives at the bottom of this doc too, because the same `*T` spelling means something *different* in a response than in a request, and that difference trips people up.

> **Background reading.** The two engineering articles explain *why* this design exists, including how Go's `encoding/json` behaves on marshal and unmarshal: [Declarative API Endpoints](../article1.md) (request decoding) and [Source Code → API Reference](../article2.md) (how these types become the OpenAPI contract).

---

## The core problem: three wire states, two native Go states

Any JSON field in an incoming request body is in exactly one of three states:

| Wire state | Example body | What the caller means |
|------------|--------------|-----------------------|
| **Absent** | `{}` | "I'm not saying anything about this field." |
| **Null** | `{"note": null}` | "Set this to null / clear it." |
| **Value** | `{"note": "hi"}` | "Set this to this value." |

Plain Go types collapse two of these into one. Unmarshal `{}` or `{"note": null}` into a `*string` and you get `nil` **both times** — Go cannot tell *absent* from *null*. Unmarshal into a `string` and both give `""` — indistinguishable from a real empty string. That lost bit of information is the entire reason `field.Optional` and `field.Clearable` exist: they carry a custom `UnmarshalJSON` that records *which* of the three states actually arrived.

---

## The four contexts

Which states you need to distinguish depends on what the request *does*. That, not personal taste, picks the type.

### 1. Create / action requests (`CreateXRequest`, action POST bodies)

A create has no existing value to "leave unchanged," so *absent* and *null* mean the same thing ("not provided"). You only need **provided vs. not**, and an explicit `null` is a client mistake we reject.

```go
type CreateCustomerRequest struct {
	// Required: value type, validate:"required", NO omit tag.
	Name string `json:"name" validate:"required,max=255"`
	// Optional scalar/enum/struct: field.Optional[T] + ,omitzero.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Optional enum with a documented default.
	StatusCode field.Optional[constants.AccountStatusCode] `json:"status,omitzero" default:"normal"`
	// Optional nested input struct — still field.Optional, NOT *QuantityInput.
	CreditLimit field.Optional[apirequest.QuantityInput] `json:"credit_limit,omitzero"`
	// Optional slice: stay []T, just use ,omitzero (do NOT wrap slices).
	PriceGroupIDs []string `json:"customer_price_group_ids,omitzero"`
}
```

### 2. Update / PATCH requests (`UpdateXRequest`)

Here all three states are meaningful: omit = leave unchanged, `null` = clear the column, value = set it. Two sub-cases:

```go
type UpdateCustomerRequest struct {
	// Path param: value type, never wrapped.
	CustomerID string `path:"id" validate:"required"`
	// Settable but NOT clearable (column is non-nullable): field.Optional[T].
	// Omit to leave unchanged; an explicit null is rejected.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Settable AND clearable (nullable column): *field.Clearable[T].
	// Omit = leave; null = clear; value = set.
	Note *field.Clearable[string] `json:"note,omitzero"`
	// "Replace the whole collection when provided" — needs absent-vs-empty,
	// so wrap the slice: field.Optional[[]T].
	PriceGroupIDs field.Optional[[]string] `json:"customer_price_group_ids,omitzero"`
}
```

> **We do not use bare `*string` for PATCH inputs anymore.** A `*string` can't be told apart from `null` at decode time and silently treats `{"name": null}` as "leave unchanged," which is surprising. `field.Optional[T]` rejects the null explicitly. `*field.Clearable[T]` is the *only* request shape that accepts `null`, and it does so on purpose (to clear).

### 3. Responses (resource structs)

Output only. There is no "absent" — every field is serialized — so plain Go types are correct. **See [`api-resource-conventions.md`](../api-resource-conventions.md); never put `omitempty` on a response field.**

```go
type Customer struct {
	Name string  `json:"name" validate:"required"` // always present
	Note *string `json:"note"`                      // value or JSON null (NO omit tag)
}
```

### 4. Shared input fragments (`pkg/request/*`: `AddressInput`, `QuantityInput`, `RateInput`, …)

These are create-style building blocks embedded in other requests. They follow the **create** rules (`field.Optional` for optional, value + `required` for mandatory).

**Compose them; do not inline their parts.** A value that already has a shared input fragment MUST be taken as that fragment, not flattened into sibling scalar fields. A rate is `rate: RateInput`, never `rate_value` + `rate_numerator_unit_id` + `rate_denominator_unit_id`; an amount with a unit is `credit_limit: QuantityInput`, never `credit_limit_value` + `credit_limit_unit_id`.

```go
// Good — one field, one concept.
Rate apirequest.RateInput `json:"rate" validate:"required"`

// Bad — the same concept smeared across three fields.
RateValue             string `json:"rate_value" validate:"required"`
RateNumeratorUnitID   string `json:"rate_numerator_unit_id" validate:"required"`
RateDenominatorUnitID string `json:"rate_denominator_unit_id" validate:"required"`
```

Inlining costs real things: the OpenAPI schema loses the shared `$ref` (so SDKs emit three loose strings instead of one reusable type), the prefix has to be repeated in every doc comment, and nothing stops a caller sending a value without its units. A fragment is also the natural unit of partial update — on PATCH use `field.Optional[apirequest.RateInput]`, which makes "replace the whole rate" the only representable operation and removes the question of what a value without units means.

Reach for a fragment whenever the concept is *value + the unit that gives it meaning*. Where a decimal is genuinely unitless — a volume-discount tier `threshold`, which is compared in whatever unit the discount already declares, or a `discount_percentage`, which is a bare multiplier — a plain `string` with `format:"decimal"` is correct, and wrapping it in a `QuantityInput` would invent a unit the resource does not store.

Add a new fragment to `pkg/request/` when a value + unit pairing shows up in a second request; a one-off stays local.

---

## The decision table

| Context | Always present / required | Optional, not clearable | Clearable (accepts `null`) |
|---------|---------------------------|-------------------------|----------------------------|
| **Create / action** | `T` + `validate:"required"`, no tag | `field.Optional[T]` + `,omitzero` | — (creates don't clear) |
| **Update / PATCH** | `T` (path params only) | `field.Optional[T]` + `,omitzero` | `*field.Clearable[T]` + `,omitzero` |
| **Response** | `T` + `validate:"required"` | `*T` (nullable, **no** omit tag) | `*T` (nullable, **no** omit tag) |

Optional **slices**: `[]T` + `,omitzero` on create (absent vs. empty doesn't matter); `field.Optional[[]T]` on update when "provided replaces the collection" must be distinguishable from "omitted."

---

## `omitzero` vs. `omitempty` (and why the wrappers force it)

This is the rule that looks arbitrary until you know it:

| You wrote | Use this json tag | Why |
|-----------|-------------------|-----|
| `field.Optional[T]` (value) | `,omitzero` | The wrapper implements `IsZero()`. `omitzero` (Go 1.24+) calls it, so an unset value is dropped from generated examples. `omitempty` does **not** call `IsZero` on a struct and would emit a broken `{}`. |
| `*field.Clearable[T]` (pointer) | `,omitzero` | Nil pointer is the zero value, so `omitzero` drops it. (`omitempty` also works on pointers, but we standardize on `omitzero` everywhere in requests so there is one rule.) |
| `[]T` / `field.Optional[[]T]` | `,omitzero` | Same: one rule. |
| `*T` in a **response** | **no omit tag** | Responses must serialize `null`, not drop the key. |

**Rule of thumb: every request field gets `,omitzero`; response fields get no omit tag.** Omit tags only affect *marshaling* (the OpenAPI examples and responses) — the server ignores them when decoding — so getting this wrong produces malformed docs examples, not a runtime accept/reject change. The accept/reject behavior comes from the *type* (below), not the tag.

> `validate:"omitempty,..."` is a different `omitempty` — it's a `go-playground/validator` keyword meaning "skip the other rules when unset." Leave it exactly as-is; it has nothing to do with the json tag.

---

## What each type accepts at runtime

The gateway request pipeline (in `services/api-gateway/pkg/endpoint/api_endpoint.go`, `Execute`) runs, in order:

1. **`DecodeJSONInto`** → `encoding/json` unmarshal. Each wrapper's custom `UnmarshalJSON` fires here.
   - `field.Optional[T].UnmarshalJSON(null)` returns `field.ErrExplicitNull`; `Execute` catches it and `field.ExplicitNullField` turns it into a field-named `400 "Field 'x' cannot be null."`.
   - `field.Clearable[T].UnmarshalJSON(null)` records the **clear** state (no error).
2. **`field.ApplyPtrClearableNulls`** — `encoding/json` leaves a `*field.Clearable[T]` nil when the key is an explicit `null`, so this pass walks the raw body and restores the clear sentinel.
3. **`validate.ApplySlicePresenceFlags`** — legacy `Has*` companions for a few slice fields.
4. **`validate.RejectExplicitJSONNulls`** — the guard for everything that is *not* a wrapper:
   - **Bare `*T` + `omitempty`**: rejects explicit `null` **and** blank/whitespace strings.
   - **`field.Clearable[T]`**: skipped (it accepts `null` by design).
   - **`field.Optional[T]`**: `null` was already rejected at step 1; this pass additionally rejects a present-but-**blank** string, so `{"name": ""}` is a `400 "Field 'x' must not be blank."` rather than silently setting `""`. (This applies uniformly to create and update Optional fields.)

Net behavior, by type:

| Type | `{}` (absent) | `{"x": null}` | `{"x": ""}` | `{"x": "v"}` |
|------|---------------|---------------|-------------|--------------|
| `T` + `required` | `400` (required) | `400` | depends on `validate` | set |
| `field.Optional[T]` | unset (OK) | **`400` cannot be null** | **`400` must not be blank** | set |
| `*field.Clearable[T]` | unset (OK) | **clear** | set to `""` | set |

---

## Reading these in the service layer

```go
// field.Optional[T]
if v, ok := req.Name.Value(); ok { /* provided */ }
pbReq.Name = req.Name.Ptr()            // *T: non-nil only when set

// *field.Clearable[T]
c := field.Coalesce(req.Note)          // nil *Clearable -> Unset
pbReq.Note = field.StringClearablePtrToProto(req.Note) // -> proto StringPatch
val := c.StringPtrAfterBackfill(existing) // clear -> nil; set -> &v; unset -> existing
```

Key accessors: `Value() (T, bool)`, `Ptr() *T`, `IsSet()/IsUnset()` (both); `IsClear()/WasProvided()`, `BackfillUnset`, `Coalesce` (Clearable). Build samples with `field.Some(v)` / `field.SomePtr(&v)` (Optional) and `field.Set(v)` / `field.Ptr(...)` (Clearable) — never `&field.Optional[T]{}`.

---

## OpenAPI mapping (derived automatically by `tools/apidocs`)

| Request shape | `required` | `nullable` | extra |
|---------------|------------|------------|-------|
| `T` (no omit) | yes | no | |
| `field.Optional[T]` | **no** (always optional) | **no** | documents inner `T`; rejects explicit null |
| `*field.Clearable[T]` | no | **yes** | in a request body, nullable means "send null to clear" |
| `*T` + `omitempty` (legacy/none left) | no | no | |
| `*T` no omit (**response only**) | yes | yes | "value or null" |

You never set `required`/`nullable` by hand — the generator reads the type + tags. After changing any request struct, run `make openapi` and commit the regenerated spec.

---

## Adding a new field — quick recipes

**Optional create input:** `field.Optional[T]` + `json:"x,omitzero"` (+ `validate:"omitempty,..."` only if the inner value has rules). Read with `.Ptr()`/`.Value()`. `make openapi`.

**PATCH, settable but not clearable:** `field.Optional[T]` + `json:"x,omitzero"`. Read with `.Ptr()`/`.Value()`.

**PATCH, clearable (nullable column):** `*field.Clearable[T]` + `json:"x,omitzero"`. Map with the `field.*ClearablePtrToProto` helpers; in core-service use `field.Clearable`/backfill. `make openapi`.

**Value that carries a unit:** the shared fragment — `apirequest.RateInput` / `QuantityInput` / `AddressInput` — not its parts as sibling scalars. On PATCH, `field.Optional[apirequest.RateInput]` (replaced whole).

**Field with a fixed set of values:** the `constants.X` type, never `string`. (See `constants-enum-patterns.md`.)

**Response, nullable:** `*T`, **no** omit tag. (See `api-resource-conventions.md`.)

---

## Response-side rule (so the overload is explicit)

A bare `*T` means different things in the two directions, distinguished only by the omit tag and the struct's role:

- **Request** `*T` + `omitempty` → "optional input, null rejected" (legacy; prefer `field.Optional`).
- **Response** `*T`, no tag → "always present, may be `null`."

If you find yourself reaching for a bare pointer on a *request*, you almost certainly want `field.Optional[T]` (set-or-absent) or `*field.Clearable[T]` (set/clear/absent) instead.
