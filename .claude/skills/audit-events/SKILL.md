---
name: audit-events
description: >-
  Publishing audit events from services via the transactional outbox, audit struct
  tags on domain models, and field-level diffs. Use when emitting or changing audit
  events, adding audit tags, or adding a public mutating POST/PATCH/PUT/DELETE route.
---

# Audit events

Mutations of user-visible data publish through `shared/audit` into the **transactional outbox** in the same DB transaction, then `platform-service` exposes them as `audit_event` resources. Human spec: `docs/patterns/audit-event-patterns.md`.

## Publish site

Inside `WithTx`, after the mutation succeeds:

1. `audit.NewPublisher().Publish(ctx, txSvc.repos.NewOutboxRepo(), audit.EventData{...})` — same tx-scoped factory.
2. `ServiceName` = the service's `domain.ServiceName`.
3. `Action` = `constants.AuditAction*`. `ResourceType` = `constants.ObjectType*`.
4. `ResourceID` = the resource **type ID**, never an internal DB id.
5. Same idempotent phase as the mutation (`RecoveryPointStarted`). Rollback rolls the outbox row back too. Cached success must not double-publish.

Public mutating gateway routes (`Public: true` + POST/PATCH/PUT/DELETE) must publish for persisted user-visible mutations. GET does not.

## Skip empty updates

`Publish` drops `AuditActionUpdate` when **both** `Changes` and `Metadata` are empty. Create/delete always publish.

- Normal `ComputeChanges(existing, updated)` — empty diff is a no-op. Fine.
- Updates with no field diff that must still record (password rotation) **must** set `Metadata` or they vanish.
- Computed changes: `audit.NewFieldChange(field, old, new)` and only append when the value changed.

`Metadata` is extra context (reason codes, ticket IDs, job labels) — **not** actor or target account (those travel with the outbox message).

## `audit` struct tags

Only fields with a non-empty `audit` tag participate in `ComputeChanges`. Tag value is the client-facing field name (snake_case). Do **not** reuse `json` tags — domain models are not the public API.

```go
type AccountGroup struct {
    OwnerAccountID string  `audit:"account_id"`
    Name           string  `audit:"name"`
    Description    *string `audit:"description"`
    // ID, CreatedAt untagged → omitted
}
```

`ComputeChanges(old, new)` diffs all tagged exported fields. An optional name subset still requires the tag.

## Exceptions (no resource audit)

- Address-validation validate action — no persisted domain state.
- Utils request-demo — marketing, not an API resource.
