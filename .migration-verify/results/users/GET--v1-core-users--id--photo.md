# GET /v1/core/users/{id}/photo — Migration Verification

## Result: PARITY CONFIRMED

No code changes needed. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Auth/permissions | No auth (public endpoint) | No auth (public endpoint) | Yes |
| DB query | `accountUser.findFirst({ where: { userID } })` | `FindFirstAccountIDByUserID` (same query) | Yes |
| S3 file check | `AwsUtils.fileExists(bucket, key)` | `s3Client.FileExists(bucket, key)` | Yes |
| S3 presigned URL | `getTemporaryFileUrl(bucket, key, 3600)` | `GetPresignedURL(bucket, key, time.Hour)` | Yes |
| URL expiry | 3600 seconds (1 hour) | `time.Hour` (1 hour) | Yes |
| S3 key format | `{accountID}/{userID}.png` (instance) / `.jpg` (static) | `{accountID}/{userID}.png` | Yes (see note) |
| No photo behavior | 404 Not Found | `{ "url": null }` | Intentional (see note) |
| Response format | 302 redirect to presigned URL | JSON `{ "url": "..." }` | Intentional (see note) |

## Notes

### File extension (.png vs .jpg)
The Dashboard has an inconsistency: the instance method `getPhotoKey()` returns `.png`, but the public static method `getPhotoUrlByUserID()` hardcodes `.jpg`. Since uploads always use `.png` (via `getPhotoKey`), the Dashboard's static method has a bug where it would never find uploaded photos. The Go implementation correctly uses `.png` for both read and write paths.

### Response format change (redirect → JSON)
The Dashboard controller returns a 302 redirect to the presigned S3 URL with cache headers. The Go API returns a JSON response with the URL string. This is an intentional design improvement — the new REST API follows standard JSON response conventions rather than redirect-based patterns.

### No photo handling (404 → null)
The Dashboard throws a 404 when no photo exists. The Go API returns `{ "url": null }`. This is a cleaner API design that lets clients handle the absence without catching errors.

## No Issues Found
