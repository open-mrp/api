package service

import (
	"encoding/json"
	"testing"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
)

// isToolBlocked mirrors the guard logic in runAgentLoop.
func isToolBlocked(runCtx *domain.HandlerRunContext, toolSlug string) bool {
	return runCtx.RequireReviewBySlug[toolSlug] &&
		!runCtx.AlwaysAllowedSlugs[toolSlug] &&
		!runCtx.OneTimeApprovedSlugs[toolSlug]
}

// isCallBlocked mirrors the full guard in runAgentLoop for a specific call (slug + input), so it also
// honors per-call approvals keyed on (slug+input) — the lever slug-only approval cannot pull.
func isCallBlocked(runCtx *domain.HandlerRunContext, toolSlug string, input json.RawMessage) bool {
	key := toolCallApprovalKey(toolSlug, input)
	return runCtx.RequireReviewBySlug[toolSlug] &&
		!runCtx.AlwaysAllowedSlugs[toolSlug] &&
		!runCtx.OneTimeApprovedSlugs[toolSlug] &&
		!runCtx.OneTimeApprovedKeys[key]
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

// approvesReviewedTool is the single authority for what a ContinueRun resume lets through. The security
// invariant under test: only an explicit approval approves — a per-tool approval names the slugs, or an
// "Approve all" sets approveAllPending. A resume that does neither (a typed-message continuation or a
// retry, both arriving with no slugs and approveAllPending=false) must approve NOTHING, so a blocked
// review-gated tool stays blocked. This is the real production predicate (not a copy), so a regression in
// the rule fails here.
func TestApprovesReviewedTool(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name              string
		toolSlug          string
		approvedToolSlugs []string
		approveAllPending bool
		want              bool
	}{
		{"per-tool approves the named slug", "update_customer", []string{"update_customer"}, false, true},
		{"per-tool rejects an unnamed slug", "delete_customer", []string{"update_customer"}, false, false},
		{"approve-all approves any pending tool", "update_customer", nil, true, true},
		{"typed-message continuation approves nothing", "update_customer", nil, false, false},
		{"retry (no slugs, no approve-all) approves nothing", "update_customer", nil, false, false},
		{"named slugs take precedence over approve-all", "delete_customer", []string{"update_customer"}, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := approvesReviewedTool(tc.toolSlug, tc.approvedToolSlugs, tc.approveAllPending); got != tc.want {
				t.Errorf("approvesReviewedTool(%q, %v, %v) = %v, want %v",
					tc.toolSlug, tc.approvedToolSlugs, tc.approveAllPending, got, tc.want)
			}
		})
	}
}

// The bug this fixes: two blocked calls of the SAME slug but different inputs (e.g. updating two different
// customers) must be approvable independently. Slug approval can't express that — it would let both through.
// Per-call approval keys on (slug+input), so approving one call's key leaves the sibling call blocked.
func TestPerCallApproval_SameSlugDifferentInputs(t *testing.T) {
	t.Parallel()
	inputA := json.RawMessage(`{"customer_id":"cust_a","name":"A"}`)
	inputB := json.RawMessage(`{"customer_id":"cust_b","name":"B"}`)

	runCtx := &domain.HandlerRunContext{
		RequireReviewBySlug: map[string]bool{"update_customer": true},
		AlwaysAllowedSlugs:  make(map[string]bool),
		// Nothing approved at the slug level — only call A's specific (slug+input) key.
		OneTimeApprovedSlugs: make(map[string]bool),
		OneTimeApprovedKeys:  map[string]bool{toolCallApprovalKey("update_customer", inputA): true},
	}

	if isCallBlocked(runCtx, "update_customer", inputA) {
		t.Error("expected the approved call (input A) to be allowed")
	}
	if !isCallBlocked(runCtx, "update_customer", inputB) {
		t.Error("expected the unapproved same-slug call (input B) to STILL be blocked — per-call approval must not leak across inputs")
	}
}

// A per-call approval is one-time: after the approved call runs, deleting its key re-blocks that exact call.
func TestPerCallApproval_ConsumedAfterExecution(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`{"customer_id":"cust_a"}`)
	key := toolCallApprovalKey("update_customer", input)
	runCtx := &domain.HandlerRunContext{
		RequireReviewBySlug:  map[string]bool{"update_customer": true},
		AlwaysAllowedSlugs:   make(map[string]bool),
		OneTimeApprovedSlugs: make(map[string]bool),
		OneTimeApprovedKeys:  map[string]bool{key: true},
	}
	if isCallBlocked(runCtx, "update_customer", input) {
		t.Error("expected the approved call to be allowed on first invocation")
	}
	delete(runCtx.OneTimeApprovedKeys, key) // mirrors the post-execution consumption
	if !isCallBlocked(runCtx, "update_customer", input) {
		t.Error("expected the call to be blocked again after its one-time per-call approval was consumed")
	}
}

// toolCallApprovalKey must be stable across JSON key order and whitespace, because the input recorded at
// block time and the input the model re-emits on resume may serialize differently. If the key weren't
// canonical, the approval would silently miss the retried call and re-pause the run.
func TestToolCallApprovalKey_StableAcrossKeyOrderAndWhitespace(t *testing.T) {
	t.Parallel()
	a := json.RawMessage(`{"customer_id":"c1","name":"Acme"}`)
	b := json.RawMessage(`{ "name": "Acme", "customer_id": "c1" }`)
	if toolCallApprovalKey("update_customer", a) != toolCallApprovalKey("update_customer", b) {
		t.Error("expected equal keys for the same object regardless of key order / whitespace")
	}
	// Different inputs (or different slugs) must yield different keys.
	c := json.RawMessage(`{"customer_id":"c2","name":"Acme"}`)
	if toolCallApprovalKey("update_customer", a) == toolCallApprovalKey("update_customer", c) {
		t.Error("expected different keys for different inputs")
	}
	if toolCallApprovalKey("update_customer", a) == toolCallApprovalKey("delete_customer", a) {
		t.Error("expected different keys for different slugs")
	}
}

// blockedCallKeysByID resolves the tool_use_ids the frontend sends back to the (slug+input) keys the runner
// matches on, reading them from this run's tool_blocked events.
func TestBlockedCallKeysByID(t *testing.T) {
	t.Parallel()
	events := []sqlc.AgentRunEvent{
		{StepType: "tool_call", Metadata: []byte(`{"tool_use_id":"ignored","tool_name":"x"}`)},
		{StepType: "tool_blocked", Metadata: []byte(`{"tool_use_id":"call_a","tool_name":"update_customer","input":{"customer_id":"a"}}`)},
		{StepType: "tool_blocked", Metadata: []byte(`{"tool_use_id":"call_b","tool_name":"update_customer","input":{"customer_id":"b"}}`)},
		{StepType: "tool_blocked", Metadata: nil}, // malformed/empty — skipped
	}
	got := blockedCallKeysByID(events)
	if len(got) != 2 {
		t.Fatalf("expected 2 blocked-call keys, got %d", len(got))
	}
	if got["call_a"] != toolCallApprovalKey("update_customer", json.RawMessage(`{"customer_id":"a"}`)) {
		t.Errorf("call_a mapped to wrong key: %q", got["call_a"])
	}
	if got["call_a"] == got["call_b"] {
		t.Error("expected the two same-slug calls to map to distinct keys")
	}
	if _, ok := got["ignored"]; ok {
		t.Error("expected non-tool_blocked events to be ignored")
	}
}
