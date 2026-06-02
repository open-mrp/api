package resourcekit

import (
	"context"
	"reflect"
	"testing"
)

func TestRequestedIncludes_AbsentReturnsNil(t *testing.T) {
	if got := RequestedIncludes(context.Background()); got != nil {
		t.Errorf("expected nil for ctx without includes, got %#v", got)
	}
}

func TestRequestedIncludes_RoundTrip(t *testing.T) {
	ctx := WithRequestedIncludes(context.Background(), []string{"actor", "actor.role"})
	if got := RequestedIncludes(ctx); !reflect.DeepEqual(got, []string{"actor", "actor.role"}) {
		t.Errorf("round trip mismatch: %#v", got)
	}
}

func TestFilterIncludes_ReturnsOnlyRequestedSupportedPreservingSupportedOrder(t *testing.T) {
	ctx := WithRequestedIncludes(context.Background(), []string{"actor.role", "actor", "unsupported"})

	got := FilterIncludes(ctx, "account", "actor", "actor.role")
	// "account" not requested -> dropped; "unsupported" requested but not in the
	// backend set -> dropped; order follows the supported argument list.
	if want := []string{"actor", "actor.role"}; !reflect.DeepEqual(got, want) {
		t.Errorf("FilterIncludes = %#v, want %#v", got, want)
	}
}

func TestFilterIncludes_NothingRequestedReturnsNil(t *testing.T) {
	// No includes on the context: the backend must not be asked to enrich
	// anything, even though it supports several includes.
	if got := FilterIncludes(context.Background(), "account", "actor"); got != nil {
		t.Errorf("expected nil when nothing requested, got %#v", got)
	}
}

func TestFilterIncludes_NoSupportedReturnsNil(t *testing.T) {
	ctx := WithRequestedIncludes(context.Background(), []string{"actor"})
	if got := FilterIncludes(ctx); got != nil {
		t.Errorf("expected nil with no supported set, got %#v", got)
	}
}
