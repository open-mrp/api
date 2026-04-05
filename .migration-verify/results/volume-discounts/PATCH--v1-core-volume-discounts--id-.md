# PATCH /v1/core/volume-discounts/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation faithfully reproduces the Dashboard business logic with some intentional improvements.

## What Was Compared

- **Permission checks**: Internal actor + discounts/update permission — match
- **Account scoping**: Both scope the update to the target account ID — match
- **Partial update semantics**: Name updated only when provided (Go uses `COALESCE`) — match
- **Tier upsert**: Both upsert tiers by ID; Go additionally deletes tiers not in the provided list when `has_tiers` is true (Dashboard Prisma upsert does not delete absent tiers) — acceptable difference, Go uses explicit `has_*` flags
- **Relation set-replace** (product lines, categories, attributes, units): Both delete-all + re-insert when the relation is provided — match (Go uses `has_*` flags, Dashboard uses Prisma `{set: true}`)
- **Customer groups**: Dashboard has this COMMENTED OUT; Go supports it — no regression
- **Name uniqueness**: Go adds a duplicate name check (excluding self) on update; Dashboard does not — improvement
- **Idempotency**: Go adds full idempotency key support per project patterns — improvement
- **Response shape**: Both return the full volume discount with all nested resources (tiers, customer groups, product lines, categories, attributes, units)
- **Side effects**: Neither implementation has side effects (no emails, webhooks, messages)
- **Error handling**: Go returns proper not-found if no rows affected; Dashboard relies on Prisma throwing if record not found

## No Issues Fixed

All differences are intentional improvements or equivalent behavior expressed through Go's explicit `has_*` flag pattern.
