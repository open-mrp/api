---
name: authentication-authorization
description: >-
  Identity model, service-layer permission checks, and cross-account authorization
  (merchant targeting a customer or supplier). Use when touching identity, permissions,
  actor checks, OpenMRP-Account, TargetRelationType, CheckHasPermission, CheckReadAccess,
  CheckEditAccess, or any tenant-scoped service method.
---

# Authentication and authorization

Human specs: `docs/patterns/authentication-patterns.md`, `docs/patterns/authorization-check-patterns.md`. Canonical implementation: `services/core-service/internal/service/address_service.go`.

The **service** is the authoritative enforcement point. Gateway middleware authenticates; it does not substitute for permission checks. Gateway `endpoints/*/service.go` files are thin RPC translators and do not duplicate checks.

`Public: true` on an endpoint means "in the public OpenAPI spec." It does **not** disable auth.

## Identity

Extract in microservices with `appctx.GetIdentityFromContext`. Missing identity is an invariant violation. In the gateway (rare, local branching only) use `httptransport.GetIdentity`.

| Check | Meaning |
|---|---|
| `CheckIsAuthenticated()` | user / api_key / agent |
| `CheckIsInternalActor()` | **same-account** internal only — **fails for cross-account** |
| `CheckIsAssignedActor()` | internal, customer, or supplier |
| `CheckHasPermission(domain, action)` | `domain:action`; admins bypass |
| `IsInternalActor()` | internal relation — works same-account **and** cross-account |
| `IsInternalUser()` | internal **and** actor account == target |
| `IsExternalTarget()` | actor account != target |
| `IsTargetCustomerAccount()` / `IsTargetSupplierAccount()` | permission-domain routing |
| `IsTargetAccountSet()` | header present (do **not** use `TargetAccountID == nil`) |

Permissions are `domain:action` (`products:read`). Domains snake_case; actions `create|read|update|delete`.

## Every tenant-aware service method

```
1. Extract identity
2. Gate actor type (usually CheckIsAssignedActor)
3. Permission — route by target relation (helpers below)
4. IsTargetAccountSet — else authentication error
5. If IsExternalTarget: CheckReadAccess / CheckEditAccess
6. Scope queries to *identity.TargetAccountID
```

Anonymous / alternate-auth RPCs must stay on the allowlist in `authentication-patterns.md` (address validation, GetAccountBySlug, RequestDemo, Stripe webhooks).

## Cross-account permission domain

Internal actor targeting **own** account → the resource's domain (`addresses:create`). Targeting a **customer** / **supplier** account → `customers` / `suppliers` domain instead.

Non-internal actors return nil from the permission helper — they have no role permissions; `CheckReadAccess` / `CheckEditAccess` is the gate.

```go
func checkThingReadPermission(identity *types.Identity) *apierror.APIError {
    if !identity.IsInternalActor() {
        return nil
    }
    if identity.IsTargetCustomerAccount() {
        return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
    }
    if identity.IsTargetSupplierAccount() {
        return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
    }
    return identity.CheckHasPermission(types.PermissionDomainThings, types.ActionRead)
}
```

## Mistakes that ship

- `CheckIsInternalActor()` on an endpoint that merchants must call against a customer account
- `IsInternalUser()` for permission branching (use `IsInternalActor()`)
- Skipping `CheckReadAccess` / `CheckEditAccess` when `IsExternalTarget()`
- Always checking the resource domain, even when `TargetRelationType` is customer/supplier
- Comparing `TargetAccountID == nil` instead of `IsTargetAccountSet()`
