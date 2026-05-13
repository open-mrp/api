# Authentication & Authorization Patterns

## Identity Model

Every HTTP request processed by the API gateway (except `/healthz`) passes through authentication middleware that resolves an **Identity** and stores it on the request context. The identity contains:

- **Type**: The kind of actor (`user`, `api_key`, `agent`, or **`unauthenticated`** when no credential was supplied)
- **Actor**: The authenticated entity with its ID, permissions map, and role type code (unset for `unauthenticated`)
- **TargetAccountID**: The account being accessed (from `Augno-Account` header when applicable)
- **AccountMode**: Whether the account is in `prod` or `test` mode

Identity is propagated from the API gateway to backend microservices via gRPC metadata using `contracts.SetIdentityInMetadata()` (see also `rpc.WithIdentity`).

## Layered Enforcement

Authorization is enforced in layers:

1. **API gateway `AuthMiddleware`** validates credentials via auth-service (`ValidateCredential`). It always attaches an identity to the context (including **`unauthenticated`** when there is no token). It does **not** substitute for permission checks on tenant data.
2. **Microservice domain services** are the **authoritative** enforcement point for tenant operations: they load identity from context with `appctx.GetIdentityFromContext(ctx)` and call `identity.Check*` / `CheckHasPermission` as appropriate. Do not rely on middleware alone.

Most gateway `endpoints/*/service.go` implementations are thin translators (HTTP → protobuf → `grpcutil.CallRPC`) and **do not** duplicate identity checks; enforcement happens in core-service, auth-service, agent-service, etc.

Use **`httptransport.GetIdentity(ctx)`** in the gateway only when the gateway handler itself needs identity (for example gateway-only branching before calling downstream RPCs). Missing identity there returns an authentication error; unauthenticated callers still have an identity object with `Type == unauthenticated`.

## Public OpenAPI (`APIEndpoint.Public`)

On gateway endpoint specs, **`Public: true`** means the operation is included in the generated **`specs/public_openapi_spec.json`** (consumer-facing OpenAPI). **`Public: false`** keeps the operation in the internal OpenAPI spec only.

This flag does **not** disable auth middleware or imply anonymous HTTP access. Logging middleware uses the same flag only to mark whether a route is a “public endpoint” on stored request logs—not to skip authentication.

Operations that intentionally allow **`unauthenticated`** callers (no JWT/API key) must still document that behavior in the owning microservice (see below). Prefer **`identity.CheckIsAuthenticated()`** (or stricter checks) on every tenant-sensitive RPC unless the handler is on the intentional allowlist.

## Extracting Identity

**Microservices (authoritative checks):**

```go
identity, ok := appctx.GetIdentityFromContext(ctx)
if !ok || identity == nil {
    return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
}
if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
    return nil, tracing.Trace(span, apiErr)
}
```

**API gateway (when local inspection is required):**

```go
identity, apiErr := httptransport.GetIdentity(ctx)
if apiErr != nil {
    return nil, apiErr
}
if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
    return nil, apiErr
}
```

## Authorization Checks

The `auth-service/pkg/types` package defines **`(*Identity)` methods** that validate identity properties and return `*apierror.APIError` on failure:

| Method | What it checks |
|--------|----------------|
| `(*Identity).CheckIsAuthenticated()` | Authenticated actor (`user`, `api_key`, or `agent`) |
| `(*Identity).CheckIsUser()` | Authenticated + user |
| `(*Identity).CheckIsAPIKey()` | Authenticated + API key |
| `(*Identity).CheckIsAgent()` | Authenticated + agent |
| `(*Identity).CheckIsInternalActor()` | Authenticated + internal relation + target account set |
| `(*Identity).CheckIsAssignedActor()` | Authenticated + internal, customer, or supplier relation |
| `(*Identity).CheckIsAdmin()` | Authenticated + admin role |
| `(*Identity).CheckHasPermission(domain, action)` | Specific permission (admins bypass) |
| `(*Identity).CheckNotSandboxMode()` | Account is not in sandbox mode |

## Standard Authorization Pattern (Microservice)

Define a helper if helpful, then call it at the start of every tenant-aware service method:

```go
func (s *mySvcImpl) MyEndpoint(ctx context.Context, req *MyRequest) (*MyResponse, *apierror.APIError) {
	ctx, span := mySvcTracer.Start(ctx, "service.my.endpoint")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainWidgets, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// ... business logic
}
```

Every service method that handles tenant data MUST begin with an authorization check unless it is intentionally anonymous or uses alternate verification (webhook signatures, etc.).

## Intentionally Anonymous or Alternate-Auth RPCs

These behaviors deliberately omit standard identity checks or use non-JWT verification. Keep this list accurate when adding endpoints.

| Area | Behavior |
|------|-----------|
| **Address validation** (`core-service` address validation service: autocomplete, place details, validate) | No tenant identity gate; proxies to Google APIs. Exposed with **`Public: true`** on gateway for checkout-style flows. |
| **Account by slug** (`GetAccountBySlug`) | Minimal public account lookup without authentication; gateway endpoint remains internal OpenAPI only unless product opts into **`Public: true`**. |
| **Request demo** (`utilsSvc.RequestDemo`) | Logs demo requests without **`CheckIsAuthenticated`**; included in public OpenAPI (**`Public: true`**). |
| **Stripe webhooks** | **`Public: false`** in OpenAPI; verified via Stripe signature in the billing path, not JWT. |

For contrast, **`SubmitFeedback`** requires **`CheckIsAssignedActor`** in core-service—it is **not** anonymous.

Regenerate the machine-readable inventory of all **`Public: true`** gateway routes (tools module):

```bash
cd tools && go run ./publicendpoints --out ../specs/public_route_inventory.tsv
```

### Auditing `Public: true` routes

1. Start from [`specs/public_route_inventory.tsv`](../../specs/public_route_inventory.tsv) (columns: HTTP method, route template, gateway endpoint file).
2. Follow `ServiceHandler` in that file to the gateway `endpoints/*/service.go` RPC wrapper, then to the **microservice** gRPC handler and domain service implementation.
3. Confirm the domain service performs `appctx.GetIdentityFromContext` and the appropriate `identity.Check*` chain **unless** the RPC is listed in **Intentionally anonymous or alternate-auth RPCs** above.

A manual spot audit of representative routes (core tenant CRUD, platform audit/request logs, address validation, utils) shows downstream checks present or intentionally omitted only on the allowlist. When adding new **`Public: true`** endpoints, extend this verification (and the allowlist when anonymity is deliberate).

## Actor Types

IdentityActorType represents **how** the caller authenticated:

| IdentityActorType | Description | Common Use |
|-------------------|-------------|------------|
| `user` | Human via session/JWT | Dashboard UI, portals |
| `api_key` | Programmatic key | External integrations, public API |
| `agent` | AI agent | Automated workflows |
| `unauthenticated` | No credential supplied | Allowed only for handlers explicitly designed for it |

## Relation Types

IdentityRelationType represents the caller's **relationship** to the account:

| IdentityRelationType | Description | Common Use |
|----------------------|-------------|------------|
| `internal` | Internal team member | Dashboard UI, most management endpoints |
| `customer` | External customer | Customer portal, order tracking |
| `supplier` | Supplier/vendor | Supplier portal |
| `unassigned` | Not yet assigned | Onboarding, pending role assignment |

## Permission Model

Permissions follow a `domain:action` pattern (e.g., `products:read`, `sales_orders:create`). Domains use snake_case (e.g., `sales_orders`, `inventory`, `agents`) and actions are `create`, `read`, `update`, or `delete`. Admin users bypass permission checks entirely. Non-admin users must have the specific permission in their `Actor.Permissions` map.
