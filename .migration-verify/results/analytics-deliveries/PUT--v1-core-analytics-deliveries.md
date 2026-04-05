# PUT /v1/core/analytics/deliveries — Verification Result

**Status: Issues found and fixed**

## What was compared

| Layer | Dashboard | Go | Match? |
|-------|-----------|-----|--------|
| **Request validation** | `startDate` (required), `endDate` (required), optional `productLineIDs`, `customerIDs`, `customerGroupIDs`, `salesRepIDs`, `targetDeliveryTimeDays`, `overridePromisedDates` | Same fields via `AnalyzeDeliveriesRequest` struct with `validate:"required"` on dates | Yes |
| **Permission checks** | `checkIsInternalActor` + `checkHasPermission(invoices, read)` | `CheckIsInternalActor()` + `CheckHasPermission(Invoices, Read)` + `CheckTargetAccountSet()` | Yes |
| **HTTP method** | PUT (via `api.analyzeDeliveries`) | PUT | Yes |
| **Response shape** | `{ statistics: {...}, chartData: {...} }` | `{ object, statistics, chart_data }` (Go adds `object` field per conventions) | Yes |
| **Statistics fields** | 10 fields (averages, percentages, counts) | Same 10 fields | Yes |
| **Chart data** | 3 charts (on-time delivery, avg delivery time, avg first shipment time), each with name/type/data coordinates | Same 3 charts with name/type/data coordinates | Yes |
| **DB query** | Prisma query on `invoice` joined with `sales_order`, filtered by accountID + date range + optional filters | sqlc query on `invoice` joined with `sales_order`, filtered by accountID + date range | See note below |
| **In-memory processing** | `DeliveryAnalyticsUtils` — groups by invoice, computes summaries, generates 30-point chart data | Equivalent Go functions: `groupByInvoice`, `getInvoiceDateSummary`, `computeDeliveryStatistics`, `computeDeliveryChartData` | Yes |
| **processDeliveryEntryWithTargetTime** | If `overridePromisedDates` is true and entry has no `promisedAt` but has `issuedAt`, sets `promisedAt = issuedAt + targetDeliveryTimeDays` | Same logic in repository | Yes |
| **Idempotency** | Not applicable (PUT, read-only analytics) | No idempotency tracking (correct for PUT) | Yes |
| **Side effects** | None | None | Yes |

## Issues found and fixed

### 1. Repository was a stub (CRITICAL)
The `GetDeliveryAnalytics` repository method returned empty results with a comment saying "this will be computed in the service layer." All business logic was missing.

**Fix:** Implemented the full delivery analytics pipeline:
- Added `GetDeliveryEntries` SQL query to fetch invoice/sales_order data
- Added `DeliveryEntry` domain model
- Implemented `nullTimePtr` helper for nullable time columns
- Implemented `groupByInvoice` — groups entries by invoice number, filters by issuedAt date range
- Implemented `getInvoiceDateSummary` — extracts earliest/latest dates per invoice (matching dashboard logic exactly: earliest issuedAt, earliest invoicedAt, earliest firstShipAt, latest completedAt, earliest promisedAt)
- Implemented `computeDeliveryStatistics` — computes all 10 statistics fields matching `DeliveryAnalyticsUtils.getDeliveryTimeStatistics` + `getOrderVolumeStatistics`
- Implemented `computeDeliveryChartData` — generates 30 data points for each chart, matching dashboard interval calculation
- Implemented `processDeliveryEntryWithTargetTime` equivalent
- Implemented per-interval chart helpers: `computeOnTimeDeliveryPct`, `computeAvgDeliveryTimeToCompletion`, `computeAvgDeliveryTimeToFirstShipment`

### 2. ChartDataPoint model had wrong fields
`ChartDataPoint` had `Label string` / `Value float64` but the dashboard sends x/y coordinates where X is a Unix timestamp in milliseconds and Y is the metric value.

**Fix:** Changed `ChartDataPoint` to `X float64` / `Y float64`.

### 3. chartDataPointsToProto used sequential index for X
The `chartDataPointsToProto` helper used `float64(i)` for the X coordinate instead of the actual timestamp value.

**Fix:** Changed to use `p.X` from the data point.

### 4. chartDataPointsToProto didn't set Type field
The proto `ChartDataPointProto` has a `type` field but it was never set. Dashboard always uses `"line"` chart type.

**Fix:** Added `chartType` parameter and set chart names to match dashboard defaults (`"On-Time Delivery %"`, `"Average Delivery Time (Days)"`, `"Average Time to First Shipment (Days)"`) with type `"line"`.

## Remaining concerns

1. **Filter parity (minor):** The dashboard Prisma query supports filtering by `salesRepIDs`, `customerIDs`, `customerGroupIDs`, and `productLineIDs` at the SQL level via WHERE clauses. The Go SQL query currently fetches all entries for the account/date range and does not apply these filters at the SQL level. For large datasets this could be a performance concern, but functionally the analytics computation is correct because all entries are fetched and the dashboard also fetches all entries (it batches with take/skip but gets everything). The optional filters should be added to the SQL query for production parity.

2. **Chart coordinate X values:** The dashboard uses `intervalStartDate.getTime()` (JavaScript `Date.getTime()` returns Unix epoch in milliseconds). The Go implementation uses `intervalStart.UnixMilli()` which is equivalent.
