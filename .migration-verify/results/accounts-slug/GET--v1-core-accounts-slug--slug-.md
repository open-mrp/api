# Verification: `GET /v1/core/accounts/slug/{slug}`

## Result: Parity confirmed — no code changes needed

The Go endpoint correctly implements the core business logic of the Dashboard `AccountSvc.findBySlug` method. All differences are intentional Go API convention changes, not bugs.

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Authentication** | `AuthOptions.None` (public) | `Public: true` | Yes |
| **Route** | `GET /v1/identity/slug/:slug` | `GET /v1/core/accounts/slug/{slug}` | Yes (prefix by design) |
| **Request params** | `slug` (path) | `slug` (path) | Yes |
| **DB query** | `account_portal.slug` → join account + left join branding | Same: `account_portal.slug` → join account + left join branding | Yes |
| **404 handling** | `HttpError.notFound('Account not found.')` | `apierror.NewResourceNotFoundError("Resource not found.")` via `MapSQLError` | Yes (message differs, code matches) |
| **Permission checks** | None | None | Yes |
| **Side effects** | None | None | Yes |
| **Idempotency** | N/A (GET) | N/A (GET) | Yes |

## Response shape comparison

| Dashboard field | Go field | Notes |
|----------------|----------|-------|
| `id` | `id` | Identical |
| — | `object` (`"public_account"`) | Go convention: all resources have object type |
| `name` | `name` | Identical |
| `slug` | `slug` | Identical |
| `email` | `support_email` | Same data (from `account_branding.support_email`), different field name |
| `logoUrl` (presigned S3 URL) | `logo_url` (raw S3 key) | See note 1 below |
| `billToAddress` (full Address) | `default_billing_address` (expandable ref) | See note 2 below |
| `customerRegistrationFlow` | — (not included) | See note 3 below |

## Known intentional differences

### 1. Logo URL: presigned URL vs raw S3 key

The Dashboard generates a presigned S3 URL (1-hour TTL) inline in the `findBySlug` response. The Go API returns the raw S3 key and provides a dedicated `GET /v1/core/accounts/{id}/logo` endpoint for presigned URLs. This is a deliberate separation of concerns — consumers should use the dedicated logo endpoint.

### 2. Default billing address: full object vs expandable reference

The Dashboard returns the full `Address` object inline. The Go API returns an expandable reference (`{ id, object }`) per its sub-resource conventions. This is consistent with Go API resource conventions.

### 3. Customer registration flow: included vs separate endpoint

The Dashboard returns the first `customerRegistrationFlow` (with customer group, payment term, and shipping term options) inline. The Go API has a dedicated `GET /v1/core/registration-flows/by-slug/{slug}` endpoint for this data. This is a deliberate separation — the registration flow is its own resource.

## Remaining concerns

None. The core business logic (lookup by portal slug, return public account data, 404 if not found) has full parity. The response shape differences follow Go API conventions and the consumer (dashboard frontend) has dedicated endpoints for the data that was previously bundled inline.
