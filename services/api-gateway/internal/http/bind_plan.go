package httptransport

import (
	"errors"
	"reflect"
	"sync"
)

const (
	stepField = iota
	stepDerefNilPtrStruct

	// maxBindPlanStructDepth caps struct nesting during plan construction.
	maxBindPlanStructDepth = 32
)

type walkStep struct {
	kind byte
	idx  int // only when kind == stepField
}

type bindPlan struct {
	fields       []bindField
	allowedQuery map[string]struct{}
}

type bindField struct {
	steps    []walkStep
	tag      reflect.StructTag
	fieldTyp reflect.Type // leaf declared type (*T, Slice, ...)
	isSlice  bool
}

var bindPlanCache sync.Map // reflect.Type (non-pointer struct elem) -> *bindPlan

func planFor(dst any) (*bindPlan, error) {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil, errors.New("destination must be a non-nil pointer")
	}
	t := rv.Type().Elem()
	if t.Kind() != reflect.Struct {
		return nil, errors.New("destination must point to a struct")
	}
	if cached, ok := bindPlanCache.Load(t); ok {
		return cached.(*bindPlan), nil
	}
	plan := buildBindPlan(t)
	if actual, loaded := bindPlanCache.LoadOrStore(t, plan); loaded {
		return actual.(*bindPlan), nil
	}
	return plan, nil
}

func fieldHasBindTag(sf reflect.StructField) bool {
	tag := sf.Tag
	return tag.Get("query") != "" || tag.Get("path") != "" ||
		tag.Get("header") != "" || tag.Get("cookie") != ""
}

func bindTagOnStruct(sf reflect.StructTag) bool {
	return sf.Get("query") != "" || sf.Get("path") != "" || sf.Get("header") != ""
}

// structTypeHasBindDescendant reports whether rt contains a query/path/header/cookie bind tag on any reachable field (same recurse rules as buildBindPlan, without leaves).
func structTypeHasBindDescendant(rt reflect.Type) bool {
	visiting := make(map[reflect.Type]struct{})
	return structTypeHasBindDescendantAt(rt, visiting, 0)
}

func structTypeHasBindDescendantAt(rt reflect.Type, visiting map[reflect.Type]struct{}, depth int) bool {
	if depth > maxBindPlanStructDepth {
		return false
	}
	if _, ok := visiting[rt]; ok {
		return false
	}
	visiting[rt] = struct{}{}
	defer delete(visiting, rt)

	for sf := range rt.Fields() {
		if sf.PkgPath != "" {
			continue
		}
		if fieldHasBindTag(sf) {
			return true
		}

		ft := sf.Type
		if sf.Anonymous {
			switch {
			case ft.Kind() == reflect.Struct:
				if structTypeHasBindDescendantAt(ft, visiting, depth+1) {
					return true
				}
			case ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct:
				if structTypeHasBindDescendantAt(ft.Elem(), visiting, depth+1) {
					return true
				}
			}
			continue
		}
		if ft.Kind() == reflect.Struct {
			if structTypeHasBindDescendantAt(ft, visiting, depth+1) {
				return true
			}
			continue
		}
		if ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct {
			if bindTagOnStruct(sf.Tag) {
				return true
			}
			if structTypeHasBindDescendantAt(ft.Elem(), visiting, depth+1) {
				return true
			}
		}
	}
	return false
}

// buildBindPlan walks the request struct type and records bindable leaves. Recursion is pruned into subtrees with no bind tags, capped by maxBindPlanStructDepth, and cycle-safe via a per-path visiting set.
func buildBindPlan(root reflect.Type) *bindPlan {
	allowedQuery := make(map[string]struct{})
	var fields []bindField
	visiting := make(map[reflect.Type]struct{})

	var walk func(rt reflect.Type, prefix []walkStep, depth int)
	walk = func(rt reflect.Type, prefix []walkStep, depth int) {
		if depth > maxBindPlanStructDepth {
			return
		}
		if _, ok := visiting[rt]; ok {
			return
		}
		visiting[rt] = struct{}{}
		defer delete(visiting, rt)

		for i := 0; i < rt.NumField(); i++ {
			sf := rt.Field(i)
			if sf.PkgPath != "" {
				continue
			}

			ft := sf.Type

			anon := sf.Anonymous
			if anon {
				next := cloneSteps(prefix)
				next = append(next, walkStep{kind: byte(stepField), idx: i})

				switch {
				case ft.Kind() == reflect.Struct:
					if structTypeHasBindDescendant(ft) {
						walk(ft, next, depth+1)
					}
				case ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct:
					if structTypeHasBindDescendant(ft.Elem()) {
						next = append(next, walkStep{kind: byte(stepDerefNilPtrStruct)})
						walk(ft.Elem(), next, depth+1)
					}
				default:
					addLeaf(sf, ft, cloneSteps(prefix), i, allowedQuery, &fields)
				}
				continue
			}
			if ft.Kind() == reflect.Struct {
				if !structTypeHasBindDescendant(ft) {
					continue
				}
				next := cloneSteps(prefix)
				next = append(next, walkStep{kind: byte(stepField), idx: i})
				walk(ft, next, depth+1)
				continue
			}

			if ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct {
				if bindTagOnStruct(sf.Tag) {
					addLeaf(sf, ft, cloneSteps(prefix), i, allowedQuery, &fields)
					continue
				}
				if !structTypeHasBindDescendant(ft.Elem()) {
					continue
				}

				next := cloneSteps(prefix)
				next = append(next,
					walkStep{kind: byte(stepField), idx: i},
					walkStep{kind: byte(stepDerefNilPtrStruct)},
				)
				walk(ft.Elem(), next, depth+1)
				continue
			}

			addLeaf(sf, ft, cloneSteps(prefix), i, allowedQuery, &fields)
		}
	}

	walk(root, nil, 0)

	return &bindPlan{fields: fields, allowedQuery: allowedQuery}
}

func addLeaf(sf reflect.StructField, ft reflect.Type, prefix []walkStep, i int,
	allowed map[string]struct{}, fields *[]bindField,
) {
	next := cloneSteps(prefix)
	next = append(next, walkStep{kind: byte(stepField), idx: i})
	queryKey := sf.Tag.Get("query")
	recordQueryAllowed(allowed, queryKey, ft.Kind())
	*fields = append(*fields, bindField{
		steps:    next,
		tag:      sf.Tag,
		fieldTyp: ft,
		isSlice:  ft.Kind() == reflect.Slice,
	})
}

func recordQueryAllowed(allowed map[string]struct{}, queryKey string, kind reflect.Kind) {
	if queryKey == "" {
		return
	}
	allowed[queryKey] = struct{}{}
	if kind == reflect.Slice {
		allowed[queryKey+"[]"] = struct{}{}
	}
}

func cloneSteps(src []walkStep) []walkStep {
	if len(src) == 0 {
		return nil
	}
	out := make([]walkStep, len(src))
	copy(out, src)
	return out
}

func navigateBindField(rootStruct reflect.Value, bf bindField) (reflect.Value, bool) {
	v := rootStruct
	for _, st := range bf.steps {
		switch st.kind {
		case byte(stepField):
			if v.Kind() != reflect.Struct {
				return reflect.Value{}, false
			}
			v = v.Field(st.idx)
		case byte(stepDerefNilPtrStruct):
			if v.Kind() != reflect.Pointer {
				return reflect.Value{}, false
			}
			if v.IsNil() {
				return reflect.Value{}, false
			}
			inner := v.Elem()
			if inner.Kind() != reflect.Struct {
				return reflect.Value{}, false
			}
			v = inner
		default:
			return reflect.Value{}, false
		}
	}
	return v, true
}
