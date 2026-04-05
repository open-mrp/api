# Entity ID Patterns

This document describes the ID generation system used across all services, implemented in `shared/id/`.

## Format

All entity IDs follow the format:

```
{prefix}_{nanoID}
```

Examples:

- `us_abc123xyz456` (user)
- `apke_9f2x7m3k1p4q` (API key)
- `acgp_a1b2c3d4e5f6` (account group)

## Vocabulary Codes

Each prefix is composed of 2-letter vocabulary codes defined in `id_prefix_values.go`. Each code represents a semantic word:


| Code | Word         | Code | Word        | Code | Word         |
| ---- | ------------ | ---- | ----------- | ---- | ------------ |
| `ac` | Account      | `ad` | Address     | `ag` | Agent        |
| `ap` | API          | `at` | Attribute   | `bl` | Billing      |
| `cr` | Carrier      | `cu` | Customer    | `dp` | Department   |
| `ds` | Discount     | `em` | Email       | `ev` | Event        |
| `gp` | Group        | `ig` | Integration | `in` | Inventory    |
| `it` | Item         | `iv` | Invoice     | `ke` | Key          |
| `lc` | Location     | `ln` | Line        | `md` | Method       |
| `og` | Organization | `or` | Order       | `pd` | Product      |
| `pk` | Pick         | `pm` | Permission  | `pn` | Production   |
| `pr` | Price        | `rl` | Role        | `sh` | Shipment     |
| `tk` | Token        | `tp` | Type        | `tx` | Transaction  |
| `un` | Unit         | `us` | User        | `ve` | Verification |


See `id_prefix_values.go` for the complete vocabulary list.

## Composable Prefixes

Prefixes are built by concatenating vocabulary codes using `composePrefix()`:

```go
// Single word: "us"
UserIDPrefix = composePrefix(VocUser)

// Two words: "apke"
APIKeyIDPrefix = composePrefix(VocAPI, VocKey)

// Three words: "pnfmit"
ProductionFormulaItemIDPrefix = composePrefix(VocProduction, VocFormula, VocItem)
```

This makes prefixes both machine-readable (each 2-char chunk is a known code) and human-readable (codes map to words).

## ID Lengths

Three lengths are available, selected based on the expected cardinality of the entity:


| Length                 | Nano ID chars | Collision threshold |
| ---------------------- | ------------- | ------------------- |
| `IDLength12` (default) | 12            | ~308M IDs           |
| `IDLength19`           | 19            | ~86T IDs            |
| `IDLength22`           | 22            | ~18,660T IDs        |


## Charset

IDs use lowercase alphanumeric only: `0123456789abcdefghijklmnopqrstuvwxyz`

## Generating an ID

```go
// Default length (12)
id, err := id.GenID(id.UserIDPrefix, nil)
// → "us_a7k3m9x2p1q4"

// Custom length
length := id.IDLength22
id, err := id.GenID(id.OrderIDPrefix, &length)
```

## Adding a New Entity

1. Pick vocabulary codes for the entity name (check existing codes in `id_prefix_values.go` to avoid conflicts)
2. Compose the prefix: `NewEntityIDPrefix = composePrefix(VocWord1, VocWord2)`
3. Add the constant to the appropriate section in `id_prefix_values.go`
4. Add an entry to the uniqueness test in `id_test.go` (`TestIDPrefixes_NoDuplicates`)
5. If you added a new vocabulary code, add it to `TestVocabulary_NoDuplicates` as well

## Reference Files

- `shared/id/id_prefix_values.go` — vocabulary codes and prefix constants
- `shared/id/gen_id.go` — `GenID()` function
- `shared/id/utils.go` — nano ID generation and charset
- `shared/id/id_test.go` — uniqueness and format tests

