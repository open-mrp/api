# Audit event patterns

Services that mutate user-visible data can publish **audit events** through `shared/audit`. Events are written to the **transactional outbox** in the same DB transaction as the business mutation, then processed asynchronously by `platform-service` and exposed via the API as `audit_event` resources.

## Publishing from a service

1. After the mutation succeeds inside `WithTx`, call `audit.NewPublisher().Publish(ctx, outboxRepo, audit.EventData{...})`.
2. Use `txSvc.repos.NewOutboxRepo()` (or the transaction-scoped factory) so the outbox row commits with the mutation.
3. Set `ServiceName` to the service’s `domain.ServiceName` constant (e.g. `core-service`).
4. Use `constants.AuditAction*` for `Action` and `constants.ObjectType*` for `ResourceType`.
5. Set `ResourceID` to the affected resource’s **type ID** (never internal DB IDs).

`EventData` and the outbox payload are defined in `shared/audit/types.go` and `shared/audit/publisher.go`.

## No-op updates are skipped

`Publish` silently drops an `AuditActionUpdate` event when **both** `Changes` and `Metadata` are empty. This is how same-value PATCH requests (which pass the `RejectEmptyPatchBody` guard but change nothing) avoid producing empty `changes: []` audit events. Create and delete events always publish.

Consequences for call sites:

- The normal `ComputeChanges(existing, updated)` flow needs no special handling — an empty diff means the event is skipped.
- An update event that intentionally records no field diff (e.g. password rotation, where recording values would leak secrets) **must** set `Metadata` (e.g. `{"password_rotated": true}`) or it will be silently dropped.
- For computed changes not derivable from two struct snapshots (e.g. inventory quantity corrections), build them with `audit.NewFieldChange(field, old, new)` and only append when the value actually changed.

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

## Public mutating HTTP endpoints (`api-gateway`)

For `api-gateway` routes with `Public: true` and `POST` / `PATCH` / `PUT` / `DELETE`, the owning microservice must publish audit for **persisted** user-visible mutations in the same transaction as the mutation (same rules as above). Read-only `GET` handlers do not publish.

**Checklist when adding or changing a public mutating route:** confirm the downstream service method wraps the mutation plus `audit.NewPublisher().Publish(..., txSvc.repos.NewOutboxRepo(), ...)` in one `WithTx` / idempotent phase.

**Regenerate the current set of public mutating endpoint files** (from the repository root):

```bash
comm -12 \
  <(grep -rl 'Public:[[:space:]]*true' services/api-gateway/endpoints \
      --include='endpoint_*.go' | sort) \
  <(grep -rlE 'Method:[[:space:]]*http\.Method(Post|Patch|Put|Delete)' \
      services/api-gateway/endpoints --include='endpoint_*.go' | sort)
```

As of the latest review, those routes are grouped under api-gateway as: `account-groups`, `account-users`, `addresses`, `address-validation`, `api-keys`, `carriers`, `customers`, `item-categories`, `locations`, `payment-terms`, `product-lines`, `properties`, `roles`, `sandboxes`, `scanning-stations`, `service-levels`, `shipping-terms`, `unit-groups`, `units`, and `utils`. Almost all call into **core-service**; **api-keys** (create, revoke, rotate) call **auth-service**. Pay special attention to **junction and linkage mutations** (properties on categories, category unit groups, merges, etc.) so they do not skip audit while “primary” CRUD paths already publish.

**Exceptions (no resource audit expected for the API resource model):**

- `address-validation` validate action: does not persist domain state in core.
- `utils` request-demo: marketing / lead capture, not a persisted API resource in the usual sense (product choice if a synthetic event is ever required).

**Conformance spot-check (publish sites):** callers use each service’s `domain.ServiceName`; `ResourceID` is always a type-prefixed ID (e.g. account subscription updates on the account row use the account type ID, not internal keys); `EventData.Metadata` is rarely needed and should not repeat actor or target account—identity is already on the AMQP payload in `shared/audit/publisher.go`.
