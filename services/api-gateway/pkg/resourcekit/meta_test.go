package resourcekit

import (
	"context"
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
)

const otMetaTest constants.ObjectType = "test_meta_obj"

func TestLoadMeta_SetGetString(t *testing.T) {
	ctx := WithLoadMeta(context.Background())
	m := GetLoadMeta(ctx)
	m.Set(otMetaTest, "id1", "k", "v")
	got, ok := m.GetString(otMetaTest, "id1", "k")
	if !ok || got != "v" {
		t.Errorf("got %q, %v; want v, true", got, ok)
	}
}

func TestLoadMeta_GetMissing(t *testing.T) {
	m := GetLoadMeta(context.Background())
	if _, ok := m.GetString(otMetaTest, "nope", "k"); ok {
		t.Errorf("missing key should report ok=false")
	}
}

func TestLoadMeta_WrongType(t *testing.T) {
	ctx := WithLoadMeta(context.Background())
	m := GetLoadMeta(ctx)
	m.Set(otMetaTest, "id1", "k", 42)
	if _, ok := m.GetString(otMetaTest, "id1", "k"); ok {
		t.Errorf("wrong type should report ok=false")
	}
}

func TestLoadMeta_Strings(t *testing.T) {
	ctx := WithLoadMeta(context.Background())
	m := GetLoadMeta(ctx)
	m.Set(otMetaTest, "id1", "ids", []string{"a", "b", "c"})
	got, ok := m.GetStrings(otMetaTest, "id1", "ids")
	if !ok {
		t.Fatalf("expected ok")
	}
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("got %v", got)
	}
}

func TestLoadMeta_Bool(t *testing.T) {
	ctx := WithLoadMeta(context.Background())
	m := GetLoadMeta(ctx)
	m.Set(otMetaTest, "id1", "more", true)
	got, ok := m.GetBool(otMetaTest, "id1", "more")
	if !ok || !got {
		t.Errorf("got %v, %v; want true, true", got, ok)
	}
}

func TestWithLoadMeta_Idempotent(t *testing.T) {
	ctx := WithLoadMeta(context.Background())
	m1 := GetLoadMeta(ctx)
	ctx2 := WithLoadMeta(ctx)
	m2 := GetLoadMeta(ctx2)
	if m1 != m2 {
		t.Errorf("repeated WithLoadMeta should keep the same instance")
	}
}

func TestGetLoadMeta_DetachedFallback(t *testing.T) {
	// No WithLoadMeta — Get should still return a usable (detached) instance.
	m := GetLoadMeta(context.Background())
	if m == nil {
		t.Fatalf("expected non-nil detached meta")
	}
	m.Set(otMetaTest, "id1", "k", "v")
	if got, ok := m.GetString(otMetaTest, "id1", "k"); !ok || got != "v" {
		t.Errorf("detached meta should still work in-process; got %q, %v", got, ok)
	}
}
