# Anatomy of an API — the four layers

> Source: https://www.danealbaugh.com/articles/api-anatomy

There is no single "right" way to structure an API; what matters is applying
separation of concerns *consistently*. This is the canonical four-layer
split. Data flows downward (routing → controller → service → repository);
each layer has a distinct concern and upper layers depend on abstractions
from lower layers, never concrete implementations.

## 1. Routing layer

Entry point for HTTP requests; maps endpoints to controller methods.

- Capture the incoming request; dispatch on URI path + HTTP method.
- Thin and stateless. **Only** decides which controller to invoke — no
  request processing, no business logic.

## 2. Controller layer

Bridge between HTTP transport and business logic — the gatekeeper.

- Parse request data: path params, query strings, bodies.
- Validate **request shape** against DTOs before anything passes upstream.
- Invoke the appropriate service method.
- Construct and return the HTTP response; own all HTTP-specific concerns.

Only valid, well-formed requests reach services.

## 3. Service layer

Core business logic, independent of HTTP and of the database.

- Implement workflows and business rules; work with domain models.
- Contain **business validation** (the rules, as opposed to the shape).
- Coordinate repository calls; return domain objects, never HTTP responses.
- Represent the "what" and "why" of an operation, not the "how" of
  persistence or transport.

Two shapes:

- **Resource-oriented services** — operations grouped around one resource
  (OrdersService). Fine for small, simple resources; bloats as systems grow.
- **Mediator services** — small, use-case-focused services per workflow
  (CreateOrder, CancelOrder). Keeps controllers thin; each workflow easy to
  locate, test, and evolve. (The article credits Brandur Leach's discussion
  of this pattern.) When a resource service grows multiple responsibilities,
  refactor toward this.

## 4. Repository layer

Abstraction over persistence and external data sources.

- Encapsulate ALL data-access logic — the only layer that knows the
  database exists.
- Expose a clean interface (Save, FindByID, Update, …) to the service layer.
- Interfaces allow swapping implementations (SQL → NoSQL, fakes for tests)
  without touching upper layers.

## The rules in one place

- **Layering**: routing → controllers → services → repositories, downward
  only.
- **Validation split**: request-shape validation in controllers; business
  validation in services.
- **Dependency direction**: depend on interfaces from below, never on
  concrete implementations.
- **Isolation**: HTTP concerns never reach services; data-access concerns
  never leave repositories.

## Mapping to this repo

The routing and controller layers are absorbed by the declarative endpoint
framework: `services/api-gateway/endpoints/<resource>/` declares endpoints
(`Materialize()`), and the shared `Execute` does the binding, validation,
and response work a hand-written controller would. Service handlers behind
the gateway map to backend services over gRPC; each backend service keeps
its own service/repository split. Layer adherence is enforced by
`services/structure_adherence_test.go`.
