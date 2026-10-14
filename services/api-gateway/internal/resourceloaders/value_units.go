package resourceloaders

import (
	"context"
	"reflect"
	"sync"

	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

// Value shapes (Quantity, Rate, UnitPrice) always carry their units in full, never behind an include. Presenters
// build them with unit stubs (apiresource.UnitStub); FillUnits swaps every stub reachable from a response for the
// unit it names, loading all of them in one batch.

var unitPtrType = reflect.TypeFor[*apiresource.Unit]()

// FillUnits replaces every unit stub reachable from values with the unit it names.
func FillUnits(ctx context.Context, values ...any) *apierror.APIError {
	stubs := map[string][]*apiresource.Unit{}
	seen := map[visit]bool{}
	for _, v := range values {
		collectUnitStubs(reflect.ValueOf(v), stubs, seen)
	}
	if len(stubs) == 0 {
		return nil
	}
	// Units already loaded for this request are reused: include fields fill one parent at a time.
	meta := include.GetLoadMeta(ctx)
	cached := func(id string) *apiresource.Unit {
		v, _ := meta.Get(constants.ObjectTypeUnit, id, valueUnitMetaKey)
		u, _ := v.(*apiresource.Unit)
		return u
	}
	var missing []string
	for id := range stubs {
		if cached(id) == nil {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		loaded, apiErr := LoadUnitsByID(ctx, missing...)
		if apiErr != nil {
			return apiErr
		}
		for id, u := range loaded {
			meta.Set(constants.ObjectTypeUnit, id, valueUnitMetaKey, u)
		}
	}
	attachUnits(stubs, cached)
	return nil
}

// valueUnitMetaKey caches, per request, the units FillUnits has loaded.
const valueUnitMetaKey = "value_shape_unit"

// UnitStubIDs lists the units the stubs reachable from v name, for an include that loads them.
func UnitStubIDs(v any) []string {
	stubs := map[string][]*apiresource.Unit{}
	collectUnitStubs(reflect.ValueOf(v), stubs, map[visit]bool{})
	ids := make([]string, 0, len(stubs))
	for id := range stubs {
		ids = append(ids, id)
	}
	return ids
}

// AttachUnits fills the stubs reachable from v from units an include loaded, keyed by ID.
func AttachUnits(v any, loaded map[string]any) {
	stubs := map[string][]*apiresource.Unit{}
	collectUnitStubs(reflect.ValueOf(v), stubs, map[visit]bool{})
	attachUnits(stubs, func(id string) *apiresource.Unit {
		u, _ := loaded[id].(*apiresource.Unit)
		return u
	})
}

func attachUnits(stubs map[string][]*apiresource.Unit, lookup func(string) *apiresource.Unit) {
	for id, refs := range stubs {
		unit := lookup(id)
		if unit == nil {
			continue
		}
		for _, ref := range refs {
			*ref = *unit
		}
	}
}

type visit struct {
	ptr uintptr
	typ reflect.Type
}

func collectUnitStubs(v reflect.Value, out map[string][]*apiresource.Unit, seen map[visit]bool) {
	if !v.IsValid() || !MayHoldUnit(v.Type()) {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		if v.Type() == unitPtrType {
			if u := v.Interface().(*apiresource.Unit); apiresource.IsUnitStub(u) {
				out[u.ID] = append(out[u.ID], u)
			}
			return
		}
		key := visit{ptr: v.Pointer(), typ: v.Type()}
		if seen[key] {
			return
		}
		seen[key] = true
		collectUnitStubs(v.Elem(), out, seen)
	case reflect.Interface:
		if !v.IsNil() {
			collectUnitStubs(v.Elem(), out, seen)
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).IsExported() {
				collectUnitStubs(v.Field(i), out, seen)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectUnitStubs(v.Index(i), out, seen)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			collectUnitStubs(iter.Value(), out, seen)
		}
	}
}

var holdsUnit sync.Map // reflect.Type → bool

// MayHoldUnit reports whether a value of type t can reach a *apiresource.Unit, so the walk skips everything else.
func MayHoldUnit(t reflect.Type) bool {
	if v, ok := holdsUnit.Load(t); ok {
		return v.(bool)
	}
	result := reachesUnit(t, map[reflect.Type]bool{})
	holdsUnit.Store(t, result)
	return result
}

func reachesUnit(t reflect.Type, visiting map[reflect.Type]bool) bool {
	if t == unitPtrType {
		return true
	}
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return reachesUnit(t.Elem(), visiting)
	case reflect.Struct:
		if visiting[t] {
			return false
		}
		visiting[t] = true
		defer delete(visiting, t)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.IsExported() && reachesUnit(f.Type, visiting) {
				return true
			}
		}
	}
	return false
}
