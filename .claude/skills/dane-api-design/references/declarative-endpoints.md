# Declarative API Endpoints (the OpenMRP registry pattern)

> Source: https://www.danealbaugh.com/articles/declarative-endpoints

**Status in this repo**: this IS the framework this repo runs on — the
article describes the pattern built here for OpenMRP's 400+ endpoint API.
Live code: `services/api-gateway/endpoints/<resource>/` (endpoint
definitions), `shared/field` (`Optional`/`Clearable`), `shared/idempotency`.
All new endpoints must use it; never hand-roll an HTTP handler.

## Problem

Hand-written Go handlers repeat the same steps — extract params, decode
JSON, validate fields, handle errors, serialize — hundreds of times.
Repetition breeds inconsistency and bugs, and is miserable to review at
scale.

## Solution shape

### Generic endpoint type

One generic struct describes any endpoint:

```go
type APIEndpoint[TReq, TResp any] struct {
    Title             string
    Method            string
    Route             string
    ContentType       string
    SuccessStatusCode int
    Public            bool
    ServiceHandler    func(svc any) ServiceHandler[TReq, TResp]
    LocationFunc      func(TResp) string
    IncludeConfig     *IncludeConfig
}
```

Request/response types are recovered with `reflect.TypeFor[T]()`.

### Struct tags as the schema

| Tag | Purpose |
|-----|---------|
| `json` | body binding |
| `query` | query params |
| `path` | path segments |
| `header` | headers |
| `validate` | validation rules |
| `expandable` | response sub-object expansion |
| `sensitive` | redaction in logs |

### Absent vs null (Go's JSON two-state problem)

`json.Unmarshal` can't distinguish a missing field from an explicit
null/zero. Custom wrapper types record what the client actually sent:

- `field.Optional[T]` — optional field that rejects explicit `null`.
- `field.Clearable[T]` — PATCH three-state: set / clear / omit.

Enforced as values (not pointers) with startup validation. This idea is
worth stealing even without the framework.

### Bind plan + Execute

A cached per-type "bind plan" (reflection walk of the struct) drives a
single shared `Execute` method that runs the whole HTTP lifecycle for every
endpoint: version check → logging/redaction → idempotency key extraction →
allocate+bind request → includes → decode/transform JSON → validate enums +
struct → call the service handler → respond (Location on 201, downloads,
includes).

Free everywhere, automatically: unknown-field rejection with Levenshtein
suggestions, explicit-null rejection, empty-PATCH validation, client
disconnect detection (499), slice presence tracking for clears.

### Registration and services

Resource groups implement `Materialize()`; a registry wires middleware +
services and registers routes at startup:

```go
apiendpoint.From(&RetrieveAPIKeyEndpoint{}).
    WithMiddleware(authMw).
    WithService(inner, apiKeySvc)
```

Service handlers receive already-validated, bound requests and only do
business mapping. A full endpoint is one struct literal:

```go
func (e *CreateAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *CreatedAPIKey] {
    return &apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *CreatedAPIKey]{
        Title:             "Create API Key",
        Method:            http.MethodPost,
        Route:             "/v1/auth/api-keys",
        SuccessStatusCode: http.StatusCreated,
        ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAPIKeyRequest) (*CreatedAPIKey, *APIError) {
            return svc.(APIKeySvc).CreateAPIKey
        },
        LocationFunc: func(resp *CreatedAPIKey) string {
            return "/v1/auth/api-keys/" + resp.APIKeyInfo.ID
        },
    }
}
```

```go
type CreateAPIKeyRequest struct {
    RoleID    string                    `json:"role_id" validate:"required"`
    Name      string                    `json:"name" validate:"required,max=255"`
    ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
}
```

## Why it wins at scale

- **Consistency** — identical error handling/validation/formatting on every
  endpoint; no copy-paste drift, because nobody writes handler code.
- **Cross-cutting concerns are one edit** — versioning, idempotency keys,
  unknown-param rejection, expand support land everywhere via `Execute`.
- **Testability** — test `Execute` once with mock handlers.
- **Single source of truth** — the definition drives behavior AND OpenAPI
  generation.
- **Readability** — the whole endpoint contract in one literal.

## Costs

Reflection overhead (mitigated by caching), a learning curve (tags, bind
plans, wrapper types), and "magic" — `Execute` does a lot implicitly. Worth
it at hundreds of endpoints; overkill below that.
