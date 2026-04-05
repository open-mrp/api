# Verification: POST /v1/core/shipments/{shipment_id}/lines

## Result: NEW ENDPOINT — No Dashboard Equivalent

This endpoint does **not** exist in the Dashboard Express.js API. It is a **new Go-only endpoint**.

### Dashboard Findings

- **No standalone "create shipment line" route** exists in the Dashboard.
- Shipment lines are created exclusively as part of the **pack flow** (`POST /v1/picks/:pickID/actions-pack`), which creates a shipment with lines inline.
- `ShipmentLineRepo.create()` exists in the Dashboard but is **never exposed via any API route**.
- `ShipmentLineSvc` is an empty class with no methods.

### Go Implementation Review

The Go endpoint is well-structured and follows all project patterns:

| Aspect | Status | Details |
|--------|--------|---------|
| **Validation** | OK | `sales_order_line_id`, `quantity_value`, `quantity_unit_id` all required via struct tags |
| **Permission checks** | OK | Internal actor only + `PermissionDomainShipments` / `ActionCreate` |
| **Account scoping** | OK | Validates shipment belongs to target account via `IsInAccount()` |
| **Idempotency** | OK | Uses idempotency keys with `RecoveryPointStarted`/`RecoveryPointFinished` |
| **Transaction** | OK | Quantity + shipment line creation wrapped in `withTx` |
| **SQL queries** | OK | `CreateShipmentLineQuantity` + `CreateShipmentLine` defined in `shipment_line.sql` |
| **Response shape** | OK | Returns `ShipmentLine` resource with nested `LightQuantity` and `LightSalesOrderLine` sub-resources |
| **Error caching** | OK | Errors cached via `CacheErrorResponse` for idempotency |

### Repository Status

All repository methods are **TODO stubs** (not yet implemented). The SQL queries are defined in `shipment_line.sql` and ready for `make sqlc core` to generate the code, after which the repository can be wired up.

### Comparison with Dashboard's Unexposed Create Logic

The Dashboard's `ShipmentLineRepo.create()`:
- Validates shipment is in account (Go does the same)
- Creates a quantity from order line's packed quantity (Go takes explicit quantity from request body — appropriate for a standalone endpoint)
- Creates shipment_line record (Go does the same)

### Conclusion

No parity issues — this is a net-new endpoint. The Go implementation follows all established patterns correctly. The only remaining work is implementing the repository methods after running `make sqlc core`.
