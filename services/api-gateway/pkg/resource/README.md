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

| Purpose                | Tag                                             | Nullable |
| ---------------------- | ----------------------------------------------- | -------- |
| Required value         | Field string `json:"field" validate:"required"` | No       |
| Required, but nullable | Field \*string `json:"field"`                   | Yes      |

Fields are **always present** in the response — either with a value or `null`. Never use `omitempty`.
