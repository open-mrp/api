# GET /v1/core/product-types — Verification Result

## Status: Parity Confirmed (No Issues)

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|---|---|---|---|
| **Endpoint registered** | No (repo exists, no controller/route/service method) | Yes, fully implemented | N/A |
| **Permission check** | N/A (not exposed) | Internal actor + `PermissionDomainProductTypes` / `ActionRead` | Consistent with other product endpoints |
| **Account scoping** | Not scoped (global) | Not scoped (global) | Yes |
| **Search/filter** | Prisma fulltext search on `name` | SQL LIKE on `name` | Equivalent intent |
| **Pagination** | Offset-based (`skip`/`take`) | Cursor-based (`cursor`/`limit`) | Intentional Go API upgrade |
| **Ordering** | By relevance (fulltext) | `created_at DESC, id DESC` | Acceptable — Go uses standard cursor-compatible ordering |
| **Response fields** | `name`, `code`, `createdAt`, `updatedAt` (no `id`) | `id`, `object`, `name`, `code`, `created_at`, `updated_at` | Go adds `id`/`object` per API resource conventions |
| **Response shape** | `{ items, count }` | `{ object, data, page_info }` | Intentional Go API list conventions |
| **Idempotency** | N/A (GET) | N/A (GET) | Correct |

## Notes

- The Dashboard never actually registered a public endpoint for listing product types — the service class is empty and no controller or route exists. Only the repository and adapter were implemented.
- The Go implementation is therefore the canonical implementation, built from the repository logic that existed in the Dashboard.
- All architectural differences (cursor pagination, LIKE search, response shape with `id`/`object`) are intentional Go API conventions, not parity gaps.
- No fixes required.
