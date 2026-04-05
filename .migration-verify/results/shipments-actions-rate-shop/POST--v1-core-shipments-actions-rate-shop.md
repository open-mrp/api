# POST /v1/core/shipments/actions/rate-shop — Migration Verification

## Status: Issues Found and Fixed

The Go `RateShop` endpoint was a **stub** returning empty results. The full business logic has been implemented to match the Dashboard's `ShipmentSvc.rateShop` method.

## What Was Compared

| Aspect | Dashboard | Go (Before) | Go (After) | Match? |
|--------|-----------|-------------|------------|--------|
| Permission checks (internal actor) | `checkHasPermission(shipments, read)` | `CheckHasPermission(shipments, read)` | Same | Yes |
| Permission checks (customer actor) | Validates `actor.accountID === customerID` | Missing customer actor validation | Added customer actor auth check | Yes |
| Product line freight exemption | `productLineRepo.isFreightExempt(ids)` → any exempt = true | Not implemented | Iterates product lines, checks `IsFreightExempt` | Yes |
| Customer/group freight exemption | `CustomerUtils.isCustomerOrGroupFreightExempt` checks `FreightPolicy` | Not implemented | Checks `FreightPolicy == FreightPolicyFree` | Yes |
| Shipping term freight exemption | `customer.defaultShippingTerm.isFreightExempt` | Not implemented | Checks `ShippingTermType == FreeFreight` | Yes |
| Carrier listing | Lists all carriers, loads options | Not implemented | Lists carriers, loads options per carrier | Yes |
| Portal filtering | Filters `isPortalEnabled` carriers/options for customer actors | Not implemented | Filters `IsPortalEnabled` for customer users | Yes |
| Shippo rate fetching | `fetchAllShippoRates` per carrier with Shippo | Not implemented | `FetchAllShippingRates` per carrier with Shippo | Yes |
| Non-Shippo carriers | Includes options with rate 0 | Not implemented | Includes options with rate 0 | Yes |
| Service level token matching | Maps Shippo rates to options by `serviceLevelToken` | Not implemented | Maps by `ServiceLevelToken` match | Yes |
| Flat rate application | Applies `shippingTerm.flatRate.measure` | Not implemented | Applies parsed `FlatRate.Value` | Yes |
| Minimum order free shipping | Checks `orderTotal > minimumOrderValue` + carrier option eligibility | Not implemented | Same logic with `freeShippingOptionIDs` set | Yes |
| Free shipping carrier options | Uses `freeShippingCarrierOptionIDs` set | Not implemented | Uses `freeShippingOptionIDs` map | Yes |
| Sort by rate | `sort((a, b) => a.rate - b.rate)` | Not implemented | `sort.Slice` by rate ascending | Yes |
| Exemption type | `none`, `flat_rate`, `minimum_order_met` | Not implemented | Same values | Yes |
| Flat rate in response | `shippingTerm.flatRate.measure` when flat rate exists | Not implemented | `flatRateValue` when `hasFlatRate` | Yes |
| Shipping markup | 10% markup applied in `fetchAllShippoRates` repo layer | Not implemented | 10% markup applied in Shippo client `FetchAllShippingRates` | Yes |
| Response shape | `{ options, exemptionType, flatRate }` | Empty options only | Full `RateShopResult` with all fields | Yes |
| Idempotency | Not required (read-only POST action) | N/A | N/A | Yes |

## Issues Found and Fixed

### 1. RateShop was a stub (Critical)
The entire `RateShop` service method was a TODO stub returning empty results. Implemented the full business logic matching the Dashboard's `ShipmentSvc.rateShop`.

### 2. Missing customer actor authorization check
Dashboard validates that customer actors can only rate-shop for their own account. Added the same check in Go.

### 3. Missing `FetchAllShippingRates` on ShippoClient interface
The `ShippoClient` interface only had `FetchShippingRate` (single rate). Added `FetchAllShippingRates` method to the interface, domain models (`FetchAllShippingRatesParams`), mock, and Shippo client implementation.

### 4. `FetchShippingRate` was also a stub
The existing `FetchShippingRate` method on the Shippo client was not implemented. Implemented it with the same logic as the Dashboard: service level token matching, BESTVALUE attribute fallback, cheapest rate fallback.

### 5. Shipping markup applied in wrong layer
The `EstimateRate` service method was applying a 10% shipping markup on top of what the Shippo client returns. In the Dashboard, the markup is applied in the repository/Shippo layer only. Moved the markup to the Shippo client (matching Dashboard) and removed it from the service layer to avoid double-application.

### 6. Missing Shippo rate response fields
Added `EstimatedDays` and `Attributes` fields to the Shippo `ShipmentRate` type to support rate shopping and BESTVALUE selection.

## Files Modified

- `services/core-service/internal/domain/clients.go` — Added `FetchAllShippingRates` to `ShippoClient` interface
- `services/core-service/internal/domain/shipping_models.go` — Added `FetchAllShippingRatesParams`
- `services/core-service/internal/domain/mock/client/shippo_client_mock.go` — Added mock for `FetchAllShippingRates`
- `services/core-service/internal/infrastructure/shippo/types.go` — Added `EstimatedDays`, `Attributes` to `ShipmentRate`
- `services/core-service/internal/infrastructure/shippo/client.go` — Implemented `FetchShippingRate`, `FetchAllShippingRates`, shipping markup, helper methods
- `services/core-service/internal/service/shipment_service.go` — Full `RateShop` implementation, removed double-markup from `EstimateRate`

## Remaining Concerns

1. **Shipping markup constant duplication**: The 10% markup constant exists in the Shippo client. If this value ever changes, it needs to be updated there (matching the Dashboard's `SHIPPING_RATE_MARKUP = 1.1`).
2. **Product line freight exempt check**: Uses iteration over individual `Get` calls rather than a batch query. For typical use (1-3 product lines), this is fine, but a batch method could be added for performance if needed.
3. **Pre-existing compile errors**: Other files in the core-service have pre-existing compile errors unrelated to this endpoint (item_repository, sales_order_repo, shipment_repository). These should be fixed separately.
