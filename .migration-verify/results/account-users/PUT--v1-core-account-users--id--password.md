# PUT /v1/core/account-users/{id}/password

## Result: Issues found and fixed

## What was compared

- **Validation rules**: Both enforce password strength (8-72 chars, lowercase, uppercase, digit, special char). Go uses the `password` validator tag; Dashboard uses `PasswordUtils.checkPasswordStrength()`. Equivalent.
- **Permission checks**: Both require internal actor with `teamUsers:update` permission and a target account ID. Equivalent.
- **Requester password verification**: Dashboard uses `authRepo.loginAccountUser()` (bcrypt compare by user ID); Go uses `UserRepo.GetHashedPassword()` + `crypto.CompareBcryptHash()`. Functionally equivalent.
- **DB queries**: Both update the `user` table's `hashed_password` column for the target user ID. Equivalent.
- **Error handling**: Minor message differences (Dashboard: "Username or password is incorrect." / Go: "Incorrect password.") — acceptable since Go's error is more precise for this context.
- **Side effects**: None in either implementation.
- **Response shape**: Dashboard returns `true` with 200; Go returns 204 No Content. Acceptable API design improvement.
- **Idempotency**: PUT endpoint — idempotent by design, no idempotency key needed. Both implementations are consistent.
- **Request body shielding**: Go correctly shields the request body (sensitive password data).

## Issues found and fixed

### 1. Missing target account user existence check (security gap)

**Dashboard behavior**: Before updating the password, the Dashboard verifies that the target user (`userID`) belongs to the requester's account by calling `accountUserRepo.find({ userID, accountID })` and returning 404 if not found.

**Go behavior (before fix)**: The Go code did not verify that the target user belongs to the requester's account. This meant any authenticated user with `teamUsers:update` permission could update any user's password across any account, which is a security issue.

**Fix**: Added a call to `AccountUserRepo.FindByAccountAndUserID(ctx, userID, *identity.TargetAccountID)` before hashing/updating the password. This returns a not-found error if the user doesn't belong to the account, matching the Dashboard's behavior.

## No remaining concerns
