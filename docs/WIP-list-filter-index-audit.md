# WIP — Keyset-list "selective filter" stall audit

**Status:** 🚧 Work in progress / triage doc. Findings below are from a static read of `*.sql` query files + `repository/*.go` builders cross-checked against the schema. **Every candidate index must be confirmed with `EXPLAIN` on production-like data before shipping.** Do not bulk-apply.

**Origin:** Surfaced from the `request_log` "searching by a user with no logs stalls forever" bug. This doc maps every other endpoint that can hit the same class of problem.

---

## The pattern

A paginated keyset/cursor list query shaped like:

```sql
WHERE  <scope_col> = ?            -- high-cardinality parent: account_id / owner_account_id / target_account_id
  AND  <selective filter>        -- col = ? or col IN (...)
ORDER BY <time_col> DESC, <id_col> DESC
LIMIT ?
```

When the selective filter matches **few or zero rows** and there is **no composite index leading `(scope_col, filter_col, time_col)`**, the planner falls back to the `(scope_col, time_col)` index, walks it newest-first, and tests the filter row-by-row. The `LIMIT` can only short-circuit once it has collected a full page of *matches* — so on a needle-in-haystack filter (or zero matches) it scans the **entire** scope partition. That is the stall.

**Why it looks intermittent:** filtering by a value that *has* rows is fast (page fills near the top of the time index); filtering by a value with no/few rows hangs. Same query, opposite outcomes — exactly the original `request_log` symptom.

**The fix:** a composite index `(scope_col, filter_col, time_col DESC, id_col DESC)`. The planner seeks straight to the matching slice in time order; zero matches returns instantly. MySQL 8.4 supports descending key parts (already used in the schema), so declare the time/id parts `DESC` to match the `ORDER BY` and avoid a filesort.

### Related correctness bug (different class)

Some queries expose a **translated** id (e.g. `COALESCE(au.id, x.actor_id) AS actor_id`, surfacing `account_user.id`) but **filter on the raw stored column**, or vice-versa. The filter then silently matches nothing. This doesn't stall (it returns fast-but-empty), so it hides. Tracked separately below.

---

## Important caveats before adding any index

These temper the raw agent findings — read before acting:

1. **Write amplification.** Several of these tables (`request_log`, `audit_event`, `transaction`, `sales_order`, `inventory_change_log`, `batch`) are high-insert. Every secondary index taxes every insert and consumes storage. **Index only filters that are both selective AND actually used by the UI** — confirm against dashboard usage, don't speculatively add the full matrix.
2. **Join-based filters can't be fixed by a single-table composite.** Many "filters" here are on a *joined* table (e.g. `sales_order.buyer_account_id` while paginating `invoice`, or `account_relation`-derived filters). A composite on the paginated table can't cover those — they need either denormalization of the filter column onto the base table, or a query rewrite (filter-first subquery). Flagged as **[JOIN]** below.
3. **`LIKE '%term%'` is non-sargable regardless of index.** Leading-wildcard search (`name`, `subject`, `path`) can't use a B-tree prefix. The fix there is `FULLTEXT … MATCH/AGAINST` or a prefix-anchored search, **not** a composite index. Flagged as **[LIKE]**.
4. **"Missing baseline `(scope, time)`" ≠ the stall.** A plain unfiltered list with only a single-column `scope` index does a filesort over the account's rows but still *terminates*; it's a latency issue, not the zero-match cliff. Lower priority — flagged **[BASELINE]**. The acute stall is specifically selective-filter + no composite.
5. **`IN (...)` with many values** uses the composite as a loose/multi-range scan and may still sort across ranges — bounded to matching rows, so it still fixes the stall, but EXPLAIN to confirm the plan.

---

## Already done

- ✅ **`request_log`** (`platform-service`): `actor_id` filter made sargable + Go-side `account_user.id → user_id` translation + new index `(target_account_id, actor_id, occurred_at DESC, id DESC)`. See `0002_request_log_actor_id_index.sql`.
- ⏳ **`request_log`** remaining selective filters (`error_code`, `normalized_route`) — identified, not yet shipped.

---

## Findings by service

> Severity = (filter selectivity) × (endpoint usage) × (table size/write-volume). High = user-facing needle-hunt on a big table. Confirm with EXPLAIN.

### platform-service

| Table | Filter | Type | Sev | Candidate index |
|---|---|---|---|---|
| `request_log` | `error_code` IN | stall | **High** | `(target_account_id, error_code, occurred_at DESC, id DESC)` |
| `request_log` | `normalized_route` IN | stall | **High** | `(target_account_id, normalized_route, occurred_at DESC, id DESC)` |
| `request_log` | `status_code` IN | stall | Med | `(target_account_id, status_code, occurred_at DESC, id DESC)` |
| `audit_event` | `resource_id` IN | stall | **High** | `(account_id, resource_id, occurred_at DESC, type_id DESC)` |
| `audit_event` | `action` IN | stall | **High** | `(account_id, action, occurred_at DESC, type_id DESC)` |
| `audit_event` | `resource_type` IN | stall | Med | `(account_id, resource_type, occurred_at DESC, type_id DESC)` |
| `audit_event` | `actor_id` IN | **correctness** | **High** | *No index — fix the translation bug (see below). Optionally `(account_id, actor_id, occurred_at DESC, type_id DESC)`.* |

Internal worker-queue tables (`message_outbox`, `message_inbox`, `idempotency_key`, `task_leases`) — **clean**, already correctly indexed for their fixed-predicate scans; no user-supplied selective filters.

### core-service (largest surface)

**High (user-facing needle-hunts on big tables):**

| Table | Filter(s) | Type | Candidate index |
|---|---|---|---|
| `inventory_change_log` | `item_id`, `action_type_code`, `responsible_user_id` | stall | `(account_id, <col>, created_at DESC, id DESC)` per filter |
| `sales_order` | `sales_order_status_code`, `buyer_account_id`, `sales_rep_id` | stall | `(owner_account_id, <col>, created_at DESC, id DESC)` |
| `sales_order` (purchase orders) | `sales_order_status_code`, `seller_account_id` | stall | `(owner_account_id, sales_order_type_code, <col>, created_at DESC, id DESC)` |
| `shipment` | `shipment_status_code` | stall | `(account_id, shipment_status_code, created_at DESC, id DESC)` |
| `delivery` | `delivery_status_code` | stall | `(account_id, delivery_status_code, created_at DESC, id DESC)` |
| `transaction` | `transaction_type_code`, `customer_account_id`, `transaction_method_code`, `adjustment_type_code` | stall | `(account_id, <col>, created_at DESC, id DESC)` |
| `account_relation` (customers/suppliers) | `account_status_code`, `account_group_id`, `default_sales_rep_id` (always alongside `account_relation_role_code`) | stall | `(owner_account_id, account_relation_role_code, <col>, created_at DESC, counterparty_account_id DESC)` |
| `account_price` | `recipient_account_id` | stall | `(owner_account_id, recipient_account_id, created_at DESC, id DESC)` |
| `batch` (scanning station) | `scanning_station_id` (mandatory) | stall | `(account_id, scanning_station_id, scanned_at DESC, id DESC)` |

**Medium / lower:**

| Table | Filter | Type | Note |
|---|---|---|---|
| `production_step` | `scanning_station_id` | stall | `(account_id, scanning_station_id, created_at DESC, id DESC)` |
| `account_user` | `status_code` (+ role_type via EXISTS) | stall/[JOIN] | `(account_id, status_code, created_at DESC, id DESC)`; role_type is a join-EXISTS |
| `edi_run` | `has_succeeded` | stall | `(account_id, has_succeeded, completed_at DESC, id DESC)` — failures are rare → selective |
| `invoice` | `buyer_account_id`, `account_group_id`, `sales_rep_id` | **[JOIN]** | filters live on joined `sales_order`/`account_relation`; composite on `invoice` won't help — needs denormalization or rewrite |
| `settlement`, `receiving_order`, `production_run` | transaction/invoice/supplier/item filters | **[JOIN]** | all via EXISTS/joined tables; `(account_id, created_at DESC, id DESC)` **[BASELINE]** only |
| `email_log` | `subject LIKE %q%` | **[LIKE]** | needs FULLTEXT/prefix, not a composite; add `(account_id, created_at DESC, id DESC)` **[BASELINE]** |
| `transaction_allocation` | type via joined `transaction` | **[JOIN]** | `(transaction_id, created_at DESC, id DESC)` helps the join anchor |
| `supplier_material` | (none beyond 2-col scope) | **[BASELINE]** | low — `(supplier_account_id, owner_account_id, created_at DESC, id DESC)` |

### auth-service

| Table | Filter | Type | Sev | Candidate index |
|---|---|---|---|---|
| `api_key` | `include_revoked` (→ `revoked_at IS NOT NULL`) | stall | **High** | `(owner_account_id, revoked_at, created_at DESC, id DESC)` |
| `api_key` | `include_expired` (→ `expires_at <= NOW()`) | stall | **High** | `(owner_account_id, expires_at, created_at DESC, id DESC)` |
| `api_key` | `name LIKE %q%` | **[LIKE]** | Med | non-sargable; add `(owner_account_id, created_at DESC, id DESC)` **[BASELINE]** for the unfiltered path |
| `registration_session` | `completed_at IS NULL` | stall | Low | few sessions per user → harmless now; `(user_id, completed_at, created_at DESC, id DESC)` |

No correctness bugs found.

### billing-service

| Table | Filter | Type | Sev | Candidate index |
|---|---|---|---|---|
| `account_plan` (public pricing list) | `name LIKE %q%` | **[LIKE]** | **High** | no `(created_at,id)` keyset index at all → add `(is_publicly_visible, created_at DESC, id DESC)`; for search use FULLTEXT `MATCH`, not LIKE |
| `token_pack_purchase` | `status = 'completed'` (aggregate, not paginated) | 2-index | Med | `(account_id, status)` — avoids the account-scan-then-filter on `SUM` |

No correctness bugs found. Outbox/queue queries clean.

### notification-service

**Clean.** No user-facing paginated list queries; only bounded `DELETE … LIMIT` batch jobs. No correctness bugs.

### agent-service (PostgreSQL — separate schema/migrations)

> ⚠️ These tables have **almost no indexes at all** — `agent_alert` has *zero*, `agent_run` has only one on `triggered_by_actor_id`. So every account-scoped list is a full sequential scan today, filter or not. Highest structural risk of any service. Postgres migration = `services/agent-service/db/migrations/00004_*.sql`.

| Table | Filter(s) | Sev | Candidate index (Postgres) |
|---|---|---|---|
| `agent_run` | `status_code`, `agent_definition_id` | **High** | `(account_id, status_code, created_at DESC, id DESC)`, `(account_id, agent_definition_id, created_at DESC, id DESC)` + baseline `(account_id, created_at DESC, id DESC)` |
| `agent_alert` | `severity_code`, `status_code` | **High** | `(account_id, severity_code, …)`, `(account_id, status_code, …)` + baseline |
| `agent_memory` | `category`, `entity_type` | Med | `(account_id, category, …)`, `(account_id, entity_type, …)` + baseline |
| `agent_definition` | `definition_type`, `trigger_type` | Low | small table; `(account_id, definition_type, …)`, `(account_id, trigger_type, …)` |
| `agent_token_usage` | `date` range (mandatory) | — | already covered by `UNIQUE (account_id, date)` |

---

## Correctness bugs (id translation) — verify before fixing

| # | Location | Bug | Confidence |
|---|---|---|---|
| 1 | `request_log` | exposed `account_user.id` vs filtered raw `actor_id` | ✅ Fixed |
| 2 | `audit_event` `ListAuditEvents{Forward,Backward}` (`audit_event.sql:60,107`) | Filters `ae.actor_id IN (…)` with **no** `account_user.id → user_id` translation, but the endpoint exposes `account_user.id` (same contract as request_log). Filtering audit by a user actor silently returns **empty**. | **High — likely real.** Mirror the request_log fix. |
| 3 | core-service `transaction` `responsible_user_id` (`transaction.sql`) | `responsible_user_id` stores `account_user.id`; a `ResolveResponsibleUserID` resolver exists, implying callers may pass either `user.id` or `account_user.id`. If the list filter doesn't route through the resolver, selective filters silently miss. | **Medium — needs tracing** of the endpoint→filter path. |
| 4 | core-service `production_run` `COALESCE(u.name, au.id, '') AS responsible_user_name` | Display-only: surfaces a UUID as a "name" when the user row is missing. Cosmetic, not a filter bug. | Low |

---

## Suggested validation & rollout

1. **Confirm UI usage.** For each High row, check the dashboard actually exposes that filter and how often it's used. Drop candidates nobody filters by.
2. **`EXPLAIN` each** on a production-sized account with (a) a matching value and (b) a zero-match value. Look for `type: index`/full scan + large `rows` on the zero-match case = confirmed stall.
3. **Fix the `audit_event` actor translation bug (#2)** alongside its indexes — it's the direct twin of what we just fixed.
4. **Batch the migrations per service**, descending key parts, and re-verify EXPLAIN flips to an index seek (`rows` ≈ page size) post-index.
5. **Watch write cost** on `request_log` / `audit_event` / `transaction` — add only the confirmed-used composites there.
6. **agent-service is structurally under-indexed** independent of this audit — its baseline `(account_id, created_at)` indexes are worth adding regardless.

---

*Generated as a triage starting point — not a green-light to apply all indexes. Validate with EXPLAIN + UI usage first.*
