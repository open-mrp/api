# Verification: GET /v1/core/addresses/autocomplete

## Status: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces all Dashboard behavior.

## What was compared

- **Authentication**: Dashboard uses `AuthOptions.None` (no auth). Go uses `Public: true`. Both are unauthenticated endpoints. **Match.**
- **Input validation**: Dashboard requires `input` as `z.string().min(1)`. Go uses `validate:"required"` on the `Input` field. Both reject empty input. **Match.**
- **Session token**: Dashboard accepts optional `sessionToken` (camelCase query param). Go accepts optional `session_token` (snake_case). Both pass it through to Google when present. **Match** (naming convention difference is expected per migration guidelines).
- **Google Places API call**: Both POST to `https://places.googleapis.com/v1/places:autocomplete` with identical body structure: `{ input, includedPrimaryTypes: ["street_address", "premise", "subpremise"], sessionToken }`. Both set `X-Goog-Api-Key` header. **Match.**
- **Response mapping**: Both extract the same fields from the Google API response: `placeId` -> `id`, `text.text` -> `description`, `structuredFormat.mainText.text` -> `main_text`, `structuredFormat.secondaryText.text` -> `secondary_text`. **Match.**
- **Error handling**: Both return 500 internal errors when the API key is missing or Google API returns a non-OK status. **Match.**
- **Side effects**: None in either implementation. **Match.**
- **Idempotency**: GET endpoint, idempotent by nature. No idempotency keys needed. **Match.**

## Response shape difference (expected)

Dashboard returns `{ suggestions: [...] }`. Go returns the standard `List` resource `{ object: "list", data: [...], page_info: {} }`. This is the expected convention for the Go API and not a parity issue.

## Files reviewed

- Dashboard: `dashboard/apps/api/src/controllers/address-validation.ctrl.ts`, `dashboard/apps/api/src/services/address-validation.svc.ts`, `dashboard/packages/dtos/src/sections/address-validation.ts`
- Go gateway: `services/api-gateway/endpoints/address-validation/endpoint_autocomplete.go`, `service.go`, `presenter.go`
- Go core service: `services/core-service/internal/service/address_validation_service.go`
- Go gRPC handler: `services/core-service/internal/infrastructure/grpc/grpc_handler.go` (lines 1610-1633)
- Go resource: `services/api-gateway/pkg/resource/address_validation_resource.go`
