---
name: constants-enums
description: >-
  shared/constants enum convention: string type alias, named constants, IsValid,
  EnumValues, and using the constant type on request fields. Use when adding or
  changing an enum, a request/response field with a fixed value set, or anything in
  shared/constants.
---

# Constants enums

Every enum in `shared/constants/` is a `string` type alias with named constants, `IsValid()`, and `EnumValues()`. Human spec: `docs/patterns/constants-enum-patterns.md`. Adherence: `shared/constants/constants_adherence_test.go`.

```go
type AccountMode string

const (
    AccountModeProduction AccountMode = "prod"
    AccountModeSandbox    AccountMode = "test"
)

func (m AccountMode) IsValid() bool {
    switch m {
    case AccountModeProduction, AccountModeSandbox:
        return true
    default:
        return false
    }
}

func (m AccountMode) EnumValues() []string {
    return []string{string(AccountModeProduction), string(AccountModeSandbox)} // declaration order
}
```

Optional extras when the domain needs them: `String()`, `Normalize()`, `Ordinal()` / `IsAfter()`, `IsX()`.

## On request/response fields

A fixed value set is `constants.X`, never `string` + `max=255`. The gateway's `ValidateEnumFields` plus the OpenAPI generator produce validation, the spec enum list, and an SDK union type.

```go
DiscountType constants.OrderDiscountType `json:"discount_type" validate:"required"`
// PATCH: field.Optional[constants.OrderDiscountType] — no value validator needed; ValidateEnumFields unwraps.
```

`validate:"required"` still belongs (rejects omit). A `max=` length cap does not. Convert to string at the proto boundary (`string(req.DiscountType)` / `req.DiscountType.Ptr().StringPtr()`), not on the request struct.

Do **not** list example values in the field doc comment — the constant type already documents them and examples go stale.

## Where it lives

| Location | Use |
|---|---|
| `shared/constants/` | Shared across services or used in gRPC |
| `services/{name}/internal/domain/constants.go` | Meaningful in one service only |
