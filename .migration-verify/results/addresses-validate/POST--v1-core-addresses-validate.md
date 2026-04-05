# POST /v1/core/addresses/validate — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Request validation**: Go uses `validate:"required"` on all fields except optional `address_line_2`; Dashboard uses Zod `z.string()` with optional `addressLine2`. Both enforce the same required fields. Go is slightly stricter (rejects empty strings), which is correct.
- **Authentication/Authorization**: Both require no auth (`AuthOptions.None` in Dashboard, `Public: true` in Go). No permission checks.
- **Business logic**: Identical flow — builds address lines array, converts country to region code, calls Google Address Validation API, parses verdict, builds validation messages, applies line2 preservation logic, falls back to user input for missing fields.
- **Google API call**: Same URL (`addressvalidation.googleapis.com/v1:validateAddress`), same request body shape, same query param auth (`key=`).
- **Validity check**: Both use `addressComplete && granularity != "OTHER" && granularity != "ROUTE"`.
- **Validation messages**: Same 3 messages for unconfirmed/inferred/replaced components, same wording.
- **Line2 preservation logic**: Same two-stage approach — first check if user's line2 appears at end of line1, then fall back to regex extraction of unit designators. Same trailing separator cleanup.
- **Unit designator regex**: Same pattern matching Suite, Ste, Apt, Apartment, Unit, Bldg, Building, Fl, Floor, Rm, Room, Dept, Department, #.
- **Region code mapping**: Identical mappings (US, CA, GB, AU, DE, FR, MX) with same fallback (first 2 chars uppercase).
- **Response shape**: Go adds `object` fields per API conventions and uses snake_case. Go always returns `validation_messages` as an array (empty `[]` vs Dashboard's `undefined`), which is correct per Go API conventions (no omitempty).
- **Idempotency**: Neither uses idempotency keys — correct since this is a stateless, non-mutating endpoint.
- **Side effects**: None in either implementation.
- **Error handling**: Both return internal server errors for Google API failures.

## Issues found and fixed

1. **Case-insensitive regex flag missing** (`address_validation_service.go:332`): The unit designator regex (`unitDesignatorRegex`) was compiled without case-insensitive matching. The Dashboard regex uses the `i` flag (`/pattern/i`), meaning it matches "suite 100", "SUITE 100", etc. The Go regex was case-sensitive, so it would only match exact casing like "Suite 100". Fixed by adding `(?i)` prefix to the regex pattern.

## Remaining concerns

None. The implementations are at parity after the regex fix.
