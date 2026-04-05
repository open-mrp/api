# POST /v1/core/actions/request-demo

## Status: PARITY CONFIRMED (with known gap)

## Comparison Summary

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Route | `POST /v1/actions/request-demo` | `POST /v1/core/actions/request-demo` | Yes (expected prefix change) |
| Authentication | None (AuthOptions.None) | None (Public: true) | Yes |
| Required fields | name, email, company | name, email (custom_email), company | Yes |
| Optional fields | phoneNumber, message | phone_number, message | Yes |
| Email validation | Zod email validator | `custom_email` struct tag | Yes |
| Response shape | `{ message: string }` | `MessageResource` with message field | Yes |
| Response message | "Demo request submitted successfully" | "Demo request submitted successfully." | Minor (trailing period) |
| Email sending | AWS SES to dane@augno.com | Not implemented (logs only) | Known gap |

## What Was Compared

- Request validation rules (required fields, email format)
- Permission/auth checks (both public, no auth required)
- Business logic (email sending vs logging)
- Response shape and status code (both HTTP 200)
- Field naming (camelCase in TS → snake_case in Go, as expected)
- Idempotency handling

## Known Gap

The Go implementation does **not** send the demo request email. It logs the request via `slog.InfoContext` and returns success. The code contains a comment: "Email sending will be wired later via the notification service." This is an intentional deferral, not a bug — the notification service integration for system emails hasn't been built yet.

## No Issues Fixed

No code changes were necessary. The endpoint structure, validation, auth, and response shape are all correct. The email-sending gap is a known, pre-existing item.
