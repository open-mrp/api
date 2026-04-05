# Verification: DELETE /v1/core/shipments/{shipment_id}/lines/{id}

## Result: No Dashboard Equivalent — New Endpoint

### Summary

The Dashboard Express.js API does **not** have a dedicated endpoint for deleting individual shipment lines. In the Dashboard, shipment lines are only deleted as a cascading side effect of deleting an entire shipment (`DELETE /v1/shipments/:shipmentID`), which:
1. Unpacks all picks associated with the shipment
2. Deletes all shipping cases
3. Deletes all shipment lines
4. Deletes the shipment itself

The Go endpoint `DELETE /v1/core/shipments/{shipment_id}/lines/{id}` is a **new endpoint** with no legacy counterpart. There is no parity to verify.

### What Was Compared

| Aspect | Dashboard | Go |
|--------|-----------|-----|
| Endpoint exists | No (lines only deleted via shipment delete) | Yes |
| Permission domain | `shipments` + `update` (on shipment delete) | `shipments` + `delete` |
| Side effects | Unpack picks, cascade delete cases/lines | Delete single line only |

### Go Implementation Status

- **API Gateway endpoint**: Complete (`endpoint_delete_shipment_line.go`)
- **gRPC handler**: Complete (`grpc_shipping_handler.go`)
- **Service logic**: Complete (`shipment_line_service.go:302-338`) — validates line belongs to shipment, deletes in transaction
- **Repository**: **Stub only** — `Delete()` and `IsInShipment()` return hardcoded values (TODO comments, awaiting `make sqlc core`)
- **SQL queries**: Defined in `shipment_line.sql` (`DeleteShipmentLine`, `DeleteShipmentLineQuantity`)

### Remaining Concerns

1. **Repository stubs**: The `Delete` and `IsInShipment` repository methods are not implemented (return `nil`/`false`). They need to be wired to the sqlc-generated code after running `make sqlc core`.
2. **Quantity cleanup**: The SQL query `DeleteShipmentLineQuantity` exists but is not called in the service flow. If shipment lines reference a `quantity_id`, orphaned quantity records could accumulate.
3. **No side effects**: Unlike the Dashboard's shipment delete (which unpacks picks), this endpoint performs no side effects. If deleting a shipment line should affect pick status or inventory, that logic is missing. However, since this endpoint never existed in the Dashboard, the expected behavior is unclear and may be intentionally minimal.
4. **New endpoint**: Since this is entirely new functionality not present in the Dashboard, it should be evaluated on its own merits rather than for migration parity.
