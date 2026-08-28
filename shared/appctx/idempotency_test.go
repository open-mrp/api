package appctx

import (
	"context"
	"testing"
)

// --- Response metadata ---

// shared/idempotency/cached_response.go calls this on every cache hit, including on contexts that
// never went through the gateway middleware; a panic here would turn a replay into a 500.
func TestMarkIdempotencyReplayed_NoMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	MarkIdempotencyReplayed(ctx)

	if meta, ok := GetIdempotencyResponseMetadata(ctx); ok || meta != nil {
		t.Errorf("expected no metadata to be created, got (%+v, %v)", meta, ok)
	}
}

func TestMarkIdempotencyReplayed_TypedNilMetadata(t *testing.T) {
	t.Parallel()
	ctx := WithIdempotencyResponseMetadata(context.Background(), nil)

	MarkIdempotencyReplayed(ctx)

	meta, ok := GetIdempotencyResponseMetadata(ctx)
	if ok {
		t.Error("expected ok=false for an explicitly stored nil metadata pointer")
	}
	if meta != nil {
		t.Errorf("expected nil metadata, got %+v", meta)
	}
}

func TestMarkIdempotencyReplayed_NilShadowsParentMetadata(t *testing.T) {
	t.Parallel()
	parent := &IdempotencyResponseMetadata{}
	ctx := WithIdempotencyResponseMetadata(WithIdempotencyResponseMetadata(context.Background(), parent), nil)

	MarkIdempotencyReplayed(ctx)

	if parent.Replayed {
		t.Error("expected the shadowed parent metadata to be left untouched")
	}
}

func TestMarkIdempotencyReplayed_MutatesSharedPointer(t *testing.T) {
	t.Parallel()
	meta := &IdempotencyResponseMetadata{}
	ctx, cancel := context.WithCancel(WithIdempotencyResponseMetadata(context.Background(), meta))
	defer cancel()

	MarkIdempotencyReplayed(ctx)

	if !meta.Replayed {
		t.Error("expected the caller's metadata pointer to observe Replayed=true through a derived context")
	}
}

// --- Key IDs ---

func TestGetIdempotencyKeyID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
		expectOK bool
	}{
		{
			name:     "absent",
			ctx:      context.Background(),
			expectOK: false,
		},
		{
			name:     "empty string",
			ctx:      WithIdempotencyKeyID(context.Background(), ""),
			expectOK: false,
		},
		{
			name:     "wrong stored type",
			ctx:      context.WithValue(context.Background(), idempotencyKeyIDKey, 42),
			expectOK: false,
		},
		{
			name:     "present",
			ctx:      WithIdempotencyKeyID(context.Background(), "idk_123"),
			expected: "idk_123",
			expectOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := GetIdempotencyKeyID(tt.ctx)
			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got %v", tt.expectOK, ok)
			}
			if id != tt.expected {
				t.Errorf("expected id %q, got %q", tt.expected, id)
			}
		})
	}
}

func TestGetIdempotencyKey_EmptyString(t *testing.T) {
	t.Parallel()
	ctx := WithIdempotencyKey(context.Background(), "")

	if key, ok := GetIdempotencyKey(ctx); ok || key != "" {
		t.Errorf("expected an empty key to report absent, got (%q, %v)", key, ok)
	}
}
