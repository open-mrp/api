package service

import (
	"testing"

	"github.com/augno/api/services/agent-service/internal/domain"
)

// isToolBlocked mirrors the guard logic in runAgentLoop.
func isToolBlocked(runCtx *domain.HandlerRunContext, toolSlug string) bool {
	return runCtx.RequireReviewBySlug[toolSlug] &&
		!runCtx.AlwaysAllowedSlugs[toolSlug] &&
		!runCtx.OneTimeApprovedSlugs[toolSlug]
}

// consumeOneTimeApproval mirrors the post-execution deletion in runAgentLoop.
func consumeOneTimeApproval(runCtx *domain.HandlerRunContext, toolSlug string) {
	delete(runCtx.OneTimeApprovedSlugs, toolSlug)
}

func TestOneTimeApproval_ConsumedAfterExecution(t *testing.T) {
	t.Parallel()
	runCtx := &domain.HandlerRunContext{
		RequireReviewBySlug:  map[string]bool{"dangerous-tool": true},
		AlwaysAllowedSlugs:   make(map[string]bool),
		OneTimeApprovedSlugs: map[string]bool{"dangerous-tool": true},
	}

	// First call: tool is approved (one-time), should not be blocked.
	if isToolBlocked(runCtx, "dangerous-tool") {
		t.Error("expected tool to be allowed on first call with one-time approval")
	}

	// Simulate execution + consumption.
	consumeOneTimeApproval(runCtx, "dangerous-tool")

	// Second call: one-time approval consumed, should be blocked.
	if !isToolBlocked(runCtx, "dangerous-tool") {
		t.Error("expected tool to be blocked after one-time approval was consumed")
	}
}

func TestAlwaysAllowed_NeverConsumed(t *testing.T) {
	t.Parallel()
	runCtx := &domain.HandlerRunContext{
		RequireReviewBySlug:  map[string]bool{"dangerous-tool": true},
		AlwaysAllowedSlugs:   map[string]bool{"dangerous-tool": true},
		OneTimeApprovedSlugs: make(map[string]bool),
	}

	// First call: always-allowed, should not be blocked.
	if isToolBlocked(runCtx, "dangerous-tool") {
		t.Error("expected always-allowed tool to not be blocked on first call")
	}

	// Consume has no effect on always-allowed.
	consumeOneTimeApproval(runCtx, "dangerous-tool")

	// Second call: still not blocked.
	if isToolBlocked(runCtx, "dangerous-tool") {
		t.Error("expected always-allowed tool to not be blocked on second call")
	}
}

func TestApproveAll_OneTimeConsumed(t *testing.T) {
	t.Parallel(
	// "Approve All" scenario: multiple tools approved one-time (no always-allow).
	)

	runCtx := &domain.HandlerRunContext{
		RequireReviewBySlug: map[string]bool{
			"tool-a": true,
			"tool-b": true,
		},
		AlwaysAllowedSlugs: make(map[string]bool),
		OneTimeApprovedSlugs: map[string]bool{
			"tool-a": true,
			"tool-b": true,
		},
	}

	// Both tools should be allowed initially.
	if isToolBlocked(runCtx, "tool-a") || isToolBlocked(runCtx, "tool-b") {
		t.Error("expected both tools to be allowed initially")
	}

	// Execute and consume both.
	consumeOneTimeApproval(runCtx, "tool-a")
	consumeOneTimeApproval(runCtx, "tool-b")

	// Both should be blocked on next invocation.
	if !isToolBlocked(runCtx, "tool-a") {
		t.Error("expected tool-a to be blocked after consumption")
	}
	if !isToolBlocked(runCtx, "tool-b") {
		t.Error("expected tool-b to be blocked after consumption")
	}
}

func TestToolNotRequiringReview_NeverBlocked(t *testing.T) {
	t.Parallel()
	runCtx := &domain.HandlerRunContext{
		RequireReviewBySlug:  map[string]bool{},
		AlwaysAllowedSlugs:   make(map[string]bool),
		OneTimeApprovedSlugs: make(map[string]bool),
	}

	if isToolBlocked(runCtx, "safe-tool") {
		t.Error("expected tool without require_review to never be blocked")
	}
}
