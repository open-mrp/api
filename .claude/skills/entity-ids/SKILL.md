---
name: entity-ids
description: >-
  Entity ID format, vocabulary codes, composable prefixes, and GenID. Use when adding
  a new entity, ID prefix, Sample*ID, or generating a type ID via shared/id.
---

# Entity IDs

Format: `{prefix}_{nanoID}`. Implemented in `shared/id/`. Human spec: `docs/patterns/entity-id-patterns.md`.

Type IDs are the public `id` field. Internal DB ids never leave the service and are never stringified onto a response.

## Prefixes

2-letter vocabulary codes concatenated with `composePrefix()`:

```go
UserIDPrefix                      = composePrefix(VocUser)                              // us
APIKeyIDPrefix                    = composePrefix(VocAPI, VocKey)                       // apke
ProductionFormulaItemIDPrefix     = composePrefix(VocProduction, VocFormula, VocItem)   // pnfmit
```

Charset: lowercase alphanumeric. Lengths: `IDLength12` (default), `IDLength19`, `IDLength22` — pick by expected cardinality.

```go
id, err := id.GenID(id.UserIDPrefix, nil)
length := id.IDLength22
id, err := id.GenID(id.OrderIDPrefix, &length)
```

## Adding a prefix

1. Reuse existing codes in `shared/id/id_prefix_values.go`; add a vocabulary code only if needed.
2. `NewEntityIDPrefix = composePrefix(...)` in the right section of that file.
3. Uniqueness test in `id_test.go` (`TestIDPrefixes_NoDuplicates`). New vocab → `TestVocabulary_NoDuplicates` too.

Reference: `id_prefix_values.go`, `gen_id.go`, `utils.go`, `id_test.go`.
