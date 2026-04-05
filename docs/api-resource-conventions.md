# API Resource Conventions

All HTTP API responses follow these conventions for consistency and developer experience.

## Object Field

Every API resource MUST have an `Object` field that identifies the resource type:

```go
type MyResource struct {
    ID     string              `json:"id" validate:"required"`
    Object constants.ObjectType `json:"object" validate:"required,enum=my_resource"`
    // ...
}
```

Register the object type in `shared/constants/object_type.go`.

## No omitempty

API resource fields must never use `omitempty`. Fields are always present in the response — either with a value or `null`.

- Use pointer types (`*string`, `*int`, `*time.Time`) for nullable fields
- Value types (`string`, `int`, `bool`) are always serialized with their value (never omitted)
- Slices serialize as `null` when nil (not omitted)
- `json.RawMessage` serializes as `null` when nil (not omitted)

```go
// Good — always present in JSON
ErrorMessage *string    `json:"error_message"`
StartedAt    *time.Time `json:"started_at"`
Input        json.RawMessage `json:"input"`

// Bad — field may be absent from JSON
ErrorMessage string     `json:"error_message,omitempty"`
StartedAt    *time.Time `json:"started_at,omitempty"`
```

## Field Naming

### Status Fields

Use `status` (not `status_code`) for status fields:

```go
// Good
Status string `json:"status" validate:"required"`

// Bad
StatusCode string `json:"status_code" validate:"required"`
```

### Date/Time Fields

All date/time fields must end with `_at`:

```go
CreatedAt   time.Time  `json:"created_at" validate:"required"`
UpdatedAt   time.Time  `json:"updated_at" validate:"required"`
StartedAt   *time.Time `json:"started_at"`
CompletedAt *time.Time `json:"completed_at"`
ReviewedAt  *time.Time `json:"reviewed_at"`
```

### General Naming

- Use `snake_case` for JSON field names
- Use `PascalCase` for Go struct fields
- Boolean fields: use descriptive names like `is_editable`, `requires_review`

## Nested Resources (Sub-Objects)

When referencing another resource, use a sub-object with `id` and `object` fields instead of a bare ID string:

```go
// Good - sub-object with id and object
type MyResource struct {
    Role *Role `json:"role"`
}

// Produces: { "role": { "id": "rl_...", "object": "role", "name": "Admin" } }
```

```go
// Bad - bare ID
type MyResource struct {
    RoleID string `json:"role_id"`
}

// Produces: { "role_id": "rl_..." }
```


## Sub-resource arrays use List

Any API resource field that is an array of other resources (sub-resources) MUST use the `List[T]` type (`*List[T]` in Go), not a raw slice. The JSON shape is always `{ "object": "list", "page_info": { ... }, "data": [ ... ] }`. This keeps responses consistent and allows future pagination of sub-resource lists if needed.

- For **embedded lists** that are not paginated, use `NewList(items, PageInfo{})` with an empty `PageInfo`.
- The same `List[T]` type is used for both top-level paginated list endpoints and for embedded sub-resource arrays; for embedded lists, `page_info` is typically empty (e.g. `next_cursor: null`, `has_next_page: false`).

```go
// Good — sub-resource array as List
type Delivery struct {
    Lines *List[DeliveryLine] `json:"lines"`
}

// Bad — raw slice of sub-resources
type Delivery struct {
    Lines []*DeliveryLine `json:"lines"`
}
```

## Export endpoints

**Export endpoints** (e.g. export items, export inventory change logs) are an exception:

- They **may use a plain array** for the payload when returning JSON (e.g. `Items []*T`) rather than `List[T]`, since the response is not a standard paginated list.
- In general, export endpoints **should return a file** (e.g. an Excel spreadsheet) rather than a JSON list. The HTTP response should be a file download with an appropriate `Content-Type` (e.g. `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`) and `Content-Disposition: attachment; filename="..."`. When returning a file, no API resource type is used for the body.

## Expandable Relations

Full nested resources that are optionally included use the `expandable` tag:

```go
type AgentRun struct {
    // Optionally expanded via ?include[]=definition
    Definition *AgentDefinition `json:"definition" expandable:"true"`

    // Optionally expanded via ?include[]=actions; when expanded, value is a List with data (and optionally page_info)
    Actions *List[AgentAction] `json:"actions" expandable:"true"`
}
```

Expandable fields are `null` when not expanded. When an expandable list field is expanded, its value is a List with `object`, `page_info`, and `data`.

## Sample Data

Every resource must have a `SchemaExample()` method and a package-level `Sample<ResourceName>` variable for OpenAPI doc generation:

```go
var SampleMyResource = &MyResource{
    ID:     "mr_01jm4r6700f8nwq3v5hx2d9ktp",
    Object: constants.ObjectTypeMyResource,
    // ... populate all required fields
}

func (*MyResource) SchemaExample() any {
    return apiexample.ValidateAndMarshalToMap(SampleMyResource)
}
```

## List Responses

Paginated list endpoints return a `List[T]` wrapper. The same `List[T]` type is used for both top-level paginated list endpoints and for embedded sub-resource arrays; for embedded lists, `page_info` is typically empty.

```json
{
    "object": "list",
    "page_info": {
        "next_cursor": "...",
        "prev_cursor": null,
        "has_next_page": true,
        "has_prev_page": false
    },
    "data": [...]
}
```

Use `apiresource.NewList(items, pageInfo)` to construct list responses in presenters. For embedded (non-paginated) sub-resource lists, use `apiresource.NewList(items, apiresource.PageInfo{})`.

## Include Subresources

The `include` system lets clients selectively expand related subresources on a response. Unexpanded fields return `null`.

### Client Usage

Clients request includes via query parameters:

```
GET /v1/ai/runs/agr_xxx?include[]=definition&include[]=actions
GET /v1/ai/runs/agr_xxx?include=definition,actions
GET /v1/ai/runs/agr_xxx?include[]=definition&include[]=definition.config
```

Both array-style (`include[]`) and comma-separated (`include`) formats are supported, and can be mixed.

### Marking Fields as Expandable

Tag expandable fields on the resource struct with `expandable:"true"`. These fields must be pointer types (including `*List[T]` for list sub-resources) so they serialize as `null` when not expanded:

```go
type AgentRun struct {
    ID         string              `json:"id" validate:"required"`
    Object     constants.ObjectType `json:"object" validate:"required,enum=agent_run"`
    Actions    *List[AgentAction]   `json:"actions" expandable:"true"`
    Definition *AgentDefinition    `json:"definition" expandable:"true"`
    Steps      *List[AgentRunStep]  `json:"steps" expandable:"true"`
}
```

When expanded, a list field is a List with `object`, `page_info`, and `data`; when not expanded, the field is `null`.

### Registering Includes

Register the expandable fields for each resource type in `pkg/endpoint/include_definitions.go`:

```go
RegisterIncludes(&ObjectIncludes{
    ObjectType: constants.ObjectTypeAgentRun,
    Fields: []IncludeFieldDef{
        {Key: "actions", ObjectType: constants.ObjectTypeAgentAction},
        {Key: "definition", ObjectType: constants.ObjectTypeAgentDefinition},
        {Key: "steps", ObjectType: constants.ObjectTypeAgentRunStep},
    },
})
```

Nested includes (e.g., `definition.config`) are resolved automatically by walking the registry graph — if `AgentDefinition` registers its own includes (config, tools, role), those become available as `definition.config`, `definition.tools`, etc.

### Declaring Includes on an Endpoint

Each endpoint that supports includes declares them in `Materialize()` using `IncludesFor`:

```go
IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
    ObjectType: constants.ObjectTypeAgentRun,
    Fields: []string{"actions", "definition", "steps",
                     "definition.config", "definition.tools", "definition.role"},
}),
```

Only the fields listed here are valid for that endpoint. Requesting an unlisted field returns a `ParameterInvalidError`.

### Conditional Data Fetching

In the service layer, use `appctx.IsIncludeRequested(ctx, key)` to avoid fetching expensive data the client didn't ask for:

```go
if appctx.IsIncludeRequested(ctx, "role.permissions") {
    // fetch permissions from backend
}
```

Use `appctx.GetRequestedIncludeKeys(ctx)` to forward the requested includes to backend gRPC services.

**Backward compatibility:** When no includes are set in the context (nil), `IsIncludeRequested()` returns `true` for all keys, so existing code that doesn't use the include system continues to work.

### How Collapsing Works

The presenter always populates all expandable fields when data is available. After the presenter returns, the endpoint framework applies `CollapseUnexpanded` which sets unrequested expandable fields to `null` in the serialized response.

For nested includes, parent objects are preserved when a child include is requested. For example, requesting `actor.role` keeps the `actor` object but only expands the `role` field within it.

### OpenAPI Documentation

The `expandable:"true"` struct tag is picked up by the OpenAPI generator to emit `x-expandable: true` in the schema, signaling to SDK generators which fields support expansion.
