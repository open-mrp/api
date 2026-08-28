package appctx

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
)

func TestGetIdentityFromContext_Absent(t *testing.T) {
	t.Parallel()
	identity, ok := GetIdentityFromContext(context.Background())
	if ok {
		t.Error("expected ok=false on a context with no identity")
	}
	if identity != nil {
		t.Errorf("expected nil identity, got %+v", identity)
	}
}

// Unlike GetRequestLog and GetHTTPResponseMetadata, this accessor's bool reports only that the
// key is present: a stored nil round-trips as (nil, true), so `ok` is not a safe guard for
// dereferencing. Callers such as registration_session_service.go read identity.Type behind
// `!ok || identity.Type != ...`, which panics for this input.
func TestGetIdentityFromContext_NilPointerReportsPresent(t *testing.T) {
	t.Parallel()
	ctx := WithIdentity(context.Background(), nil)

	identity, ok := GetIdentityFromContext(ctx)
	if !ok {
		t.Error("expected ok=true for an explicitly stored nil identity")
	}
	if identity != nil {
		t.Errorf("expected nil identity, got %+v", identity)
	}
}

func TestGetIdentityFromContext_NilShadowsParentIdentity(t *testing.T) {
	t.Parallel()
	parent := WithIdentity(context.Background(), &types.Identity{Type: types.IdentityActorTypeUser})
	child := WithIdentity(parent, nil)

	if identity, ok := GetIdentityFromContext(child); !ok || identity != nil {
		t.Errorf("expected child to shadow the parent identity with (nil, true), got (%+v, %v)", identity, ok)
	}
	if identity, ok := GetIdentityFromContext(parent); !ok || identity == nil {
		t.Errorf("expected parent identity to be unchanged, got (%+v, %v)", identity, ok)
	}
}

func TestGetIdentityFromContext_WrongStoredType(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), identityKey, types.Identity{Type: types.IdentityActorTypeUser})

	if identity, ok := GetIdentityFromContext(ctx); ok || identity != nil {
		t.Errorf("expected a non-pointer identity value to be ignored, got (%+v, %v)", identity, ok)
	}
}
