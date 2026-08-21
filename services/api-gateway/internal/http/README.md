# API Endpoint Parameter Binding

This package provides a powerful parameter binding system for API endpoints that automatically extracts and validates request data from multiple sources.

## Overview

The handler converter supports binding request parameters from multiple sources using struct tags:

-   **Path parameters** from URL segments like `/users/{id}`
-   **Query parameters** from URL query strings like `?page=1&limit=10`
-   **Header parameters** from HTTP headers like `Authorization: Bearer <token>`
-   **Cookie parameters** from HTTP cookies with automatic fallback support
-   **JSON body parameters** from request body data
-   **Raw body bytes** for endpoints that must verify a signature over the exact payload

### Example

```go
type ComplicatedRequest struct {
    Token     string   `header:"Authorization" scheme:"Bearer"`   // Authorization: Bearer <token>
    SessionID string   `cookie:"session_id" validate:"required"`  // Only from cookie
    Name      string   `json:"name" validate:"required"`
    Email     string   `json:"email" validate:"required,email"`
    Age       int      `json:"age" validate:"min=18"`
    UserID    string   `path:"id"`                                // Binds to {id} in URL path
    Page      int      `query:"page" default:"1"`                 // ?page=1
    Limit     int      `query:"limit" default:"20"`               // ?limit=10
}
```

## Parameter Binding Examples

### 1. Path Parameters

Extract values from URL path segments:

**Route:** `/v1/users/{id}/posts/{postId}`

```go
type GetUserPostRequest struct {
    UserID  string `path:"id"`      // Binds to {id} in URL path
    PostID  int    `path:"postId"`  // Binds to {postId} in URL path
}
```

### 2. Query Parameters

Handle URL query strings with various data types:

**URL:** `/v1/users?page=1&limit=10&tags=go,api&active=true`

```go
type ListUsersRequest struct {
    Page    int      `query:"page" default:"1"`           // ?page=1
    Limit   int      `query:"limit" default:"20"`         // ?limit=10
    Tags    []string `query:"tags"`                       // ?tags=go,api (comma-separated or repeated)
    Active  bool     `query:"active"`                     // ?active=true
    Search  string   `query:"search"`                     // ?search=term
}
```

### 3. Header Parameters

Extract values from HTTP headers:

```go
type AuthRequest struct {
    Token     string `header:"Authorization" scheme:"Bearer"`  // Authorization: Bearer <token>
    UserAgent string `header:"User-Agent"`                    // User-Agent: <value>
    APIKey    string `header:"X-API-Key"`                     // X-API-Key: <value>
}
```

#### Flexible Authorization Schemes

For Authorization headers, you can support multiple schemes (Bearer or Basic) without specifying a scheme:

```go
type RefreshTokenRequest struct {
    // Supports both "Bearer <token>" and "Basic <base64(token:)>" formats
    RefreshToken string `header:"Authorization" validate:"required"`
}
```

When `scheme` is omitted, the binding system automatically validates and extracts tokens from either Bearer or Basic auth schemes.

### 4. Cookie Parameters

Extract values from HTTP cookies with automatic fallback support:

```go
type RefreshTokenRequest struct {
    // Tries Authorization header first, falls back to cookie if header is missing or invalid
    RefreshToken string `header:"Authorization" cookie:"__Secure-augno.refresh-token" validate:"required"`
}
```

**Cookie Binding Behavior:**

-   If both `header` and `cookie` tags are present, the header is checked first
-   If the header is missing or invalid, the system automatically falls back to the cookie
-   For Authorization headers, supports both Bearer and Basic schemes when no `scheme` is specified
-   Cookie fallback also works when Authorization header validation fails

**Example with cookie-only binding:**

```go
type CookieOnlyRequest struct {
    SessionID string `cookie:"session_id" validate:"required"`  // Only from cookie
}
```

### 5. JSON Body Parameters

Handle request body data with validation:

```go
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=18"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

### 6. Raw Body

Webhook endpoints need the exact bytes the sender signed, before any JSON decoding. Tag a
`[]byte` field with `rawbody`:

```go
type StripeWebhookRequest struct {
    Signature string `header:"Stripe-Signature" validate:"required"`
    Payload   []byte `rawbody:"true"`
}
```

The tag only works on `[]byte` fields, the body is read once and shared across every `rawbody`
field on the struct, and the read is capped at 1MB. The tag's value is not interpreted — any
non-empty value works, and `"true"` is the convention.

### 7. Combined Example

A realistic example showing all parameter types together:

**Route:** `/v1/users/{userId}/posts?page=1&limit=10`  
**Headers:** `Authorization: Bearer <token>`  
**Body:** `{"title": "My Post", "content": "..."}`

```go
type CreatePostRequest struct {
    // Path parameters
    UserID string `path:"userId" validate:"required"`

    // Query parameters
    Page   int `query:"page" default:"1"`
    Limit  int `query:"limit" default:"20"`

    // Header parameters
    Token string `header:"Authorization" scheme:"Bearer" validate:"required"`

    // JSON body parameters
    Title   string `json:"title" validate:"required,min=1,max=100"`
    Content string `json:"content" validate:"required"`
}
```

### 8. Time Parsing

Handle date/time parameters with custom layouts:

```go
type TimeRequest struct {
    CreatedAt time.Time `query:"created_at" time_layout:"2006-01-02"`  // ?created_at=2023-12-25
    UpdatedAt time.Time `query:"updated_at"`                           // Uses RFC3339 by default
}
```

### 9. Default Values

Provide fallback values when parameters are missing:

```go
type OptionalRequest struct {
    Page     int    `query:"page" default:"1"`           // Defaults to 1 if not provided
    Sort     string `query:"sort" default:"created_at"`  // Defaults to "created_at"
    Include  bool   `query:"include" default:"true"`     // Defaults to true
}
```

## Binding Precedence

Parameters are bound in the following order (later sources override earlier ones):

1. **Headers** - HTTP header values
2. **Cookies** - HTTP cookie values (used as fallback when header is missing or invalid)
3. **Path parameters** - URL path segments
4. **Query parameters** - URL query string values
5. **JSON body** - Request body data (for POST/PUT/PATCH requests)

**Note:** When both `header` and `cookie` tags are present on the same field, the header takes precedence. The cookie is only used if the header is missing or fails validation.

## Validation

All fields are automatically validated after binding:

-   Use `validate:"required,email,min=5"` tags for validation rules
-   Validation errors are automatically converted to user-friendly API error responses
-   The validation system supports all standard validator tags

## Supported Data Types

The binding system supports automatic conversion for:

-   `string` - Direct string values
-   `int`, `int8`, `int16`, `int32`, `int64` - Integer values
-   `uint`, `uint8`, `uint16`, `uint32`, `uint64` - Unsigned integer values
-   `float32`, `float64` - Floating point values
-   `bool` - Boolean values (true/false, 1/0, etc.)
-   `time.Time` - Date/time values with custom layouts
-   `[]string` - String arrays (for query parameters)
-   `map[string]interface{}` - JSON objects

## Struct Tags Reference

| Tag                    | Description                            | Example                     |
| ---------------------- | -------------------------------------- | --------------------------- |
| `path:"name"`          | Binds to URL path parameter `{name}`   | `path:"userId"`             |
| `query:"name"`         | Binds to query parameter `?name=value` | `query:"page"`              |
| `header:"Name"`        | Binds to HTTP header `Name: value`     | `header:"Authorization"`    |
| `cookie:"name"`        | Binds to HTTP cookie `name`            | `cookie:"session_id"`       |
| `scheme:"Bearer"`      | Strips scheme prefix from header value | `scheme:"Bearer"`           |
| `json:"name"`          | Binds to JSON field in request body    | `json:"title"`              |
| `rawbody:"true"`       | Binds the raw request body to `[]byte` | `rawbody:"true"`            |
| `default:"value"`      | Fallback value if parameter is missing | `default:"1"`               |
| `time_layout:"layout"` | Custom time parsing layout             | `time_layout:"2006-01-02"`  |
| `validate:"rules"`     | Validation rules for the field         | `validate:"required,email"` |

**Cookie Fallback:** When both `header` and `cookie` tags are present, the header is checked first. If the header is missing or invalid, the system automatically falls back to the cookie value.

## Error Handling

When validation fails, the system automatically:

1. Converts validation errors to user-friendly messages
2. Returns appropriate HTTP status codes (400 for validation errors)
3. Formats errors as JSON responses using the `apierror.APIErrorResponse` envelope

Unknown fields are rejected rather than ignored: an unrecognized JSON body field or query
parameter produces a validation error naming the offending field.

Example error response:

```json
{
    "error": {
        "code": "validation_failed",
        "type": "invalid_request_error",
        "message": "Title must be at least 1 characters long",
        "param": "title",
        "doc_url": "https://docs.augno.com/api/errors#validation_failed",
        "is_transient": false,
        "quota": null,
        "request_log_url": "https://augno.com/dashboard/request-logs/rq_fbv1ygmybo3eauykr74"
    }
}
```

Every field is always present — `param`, `doc_url`, `quota`, and `request_log_url` are `null`
when they do not apply. The shape is defined by `apierror.ResponseError` in `shared/errors`.
