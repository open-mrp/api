package patch

import "reflect"

// IsFieldType reports whether typ is patch.Field[T] or *patch.Field[T].
func IsFieldType(typ reflect.Type) bool {
	return openAPIKind(typ) == "clearable"
}

// IsNullableType reports whether typ is patch.Nullable[T] (value type only, not *Nullable[T]).
func IsNullableType(typ reflect.Type) bool {
	return nullableStructType(typ) != nil
}

// FieldElemType returns the inner type T for patch.Field[T] or *patch.Field[T].
// Returns nil when typ is not a patch field type.
func FieldElemType(typ reflect.Type) reflect.Type {
	return elemTypeForOpenAPIWrapper(typ, "clearable")
}

// NullableElemType returns the inner type T for patch.Nullable[T].
// Returns nil when typ is not a nullable input type.
func NullableElemType(typ reflect.Type) reflect.Type {
	ft := nullableStructType(typ)
	if ft == nil {
		return nil
	}
	inner := reflect.New(ft).Elem().MethodByName("OpenAPINullableInner").Call(nil)[0].Interface().(reflect.Type)
	return inner
}

func elemTypeForOpenAPIWrapper(typ reflect.Type, wantKind string) reflect.Type {
	ft := patchWrapperStructType(typ)
	if ft == nil || openAPIKindOnStruct(ft) != wantKind {
		return nil
	}
	inner := reflect.New(ft).Elem().MethodByName("OpenAPINullableInner").Call(nil)[0].Interface().(reflect.Type)
	return inner
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

// nullableStructType returns typ when it is patch.Nullable[T] (not a pointer).
func nullableStructType(typ reflect.Type) reflect.Type {
	if typ == nil || typ.Kind() == reflect.Pointer {
		return nil
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}
	if _, ok := typ.MethodByName("OpenAPINullableInner"); !ok {
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
	if _, ok := typ.MethodByName("OpenAPINullableInner"); !ok {
		return nil
	}
	return typ
}
