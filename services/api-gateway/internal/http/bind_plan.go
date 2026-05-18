package httptransport

import (
	"errors"
	"reflect"
	"sync"
)

const (
	stepField = iota
	stepDerefNilPtrStruct
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

func bindTagOnStruct(sf reflect.StructTag) bool {
	return sf.Get("query") != "" || sf.Get("path") != "" || sf.Get("header") != ""
}

// buildBindPlan mirrors walkStruct in handler.go — same recurse / leaf distinction.
func buildBindPlan(root reflect.Type) *bindPlan {
	allowedQuery := make(map[string]struct{})
	var fields []bindField

	var walk func(rt reflect.Type, prefix []walkStep)
	walk = func(rt reflect.Type, prefix []walkStep) {
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
					walk(ft, next)
				case ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct:
					next = append(next, walkStep{kind: byte(stepDerefNilPtrStruct)})
					walk(ft.Elem(), next)
				default:
					addLeaf(sf, ft, cloneSteps(prefix), i, allowedQuery, &fields)
				}
				continue
			}
			if ft.Kind() == reflect.Struct {
				next := cloneSteps(prefix)
				next = append(next, walkStep{kind: byte(stepField), idx: i})
				walk(ft, next)
				continue
			}

			if ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct {
				if bindTagOnStruct(sf.Tag) {
					addLeaf(sf, ft, cloneSteps(prefix), i, allowedQuery, &fields)
					continue
				}

				next := cloneSteps(prefix)
				next = append(next,
					walkStep{kind: byte(stepField), idx: i},
					walkStep{kind: byte(stepDerefNilPtrStruct)},
				)
				walk(ft.Elem(), next)
				continue
			}

			addLeaf(sf, ft, cloneSteps(prefix), i, allowedQuery, &fields)
		}
	}

	walk(root, nil)

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
