package appctx

import (
	"context"
	"testing"
)

func TestGetRequestLog_Absent(t *testing.T) {
	t.Parallel()
	if rl, ok := GetRequestLog(context.Background()); ok || rl != nil {
		t.Errorf("expected (nil, false) on a bare context, got (%+v, %v)", rl, ok)
	}
}

func TestGetRequestLog_TypedNil(t *testing.T) {
	t.Parallel()
	ctx := WithRequestLog(context.Background(), nil)

	if rl, ok := GetRequestLog(ctx); ok || rl != nil {
		t.Errorf("expected an explicitly stored nil to report absent, got (%+v, %v)", rl, ok)
	}
}

func TestGetRequestLog_NilShadowsParent(t *testing.T) {
	t.Parallel()
	parent := WithRequestLog(context.Background(), &RequestLog{ID: "rql_1"})
	child := WithRequestLog(parent, nil)

	if rl, ok := GetRequestLog(child); ok || rl != nil {
		t.Errorf("expected the shadowing nil to report absent, got (%+v, %v)", rl, ok)
	}
	if rl, ok := GetRequestLog(parent); !ok || rl == nil {
		t.Errorf("expected the parent request log to survive, got (%+v, %v)", rl, ok)
	}
}

// The endpoint layer sets SensitiveResponseFields on the pointer it pulls out of context and the
// logging middleware redacts from the pointer it created; if the context ever handed back a copy,
// secrets would be persisted unredacted.
func TestGetRequestLog_MutationsAliasTheOriginal(t *testing.T) {
	t.Parallel()
	rl := &RequestLog{ID: "rql_1"}
	ctx, cancel := context.WithCancel(WithRequestLog(context.Background(), rl))
	defer cancel()

	fromCtx, ok := GetRequestLog(ctx)
	if !ok {
		t.Fatal("expected request log in context")
	}
	if fromCtx != rl {
		t.Fatal("expected the same pointer back out of context")
	}

	fromCtx.SensitiveResponseFields = map[string]bool{"secret": true}
	fromCtx.StatusCode = 200
	fromCtx.SkipSave = true

	if !rl.SensitiveResponseFields["secret"] {
		t.Error("expected SensitiveResponseFields set through the context to be visible to the original holder")
	}
	if rl.StatusCode != 200 {
		t.Errorf("expected StatusCode 200 on the original holder, got %d", rl.StatusCode)
	}
	if !rl.SkipSave {
		t.Error("expected SkipSave to be visible to the original holder")
	}
}
