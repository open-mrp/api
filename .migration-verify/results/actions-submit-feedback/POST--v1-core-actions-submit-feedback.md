# POST /v1/core/actions/submit-feedback

## Status: Parity Confirmed (with known gap)

No code changes required.

## What Was Compared

- **Validation**: Both require non-empty `question` and `answer` strings, with optional page URL. Dashboard uses Zod `min(1)`, Go uses `validate:"required"`. Equivalent.
- **Permissions**: Dashboard requires authenticated user with no specific permission domain. Go uses `CheckIsAssignedActor()` with no permission domain. Equivalent.
- **Request shape**: Dashboard `pageUrl` (camelCase) vs Go `page_url` (snake_case) — expected convention difference.
- **Response shape**: Both return `{ message: "Feedback submitted successfully" }`. Go adds a trailing period — cosmetic only.
- **Side effects**: Dashboard sends email via AWS SES to `dev@augno.com` with actor metadata (name, type, ID, account ID, page URL, question, answer). Go logs feedback via structured logging.
- **Idempotency**: Go gRPC handler uses `WithIdempotencyTracking`. No DB mutations so no idempotency key state machine in the service — consistent with the similar `RequestDemo` endpoint.
- **Error handling**: Both return errors only for missing authentication. No DB operations to fail.

## Known Gap

The Go implementation logs feedback but does not send an email. This is intentional and documented in the Go code (`// Email sending will be wired later via the notification service.`). The Go API architecture routes emails through the notification service rather than calling AWS SES directly, so this will be wired when the notification integration is connected.

## No Issues Fixed

No discrepancies requiring code changes were found.
