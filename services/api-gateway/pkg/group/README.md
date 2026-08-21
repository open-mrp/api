# `httpgroup`

This package defines **grouped API endpoint definitions** for resource types.

Each group:

- Materializes HTTP endpoints for a given resource (e.g. `/customers`)
- Wires together handlers, services, and request/response types
- Should be declarative and descriptive

Define REST endpoints and route metadata here  
Do not put business logic in this layer

Each group has a `Materialize` method taking a config struct that it validates (and panics on, at
startup) before wiring the endpoints. See `docs/patterns/architecture-patterns.md`.
