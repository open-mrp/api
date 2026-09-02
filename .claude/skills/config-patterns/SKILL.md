---
name: config-patterns
description: >-
  Config struct conventions for Go utilities: field docs with (required)/(optional;
  default), WithDefaults, validate, constructor order. Use when adding or changing a
  Config struct, WithDefaults, or a configurable component constructor.
---

# Config structs

Every configurable component: a documented `Config`, `WithDefaults()`, `validate()`, and a constructor that applies defaults **then** validates. Human spec: `docs/patterns/config-patterns.md`.

```go
cfg = cfg.WithDefaults()
if err := cfg.validate(); err != nil { return nil, err }
```

## Field comments

Lead with `(required)` or `(optional; default: <value>)`. Include units, bounds, zero-value meaning, and side effects of the default (especially nil deps that no-op or panic).

```go
// PollInterval (optional; default: 30s) controls how frequently we poll the service.
PollInterval time.Duration
```

`WithDefaults` treats Go zero values as unset (`0`, `""`, `nil`) unless a field documents otherwise (`0` means disable) and implements that consistently.

## `WithDefaults()`

- Nil-safe: `if cfg == nil { cfg = &MyConfig{} }`.
- Fills every optional field that has a default. Production-safe defaults, in code.
- Mutates and returns the pointer. Only overwrites unset fields.

## `validate()`

Authoritative entry point — struct tags do not replace it. Enforce required fields, bounds, and cross-field invariants. Errors are component-scoped: `fmt.Errorf("<component>: <message>")`.

## Constructor

Apply defaults, validate, store the effective config by value. Non-trivial components get an `ExampleNewThing` in `examples_test.go`.
