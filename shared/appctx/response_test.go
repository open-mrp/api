package appctx

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestAddCookies_NoMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	AddCookies(ctx, []*http.Cookie{{Name: "access_token", Value: "at_1"}})

	if meta, ok := GetHTTPResponseMetadata(ctx); ok || meta != nil {
		t.Errorf("expected no metadata to be created, got (%+v, %v)", meta, ok)
	}
}

func TestSetHTTPReplayed_NoMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	SetHTTPReplayed(ctx, true)

	if meta, ok := GetHTTPResponseMetadata(ctx); ok || meta != nil {
		t.Errorf("expected no metadata to be created, got (%+v, %v)", meta, ok)
	}
}

func TestAddCookies_TypedNilMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), httpResponseMetadataKey, (*HTTPResponseMetadata)(nil))

	AddCookies(ctx, []*http.Cookie{{Name: "access_token", Value: "at_1"}})
	SetHTTPReplayed(ctx, true)

	if meta, ok := GetHTTPResponseMetadata(ctx); ok || meta != nil {
		t.Errorf("expected a stored nil pointer to report absent, got (%+v, %v)", meta, ok)
	}
}

// Auth cookies are written by handlers deep below the middleware that eventually flushes them, so
// the whole mechanism is the shared pointer: appends made through any derived context must land on
// the pointer the middleware holds, in call order.
func TestAddCookies_AppendsAcrossCallsInOrder(t *testing.T) {
	t.Parallel()
	root, meta := WithHTTPResponseMetadata(context.Background())
	derived, cancel := context.WithTimeout(context.WithValue(root, handlerKey, "auth.Login"), time.Hour)
	defer cancel()

	AddCookies(root, []*http.Cookie{{Name: "access_token", Value: "at_1"}})
	AddCookies(derived, []*http.Cookie{
		{Name: "refresh_token", Value: "rt_1"},
		{Name: "session", Value: "s_1"},
	})

	expected := []string{"access_token", "refresh_token", "session"}
	if len(meta.Cookies) != len(expected) {
		t.Fatalf("expected %d cookies, got %d", len(expected), len(meta.Cookies))
	}
	for i, name := range expected {
		if meta.Cookies[i].Name != name {
			t.Errorf("expected cookie %d to be %q, got %q", i, name, meta.Cookies[i].Name)
		}
	}

	fromCtx, ok := GetHTTPResponseMetadata(derived)
	if !ok {
		t.Fatal("expected metadata to be reachable from the derived context")
	}
	if fromCtx != meta {
		t.Error("expected the derived context to carry the same metadata pointer")
	}
}

func TestSetHTTPReplayed_MutatesSharedPointer(t *testing.T) {
	t.Parallel()
	ctx, meta := WithHTTPResponseMetadata(context.Background())

	SetHTTPReplayed(ctx, true)
	if !meta.Replayed {
		t.Fatal("expected Replayed=true on the caller's pointer")
	}

	SetHTTPReplayed(ctx, false)
	if meta.Replayed {
		t.Error("expected Replayed=false after being cleared")
	}
}

func TestWithHTTPResponseMetadata_ChildScopeIsIsolated(t *testing.T) {
	t.Parallel()
	outer, outerMeta := WithHTTPResponseMetadata(context.Background())
	inner, innerMeta := WithHTTPResponseMetadata(outer)

	AddCookies(inner, []*http.Cookie{{Name: "access_token", Value: "at_1"}})

	if len(innerMeta.Cookies) != 1 {
		t.Errorf("expected 1 cookie on the inner metadata, got %d", len(innerMeta.Cookies))
	}
	if len(outerMeta.Cookies) != 0 {
		t.Errorf("expected the outer metadata to be untouched, got %d cookies", len(outerMeta.Cookies))
	}
}
