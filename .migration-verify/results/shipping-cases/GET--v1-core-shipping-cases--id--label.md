# GET /v1/core/shipping-cases/{id}/label

**Status: PARITY CONFIRMED** — no fixes needed.

## What was compared

- **Validation**: Path parameter extraction (shipping case ID) — matches
- **Permission checks**: Internal actor + shipments/read — matches
- **DB queries**: Both fetch shipping case number by ID + account ID, return 404 if not found — matches
- **S3 key format**: `shipping-labels/{accountID}/{number}.gif` — matches
- **Presigned URL expiry**: 1 hour — matches
- **Error handling**: 404 for missing shipping case, auth errors for missing permissions — matches
- **Side effects**: None (correct for GET) — matches
- **Idempotency**: Not applicable (GET endpoint) — matches

## Intentional differences (by design)

1. **Response shape**: Dashboard returns a plain string (the URL). Go returns a structured resource `{ object: "shipping_case_label_url", url: "..." }` per API resource conventions requiring an `object` field on all responses.

2. **S3 file existence check**: Dashboard always generates a presigned URL even if the file doesn't exist in S3 (client would get a 404 from S3 when accessing the URL). Go checks file existence first and returns `null` URL if the label file is missing. This is an intentional improvement.

## Files reviewed

- Dashboard: `services/shipping-case.svc.ts` (fetchLabel), `repositories/shipping-case.repo.ts` (findLabel), `controllers/shipping-case.ctrl.ts`
- Go gateway: `endpoints/shipping-cases/endpoint_get_shipping_case_label.go`, `endpoints/shipping-cases/service.go`
- Go core: `service/shipping_case_service.go` (GetShippingCaseLabel), `infrastructure/grpc/grpc_shipping_case_handler.go`
- Go resource: `pkg/resource/shipping_case_resource.go` (ShippingCaseLabelURL)
