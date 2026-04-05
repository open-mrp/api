# Constants Enum Patterns

This document describes the enum convention used across `shared/constants/`.

## Type Definition

Every enum is a type alias on `string`:

```go
// AccountMode is the intended mode of operation for a request.
type AccountMode string
```

## Named Constants

Constants use type-prefixed names with doc comments:

```go
const (
    // AccountModeProduction indicates that the request is targeting production resources.
    AccountModeProduction AccountMode = "prod"
    // AccountModeSandbox indicates that the request is targeting sandbox resources.
    AccountModeSandbox AccountMode = "test"
)
```

## Required Methods

Every enum type must implement two methods:

### `IsValid() bool`

Validates that a value is one of the defined constants using a switch statement:

```go
func (m AccountMode) IsValid() bool {
    switch m {
    case AccountModeProduction, AccountModeSandbox:
        return true
    default:
        return false
    }
}
```

### `EnumValues() []string`

Returns all valid values as strings. The order must match the constant declaration order:

```go
func (m AccountMode) EnumValues() []string {
    return []string{string(AccountModeProduction), string(AccountModeSandbox)}
}
```

## Optional Methods

Some enums have additional domain-specific methods:

- `String() string` — explicit string conversion
- `Normalize() Type` — parse/normalize a raw string into the enum (e.g., `Protocol`)
- `Ordinal() int` / `IsAfter(other) bool` — ordering (e.g., `RegistrationStep`)
- `IsX() bool` — convenience boolean checks (e.g., `PlatformMode.IsProduction()`)

## Comment Conventions

When documenting fields that use a constant type, do **not** include example values (e.g. `(e.g. "foo", "bar")`) in the doc comment. The valid values are already defined by the constant type and will be automatically populated in the generated API documentation. Inline examples create maintenance burden and risk going stale.

```go
// Good — the type communicates the valid values.
// The unit dimension.
Type constants.UnitType `json:"type" validate:"required"`

// Bad — redundant examples that can go stale.
// The unit dimension (e.g. "mass", "quantity").
Type constants.UnitType `json:"type" validate:"required"`
```

Fields that are plain `string` (not backed by a constant type) may still use e.g. examples in their comments to aid readability.

## Cross-Service vs Service-Local

| Location | Use case |
|----------|----------|
| `shared/constants/` | Values shared across multiple services or used in gRPC contracts |
| `services/{name}/internal/domain/constants.go` | Values only meaningful within one service |

## Adherence Tests

Pattern-adherence tests in `shared/constants/constants_adherence_test.go` automatically verify that all enum types implement `IsValid()` and `EnumValues()`, and that the two methods are consistent with each other.

## Reference Files

All enum files are in `shared/constants/`. Non-enum files in this package (e.g., `registration_limit.go`) contain helper structs and are excluded from adherence testing.
