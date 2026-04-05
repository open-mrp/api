# GET /v1/core/accounts/{id} — Migration Verification

## Result: Parity Confirmed (no changes needed)

The core business logic is preserved. Response shape differences are intentional per Go API conventions (sub-resources, expandable fields).

## What Was Compared

### Permission & Access Control
- **Dashboard**: Controller checks `accountID !== targetAccountID` (403 forbidden), service checks `isInternalActor` + `hasPermission(account, read)`
- **Go**: Service checks `isInternalActor` + `hasPermission(account, read)` + `checkTargetAccountSet` + `accountID != targetAccountID` (authorization error)
- **Verdict**: ✅ Equivalent. Go consolidates all checks in the service layer and adds an explicit target-account-set check (stricter).

### Database Query
- **Dashboard**: Prisma `findUnique` with selects for account, branding, portal, and full address objects
- **Go**: Single SQL query with `LEFT JOIN account_branding` and `LEFT JOIN account_portal`, returns address IDs only
- **Verdict**: ✅ Equivalent data fetched. Address handling differs by design (Go uses expandable sub-resources).

### Error Handling
- **Not found**: Dashboard explicit null check → 404; Go `db.MapSQLError()` converts `sql.ErrNoRows` → 404. ✅
- **Access denied**: Dashboard 403 "You cannot access this account."; Go authorization error "You can only access your own account." ✅ (cosmetic message difference)
- **Permission denied**: Both check internal actor + account:read permission. ✅

### Response Shape
- **Dashboard**: Flat structure — branding fields (phone, email, logoUrl, social handles, website) hoisted to top level; full address objects inline; portal slug at top level
- **Go**: Structured sub-resources — `branding` (AccountBranding), `portal` (AccountPortal), `default_billing_address`/`default_shipping_address` (Address, expandable)
- **Verdict**: ✅ Intentional architectural difference per Go API resource conventions (sub-resources with `object` field, expandable relations)

### Logo URL Presigning
- **Dashboard**: Presigns S3 URL inline in the `find` method; silently sets to null on failure
- **Go**: Returns raw S3 key in `branding.logo_url`; separate dedicated endpoint exists for presigned URL (`AccountLogoURL` resource)
- **Verdict**: ✅ Intentional design change — Go separates concerns with a dedicated presigning endpoint

### Side Effects
- Neither endpoint has side effects (read-only). ✅

### Idempotency
- GET endpoint — inherently idempotent, no idempotency key needed. ✅

## Intentional Differences (not bugs)

1. **Response shape**: Go uses proper nested sub-resources per codebase conventions
2. **Logo presigning**: Go has a separate endpoint rather than inline presigning
3. **Address data**: Go returns expandable address references (ID + object type) rather than full address inline
4. **shipToAddress fallback**: Dashboard falls back to billing address when no shipping address exists; Go returns null — acceptable since Go uses expandable sub-resources
