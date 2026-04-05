# PUT /v1/core/analytics/new-customers — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Validation**: Request fields (startDate, endDate, customerGroupIDs, salesRepIDs) — match
- **Permission checks**: Internal actor + `customers:read` — match
- **Response shape**: Dashboard returns array of data points (one per customer, `x: createdAt, y: 1`); Go was returning a single aggregated count
- **DB query & filters**: Dashboard filters by customerGroupIDs (account_group OR price_group membership) and salesRepIDs; Go had no filter support
- **Label**: Dashboard uses `"New Customers"`; Go used `"new_customers"`
- **Side effects**: None in either implementation
- **Idempotency**: PUT method, no idempotency keys needed — correct

## Issues found and fixed

### 1. Response shape mismatch (critical)
- **Dashboard**: Returns one `{x: createdAt, y: 1}` data point per new customer found in the date range
- **Go (before)**: Returned a single aggregated `COUNT(*)` as one data point with no X timestamp
- **Fix**: Changed SQL from `GetNewCustomersCount :one` (COUNT) to `GetNewCustomerEntries :many` (SELECT created_at), and updated the entire chain (repo → service → gRPC handler) to return individual entries. Each entry becomes a `DateTimeCoordinateProto{X: createdAt, Y: 1}`.

### 2. Missing customer group filter (critical)
- **Dashboard**: Filters by `customerGroupIDs` matching EITHER `accountRelation.accountGroup.id` OR `accountRelationPriceGroups.group.id`
- **Go (before)**: No filter support at all — `customer_group_ids` and `sales_rep_ids` request fields were accepted but ignored
- **Fix**: Added `include_customer_group_filter` boolean toggle + `customer_group_ids` / `price_group_ids` slices to SQL. The query now matches against both `ar.account_group_id` and `account_relation_price_group.account_group_id` via an EXISTS subquery, matching Dashboard's OR logic.

### 3. Missing sales rep filter (critical)
- **Dashboard**: Filters by `salesRepIDs` matching `defaultSalesRep.id`
- **Go (before)**: Ignored
- **Fix**: Added `include_sales_rep_filter` boolean toggle + `sales_rep_ids` slice to SQL, filtering on `ar.default_sales_rep_id`.

### 4. Label mismatch (minor)
- **Dashboard**: `"New Customers"`
- **Go (before)**: `"new_customers"`
- **Fix**: Changed label to `"New Customers"` in gRPC handler.

### 5. Domain model missing filter fields
- **Fix**: Added `CustomerGroupIDs []string` and `SalesRepIDs []string` to `GetNewCustomersAnalyticsParams`. Added `NewCustomerEntry` struct with `CreatedAt time.Time`.

### 6. gRPC handler not passing filter params
- **Fix**: Handler now passes `req.CustomerGroupIds` and `req.SalesRepIds` to domain params.

## Files modified

- `services/core-service/internal/infrastructure/queries/analytics.sql` — rewrote query
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` — new repo method
- `services/core-service/internal/domain/analytics_models.go` — added fields + new struct
- `services/core-service/internal/domain/repositories.go` — updated interface
- `services/core-service/internal/domain/services.go` — updated interface
- `services/core-service/internal/service/analytics_service.go` — updated return type
- `services/core-service/internal/infrastructure/grpc/grpc_analytics_handler.go` — full rewrite of handler
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` — regenerated via `make sqlc core`

## Remaining concerns

- Proto definition already supports `customer_group_ids` and `sales_rep_ids` fields — no proto changes needed.
- API gateway service and presenter already handle the `DateTimeCoordinateProto` array correctly — no changes needed there.
- The `account_relation_role_code = 'customer'` filter in Go matches Dashboard's implicit Prisma relation filtering (the repo queries `accountRelation` which is typed as customer relations).
