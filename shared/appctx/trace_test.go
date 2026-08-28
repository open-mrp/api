package appctx

import (
	"context"
	"testing"
	"time"
)

func TestShouldTrace_FreshContext(t *testing.T) {
	t.Parallel()
	if !ShouldTrace(context.Background()) {
		t.Error("expected tracing to be enabled on a fresh context")
	}
	if !ShouldTrace(t.Context()) {
		t.Error("expected tracing to be enabled on a derived test context")
	}
}

// Background workers suppress tracing once and then wrap the context in timeouts and cancels per
// batch (messaging/enqueuer.go, cleanup.go, inbox_purger.go); suppression has to survive every one
// of those wrappers or high-volume polling floods the trace backend.
func TestShouldTrace_SuppressionSurvivesWrappers(t *testing.T) {
	t.Parallel()
	base := WithNoTrace(context.Background())

	cancelCtx, cancel := context.WithCancel(base)
	defer cancel()
	timeoutCtx, cancelTimeout := context.WithTimeout(base, time.Hour)
	defer cancelTimeout()
	deadlineCtx, cancelDeadline := context.WithDeadline(base, time.Now().Add(time.Hour))
	defer cancelDeadline()
	withoutCancelCtx := context.WithoutCancel(base)

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "direct", ctx: base},
		{name: "with value", ctx: context.WithValue(base, handlerKey, "worker.Enqueue")},
		{name: "with cancel", ctx: cancelCtx},
		{name: "with timeout", ctx: timeoutCtx},
		{name: "with deadline", ctx: deadlineCtx},
		{name: "without cancel", ctx: withoutCancelCtx},
		{name: "nested", ctx: context.WithValue(cancelCtx, requestIDKey, "req_1")},
		{name: "re-suppressed", ctx: WithNoTrace(timeoutCtx)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if ShouldTrace(tt.ctx) {
				t.Error("expected tracing to stay suppressed through the wrapper")
			}
		})
	}
}

func TestShouldTrace_SuppressionDoesNotEscapeToSiblings(t *testing.T) {
	t.Parallel()
	root := context.WithValue(context.Background(), handlerKey, "worker.Enqueue")
	suppressed := WithNoTrace(root)

	if ShouldTrace(suppressed) {
		t.Error("expected the derived context to suppress tracing")
	}
	if !ShouldTrace(root) {
		t.Error("expected the parent context to be unaffected")
	}
}

func TestShouldTrace_NonBoolValueUnderKey(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), noTraceKey, "true")

	if !ShouldTrace(ctx) {
		t.Error("expected a non-bool value to leave tracing enabled")
	}
}
