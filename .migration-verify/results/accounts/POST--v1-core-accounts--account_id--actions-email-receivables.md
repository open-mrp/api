# POST /v1/core/accounts/{account_id}/actions/email-receivables

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Request requires `account_id` (path) and `recipient_emails` (body, min=1). Both match.
- **Permission checks**: Both require internal actor + `customers:read`. Match.
- **Business logic flow**: Both fetch unpaid invoices, open credits, customer name, generate Excel statement, send email. Match.
- **Excel generation**: Columns, aging buckets (Current/30/60/90/120+), sorting (newest first), open credits as negative amounts, totals row with bold+gray styling. All match.
- **Email subject/body**: Both use `"Statement of Account for {name}"` and matching body text. Match.
- **Idempotency**: Go correctly uses idempotency keys with recovery points for this POST endpoint. Dashboard doesn't have idempotency (expected).
- **Response**: Dashboard returns 200 OK with `{ message: "..." }`, Go returns 202 Accepted with empty resource. Acceptable — 202 is more appropriate for async email sending.

## Issues found and fixed

1. **Attachment filename missing date** (fixed): Dashboard generates filename as `account-statement-{customerName}-{accountName}-{MM-DD-YYYY}.xlsx`. Go was generating `statement-of-account-{customerName}.xlsx` — missing the date and using a different prefix. Fixed to `account-statement-{customerName}-{MM-DD-YYYY}.xlsx` matching the Dashboard's prefix and date format.

2. **Missing SentByID audit field** (fixed): Dashboard passes `sentBy: this.identity` for email audit logging. Go was not setting `SentByID` on the `EmailSendData`. Fixed to set `SentByID` from `identity.Actor.ID`.

## Minor differences (not fixed, acceptable)

- **formatRecordNumber in Excel**: Dashboard pads invoice numbers with leading zeros (e.g., `123` -> `000123`) in the Excel output. Go uses raw invoice numbers. This is cosmetic and the numbers are still correct.
- **Dashboard filename includes both customer.name and account.name**: The Dashboard fetches both a customer relationship name and the account name, using both in the filename. Go uses only the account name (from `AccountRepo.GetName`). These values are typically identical.
- **Dashboard sender overrides**: Dashboard sets `senderEmail`/`senderName` from the customer's account record. Go uses the default notification-service sender. This appears to be a Dashboard-specific behavior that may actually be a bug (setting sender as the customer's own email when emailing that customer).
