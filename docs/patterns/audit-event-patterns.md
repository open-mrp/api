# Audit event patterns

Services that mutate user-visible data can publish **audit events** through `shared/audit`. Events are written to the **transactional outbox** in the same DB transaction as the business mutation, then processed asynchronously by `platform-service` and exposed via the API as `audit_event` resources.

## Publishing from a service

1. After the mutation succeeds inside `WithTx`, call `audit.NewPublisher().Publish(ctx, outboxRepo, audit.EventData{...})`.
2. Use `txSvc.repos.NewOutboxRepo()` (or the transaction-scoped factory) so the outbox row commits with the mutation.
3. Set `ServiceName` to the service’s `domain.ServiceName` constant (e.g. `core-service`).
4. Use `constants.AuditAction*` for `Action` and `constants.ObjectType*` for `ResourceType`.
5. Set `ResourceID` to the affected resource’s **type ID** (never internal DB IDs).

`EventData` and the outbox payload are defined in `shared/audit/types.go` and `shared/audit/publisher.go`.

## `Metadata`

Use `EventData.Metadata` only for **extra context** that is not a normal attribute of the resource (reason codes, ticket IDs, bulk-job labels, correlation IDs). Do **not** duplicate **actor** or **target account** there: those come from request identity and are propagated with the outbox message; persisting them again in metadata is redundant.

## Field-level changes: `audit` struct tags

`audit.ComputeChanges(old, new, fieldNames...)` compares two values (usually two versions of the same domain struct) and returns `[]audit.FieldChange` with JSON fragments for `OldValue` / `NewValue`.

**Only fields with a non-empty `audit` struct tag participate.** Untagged exported fields (e.g. `ID`, `CreatedAt`, internal columns) are never included in the diff.

The **`Field` string** in each change is the tag value (first segment before `,` if you add options later), typically snake_case for clients.

**Convention:** On domain structs you diff for audit, tag every field that should appear in timelines:

```go
type AccountGroup struct {
    OwnerAccountID       string  `audit:"account_id"`
    Name                 string  `audit:"name"`
    Description          *string `audit:"description"`
    // ID, CreatedAt, ... left untagged → omitted from audit diffs
}
```

### Calling `ComputeChanges`

- **`ComputeChanges(old, new)`** — diffs all **exported** fields that have an `audit` tag. This is the usual call site after create/update/delete.
- **`ComputeChanges(old, new, "Name", ...)`** — optional subset, but each name must still refer to a field **with** an `audit` tag; untagged names are ignored.

### Relation to `json` tags

Do **not** rely on `json` tags for audit field names. Domain models are not guaranteed to match public API JSON. The `audit` tag keeps audit vocabulary explicit and independent of serialization.

## Actions and resource types

- **Actions:** `constants.AuditAction` (`shared/constants/audit_action.go`) — e.g. `AuditActionCreate`, `AuditActionUpdate`, `AuditActionDelete`.
- **Resource types:** `constants.ObjectType` — e.g. `ObjectTypeAccountGroup` for account groups.

## Idempotency and retries

Publish audit events inside the same idempotent phase as the mutation (e.g. `RecoveryPointStarted`). If the transaction rolls back, the outbox row rolls back with it. Cached idempotent success responses should not double-publish; the outbox write is part of the atomic phase that idempotency replays.
