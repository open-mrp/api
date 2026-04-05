# PUT /v1/core/analytics/oee - Migration Verification

## Result: Issues found and fixed

## What was compared

- Validation rules (required fields, date parameters)
- Permission checks (internal actor, invoices read permission)
- DB queries and logic (filters, joins, aggregation)
- Response shape (field names, types)
- Side effects (none expected)

## Issues found and fixed

### 1. Wrong date filter column (FIXED)
- **Dashboard**: Filters batches by `b.scanned_at >= startDate AND b.scanned_at <= endDate`
- **Go (before)**: Filtered by `b.closed_at IS NOT NULL AND b.closed_at >= startDate AND b.closed_at <= endDate`
- **Fix**: Changed to use `b.scanned_at` to match Dashboard behavior

### 2. Missing unit ratio normalization (FIXED)
- **Dashboard**: Normalizes all quantity values using `value * (unit.ratio_numerator / unit.ratio_denominator)` by joining the `unit` table for each quantity
- **Go (before)**: Summed raw `quantity.value` without any unit conversion
- **Fix**: Added unit table joins and ratio multiplication for good_units, waste_units, and seconds_units

### 3. Wrong JOIN types (FIXED)
- **Dashboard**: Uses LEFT JOINs for scanning_station and department, with `COALESCE(d.id, 'unassigned')` / `COALESCE(d.name, 'Unassigned')` to handle batches without department assignment
- **Go (before)**: Used INNER JOINs, which would exclude batches without a scanning station or department
- **Fix**: Changed to LEFT JOINs with COALESCE, matching Dashboard behavior

### 4. Missing department filtering (FIXED)
- **Dashboard**: Filters results by `departmentIDs` when provided via `AND ss.department_id IN (...)`
- **Go (before)**: `DepartmentIDs` parameter existed in domain model but was never used
- **Fix**: Added in-memory department filtering in the repository (consistent with how other optional filters like inventory receipt filtering are handled in this codebase)

### 5. Missing estimated runtime hours calculation (FIXED)
- **Dashboard**: Runs a separate query (Query B) that computes daily scan windows per department using `TIMESTAMPDIFF(SECOND, MIN(scanned_at), MAX(scanned_at))` grouped by department and date, then sums across days. The service converts seconds to hours by dividing by 3600.
- **Go (before)**: Hardcoded `EstimatedRuntimeHours` to 0
- **Fix**: Added new SQL query `GetOeeEstimatedRuntime` that computes daily scan windows, and updated the repository to merge runtime data and convert to hours

## Files modified
- `services/core-service/internal/infrastructure/queries/analytics.sql` - Updated `GetOeeDepartmentData` query and added `GetOeeEstimatedRuntime` query
- `services/core-service/internal/infrastructure/repository/analytics_repository.go` - Updated `GetOeeByDepartment` to use both queries, apply department filtering, and compute runtime hours
- `services/core-service/internal/infrastructure/sqlc/analytics.sql.go` - Regenerated via `make sqlc core`

## Remaining concerns

- The Dashboard also computes `standardTimeGoodSeconds` and `standardTimeAllSeconds` in Query A (involving production_step, labor_time, leveling_factor, allowances, and a production subquery), but these fields are **not included in the response** - the service only returns goodUnits, wasteUnits, secondsUnits, and estimatedRuntimeHours. So omitting them from the Go query is correct for response parity.
- Permission check matches: both use `invoices` domain with `read` action and require internal actor.
- Response shape matches: both return an array of departments with the same fields. Go adds the standard `object` field per API conventions.
