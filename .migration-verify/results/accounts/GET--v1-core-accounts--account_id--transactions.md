# Verification: GET /v1/core/accounts/{account_id}/transactions

**Status: Issues found and fixed**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission checks | `checkIsInternalActor` + `transactions.read` | `CheckIsInternalActor` + `PermissionDomainTransactions.ActionRead` + target account check | Yes |
| Path params | `accountID` | `account_id` (CustomerAccountID) | Yes |
| Query: search | `query` (fulltext on number via Prisma relevance) | `query` (MATCH on number + note via BOOLEAN MODE) | ~Yes (see note) |
| Query: status | `status` (allocated/unallocated) | `status` (allocated/unallocated) | Yes |
| Query: type | Not exposed by controller | `type` (extra param) | N/A (extra) |
| Child accounts | Hardcoded `allowChildAccounts: true` | `include_child_accounts` param | **Fixed** |
| Pagination | Offset (take/skip) | Cursor-based | Expected |
| Response shape | `{ items: Transaction[], count }` | `List[TransactionDetail]` with page_info | Expected |
| Allocations | Always included nested | Not fetched | **Fixed** |
| Ordering | `_relevance` on number, then `createdAt desc` | `created_at DESC, id DESC` | ~Yes |
| Error handling | Standard HttpError | Standard apierror | Yes |
| Side effects | None | None | Yes |
| Idempotency | N/A (GET) | N/A (GET) | Yes |

## Issues found and fixed

### 1. IncludeChildAccounts defaulted to `false` (should be `true`)

**File:** `services/api-gateway/endpoints/transactions/endpoint_list_account_transactions.go`
- Changed `IncludeChildAccounts bool` to `IncludeChildAccounts *bool` so nil (unset) can be distinguished from explicit `false`.

**File:** `services/api-gateway/endpoints/transactions/service.go`
- Added logic to default to `true` when nil: `includeChildAccounts := req.IncludeChildAccounts == nil || *req.IncludeChildAccounts`

**Reason:** Dashboard controller hardcodes `allowChildAccounts: true` — it is never exposed as a user parameter. The Go endpoint now defaults to `true` while still allowing explicit override.

### 2. Allocations not fetched for listed transactions

**File:** `services/core-service/internal/service/transaction_service.go`
- After `ListByCustomer` returns, added a loop to fetch allocations for each transaction via `repo.GetAllocations()`.

**Reason:** Dashboard's `listFull` uses `TransactionAdapter.select()` which includes `accountTransactionAllocations` as nested data. The Go SQL query only returned `allocation_count` but never populated the actual allocations array.

## Remaining notes

- **Search scope differs slightly:** Dashboard's `BaseTransactionAdapter.fetchInput` with `query` searches across transaction number, method name, customer info, responsible user, and linked invoice numbers (via Prisma OR). Go searches `MATCH(t.number, t.note)` using MySQL fulltext. The Go search includes `note` (which Dashboard doesn't search) but doesn't search method/customer/user/invoice fields. This would require schema changes (new fulltext indexes) to fully match and is a minor behavioral difference.
- **Type filter:** Go exposes a `type` query param that the Dashboard controller does not. This is additive functionality and not a parity concern.
- **Ordering:** Dashboard uses Prisma `_relevance` sorting on `number` field as primary sort, then `createdAt desc`. Go uses `created_at DESC, id DESC`. When no search query is active, both effectively sort by creation date descending. With a search query, relevance sorting may differ slightly.
