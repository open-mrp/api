# `apiresource`

This package defines **canonical, shared API resource types** used in public API responses.

These types:

-   Represent what is returned to clients (e.g. `Address`, `Customer`)
-   Contain `json` tags and flatten/compose internal data for client consumption
-   Are version-stable and reused across endpoints
-   Follow strict conventions for nullable vs optional fields

Use these in handlers as response types  
Do not use these internally in services or repositories

## Tag Reference

| Purpose                           | Tag                                             | Nullable | Optional |
| --------------------------------- | ----------------------------------------------- | -------- | -------- |
| Required string                   | Field string `json:"field" validate:"required"` | No       | No       |
| Required, but nullable            | Field \*string `json:"field"`                   | Yes      | No       |
| Optional (string if present)      | Field \*string `json:"field,omitempty"`         | No       | Yes      |
| Optional, but nullable if present | Field Optional[string] `json:"field,omitempty"` | Yes      | Yes      |
