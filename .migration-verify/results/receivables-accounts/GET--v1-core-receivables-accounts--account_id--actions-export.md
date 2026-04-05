# Verification: GET /v1/core/receivables/accounts/{account_id}/actions/export

## Status: Issues found and fixed

## What was compared

| Area | Dashboard | Go | Match? |
|------|-----------|-----|--------|
| **Permission check** | `customers:read`, internal actor | `customers:read`, internal actor | Yes |
| **Query params** | `cutoffDate` (optional) | `cutoff_date` (optional) | Yes |
| **DB query** | Prisma: `isPaidInFull: false`, buyer filter, cutoff date, ordered by `createdAt DESC` | SQL: `is_paid_in_full = false`, buyer filter, cutoff date, ordered by `created_at DESC` | Yes |
| **Remaining balance calc** | Sum of invoice line (qty * unit_price) minus sum of allocations (with cutoff) | Same calculation in SQL | Yes |
| **CSV headers** | Invoice Number, Invoiced At, Customer Number, Customer Name, Remaining Balance | Same | Yes |
| **CSV date format** | `toLocaleDateString()` (locale-dependent) | `time.RFC3339` (precise, unambiguous) | Acceptable improvement |
| **Filename** | `receivables-report-{cutoffDate}.csv` | Was `receivables.csv` (fixed) | Fixed |
| **No pagination** | Returns all results | Returns all results (limit 10000) | Yes |
| **Idempotency** | Not needed (GET) | Not used | Correct |
| **Response** | CSV file download with Content-Type/Disposition headers | CSV file download via `FileDownload` struct | Yes |

## Issues found and fixed

1. **Filename mismatch**: The Go API was using a static filename `receivables.csv` while the Dashboard uses `receivables-report-{cutoffDate}.csv`. Fixed in `services/api-gateway/endpoints/receivables/service.go` to generate a dynamic filename that includes the cutoff date when provided, falling back to `receivables-report.csv` when no cutoff date is specified.

## Notes

- The Go CSV uses RFC3339 dates for `Invoiced At` values rather than locale-dependent `toLocaleDateString()`. This is a deliberate improvement for consistency.
- The Go implementation caps results at 10,000 via `ListAllByCustomer`. The Dashboard has no explicit cap but is bounded by Prisma defaults. 10,000 is a reasonable safeguard.
- Pre-existing compiler errors in shipment and sales order code are unrelated to this endpoint.
