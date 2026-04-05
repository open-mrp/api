# DELETE /v1/core/registration-flows/{id} — Migration Verification

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Actor type, permission domain, action
- **DB queries**: Delete query with account scoping, cascade cleanup of related records
- **Error handling**: Not-found detection, SQL error mapping
- **Side effects**: Cleanup of payment term options, shipping term options, customer group references
- **Response shape**: Dashboard returns deleted object (200), Go returns empty (204)

## Issues found and fixed

### 1. Permission action mismatch (FIXED)
- **Dashboard**: Checks `update` permission on `account` domain
- **Go (before fix)**: Checked `delete` permission on `account` domain
- **Fix**: Changed `types.ActionDelete` → `types.ActionUpdate` in `registration_flow_service.go:261` to match Dashboard behavior
- **Note**: The Go Create and Update methods already correctly used `ActionUpdate`, consistent with the Dashboard

## No issues (confirmed parity)

- **Actor type check**: Both require internal actor
- **Account isolation**: Both scope the delete query to the target account ID
- **Cascade cleanup**: Go manually clears payment term options, shipping term options, and customer group references before deleting the flow — matches Prisma cascade behavior in Dashboard
- **Not-found handling**: Go explicitly checks rows affected and returns a not-found error; Dashboard relies on Prisma throwing on delete of non-existent record — functionally equivalent

## Accepted differences (by design)

- **Response shape**: Dashboard returns the deleted object with HTTP 200; Go returns 204 No Content with empty body. This follows Go API conventions for DELETE endpoints and is an intentional design choice.
