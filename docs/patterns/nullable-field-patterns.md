# Nullable Field Patterns for PATCH Endpoints

This document describes how PATCH endpoints handle nullable fields — fields that users can explicitly set to `null` to clear their value.

## The Problem

Go's `encoding/json` decoder treats both "field absent from JSON" and "field explicitly set to null" identically: both leave a `*string` pointer as `nil`. PATCH semantics require distinguishing these cases:

- **Absent** — don't change the field
- **Explicit null** — clear the field (set to NULL in the database)
- **Value provided** — update the field

## The `nullable` Struct Tag

The API gateway uses a `nullable` struct tag on request struct fields to control null handling:

| Tag | Meaning | Behavior |
|-----|---------|----------|
| `nullable:"true"` | Field can be cleared by sending `null` | Explicit null sets pointer to `ptr("")` (empty string sentinel) |
| `nullable:"false"` | Field must not be `null` | Explicit null returns a validation error |
| _(no tag)_ | Default PATCH behavior | Explicit null treated same as absent (no update) |

### Example

```go
type UpdateCustomerRequest struct {
    // Can be cleared by sending null.
    DefaultCarrierID *string `json:"default_carrier_id,omitempty" nullable:"true"`

    // Cannot be null — validation error if null is sent.
    CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`

    // Standard optional — null and absent both mean "don't update".
    Name *string `json:"name,omitempty"`
}
```

## Three-State Model for `nullable:"true"` Fields

After the handler pipeline processes the request, a `*string` field tagged `nullable:"true"` has three possible states:

| Pointer value | JSON input | Meaning |
|---------------|------------|---------|
| `nil` | Field absent | Don't update this field |
| `ptr("")` | `"field": null` | Clear this field (set to NULL) |
| `ptr("value")` | `"field": "value"` | Update to this value |

## How Each Layer Handles It

### 1. API Gateway Handler (`api_endpoint.go`)

After JSON decoding, `validate.ApplyExplicitNulls(body, req)` scans the raw JSON for explicit nulls on `nullable:"true"` fields and sets their pointer to `ptr("")`. This runs before `RejectExplicitJSONNulls` (which handles `nullable:"false"`).

### 2. Gateway Service → Proto

No special handling needed. Proto `optional string` distinguishes `nil` from `""` naturally.

### 3. gRPC Handler → Domain Params

No special handling needed. The `*string` values pass through as-is.

### 4. Service Layer

The service must handle all three states:

```go
// Example: resolving a sales rep user ID
if params.DefaultSalesRepUserID != nil && *params.DefaultSalesRepUserID != "" {
    // Value provided — resolve user_id to account_user_id
    accountUser, apiErr := repo.FindByAccountAndUserID(ctx, *params.DefaultSalesRepUserID, ownerAccountID)
    // ...
} else if params.DefaultSalesRepUserID == nil {
    // Not provided — keep existing value
    params.DefaultSalesRepUserID = old.DefaultSalesRepID
}
// ptr("") falls through — means "clear", repository maps "" to SQL NULL
```

For simple ID fields without resolution logic:

```go
if params.DefaultCarrierID == nil {
    params.DefaultCarrierID = old.DefaultCarrierID // keep existing
}
// ptr("") and ptr("value") both pass through to repository
```

### 5. Repository Layer

The `stringToNullString()` helper already handles this correctly:

```go
func stringToNullString(s *string) gosql.NullString {
    if s == nil || *s == "" {
        return gosql.NullString{} // SQL NULL
    }
    return gosql.NullString{String: *s, Valid: true}
}
```

### 6. SQL Layer

Nullable fields use **direct assignment** (not COALESCE):

```sql
-- Clearable field: direct assignment. Service must always provide a value.
default_carrier_id = sqlc.narg('default_carrier_id'),

-- Non-clearable field: COALESCE preserves existing value when param is NULL.
alias = COALESCE(sqlc.narg('alias'), alias),
```

The key difference: with COALESCE, a NULL parameter means "keep existing value" (the SQL handles it). Without COALESCE, a NULL parameter sets the column to NULL (the service must backfill the existing value for unchanged fields).

## When to Use Each Tag

- **`nullable:"true"`** — Use on optional foreign-key reference fields (`*ID` fields) and other fields that the user should be able to clear. These are typically IDs pointing to related entities (carriers, payment terms, addresses, sales reps, etc.).

- **`nullable:"false"`** — Use on fields where `null` is never valid input, such as enum fields that must always have a value once set (commission policy, freight policy).

- **No tag** — Use on fields like `name`, `email`, `note` where sending `null` has no meaningful use case distinct from simply omitting the field.

## Adding a New Nullable Field

1. Add `nullable:"true"` to the field in the endpoint request struct
2. In the SQL update query, use `sqlc.narg('field')` (no COALESCE wrapper)
3. In the service layer, backfill unchanged fields from the existing record:
   ```go
   if params.FieldID == nil {
       params.FieldID = old.FieldID
   }
   ```
4. Run `make gen-sqlc <service>` to regenerate
