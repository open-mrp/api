# Verification: GET /v1/core/account-prices

**Status: Parity confirmed** — no fixes required.

## What was compared

- **Permission checks**: `CheckIsAssignedActor` + `discounts:read` for internal actors; customer actors scoped to own account
- **Query/search logic**: Search by recipient account name and external number via `account_relation`
- **Recipient account filtering**: Optional `recipientAccountID` filter; forced for customer actors
- **Response shape**: Sub-resources (recipient_account, product_line, rate with units, categories, attributes) with `object` type fields
- **Error handling**: Not-found, validation, and auth errors all map correctly
- **SQL joins and filters**: owner_account_id scoping, cursor-based pagination, LIKE search

## Minor behavioral differences (intentional improvements, not parity gaps)

1. **Search + recipientAccountID are independent in Go, mutually exclusive in Dashboard**
   - Dashboard adapter's `fetchInput()` uses if/else if: when `recipientAccountID` is set, `query` search is ignored entirely.
   - Go SQL applies both as independent AND clauses, allowing search within a filtered recipient.
   - Effect: Customer actors (who always have recipientAccountID forced) can search their own prices in Go but could not in Dashboard. This is strictly better behavior.

2. **No explicit else guard for unknown actor types**
   - Dashboard throws `HttpError.badRequest('Invalid actor type')` if actor is neither internal nor customer.
   - Go omits this guard because `CheckIsAssignedActor()` already restricts to valid assigned actor types, making the else branch unreachable.

3. **Cursor-based pagination replaces offset pagination**
   - Dashboard uses `take`/`skip` + `count` (offset pagination).
   - Go uses cursor-based pagination with `page_info`. This is an intentional architectural upgrade consistent with all other Go endpoints.

## Issues found and fixed

None.

## Remaining concerns

None.
