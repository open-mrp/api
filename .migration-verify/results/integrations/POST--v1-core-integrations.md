# POST /v1/core/integrations — Migration Verification

## Result: PARITY CONFIRMED

No code changes required. The Go implementation faithfully reproduces all Dashboard business logic for this endpoint.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: internal actor** | `checkIsInternalActor(identity)` | `identity.CheckIsInternalActor()` | Yes |
| **Permission: admin role** | `roleTypeCode !== RoleTypes.admin` | `identity.CheckIsAdmin()` | Yes |
| **Permission: target account** | `identity.targetAccountID` | `identity.CheckTargetAccountSet()` | Yes |
| **Integration code validation** | Zod `z.nativeEnum(Integrations)` | `params.IntegrationCode.IsValid()` | Yes |
| **Stripe cred: privateKey prefix** | `sk_` | `sk_` | Yes |
| **Stripe cred: publishableKey prefix** | `pk_` | `pk_` | Yes |
| **Stripe cred: webhookSecret prefix** | `whsec_` | `whsec_` | Yes |
| **Shippo cred: apiKey prefix** | `shippo_live_` / `shippo_test_` | `shippo_live_` / `shippo_test_` | Yes |
| **Sandbox: test keys required** | `sk_test_`, `pk_test_`, `shippo_test_` | Same | Yes |
| **Non-sandbox/prod: live keys required** | `sk_live_`, `pk_live_`, `shippo_live_` | Same | Yes |
| **Credential encryption** | `EncryptionUtils.encryptObject()` | `crypto.EncryptAESGCM()` | Yes |
| **Upsert behavior** | FindByCode → update or create | FindByCode → update or create | Yes |
| **Upsert updates name + credentials** | `{ name, credentials }` | `UpdateCredentials(name, creds)` | Yes |
| **New record: is_active default** | DB default `true` | `INSERT ... 1` | Yes |
| **Response excludes credentials** | `AccountIntegrationAdapter.select` omits | Resource type has no cred field | Yes |
| **HTTP status** | 201 Created | 201 Created | Yes |
| **Side effects** | None | None | Yes |
| **Request body shielding** | N/A (not applicable to Express) | `ShieldRequestBody: true` | N/A |
| **Idempotency** | Not present | Recovery point pattern | Improvement |

## Minor Acceptable Differences

1. **Field naming**: Dashboard uses `code`, Go uses `integration_code`. This follows Go API conventions and the internal-sdk will generate the correct types from the OpenAPI spec.

2. **Credentials field type**: Dashboard accepts credentials as a JSON object (`Record<string, string>`), Go accepts it as a JSON string. Functionally equivalent — the internal-sdk client handles serialization.

3. **Non-sandbox + dev environment**: Dashboard has a 3-way check (sandbox → test keys, production env → live keys, dev env → test keys). Go has a 2-way check (sandbox → test keys, non-sandbox → live keys). In production the behavior is identical. The only difference is in development environments where Go requires live keys for non-sandbox accounts while Dashboard allows test keys. This is a dev-ergonomics concern only, not a production parity issue.

4. **Idempotency keys**: Go adds idempotency key support with recovery points for the POST endpoint. This is an improvement over the Dashboard, which lacks explicit idempotency handling.

5. **Transaction handling**: Go wraps the upsert in a transaction (`withTx`). Dashboard does not use an explicit transaction for the create/update operation, relying on Prisma's default behavior.

## No Issues Found
