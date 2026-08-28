package appctx

import (
	"context"
	"testing"
)

// Unlike the string accessors elsewhere in the package, this one does not fold empty to ok=false,
// so shared/tracing/http.go and idempotency_middleware.go carry their own `!= ""` guards: an empty
// pattern would otherwise become an unattributable span name and a colliding idempotency scope key.
func TestGetRoutePattern(t *testing.T) {
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
			name:     "wrong stored type",
			ctx:      context.WithValue(context.Background(), routePatternKey, 42),
			expectOK: false,
		},
		{
			name:     "empty pattern",
			ctx:      WithRoutePattern(context.Background(), ""),
			expected: "",
			expectOK: true,
		},
		{
			name:     "pattern",
			ctx:      WithRoutePattern(context.Background(), "/v1/orders/{order_id}"),
			expected: "/v1/orders/{order_id}",
			expectOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pattern, ok := GetRoutePattern(tt.ctx)
			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got %v", tt.expectOK, ok)
			}
			if pattern != tt.expected {
				t.Errorf("expected pattern %q, got %q", tt.expected, pattern)
			}
		})
	}
}

func TestGetAllowedMethods_NilSliceReportsPresent(t *testing.T) {
	t.Parallel()
	ctx := WithAllowedMethods(context.Background(), nil)

	methods, ok := GetAllowedMethods(ctx)
	if !ok {
		t.Error("expected ok=true for an explicitly stored nil slice")
	}
	if methods != nil {
		t.Errorf("expected nil methods, got %v", methods)
	}

	if methods, ok := GetAllowedMethods(context.Background()); ok || methods != nil {
		t.Errorf("expected absent methods to report (nil, false), got (%v, %v)", methods, ok)
	}
}

func TestGetAllowedMethods_EmptySlice(t *testing.T) {
	t.Parallel()
	ctx := WithAllowedMethods(context.Background(), []string{})

	if methods, ok := GetAllowedMethods(ctx); !ok || len(methods) != 0 {
		t.Errorf("expected an empty slice to round-trip as present, got (%v, %v)", methods, ok)
	}
}

func TestGetPathParams_NilMapReportsPresent(t *testing.T) {
	t.Parallel()
	ctx := WithPathParams(context.Background(), nil)

	params, ok := GetPathParams(ctx)
	if !ok {
		t.Error("expected ok=true for an explicitly stored nil map")
	}
	if params != nil {
		t.Errorf("expected nil params, got %v", params)
	}
	if params["order_id"] != "" {
		t.Error("expected reads from the nil map to yield the zero value")
	}

	if params, ok := GetPathParams(context.Background()); ok || params != nil {
		t.Errorf("expected absent params to report (nil, false), got (%v, %v)", params, ok)
	}
}

func TestGetPathParams_SharesUnderlyingMap(t *testing.T) {
	t.Parallel()
	params := map[string]string{"order_id": "or_1"}
	ctx := WithPathParams(context.Background(), params)

	fromCtx, ok := GetPathParams(ctx)
	if !ok {
		t.Fatal("expected params in context")
	}
	fromCtx["account_id"] = "acct_1"

	if params["account_id"] != "acct_1" {
		t.Error("expected the context to carry the caller's map, not a copy")
	}
}
