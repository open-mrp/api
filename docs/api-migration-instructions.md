# Dashboard API → Go API

## Context

There are many endpoints currently live in the dashboard Express.js API (`dashboard/apps/api`). We need to migrate them to the Go API (`api/`) following the conventions in `api/docs/`. We should be careful to preserve all existing business logic in the Express.js API so that it will work as a drop in replacement.

**Decision: No new microservices** — sales orders, shipments, and all other core services will be in the core microservice for the time being.

**Key existing infrastructure:**
- DB tables already exist. We will not need to migrate anything.
- ID prefixes already exist in `shared/id`.
- Permission domains already exist.

## Technical Notes

- **SQL queries:** Separate files per entity (sales_order.sql, pick.sql, shipment.sql, etc.)
- **Proto:** Create a separate proto per entity.
- **Customer actor access:** Dashboard supports customer actors for some endpoints. This should be preserved when applicable.

## Verification

For each batch:
1. Run `make proto` and `make sqlc` after adding proto/SQL
2. Run `make test` to verify compilation and unit tests pass
3. Run `make dev` and test endpoints via curl/Postman against local environment
4. Compare response shapes with dashboard API to ensure parity

## Key Source Files (Dashboard)
- `dashboard/apps/api/src/services` — core business logic of the legacy API
- `dashboard/apps/api/src/repositories` — where interfaces with the database live

## Key Reference Files (Go API)
- `docs/api-resource-conventions` — patterns we should observe always when returning data to users
- `docs/architecture-patterns` — patterns we should observe when designing our various layers
- `shared/db/migrations/0001_initial.sql` — existing DB schema (large, so don't try to read it all at once)

