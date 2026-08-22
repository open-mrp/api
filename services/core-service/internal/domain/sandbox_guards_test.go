package domain

import (
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
)

// SAFETY: DO NOT REMOVE — these tests ensure sandbox guard functions
// correctly protect production accounts from sandbox-only operations.

func TestSafety_RequireSandboxAccount_RejectsNonSandbox(t *testing.T) {
	t.Parallel()
	accountCtx := &AccountContext{AccountID: "ac_prod", IsSandbox: false}
	err := RequireSandboxAccount(accountCtx)
	if err == nil {
		t.Fatal("expected error for non-sandbox account, got nil")
	}
	if err.Code != apierror.ErrorCodeInternalError {
		t.Fatalf("expected ErrorCodeInternalError, got %s", err.Code)
	}
}

func TestSafety_RequireSandboxAccount_AcceptsSandbox(t *testing.T) {
	t.Parallel()
	accountCtx := &AccountContext{AccountID: "ac_sandbox", IsSandbox: true}
	err := RequireSandboxAccount(accountCtx)
	if err != nil {
		t.Fatalf("expected nil error for sandbox account, got %v", err)
	}
}

func TestSafety_RequireSandboxAccount_RejectsNil(t *testing.T) {
	t.Parallel()
	err := RequireSandboxAccount(nil)
	if err == nil {
		t.Fatal("expected error for nil AccountContext, got nil")
	}
	if err.Code != apierror.ErrorCodeInternalError {
		t.Fatalf("expected ErrorCodeInternalError, got %s", err.Code)
	}
}

func TestSafety_RequireNotSandboxAccount_RejectsSandbox(t *testing.T) {
	t.Parallel()
	accountCtx := &AccountContext{AccountID: "ac_sandbox", IsSandbox: true}
	err := RequireNotSandboxAccount(accountCtx)
	if err == nil {
		t.Fatal("expected error for sandbox account, got nil")
	}
	if err.Code != apierror.ErrorCodeValidationFailed {
		t.Fatalf("expected ErrorCodeValidationFailed, got %s", err.Code)
	}
}

func TestSafety_RequireNotSandboxAccount_AcceptsNonSandbox(t *testing.T) {
	t.Parallel()
	accountCtx := &AccountContext{AccountID: "ac_prod", IsSandbox: false}
	err := RequireNotSandboxAccount(accountCtx)
	if err != nil {
		t.Fatalf("expected nil error for non-sandbox account, got %v", err)
	}
}

func TestSafety_RequireNotSandboxAccount_RejectsNil(t *testing.T) {
	t.Parallel()
	err := RequireNotSandboxAccount(nil)
	if err == nil {
		t.Fatal("expected error for nil AccountContext, got nil")
	}
	if err.Code != apierror.ErrorCodeInternalError {
		t.Fatalf("expected ErrorCodeInternalError, got %s", err.Code)
	}
}
