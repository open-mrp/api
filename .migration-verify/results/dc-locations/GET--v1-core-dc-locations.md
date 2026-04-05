# GET /v1/core/dc-locations — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Actor type, permission domain, action
- **Query parameters**: Search, pagination
- **SQL queries**: Filters, joins, ordering
- **Response shape**: Field names, types, nested resources
- **Error handling**: Error types
- **Side effects**: None expected for GET

## Issues found and fixed

### 1. Permission domain mismatch (fixed)
- **Dashboard**: All DC location operations use `PermissionDomains.ediRuns` (`edi_runs`) permission domain
- **Go (before fix)**: Used `PermissionDomainEdiLocations` (`edi_locations`) — a separate permission domain
- **Fix**: Changed all five DC location service methods (List, Get, Create, Update, Delete) in `edi_service.go` to use `PermissionDomainEdiRuns` instead of `PermissionDomainEdiLocations`

### 2. Search scope missing customer name (fixed)
- **Dashboard**: `EdiDCLocationAdapter.fetchInput` searches both `location` AND customer account name via `CustomerAccountAdapter.fetchInput`
- **Go (before fix)**: SQL only searched `dcl.location LIKE`
- **Fix**: Added `OR a.name LIKE sqlc.narg('search_query')` to both `ListDCLocationsForward` and `ListDCLocationsBackward` SQL queries in `edi.sql`, then regenerated sqlc

## Acceptable differences (not bugs)

- **Pagination style**: Dashboard uses offset-based (`take`/`skip`), Go uses cursor-based (`cursor`/`limit`). This is an intentional platform-wide migration pattern.
- **Sort order**: Dashboard sorts by MySQL full-text relevance on `location`; Go sorts by `created_at DESC`. Cursor-based pagination requires stable ordering, and relevance sorting is not compatible with cursor pagination.
- **Response shape**: Dashboard returns a rich `CustomerSummary` (id, name, number, email, createdAt, customerTypeGroup, status), Go returns a lightweight `DCLocationCustomer` sub-resource (id, object, name). The Go version follows API resource conventions for sub-resources. The customer can be fetched independently for full details.
- **Response wrapper**: Dashboard returns `{ items, count }`, Go returns `{ data, page_info }` per standard list response conventions.
