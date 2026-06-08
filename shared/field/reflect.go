package field

import "reflect"

// IsClearableType reports whether typ is field.Clearable[T] or *field.Clearable[T].
func IsClearableType(typ reflect.Type) bool {
	return openAPIKind(typ) == "clearable"
}

// AssertValuePatchFields panics if typ (or any embedded struct within it) declares a
// field.Clearable[T] or field.Optional[T] as a pointer. Both must be used as values with
// json:"<name>,omitzero": encoding/json short-circuits an explicit null on a pointer field
// to a nil pointer without calling UnmarshalJSON. For Clearable that makes "clear"
// indistinguishable from "unset"; for Optional it bypasses the null rejection so an explicit
// null is silently accepted instead of erroring. This is invoked at endpoint registration so
// a pointer field fails fast at startup rather than misbehaving at request time.
func AssertValuePatchFields(typ reflect.Type) {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		return
	}
	for sf := range typ.Fields() {
		if sf.Anonymous {
			AssertValuePatchFields(sf.Type)
			continue
		}
		if sf.Type.Kind() != reflect.Pointer {
			continue
		}
		elem := sf.Type.Elem()
		switch {
		case IsClearableType(elem):
			panic("field: " + typ.String() + "." + sf.Name +
				" must be field.Clearable[T] (value), not *field.Clearable[T]; a pointer drops explicit-null clears at decode time")
		case IsOptionalType(elem):
			panic("field: " + typ.String() + "." + sf.Name +
				" must be field.Optional[T] (value), not *field.Optional[T]; a pointer bypasses the explicit-null rejection at decode time")
		}
	}
}

// IsOptionalType reports whether typ is field.Optional[T] (value type only, not *Optional[T]).
func IsOptionalType(typ reflect.Type) bool {
	return optionalStructType(typ) != nil
}

func openAPIKind(typ reflect.Type) string {
	ft := patchWrapperStructType(typ)
	if ft == nil {
		return ""
	}
	return openAPIKindOnStruct(ft)
}

func openAPIKindOnStruct(typ reflect.Type) string {
	if typ == nil || typ.Kind() != reflect.Struct {
		return ""
	}
	m, ok := typ.MethodByName("OpenAPIKind")
	if !ok || m.Type.NumOut() != 1 || m.Type.Out(0).Kind() != reflect.String {
		return ""
	}
	return reflect.New(typ).Elem().MethodByName("OpenAPIKind").Call(nil)[0].String()
}

// optionalStructType returns typ when it is field.Optional[T] (not a pointer).
func optionalStructType(typ reflect.Type) reflect.Type {
	if typ == nil || typ.Kind() == reflect.Pointer {
		return nil
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}
	if _, ok := typ.MethodByName("OpenAPIInnerType"); !ok {
		return nil
	}
	if openAPIKindOnStruct(typ) != "nullable_input" {
		return nil
	}
	return typ
}

func patchWrapperStructType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}
	if _, ok := typ.MethodByName("OpenAPIInnerType"); !ok {
		return nil
	}
	return typ
}
