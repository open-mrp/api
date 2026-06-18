package domain

import (
	apierror "github.com/augno/api/shared/errors"
)

// RequireSandboxAccount returns an invariant violation error if the account is not a sandbox. Used before sandbox-mutating operations (delete, purge).
// SAFETY: DO NOT REMOVE — protects production accounts from sandbox-only operations.
func RequireSandboxAccount(accountCtx *AccountContext) *apierror.APIError {
	if accountCtx == nil {
		return apierror.NewInvariantViolationError("RequireSandboxAccount: nil AccountContext")
	}
	if !accountCtx.IsSandbox {
		return apierror.NewInvariantViolationError("Cannot delete: account is not a sandbox account.")
	}
	return nil
}

// RequireNotSandboxAccount returns a validation error if the account IS a sandbox. Used to block sandbox-from-sandbox creation.
// SAFETY: DO NOT REMOVE — prevents sandbox accounts from spawning nested sandboxes.
func RequireNotSandboxAccount(accountCtx *AccountContext) *apierror.APIError {
	if accountCtx == nil {
		return apierror.NewInvariantViolationError("RequireNotSandboxAccount: nil AccountContext")
	}
	if accountCtx.IsSandbox {
		return apierror.NewValidationError("Sandbox accounts cannot create sandboxes.")
	}
	return nil
}
