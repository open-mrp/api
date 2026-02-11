# Config conventions for configurable utilities (Go)

This document defines how configurable utility components should expose, document, default, and validate configuration.

## Goals

- Make configuration **self-documenting** in code via comments.
- Ensure safe, predictable behavior via **production defaults**.
- Ensure misconfiguration fails fast via **validation**.
- Avoid drift between docs and behavior by keeping the “source of truth” next to the config type.

## Standard pattern

Each configurable component should have:

1. A `Config` struct that documents every field.
2. A `WithDefaults()` method that populates production defaults. `cmp.Or` can be useful here.
3. A `validate()` method that ensures all requirements are met.
4. A constructor that applies defaults then validates:
   - `cfg = cfg.WithDefaults()`
   - `if err := cfg.validate(); err != nil { ... }`

> Note: should you wish to populate defaults in a new config, it will be as simple as `cfg := new(config).withDefaults()`

### Required ordering

Defaults must be applied before validation:

- Defaults may populate fields needed for validation.
- Validation should validate the final, effective configuration.

## Config struct documentation requirements

The config struct is the primary documentation surface. Every field must have a doc comment that includes:

- Whether it is **required** or **optional**
- The **default value** if optional (including computed defaults)
- Any important semantics (units, bounds, meaning of zero, side effects)

Recommended format:

- `(required)` or `(optional; default: <value>)` at the start of the field comment.
- Keep the first sentence short and descriptive; add detail below if needed.

Example:

```go
type Config struct {
  // ServiceName (required) identifies which service owns this instance.
  ServiceName string

  // PollInterval (optional; default: 30s) controls how frequently we poll the service.
  PollInterval time.Duration
}
```

### Zero-value semantics

By default, `WithDefaults()` treats Go zero values as “unset” (e.g., `0`, `""`, `nil`).
If a field uses a different meaning (e.g., `0` means “disable”), it must be explicitly documented and implemented consistently.

## Defaults: `WithDefaults()`

### Rules

- `WithDefaults()` must fill all optional fields that have a default.
- Defaults should be **production-safe**.
- Defaults are implemented in code.
- `WithDefaults()` must be safe to call with `nil` receivers:
  - If `cfg == nil`, create a new config and populate defaults.

### Signature

```go
func (c *MyConfig) WithDefaults() *MyConfig
```

### Behavior

- Mutates and returns the config pointer.
- Only overwrites fields that are considered “unset” (per documented zero-value semantics).

## Validation: `validate()`

### Rules

- `validate()` must enforce all required fields and constraints:
  - Required fields present
  - Value bounds (e.g., `> 0`)
  - Cross-field invariants (e.g., “A requires B”, “X must be >= Y”)
- `validate()` must return clear, stable errors suitable for logs and debugging.
- Prefer component-scoped messages:
  - `return fmt.Errorf("<component>: <message>")`

### Validate tags (optional)

Struct tags may be used for simple field-local validation (e.g., `required`, `gt=0`) if consistent with the codebase’s validator choice, but:

- Tags do **not** replace the requirement for `validate()` as the authoritative entry point.
- Cross-field checks must be implemented in `validate()`.

## Constructor requirements

Every configurable component constructor must:

1. Apply defaults
2. Validate
3. Store the effective config (usually by value)

Example:

```go
func NewThing(cfg *ThingConfig, deps ...) (*Thing, error) {
  cfg = cfg.WithDefaults()
  if err := cfg.validate(); err != nil {
    return nil, err
  }
  return &Thing{config: *cfg, ...}, nil
}
```

## Examples and discoverability

For non-trivial components, add at least one example showing minimal configuration:

- `examples_test.go` with `ExampleNewThing()` (preferred for Go doc rendering), or
- package README snippet if examples need more narrative.

## Review checklist

- [ ] Every config field has a doc comment including required/optional + default.
- [ ] `WithDefaults()` exists, is nil-safe, and matches documented defaults.
- [ ] `validate()` exists and enforces requirements and constraints (including cross-field).
- [ ] Constructor calls `WithDefaults()` then `validate()` in that order.
- [ ] Error messages are component-scoped and actionable.
- [ ] Zero-value semantics are explicit for any field where it’s ambiguous.