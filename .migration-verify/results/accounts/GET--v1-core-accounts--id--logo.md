# Verification: GET /v1/core/accounts/{id}/logo

## Result: Parity Confirmed

No code changes required. The Go implementation correctly replicates the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Auth | `AuthOptions.None` (public) | `Public: true` | Yes |
| Permission checks | None | None | Yes |
| S3 file existence check | Yes (`AwsUtils.fileExists`) | Yes (`s3Client.FileExists`) | Yes |
| Presigned URL expiry | 3600 seconds (1 hour) | `time.Hour` (1 hour) | Yes |
| No-logo behavior | Returns null (controller throws 404) | Returns nil (response has `url: null`) | Yes (see note) |
| Idempotency | N/A (GET) | N/A (GET) | Yes |
| Side effects | None | None | Yes |

## Notes

### Logo key lookup difference (acceptable)

- **Dashboard**: Constructs the S3 key directly as `${accountID}/logo.png` without a database lookup.
- **Go**: Reads the logo key from `account_branding.logo_url` in the database.

These are functionally equivalent because the Dashboard `updatePhoto` method stores the same `${accountID}/logo.png` key into the `account_branding.logoURL` field. Both paths resolve to the same S3 object. The Go approach is actually more robust since it uses the stored key rather than reconstructing it.

### Response format difference (by design)

- **Dashboard**: Returns HTTP 302 redirect to the presigned URL with `Cache-Control: no-cache, no-store, must-revalidate` and `Cross-Origin-Resource-Policy: cross-origin` headers. Returns 404 if no logo.
- **Go**: Returns HTTP 200 with JSON `{ "url": "..." }` or `{ "url": null }`.

This is an intentional change for the new `/v1/core/` API, which follows the standard JSON response pattern. The redirect behavior was specific to the legacy API where this endpoint was used directly as an image source.

## No Issues Found

The core business logic (S3 presigned URL generation with 1-hour expiry, public access, no auth) is fully preserved.
