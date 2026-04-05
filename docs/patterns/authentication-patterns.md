# Authentication & Authorization Patterns

## Identity Model

Every authenticated HTTP request flowing through the API gateway carries an **Identity** that is extracted from the request context. The identity contains:

- **Type**: The kind of actor making the request (`user`, `api_key`, `agent`)
- **Actor**: The authenticated entity with its ID, permissions map, and role type code
- **TargetAccountID**: The account being accessed (from `Augno-Account` header)
- **AccountMode**: Whether the account is in `prod` or `test` mode

Identity is propagated from the API gateway to backend microservices via gRPC metadata using `contracts.SetIdentityInMetadata()`.

## Extracting Identity

In API gateway endpoint service methods, use `httptransport.GetIdentity(ctx)`:

```go
identity, apiErr := httptransport.GetIdentity(ctx)
if apiErr != nil {
    return nil, apiErr
}
```

## Authorization Checks

The `auth-service/pkg/types` package provides check functions that validate identity properties and return `*apierror.APIError` on failure:

| Function | What it checks |
|----------|---------------|
| `CheckIsAuthenticated(identity)` | Identity exists and is authenticated |
| `CheckIsUser(identity)` | Authenticated + is a user (not API key or agent) |
| `CheckIsAPIKey(identity)` | Authenticated + is an API key |
| `CheckIsAgent(identity)` | Authenticated + is an agent |
| `CheckIsInternalActor(identity)` | Authenticated + is an internal user for the account |
| `CheckIsAdmin(identity)` | Authenticated + has admin role |
| `CheckHasPermission(identity, domain, action)` | Has specific permission (admins bypass) |
| `CheckNotSandboxMode(identity)` | Account is not in sandbox mode |

## Standard Authorization Pattern

Define a package-level helper that extracts identity and runs the appropriate check, then call it at the start of every service method:

```go
func requireInternalActor(ctx context.Context) *apierror.APIError {
    identity, apiErr := httptransport.GetIdentity(ctx)
    if apiErr != nil {
        return apiErr
    }
    return identitytypes.CheckIsInternalActor(identity)
}

func (m *mySvcImpl) MyEndpoint(ctx context.Context, req *MyRequest) (*MyResponse, *apierror.APIError) {
    if apiErr := requireInternalActor(ctx); apiErr != nil {
        return nil, apiErr
    }
    // ... business logic
}
```

Every service method that handles a request MUST begin with an authorization check. Do not rely on middleware alone; the service layer is the authoritative enforcement point.

## Actor Types

IdentityActorType represents **how** the caller authenticated:

| IdentityActorType | Description | Common Use |
|-------------------|-------------|------------|
| `user` | A human user authenticated via session/JWT | Dashboard UI, portals |
| `api_key` | A programmatic access key | External integrations, public API |
| `agent` | An AI agent executing on behalf of the account | Automated workflows |
| `unauthenticated` | No authentication provided | Public endpoints only |

## Relation Types

IdentityRelationType represents the caller's **relationship** to the account:

| IdentityRelationType | Description | Common Use |
|----------------------|-------------|------------|
| `internal` | A member of the account's internal team | Dashboard UI, most management endpoints |
| `customer` | An external customer of the account | Customer portal, order tracking |
| `supplier` | A supplier/vendor of the account | Supplier portal |
| `unassigned` | No relation type assigned yet | Onboarding, pending role assignment |

## Permission Model

Permissions follow a `domain:action` pattern (e.g., `products:read`, `sales_orders:create`). Domains use snake_case (e.g., `sales_orders`, `inventory`, `agents`) and actions are `create`, `read`, `update`, or `delete`. Admin users bypass permission checks entirely. Non-admin users must have the specific permission in their `Actor.Permissions` map.
