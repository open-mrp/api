# Authorization Check Patterns for Cross-Account Access

This document covers how service-layer authorization should work when an API key or user targets an account other than their own — specifically when a merchant targets a customer or supplier account.

For general authentication and identity model background, see [authentication-patterns.md](authentication-patterns.md).

## Cross-Account Identity Model

When a request targets the caller's own account, the **actor account** and **target account** are the same. Cross-account access occurs when the `Augno-Account` header specifies a different account — for example, a merchant's API key operating on a customer's account.

The identity carries two pieces of information for this:

- **Actor account** (`identity.ActorAccountID()`): The account the caller belongs to (the API key's owner account).
- **Target account** (`identity.TargetAccountID`): The account being accessed (from the `Augno-Account` header).
- **Target relation type** (`identity.TargetRelationType`): When the target differs from the actor, this indicates the relationship — `"customer"` or `"supplier"`. It is `nil` when the actor targets their own account.

### How owner-side auth works

When a merchant API key sets the `Augno-Account` header to a customer's account ID, the auth system:

1. Finds the `account_relation` where the merchant is the owner and the customer is the counterparty.
2. Resolves the API key's permissions from the **actor** (merchant) account — not the target.
3. Builds an identity with actor type `internal`, the merchant's permissions, and `TargetRelationType = "customer"`.

This means the identity looks like a normal internal actor but with the target account pointing at the customer. Service-layer code must use the correct permission domain based on the target relation type.

## Identity Check Reference

### Boolean helpers (for branching logic)

| Method | What it checks | Use for |
|--------|---------------|---------|
| `IsInternalActor()` | Actor type is `internal` | Permission check branching — works for both same-account and cross-account |
| `IsInternalUser()` | Actor type is `internal` AND actor account == target account | Logic that should only run when targeting the actor's own account |
| `IsExternalTarget()` | Actor account != target account | Gating access checks (`CheckReadAccess` / `CheckEditAccess`) |
| `IsTargetCustomerAccount()` | `TargetRelationType` is `"customer"` | Permission domain routing |
| `IsTargetSupplierAccount()` | `TargetRelationType` is `"supplier"` | Permission domain routing |
| `IsTargetAccountSet()` | `TargetAccountID` is non-nil and non-empty | Verifying the account header was provided |

### Error-returning checks (for gating)

| Method | What it checks | Caution |
|--------|---------------|---------|
| `CheckIsAssignedActor()` | Actor is `internal`, `customer`, or `supplier` | Use for endpoints accessible to all actor types |
| `CheckIsInternalActor()` | Calls `IsInternalUser()` internally | **Fails for cross-account access.** Only use for endpoints that must be restricted to same-account internal users. |

## Standard Authorization Flow

Every service method that handles a request must follow this flow:

```go
func (s *mySvcImpl) ListThings(ctx context.Context, params domain.ListThingsParams) (*domain.ListThingsResult, *apierror.APIError) {
    ctx, span := tracer.Start(ctx, "service.thing.list")
    defer span.End()

    // 1. Extract identity
    identity, ok := appctx.GetIdentityFromContext(ctx)
    if !ok || identity == nil {
        return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
    }

    // 2. Gate on actor type
    if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
        return nil, tracing.Trace(span, apiErr)
    }

    // 3. Check permission (route by target relation type)
    if apiErr := checkThingReadPermission(identity); apiErr != nil {
        return nil, tracing.Trace(span, apiErr)
    }

    // 4. Verify target account is set (use the method, not TargetAccountID == nil)
    if !identity.IsTargetAccountSet() {
        return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
    }

    // 5. Check cross-account access
    if identity.IsExternalTarget() {
        meds := s.mediators()
        if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), *identity.TargetAccountID); apiErr != nil {
            return nil, tracing.Trace(span, apiErr)
        }
    }

    // 6. Scope to target account
    params.AccountID = *identity.TargetAccountID

    return s.repos.NewThingRepo().List(ctx, params)
}
```

For write operations, use `meds.EditAccess.CheckEditAccess(...)` instead of `CheckReadAccess`.

## Permission Domain Resolution

When an internal actor targets their own account, use the resource's own permission domain (e.g., `addresses:create`). When targeting a customer or supplier account, use the relationship domain instead.

Define a pair of permission helpers per service file:

```go
// checkThingReadPermission checks the appropriate read permission based on the target context.
func checkThingReadPermission(identity *types.Identity) *apierror.APIError {
    if !identity.IsInternalActor() {
        return nil // Non-internal actors are gated by access checks, not permissions
    }
    if identity.IsTargetCustomerAccount() {
        return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
    }
    if identity.IsTargetSupplierAccount() {
        return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
    }
    return identity.CheckHasPermission(types.PermissionDomainThings, types.ActionRead)
}

// checkThingWritePermission checks the appropriate write permission based on the target context.
func checkThingWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
    if !identity.IsInternalActor() {
        return nil
    }
    if identity.IsTargetCustomerAccount() {
        return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
    }
    if identity.IsTargetSupplierAccount() {
        return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
    }
    return identity.CheckHasPermission(types.PermissionDomainThings, action)
}
```

**Why non-internal actors return nil:** Customer and supplier actors don't carry role-based permissions. Their access is controlled by the `CheckReadAccess` / `CheckEditAccess` mediators in step 5, which verify the account relation exists and that the target account allows external edits.

## Common Mistakes

### Using `TargetAccountID == nil` directly

```go
// Bad — doesn't check for empty string
if identity.TargetAccountID == nil {

// Good
if !identity.IsTargetAccountSet() {
```

### Using `IsInternalUser()` for permission branching

```go
// Bad — fails when a merchant API key targets a customer account,
// because IsInternalUser() requires actor account == target account
if identity.IsInternalUser() {
    if apiErr := identity.CheckHasPermission(...); apiErr != nil {
        return apiErr
    }
}

// Good — works for both same-account and cross-account internal actors
if identity.IsInternalActor() {
    if apiErr := identity.CheckHasPermission(...); apiErr != nil {
        return apiErr
    }
}
```

### Missing external target access check

Every method that accesses account-scoped data must check external target access when the actor and target accounts differ:

```go
// Bad — no cross-account access check
params.AccountID = *identity.TargetAccountID
return s.repos.NewThingRepo().List(ctx, params)

// Good
if identity.IsExternalTarget() {
    meds := s.mediators()
    if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), *identity.TargetAccountID); apiErr != nil {
        return nil, tracing.Trace(span, apiErr)
    }
}
params.AccountID = *identity.TargetAccountID
```

### Not routing permission domain by target relation type

```go
// Bad — always checks the resource domain, even when targeting a customer account
if identity.IsInternalActor() {
    return identity.CheckHasPermission(types.PermissionDomainAddresses, types.ActionRead)
}

// Good — routes to the correct domain
if identity.IsTargetCustomerAccount() {
    return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
}
if identity.IsTargetSupplierAccount() {
    return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
}
return identity.CheckHasPermission(types.PermissionDomainAddresses, types.ActionRead)
```

## Reference Implementation

See `services/core-service/internal/service/address_service.go` for the canonical implementation of all patterns described here.

Other services with correct external target handling:
- `services/core-service/internal/service/carrier_service.go`
- `services/core-service/internal/service/service_level_service.go`
- `services/core-service/internal/service/unit_group_service.go`
- `services/core-service/internal/service/account_user_service.go`

## Key Source Files

- `services/auth-service/pkg/types/identity_model.go` — Identity struct, boolean helpers (`IsInternalActor`, `IsExternalTarget`, etc.)
- `services/auth-service/pkg/types/identity_utils.go` — Error-returning checks (`CheckIsAssignedActor`, `CheckHasPermission`, etc.)
- `services/auth-service/pkg/types/permissions.go` — Permission domain constants
- `services/core-service/internal/mediator/read_access_mediator.go` — `CheckReadAccess` implementation
- `services/core-service/internal/mediator/edit_access_mediator.go` — `CheckEditAccess` implementation
