-- name: GetAccountTypeCode :one
-- Resolves whether an account is a sandbox. The send path reads this rather than trusting a caller
-- to have populated AccountMode on the message: a publisher that forgets the field would otherwise
-- mail a sandbox account's invoice to its real customer.
SELECT account_type_code FROM account WHERE id = sqlc.arg('id');
